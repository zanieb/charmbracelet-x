package vt

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
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

	// Restore terminal modes that affect client-side input/output behavior.
	// Without this, a reconnecting client's terminal will be in default mode
	// while the remote application expects modes like bracketed paste or
	// mouse tracking to be enabled, causing input corruption.
	writeModes(&buf, e.modes)

	// Restore kitty keyboard protocol state. The protocol uses a stack
	// (CSI > flags u), not a DEC mode. If the remote application pushed
	// flags, we need to re-push them on the client terminal so keystrokes
	// are encoded correctly.
	if flags := e.KittyKeyboardFlags(); flags > 0 {
		buf.WriteString(ansi.PushKittyKeyboard(flags))
	}

	return buf.String()
}

// modesToRestore lists terminal modes that should be restored on reconnect.
// These are modes that affect how the client terminal processes input or output
// and whose state must match what the remote application expects.
//
// We exclude modes that are purely internal to the emulator (origin, margins)
// or that are handled separately (cursor visibility, alt screen).
var modesToRestore = map[ansi.Mode]ansi.ModeSetting{
	// Input modes — if the remote app enabled these, the client terminal
	// must also have them enabled or keystrokes will be misencoded.
	ansi.ModeCursorKeys:   ansi.ModeReset, // ?1  — cursor key mode (normal vs application)
	ansi.ModeNumericKeypad: ansi.ModeReset, // ?66 — numeric keypad mode

	// Mouse tracking — if enabled, the client terminal must know.
	ansi.ModeMouseX10:         ansi.ModeReset, // ?9
	ansi.ModeMouseNormal:      ansi.ModeReset, // ?1000
	ansi.ModeMouseHighlight:   ansi.ModeReset, // ?1001
	ansi.ModeMouseButtonEvent: ansi.ModeReset, // ?1002
	ansi.ModeMouseAnyEvent:    ansi.ModeReset, // ?1003
	ansi.ModeMouseExtSgr:      ansi.ModeReset, // ?1006

	// Focus events — if enabled, client terminal reports focus in/out.
	ansi.ModeFocusEvent: ansi.ModeReset, // ?1004

	// Bracketed paste — affects how pasted text is sent.
	ansi.ModeBracketedPaste: ansi.ModeReset, // ?2004
}

// writeModes emits DECSET sequences for modes that differ from their defaults.
func writeModes(buf *strings.Builder, modes ansi.Modes) {
	for mode, defaultSetting := range modesToRestore {
		current, ok := modes[mode]
		if !ok {
			continue
		}
		if current == defaultSetting {
			continue
		}
		// Mode is non-default — emit the appropriate set/reset sequence.
		if current.IsSet() {
			buf.WriteString(ansi.SetMode(mode))
		}
	}
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
