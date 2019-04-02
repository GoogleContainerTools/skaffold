/*
Copyright 2019 The Skaffold Authors

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

package kubernetes

import (
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	v1 "k8s.io/api/core/v1"
)

var colorCodes = []color.Color{
	color.LightRed,
	color.LightGreen,
	color.LightYellow,
	color.LightBlue,
	color.LightPurple,
	color.Red,
	color.Green,
	color.Yellow,
	color.Blue,
	color.Purple,
	color.Cyan,
}

// ColorPicker is used to associate colors for with pods so that the container logs
// can be output to the terminal with a consistent color being used to identify logs
// from each pod.
type ColorPicker interface {
	Pick(pod *v1.Pod) color.Color
	Register(pod *v1.Pod)
}

type colorPicker struct {
	sync.RWMutex
	podColors map[string]color.Color
}

// NewColorPicker creates a new ColorPicker. The formatter for the associated color will
// then be returned by `Pick` each time it is called for the pod and can be used to write
// to out in that color.
func NewColorPicker() ColorPicker {
	return &colorPicker{podColors: make(map[string]color.Color)}
}

// Pick will return the color that was associated with pod when `Register` was called.
// If no color was associated with the pod, the none color will be returned, which will
// write with no formatting.
func (p *colorPicker) Pick(pod *v1.Pod) color.Color {
	p.RLock()
	c, present := p.podColors[pod.GetName()]
	p.RUnlock()

	if present {
		return c
	}

	// If no mapping is found, don't add any color formatting
	return color.None
}

// Register associates a color with the given pod by its name. For each registered pod,
// a color will be selected sequentially from `colorCodes`. If all colors are used,
// the first color will be used again.
func (p *colorPicker) Register(pod *v1.Pod) {
	// assume that pods are already registered most of the time
	p.RLock()
	_, ok := p.podColors[pod.GetName()]
	p.RUnlock()

	if ok {
		return
	}

	p.Lock()
	p.podColors[pod.GetName()] = colorCodes[len(p.podColors)%len(colorCodes)]
	p.Unlock()
}
