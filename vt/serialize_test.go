package vt

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
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

func TestStateDumpRestoresModes(t *testing.T) {
	w, h := 40, 10

	src := newTestTerminal(t, w, h)
	src.WriteString("prompt$ ")

	// Enable bracketed paste and mouse tracking (as a shell would).
	src.WriteString("\x1b[?2004h") // bracketed paste
	src.WriteString("\x1b[?1000h") // mouse normal
	src.WriteString("\x1b[?1006h") // mouse SGR extended
	src.WriteString("\x1b[?1004h") // focus events
	src.WriteString("\x1b[?1h")    // cursor keys (application mode)

	dump := src.StateDump()

	// Apply dump to a fresh emulator (simulates client terminal after DECSTR).
	dst := newTestTerminal(t, w, h)
	dst.WriteString(dump)

	// Verify modes were restored.
	tests := []struct {
		name string
		mode ansi.Mode
	}{
		{"BracketedPaste", ansi.ModeBracketedPaste},
		{"MouseNormal", ansi.ModeMouseNormal},
		{"MouseExtSgr", ansi.ModeMouseExtSgr},
		{"FocusEvent", ansi.ModeFocusEvent},
		{"CursorKeys", ansi.ModeCursorKeys},
	}
	for _, tt := range tests {
		if !dst.IsModeSet(tt.mode) {
			t.Errorf("mode %s not restored after StateDump round-trip", tt.name)
		}
	}
}

func TestStateDumpSkipsDefaultModes(t *testing.T) {
	src := newTestTerminal(t, 20, 5)
	src.WriteString("hello")
	// Don't enable any non-default modes.

	dump := src.StateDump()

	// The dump should not contain any DECSET sequences for modes that are
	// already at their default values.
	// Bracketed paste default is off (?2004), mouse default is off (?1000).
	if contains(dump, "\x1b[?2004h") {
		t.Error("StateDump should not emit DECSET for default-off bracketed paste")
	}
	if contains(dump, "\x1b[?1000h") {
		t.Error("StateDump should not emit DECSET for default-off mouse")
	}
}

func TestStateDumpRestoresKittyKeyboard(t *testing.T) {
	src := newTestTerminal(t, 40, 10)
	src.WriteString("prompt$ ")

	// Push kitty keyboard flags (disambiguate + report events).
	src.WriteString("\x1b[>3u")

	if src.KittyKeyboardFlags() != 3 {
		t.Fatalf("expected kitty flags=3, got %d", src.KittyKeyboardFlags())
	}

	dump := src.StateDump()

	// Apply dump to a fresh emulator.
	dst := newTestTerminal(t, 40, 10)
	dst.WriteString(dump)

	// The kitty keyboard flags should be restored.
	if dst.KittyKeyboardFlags() != 3 {
		t.Fatalf("expected kitty flags=3 after StateDump, got %d", dst.KittyKeyboardFlags())
	}
}

func TestKittyKeyboardPushPop(t *testing.T) {
	em := newTestTerminal(t, 40, 10)

	// Initially empty stack.
	if em.KittyKeyboardFlags() != 0 {
		t.Fatalf("expected initial flags=0, got %d", em.KittyKeyboardFlags())
	}

	// Push flags=1.
	em.WriteString("\x1b[>1u")
	if em.KittyKeyboardFlags() != 1 {
		t.Fatalf("after push 1: expected flags=1, got %d", em.KittyKeyboardFlags())
	}

	// Push flags=3.
	em.WriteString("\x1b[>3u")
	if em.KittyKeyboardFlags() != 3 {
		t.Fatalf("after push 3: expected flags=3, got %d", em.KittyKeyboardFlags())
	}

	// Pop 1.
	em.WriteString("\x1b[<1u")
	if em.KittyKeyboardFlags() != 1 {
		t.Fatalf("after pop 1: expected flags=1, got %d", em.KittyKeyboardFlags())
	}

	// Pop remaining.
	em.WriteString("\x1b[<1u")
	if em.KittyKeyboardFlags() != 0 {
		t.Fatalf("after pop all: expected flags=0, got %d", em.KittyKeyboardFlags())
	}
}

func TestStateDumpNoKittyKeyboardWhenDisabled(t *testing.T) {
	src := newTestTerminal(t, 20, 5)
	src.WriteString("hello")
	// Don't enable kitty keyboard.

	dump := src.StateDump()

	// Should not contain CSI > u sequence.
	if contains(dump, "\x1b[>") {
		t.Error("StateDump should not emit kitty keyboard push when disabled")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
