/*
Copyright 2022 The Skaffold Authors

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

package deploy

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

// CheckHelmInitStruct test if helm init struct
func CheckHelmInitStruct(t *testutil.T, expected []latest.HelmRelease, actual []latest.HelmRelease) {
	expectedC, evf := convertToHelmInit(expected)
	actualC, avf := convertToHelmInit(actual)
	t.CheckElementsMatch(expectedC, actualC)
	t.CheckMapsMatch(evf, avf)
}

type helmInit struct {
	name        string
	chartPath   string
	remoteChart string
}

func convertToHelmInit(releases []latest.HelmRelease) ([]helmInit, map[string][]string) {
	hs := make([]helmInit, len(releases))
	vf := map[string][]string{}
	for i, r := range releases {
		hs[i] = helmInit{
			name:        r.Name,
			remoteChart: r.RemoteChart,
			chartPath:   r.ChartPath,
		}
		vf[r.Name] = r.ValuesFiles
	}
	return hs, vf
}
