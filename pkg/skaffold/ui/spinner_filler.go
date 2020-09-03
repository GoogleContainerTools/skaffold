package ui

import (
	"io"

	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
)

type spinnerFiller struct {
	frames []string
	count  uint
}

// NewSpinnerFiller constructs a mpb.BarFiller, to be used with *Progress.Add(...) method.
func NewSpinnerFiller(style []string) mpb.BarFiller {
	if style == nil || len(style) == 0 {
		style = mpb.DefaultSpinnerStyle
	}
	filler := &spinnerFiller{
		frames: style,
	}
	return filler
}

// To fulfill the implementation of mpb.BarFiller interface
func (s *spinnerFiller) Fill(w io.Writer, _ int, _ decor.Statistics) {
	frame := s.frames[s.count%uint(len(s.frames))]

	io.WriteString(w, frame)
	s.count++
}
