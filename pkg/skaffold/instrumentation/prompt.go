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

package instrumentation

import (
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
)

const Prompt = `To help improve the quality of this product, we collect anonymized usage data for details on what is tracked and how we use this data visit <https://skaffold.dev/docs/telemetry/>. This data is handled in accordance with our privacy policy <https://policies.google.com/privacy/>

You may choose to opt out of this collection by running the following command:
	skaffold config set --global collect-metrics false
`

var (
	// for testing
	isStdOut     = color.IsStdout
	updateConfig = config.UpdateGlobalCollectMetrics
	getConfig    = config.GetConfigForCurrentKubectx
	setStatus    = SetOnlineStatus
)

func ShouldDisplayMetricsPrompt(configfile string) bool {
	cfg, err := getConfig(configfile)
	if err != nil {
		return false
	}
	if cfg == nil || cfg.CollectMetrics == nil {
		return true
	}
	shouldExportMetrics = *cfg.CollectMetrics
	setStatus()
	return false
}

func DisplayMetricsPrompt(configFile string, out io.Writer) error {
	if isStdOut(out) {
		color.Green.Fprintf(out, Prompt)
		return updateConfig(configFile, true)
	}
	return nil
}
