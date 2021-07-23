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

package logger

import (
	"fmt"
	"io"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker/tracker"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
)

type DockerLogFormatter struct {
	colorPicker   output.ColorPicker
	tracker       *tracker.ContainerTracker
	lock          sync.Mutex
	isMuted       func() bool
	id            string
	containerName string
	prefix        string
}

func NewDockerLogFormatter(colorPicker output.ColorPicker, tracker *tracker.ContainerTracker, isMuted func() bool, id string) *DockerLogFormatter {
	return &DockerLogFormatter{
		colorPicker:   colorPicker,
		tracker:       tracker,
		isMuted:       isMuted,
		id:            id,
		containerName: tracker.ArtifactForContainer(id).ImageName,
		prefix:        prefix(tracker.ArtifactForContainer(id).ImageName),
	}
}

func (d *DockerLogFormatter) Name() string { return d.prefix }

func (d *DockerLogFormatter) PrintLine(out io.Writer, line string) {
	if d.isMuted() {
		return
	}
	d.lock.Lock()
	defer d.lock.Unlock()

	formattedPrefix := d.prefix
	// if our original prefix wasn't empty, append a space to the line
	if d.prefix != "" {
		formattedPrefix = fmt.Sprintf("%s ", formattedPrefix)
	}
	if output.IsColorable(out) {
		formattedPrefix = d.color().Sprintf("%s", formattedPrefix)
	}
	formattedLine := fmt.Sprintf("%s%s", formattedPrefix, line)
	eventV2.ApplicationLog("", d.containerName, formattedPrefix, line, formattedLine)
	fmt.Fprint(out, formattedLine)
}

func (d *DockerLogFormatter) color() output.Color {
	return d.colorPicker.Pick(d.tracker.ArtifactForContainer(d.id).Tag)
}

func prefix(containerName string) string {
	if containerName == "" { // should only happen in testing
		return ""
	}
	return fmt.Sprintf("[%s]", containerName)
}
