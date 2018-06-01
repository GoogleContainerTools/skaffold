/*
Copyright 2018 The Skaffold Authors

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
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"k8s.io/api/core/v1"
)

type color int

var (
	colorCodeWhite = color(97)
	colorCodes     = []color{
		91, // lightRed
		92, // lightGreen
		93, // lightYellow
		94, // lightBlue
		95, // lightPurple
		96, // lightCyan
		31, // red
		32, // green
		33, // yellow
		34, // blue
		35, // purple
		36, // cyan
	}
)

func (c color) Sprint(text string) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", c, text)
}

// ColorPicker is used to pick colors for pods and container logs.
type ColorPicker interface {
	Pick(pod *v1.Pod) color
}

type colorPicker struct {
	imageColors map[string]color
}

// NewColorPicker creates a new ColorPicker.
func NewColorPicker(artifacts []*v1alpha2.Artifact) ColorPicker {
	colors := map[string]color{}
	for i, artifact := range artifacts {
		colors[artifact.ImageName] = colorCodes[i%len(colorCodes)]
	}

	return &colorPicker{
		imageColors: colors,
	}
}

func (p *colorPicker) Pick(pod *v1.Pod) color {
	for _, container := range pod.Spec.Containers {
		if color, present := p.imageColors[stripTag(container.Image)]; present {
			return color
		}
	}

	return colorCodeWhite
}

func stripTag(image string) string {
	if !strings.Contains(image, ":") {
		return image
	}

	return strings.SplitN(image, ":", 2)[0]
}
