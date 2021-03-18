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

package config

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestValidateModes(t *testing.T) {
	tests := []struct {
		modes     []string
		shouldErr bool
	}{
		{modes: nil, shouldErr: false},
		{modes: []string{"compat"}, shouldErr: true},
		{modes: []string{"off"}, shouldErr: false},
		{modes: []string{"true"}, shouldErr: false},
		{modes: []string{"false"}, shouldErr: false},
		{modes: []string{"TRUE"}, shouldErr: false},
		{modes: []string{"FALSE"}, shouldErr: false},
		{modes: []string{"1"}, shouldErr: false},
		{modes: []string{"0"}, shouldErr: false},
		{modes: []string{"user", "debug", "pods", "services"}, shouldErr: false},
		{modes: []string{"user", "true", "debug"}, shouldErr: true},
		{modes: []string{"off", "debug"}, shouldErr: true},
		{modes: []string{"pods", "false"}, shouldErr: true},
	}
	for _, test := range tests {
		testutil.Run(t, fmt.Sprintf("%v", test.modes), func(t *testutil.T) {
			err := validateModes(test.modes)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestPortForwardOptions_Enabled(t *testing.T) {
	tests := []struct {
		modes   []string
		enabled bool
	}{
		{modes: nil, enabled: false},
		{modes: []string{"off"}, enabled: false},
		{modes: []string{"true"}, enabled: true},
		{modes: []string{"false"}, enabled: false},
		{modes: []string{"TRUE"}, enabled: true},
		{modes: []string{"FALSE"}, enabled: false},
		{modes: []string{"1"}, enabled: true},
		{modes: []string{"0"}, enabled: false},
		{modes: []string{"user", "debug", "pods", "services"}, enabled: true},
		{modes: []string{"user", "debug"}, enabled: true},
	}
	for _, test := range tests {
		testutil.Run(t, fmt.Sprintf("modes: %v", test.modes), func(t *testutil.T) {
			opts := PortForwardOptions{}
			t.CheckError(false, opts.Replace(test.modes))
			result := opts.Enabled()
			t.CheckDeepEqual(test.enabled, result)
		})
	}
}

func TestPortForwardOptions_Forwards(t *testing.T) {
	tests := []struct {
		runModes        []RunMode // if empty, then all of Deploy, Dev, Debug, Run
		modes           []string
		forwardUser     bool
		forwardServices bool
		forwardPods     bool
		forwardDebug    bool
	}{
		{modes: nil},               // all disabled
		{modes: []string{"off"}},   // all disabled
		{modes: []string{"false"}}, // all disabled
		{modes: []string{"true"}, runModes: []RunMode{RunModes.Deploy, RunModes.Run, RunModes.Dev}, forwardUser: true, forwardServices: true},
		{modes: []string{"true"}, runModes: []RunMode{RunModes.Debug}, forwardUser: true, forwardServices: true, forwardDebug: true},

		{modes: []string{"user", "debug", "pods", "services"}, forwardUser: true, forwardServices: true, forwardPods: true, forwardDebug: true},
		{modes: []string{"user"}, forwardUser: true},
		{modes: []string{"services"}, forwardServices: true},
		{modes: []string{"pods"}, forwardPods: true},
		{modes: []string{"debug"}, forwardDebug: true},
	}
	for _, test := range tests {
		runModes := test.runModes
		if len(runModes) == 0 {
			runModes = []RunMode{RunModes.Deploy, RunModes.Run, RunModes.Dev, RunModes.Debug}
		}
		for _, rm := range runModes {
			testutil.Run(t, fmt.Sprintf("modes: %v runMode: %v", test.modes, rm), func(t *testutil.T) {
				opts := PortForwardOptions{}
				t.CheckError(false, opts.Replace(test.modes))
				t.CheckDeepEqual(test.forwardUser, opts.ForwardUser(rm))
				t.CheckDeepEqual(test.forwardServices, opts.ForwardServices(rm))
				t.CheckDeepEqual(test.forwardPods, opts.ForwardPods(rm))
				t.CheckDeepEqual(test.forwardDebug, opts.ForwardDebug(rm))
			})
		}
	}
}
