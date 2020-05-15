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

package binpack

import (
	"fmt"
	"sort"
)

type timing struct {
	name string
	time float64
}

// we'll need to regenerate this list time to time
var timings = []timing{
	{"TestRun", 183.68},
	{"TestDebug", 100.51},
	{"TestDevAPITriggers", 81.05},
	{"TestBuild", 68.59},
	{"TestDevAutoSync", 68.45},
	{"TestDiagnose", 35.67},
	{"TestDevPortForward", 35.44},
	{"TestCancellableBuild", 30.83},
	{"TestCancellableDeploy", 29.04},
	{"TestDevSync", 25.69},
	{"TestDevNotification", 22.95},
	{"TestDevPortForwardDeletePod", 16.27},
	{"TestRunUnstableChecked", 15.32},
	{"TestEventLogHTTP", 14.19},
	{"TestDebugEventsRPC_StatusCheck", 13.68},
	{"TestRunTailPod", 12.96},
	{"TestInitManifestGeneration", 8.82},
	{"TestCacheAPITriggers", 8.75},
	{"TestGetStateRPC", 8.50},
	{"TestInitKustomize", 8.45},
	{"TestDebugEventsRPC_NoStatusCheck", 7.90},
	{"TestDevSyncAPITrigger", 7.62},
	{"TestEventsRPC", 7.61},
	{"TestGetStateHTTP", 7.51},
	{"TestInitCompose", 6.88},
	{"TestFix", 6.49},
	{"TestRunIdempotent", 5.37},
	{"TestRunPortForward", 5.31},
	{"TestDeploy", 4.81},
	{"TestRunTailDeployment", 4.70},
	{"TestPortForward", 4.28},
	{"TestDev_WithKubecontextOverride", 3.86},
	{"TestDeployTail", 2.51},
	{"TestRunUnstableNotChecked", 1.97},
	{"TestExpectedBuildFailures", 1.42},
	{"TestKubectlRender", 0.94},
	{"TestDeployWithInCorrectConfig", 0.45},
	{"TestGeneratePipeline", 0.23},
	{"TestCredits", 0.07},
	{"TestSetGlobalDefaultRepo", 0.07},
	{"TestSchema", 0.07},
	{"TestSetDefaultRepoForContext", 0.06},
	{"TestFailToSetUnrecognizedValue", 0.04},
	{"TestConfigListForContext", 0.03},
	{"TestConfigListForAll", 0.03},
}

type bin struct {
	timings []timing
	total   float64
}

func (b *bin) Add(t timing) {
	b.timings = append(b.timings, t)
	b.total += t.time
}

func (b *bin) String() string {
	return fmt.Sprintf("total: %f, size: %d", b.total, len(b.timings))
}

func Partitions() (map[string]int, int) {
	// binpack with first fit decreasing
	sort.Slice(timings, func(i, j int) bool {
		return timings[i].time > timings[j].time
	})
	maxTime := 240.0

	result := make(map[string]int)

	var bins []bin
	for _, timing := range timings {
		fit := false
		for i := range bins {
			if bins[i].total+timing.time < maxTime {
				bins[i].Add(timing)
				result[timing.name] = i
				fit = true
				break
			}
		}
		if !fit {
			newBin := bin{}
			bins = append(bins, newBin)
			newBin.Add(timing)
			result[timing.name] = len(bins) - 1
		}
	}
	for _, b := range bins {
		fmt.Printf("%s\n", b.String())
	}
	return result, len(bins) - 1
}
