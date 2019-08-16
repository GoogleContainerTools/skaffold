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

package util

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

type testConfig struct {
	Pipeline testPipeline
	Profiles []testConfig
}

type testPipeline struct {
	name string
}

func TestUpgradePipelines(t *testing.T) {
	from := testConfig{
		Pipeline: testPipeline{name: "main"},
		Profiles: []testConfig{
			{Pipeline: testPipeline{name: "profile-0"}},
			{Pipeline: testPipeline{name: "profile-1"}},
		},
	}
	to := testConfig{
		Profiles: []testConfig{{}, {}},
	}

	testUpgrade := func(o, n interface{}) error {
		src := o.(*testPipeline)
		dest := n.(*testPipeline)

		dest.name = src.name + "-upgraded"
		return nil
	}

	err := UpgradePipelines(&from, &to, testUpgrade)
	testutil.CheckError(t, false, err)

	if to.Pipeline.name != "main-upgraded" {
		t.Error("expected main pipeline to be upgraded")
	}
	if to.Profiles[0].Pipeline.name != "profile-0-upgraded" {
		t.Error("expected profile-0 to be upgraded")
	}
	if to.Profiles[1].Pipeline.name != "profile-1-upgraded" {
		t.Error("expected profile-1 to be upgraded")
	}
}
