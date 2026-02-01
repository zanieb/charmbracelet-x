package vt

import (
	"fmt"
	"strings"
)

// StateDump returns ANSI sequences that reproduce current screen contents,
// cursor position, and cursor state when written to a terminal of the same
// dimensions. The output uses \r\n between rows so it works correctly when
// fed to another [Emulator] (which does not have ONLCR).
func (e *Emulator) StateDump() string {
	var buf strings.Builder

	// Clear screen and home cursor
	buf.WriteString("\x1b[0m\x1b[2J\x1b[H")

	// Render screen content using the buffer's styled renderer, replacing
	// bare \n with \r\n for correct positioning in raw terminal mode.
	rendered := e.Render()
	buf.WriteString(strings.ReplaceAll(rendered, "\n", "\r\n"))

	// Position cursor
	pos := e.CursorPosition()
	buf.WriteString(fmt.Sprintf("\x1b[%d;%dH", pos.Y+1, pos.X+1))

	// Cursor visibility
	if e.scr.cur.Hidden {
		buf.WriteString("\x1b[?25l")
	}

	// Cursor style
	writeCursorStyle(&buf, e.scr.cur.Style, e.scr.cur.Steady)

	return buf.String()
}

// writeCursorStyle emits the DECSCUSR sequence for cursor shape.
func writeCursorStyle(buf *strings.Builder, style CursorStyle, steady bool) {
	// DECSCUSR: 0=default, 1=blinking block, 2=steady block,
	// 3=blinking underline, 4=steady underline, 5=blinking bar, 6=steady bar
	var n int
	switch style {
	case CursorBlock:
		n = 1
	case CursorUnderline:
		n = 3
	case CursorBar:
		n = 5
	}
	if steady {
		n++
	}
	if n > 0 {
		buf.WriteString(fmt.Sprintf("\x1b[%d q", n))
	}
}
