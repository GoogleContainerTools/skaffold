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
	"strings"

	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
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
}

type colorPicker struct {
	imageColors map[string]color.Color
}

// NewColorPicker creates a new ColorPicker. For each artifact, a color will be selected
// sequentially from `colorCodes`. If all colors are used, the first color will be used
// again. The formatter for the associated color will then be returned by `Pick` each
// time it is called for the artifact and can be used to write to out in that color.
func NewColorPicker(imageNames []string) ColorPicker {
	imageColors := make(map[string]color.Color)

	for i, imageName := range imageNames {
		imageColors[stripTag(imageName)] = colorCodes[i%len(colorCodes)]
	}

	return &colorPicker{
		imageColors: imageColors,
	}
}

// Pick will return the color that was associated with pod when `NewColorPicker` was called.
// If no color was associated with the pod, the none color will be returned, which will
// write with no formatting.
func (p *colorPicker) Pick(pod *v1.Pod) color.Color {
	for _, container := range pod.Spec.Containers {
		if c, present := p.imageColors[stripTag(container.Image)]; present {
			return c
		}
	}

	// If no mapping is found, don't add any color formatting
	return color.None
}

func stripTag(image string) string {
	if !strings.Contains(image, ":") {
		return image
	}

	return strings.SplitN(image, ":", 2)[0]
}
