/*
Copyright 2020 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	if len(style) == 0 {
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
