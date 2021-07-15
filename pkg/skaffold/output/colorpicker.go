/*
Copyright 2021 The Skaffold Authors

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

package output

import (
	tag "github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag/util"
)

// ColorPicker associates colors with images such that container logs can be output to
// the terminal with a consistent color being used to identify individual log streams.
type ColorPicker struct {
	imageColors map[string]Color
}

// NewColorPicker creates a new ColorPicker.
func NewColorPicker() ColorPicker {
	imageColors := make(map[string]Color)

	return ColorPicker{
		imageColors: imageColors,
	}
}

// AddImage adds an image to the ColorPicker. Each image added will be paired with a color
// selected sequentially from `DefaultColorCodes`. If all colors are used, the first color
// will be used again. The formatter for the associated color will then be returned by `Pick`
// each time it is called for the artifact and can be used to write to out in that color.
func (p *ColorPicker) AddImage(image string) {
	imageName := tag.StripTag(image, false)
	if _, ok := p.imageColors[imageName]; ok {
		return
	}
	p.imageColors[imageName] = DefaultColorCodes[len(p.imageColors)%len(DefaultColorCodes)]
}

// Pick will return the color that was associated with the image when it was added to the
// ColorPicker. If no color was associated with the image, the none color will be returned,
// which will write with no formatting.
func (p *ColorPicker) Pick(image string) Color {
	if c, present := p.imageColors[tag.StripTag(image, false)]; present {
		return c
	}

	// If no mapping is found, don't add any color formatting
	return None
}
