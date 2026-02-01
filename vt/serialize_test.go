package vt

import (
	"testing"
)

func TestStateDumpRoundTrip(t *testing.T) {
	w, h := 40, 10

	// Create source emulator with styled content
	src := newTestTerminal(t, w, h)
	src.WriteString("Hello World")
	src.WriteString("\x1b[2;1H") // move to row 2
	src.WriteString("\x1b[31mRed Text\x1b[0m")
	src.WriteString("\x1b[3;1H") // move to row 3
	src.WriteString("\x1b[1;4mBold Underline\x1b[0m")
	src.WriteString("\x1b[5;10H") // position cursor

	dump := src.StateDump()

	// Apply dump to a fresh emulator
	dst := newTestTerminal(t, w, h)
	dst.WriteString(dump)

	// Verify plain text matches
	if src.String() != dst.String() {
		t.Errorf("text mismatch:\nsrc=%q\ndst=%q", src.String(), dst.String())
	}

	// Verify cursor position
	srcPos := src.CursorPosition()
	dstPos := dst.CursorPosition()
	if srcPos != dstPos {
		t.Errorf("cursor position mismatch: src=%v dst=%v", srcPos, dstPos)
	}

	// Verify styled cells match
	for y := range h {
		for x := range w {
			sc := src.CellAt(x, y)
			dc := dst.CellAt(x, y)
			if sc == nil && dc == nil {
				continue
			}
			if sc == nil || dc == nil {
				t.Errorf("cell nil mismatch at (%d,%d)", x, y)
				continue
			}
			if !sc.Equal(dc) {
				t.Errorf("cell mismatch at (%d,%d): src=%+v dst=%+v", x, y, sc, dc)
			}
		}
	}
}

func TestStateDumpPlainText(t *testing.T) {
	src := newTestTerminal(t, 20, 3)
	src.WriteString("ABC")
	src.WriteString("\x1b[2;1HDEF")
	src.WriteString("\x1b[3;1HGHI")

	dump := src.StateDump()
	dst := newTestTerminal(t, 20, 3)
	dst.WriteString(dump)

	if src.String() != dst.String() {
		t.Errorf("plain text mismatch:\nsrc=%q\ndst=%q", src.String(), dst.String())
	}
}

func TestStateDumpEmptyScreen(t *testing.T) {
	src := newTestTerminal(t, 10, 5)
	dump := src.StateDump()
	dst := newTestTerminal(t, 10, 5)
	dst.WriteString(dump)

	srcPos := src.CursorPosition()
	dstPos := dst.CursorPosition()
	if srcPos != dstPos {
		t.Errorf("cursor mismatch on empty screen: src=%v dst=%v", srcPos, dstPos)
	}
}

func TestStateDumpWithStyles(t *testing.T) {
	w, h := 30, 5
	src := newTestTerminal(t, w, h)

	// Write with various styles
	src.WriteString("\x1b[31;1mBold Red\x1b[0m Normal \x1b[42mGreen BG\x1b[0m")

	dump := src.StateDump()
	dst := newTestTerminal(t, w, h)
	dst.WriteString(dump)

	// Check styled cells match
	for x := range w {
		sc := src.CellAt(x, 0)
		dc := dst.CellAt(x, 0)
		if sc == nil && dc == nil {
			continue
		}
		if sc == nil || dc == nil {
			t.Errorf("cell nil mismatch at (%d,0)", x)
			continue
		}
		if !sc.Style.Equal(&dc.Style) {
			t.Errorf("style mismatch at (%d,0): src=%+v dst=%+v", x, sc.Style, dc.Style)
		}
	}
}

func TestStateDumpHiddenCursor(t *testing.T) {
	src := newTestTerminal(t, 20, 5)
	src.WriteString("text")
	src.WriteString("\x1b[?25l") // hide cursor

	dump := src.StateDump()
	dst := newTestTerminal(t, 20, 5)
	dst.WriteString(dump)

	if !dst.scr.cur.Hidden {
		t.Error("expected hidden cursor after StateDump round-trip")
	}
}
