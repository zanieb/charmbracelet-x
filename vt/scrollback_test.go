package vt

import (
	"fmt"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

func makeLine(s string) uv.Line {
	line := make(uv.Line, len(s))
	for i, r := range s {
		line[i] = uv.Cell{Content: string(r), Width: 1}
	}
	return line
}

func lineString(line uv.Line) string {
	var s string
	for _, c := range line {
		if c.Content == "" {
			s += " "
		} else {
			s += c.Content
		}
	}
	return s
}

func TestScrollbackPushAndLine(t *testing.T) {
	sb := NewScrollback(5)
	for i := range 3 {
		sb.Push(makeLine(fmt.Sprintf("line%d", i)))
	}

	if sb.Len() != 3 {
		t.Fatalf("expected Len()=3, got %d", sb.Len())
	}
	if got := lineString(sb.Line(0)); got != "line0" {
		t.Fatalf("Line(0) = %q, want %q", got, "line0")
	}
	if got := lineString(sb.Line(2)); got != "line2" {
		t.Fatalf("Line(2) = %q, want %q", got, "line2")
	}
}

func TestScrollbackOverflow(t *testing.T) {
	sb := NewScrollback(3)
	for i := range 7 {
		sb.Push(makeLine(fmt.Sprintf("L%d", i)))
	}

	if sb.Len() != 3 {
		t.Fatalf("expected Len()=3, got %d", sb.Len())
	}
	// Oldest should be L4, newest L6
	if got := lineString(sb.Line(0)); got != "L4" {
		t.Fatalf("Line(0) = %q, want %q", got, "L4")
	}
	if got := lineString(sb.Line(2)); got != "L6" {
		t.Fatalf("Line(2) = %q, want %q", got, "L6")
	}
}

func TestScrollbackReset(t *testing.T) {
	sb := NewScrollback(5)
	sb.Push(makeLine("hello"))
	sb.Reset()

	if sb.Len() != 0 {
		t.Fatalf("expected Len()=0 after Reset, got %d", sb.Len())
	}
	if sb.Cap() != 5 {
		t.Fatalf("expected Cap()=5 after Reset, got %d", sb.Cap())
	}
}

func TestScrollbackLines(t *testing.T) {
	sb := NewScrollback(3)
	for i := range 5 {
		sb.Push(makeLine(fmt.Sprintf("%d", i)))
	}

	lines := sb.Lines()
	if len(lines) != 3 {
		t.Fatalf("Lines() len = %d, want 3", len(lines))
	}
	if got := lineString(lines[0]); got != "2" {
		t.Fatalf("Lines()[0] = %q, want %q", got, "2")
	}
	if got := lineString(lines[2]); got != "4" {
		t.Fatalf("Lines()[2] = %q, want %q", got, "4")
	}
}

func TestScrollbackClone(t *testing.T) {
	sb := NewScrollback(3)
	orig := makeLine("abc")
	sb.Push(orig)

	// Modify original — should not affect scrollback (copy semantics)
	orig[0].Content = "X"
	if got := lineString(sb.Line(0)); got != "abc" {
		t.Fatalf("scrollback was mutated: got %q, want %q", got, "abc")
	}
}

func TestScrollbackZeroCap(t *testing.T) {
	sb := NewScrollback(0)
	sb.Push(makeLine("test"))
	if sb.Len() != 0 {
		t.Fatalf("expected Len()=0 for zero-cap scrollback, got %d", sb.Len())
	}
}

func TestScrollbackIntegration(t *testing.T) {
	term := newTestTerminal(t, 20, 5)
	term.SetScrollbackSize(100)

	// Write more lines than the screen height to trigger scrolling
	for i := range 10 {
		term.WriteString(fmt.Sprintf("line %02d\n", i))
	}

	sb := term.Scrollback()
	if sb == nil {
		t.Fatal("expected scrollback to be non-nil")
	}
	if sb.Len() == 0 {
		t.Fatal("expected scrollback to have captured lines")
	}

	// The first scrolled-off line should be "line 00"
	first := lineString(sb.Line(0))
	if len(first) < 7 || first[:7] != "line 00" {
		t.Fatalf("first scrollback line = %q, want prefix %q", first, "line 00")
	}
}

func TestScrollbackCSI3J(t *testing.T) {
	term := newTestTerminal(t, 20, 5)
	term.SetScrollbackSize(100)

	for i := range 10 {
		term.WriteString(fmt.Sprintf("line %d\n", i))
	}

	if term.Scrollback().Len() == 0 {
		t.Fatal("expected scrollback lines before CSI 3J")
	}

	// CSI 3 J should clear scrollback
	term.WriteString("\x1b[3J")
	if term.Scrollback().Len() != 0 {
		t.Fatalf("expected scrollback cleared after CSI 3J, got %d", term.Scrollback().Len())
	}
}
