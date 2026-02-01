package vt

import uv "github.com/charmbracelet/ultraviolet"

// Scrollback is a fixed-capacity ring buffer of terminal lines that have
// scrolled off the top of the main screen.
type Scrollback struct {
	lines []uv.Line // pre-allocated ring buffer
	head  int       // next write position
	len   int       // number of stored lines (≤ cap)
}

// NewScrollback creates a new scrollback buffer with the given capacity.
// A capacity of 0 effectively disables scrollback.
func NewScrollback(cap int) *Scrollback {
	if cap < 0 {
		cap = 0
	}
	return &Scrollback{
		lines: make([]uv.Line, cap),
	}
}

// Push appends a cloned copy of the line to the ring buffer.
// If the buffer is full, the oldest line is overwritten.
func (s *Scrollback) Push(line uv.Line) {
	if len(s.lines) == 0 {
		return
	}
	s.lines[s.head] = cloneLine(line)
	s.head = (s.head + 1) % len(s.lines)
	if s.len < len(s.lines) {
		s.len++
	}
}

// Len returns the number of lines currently stored.
func (s *Scrollback) Len() int {
	return s.len
}

// Cap returns the maximum capacity of the scrollback buffer.
func (s *Scrollback) Cap() int {
	return len(s.lines)
}

// Line returns the line at index i, where 0 is the oldest and Len()-1 is
// the newest. Returns nil if i is out of range.
func (s *Scrollback) Line(i int) uv.Line {
	if i < 0 || i >= s.len {
		return nil
	}
	idx := (s.head - s.len + i) % len(s.lines)
	if idx < 0 {
		idx += len(s.lines)
	}
	return s.lines[idx]
}

// Lines returns all stored lines from oldest to newest as a new slice.
func (s *Scrollback) Lines() []uv.Line {
	out := make([]uv.Line, s.len)
	for i := range s.len {
		out[i] = s.Line(i)
	}
	return out
}

// Reset clears all stored lines without deallocating the underlying storage.
func (s *Scrollback) Reset() {
	for i := range s.lines {
		s.lines[i] = nil
	}
	s.head = 0
	s.len = 0
}

// cloneLine creates a deep copy of a terminal line.
func cloneLine(line uv.Line) uv.Line {
	if line == nil {
		return nil
	}
	out := make(uv.Line, len(line))
	copy(out, line)
	return out
}
