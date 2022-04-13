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

	v1 "k8s.io/api/core/v1"

	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	tagutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag/util"
)

type Formatter func(pod v1.Pod, containerStatus v1.ContainerStatus, isMuted func() bool) log.Formatter

type kubernetesLogFormatter struct {
	colorPicker     output.ColorPicker
	prefix          string
	JSONParseConfig latest.JSONParseConfig

	pod       *v1.Pod
	container v1.ContainerStatus

	lock    sync.Mutex
	isMuted func() bool
}

func newKubernetesLogFormatter(config Config, colorPicker output.ColorPicker, isMuted func() bool, pod *v1.Pod, container v1.ContainerStatus) *kubernetesLogFormatter {
	return &kubernetesLogFormatter{
		colorPicker:     colorPicker,
		prefix:          prefix(config, pod, container),
		JSONParseConfig: config.JSONParseConfig(),
		pod:             pod,
		container:       container,
		isMuted:         isMuted,
	}
}

func (k *kubernetesLogFormatter) Name() string { return k.prefix }

func (k *kubernetesLogFormatter) PrintLine(out io.Writer, line string) {
	if k.isMuted() {
		return
	}
	formattedPrefix := k.prefix
	if output.IsColorable(out) {
		formattedPrefix = k.color().Sprintf("%s", k.prefix)
		// if our original prefix was empty, don't prepend a space to the line,
		// but keep the color prefix we just added.
		if k.prefix != "" {
			formattedPrefix = fmt.Sprintf("%s ", formattedPrefix)
		}
	}

	line = log.ParseJSON(k.JSONParseConfig, line)
	formattedLine := fmt.Sprintf("%s%s", formattedPrefix, line)
	eventV2.ApplicationLog(k.pod.Name, k.container.Name, formattedPrefix, line, formattedLine)

	k.lock.Lock()
	defer k.lock.Unlock()
	fmt.Fprint(out, formattedLine)
}

func (k *kubernetesLogFormatter) color() output.Color {
	for _, container := range k.pod.Spec.Containers {
		if c := k.colorPicker.Pick(container.Image); c != output.None {
			return c
		}
	}

	// If no mapping is found, don't add any color formatting
	return output.None
}

func prefix(config Config, pod *v1.Pod, container v1.ContainerStatus) string {
	var c latest.Pipeline
	var present bool
	for _, container := range pod.Spec.Containers {
		if c, present = config.PipelineForImage(tagutil.StripTag(container.Image, false)); present {
			break
		}
	}
	if !present {
		c = config.DefaultPipeline()
	}
	switch c.Deploy.Logs.Prefix {
	case "auto":
		if pod.Name != container.Name {
			return podAndContainerPrefix(pod, container)
		}
		return autoPrefix(pod, container)
	case "container":
		return containerPrefix(container)
	case "podAndContainer":
		return podAndContainerPrefix(pod, container)
	case "none":
		return ""
	default:
		panic("unsupported prefix: " + c.Deploy.Logs.Prefix)
	}
}

func autoPrefix(pod *v1.Pod, container v1.ContainerStatus) string {
	if pod.Name != container.Name {
		return fmt.Sprintf("[%s %s]", pod.Name, container.Name)
	}
	return fmt.Sprintf("[%s]", container.Name)
}

func containerPrefix(container v1.ContainerStatus) string {
	return fmt.Sprintf("[%s]", container.Name)
}

func podAndContainerPrefix(pod *v1.Pod, container v1.ContainerStatus) string {
	return fmt.Sprintf("[%s %s]", pod.Name, container.Name)
}
