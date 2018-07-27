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
	"io"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	v1 "k8s.io/api/core/v1"
)

var colorCodes = []output.Color{
	output.ColorCodeLightRed,
	output.ColorCodeLightGreen,
	output.ColorCodeLightYellow,
	output.ColorCodeLightBlue,
	output.ColorCodeLightPurple,
	output.ColorCodeRed,
	output.ColorCodeGreen,
	output.ColorCodeYellow,
	output.ColorCodeBlue,
	output.ColorCodePurple,
	output.ColorCodeCyan,
}

// ColorPicker is used to associate colors for with pods so that the container logs
// can be output to the terminal with a consistent color being used to identify logs
// from each pod.
type ColorPicker interface {
	Pick(pod *v1.Pod) output.ColorFormatter
}

type colorPicker struct {
	imageFormatters map[string]output.ColorFormatter
	out             io.Writer
}

// NewColorPicker creates a new ColorPicker. For each artfact, a color will be selected
// sequentially from `colorCodes`. If all colors are used, the first color will be used
// again. The formatter for the associated color will then be returned by `Pick` each
// time it is called for the artifact and can be used to write to out in that color.
func NewColorPicker(out io.Writer, artifacts []*v1alpha2.Artifact) ColorPicker {
	formatters := map[string]output.ColorFormatter{}
	for i, artifact := range artifacts {
		c := colorCodes[i%len(colorCodes)]
		formatters[artifact.ImageName] = output.NewColorFormatter(out, c)
	}

	return &colorPicker{
		imageFormatters: formatters,
	}
}

// Pick will return the color formatter that was associated with pod when
// `NewColorPicker` was called. If no color was associated with the pod,
// the default color (white) will be returned.
func (p *colorPicker) Pick(pod *v1.Pod) output.ColorFormatter {
	for _, container := range pod.Spec.Containers {
		if f, present := p.imageFormatters[stripTag(container.Image)]; present {
			return f
		}
	}

	// If no mapping is found, don't add any color formatting
	return output.NewColorFormatter(p.out, output.ColorCodeNone)
}

func stripTag(image string) string {
	if !strings.Contains(image, ":") {
		return image
	}

	return strings.SplitN(image, ":", 2)[0]
}
