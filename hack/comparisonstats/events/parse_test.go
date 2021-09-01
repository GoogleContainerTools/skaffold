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

package events

import (
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

var validEventsFile = `{"timestamp":"2021-08-11T19:19:41.711480752Z","event":{"metaEvent":{"entry":"Starting Skaffold: \u0026{Version:v1.29.0 ConfigVersion:skaffold/v2beta20 GitVersion: GitCommit:39371bb996a3c39c3d4fa8749cabe173c5f45b3a BuildDate:2021-08-02T17:52:01Z GoVersion:go1.14.14 Compiler:gc Platform:linux/amd64 User:}","metadata":{"build":{"numberOfArtifacts":1,"builders":[{"type":"DOCKER","count":1}],"type":"LOCAL"},"deploy":{"deployers":[{"type":"HELM","count":1}],"cluster":"MINIKUBE"}}}}}
{"timestamp":"2021-08-11T19:19:41.756663171Z","event":{"devLoopEvent":{"status":"In Progress"}},"entry":"Update initiated"}
{"timestamp":"2021-08-11T19:19:41.763416940Z","event":{"buildEvent":{"artifact":"skaffold-helm","status":"In Progress"}},"entry":"Build started for artifact skaffold-helm"}
{"timestamp":"2021-08-11T19:19:45.685909133Z","event":{"buildEvent":{"artifact":"skaffold-helm","status":"Complete"}},"entry":"Build completed for artifact skaffold-helm"}
{"timestamp":"2021-08-11T19:19:45.686277380Z","event":{"deployEvent":{"status":"In Progress"}},"entry":"Deploy started"}
{"timestamp":"2021-08-11T19:19:46.624504850Z","event":{"deployEvent":{"status":"Complete"}},"entry":"Deploy completed"}
{"timestamp":"2021-08-11T19:19:46.624550647Z","event":{"statusCheckEvent":{"status":"Started"}},"entry":"Status check started"}
{"timestamp":"2021-08-11T19:19:48.719236104Z","event":{"resourceStatusCheckEvent":{"resource":"deployment/skaffold-helm","status":"Succeeded","message":"Succeeded","statusCode":"STATUSCHECK_SUCCESS"}},"entry":"Resource deployment/skaffold-helm status completed successfully"}
{"timestamp":"2021-08-11T19:19:48.719307379Z","event":{"statusCheckEvent":{"status":"Succeeded"}},"entry":"Status check succeeded"}
{"timestamp":"2021-08-11T19:19:48.734465633Z","event":{"devLoopEvent":{"status":"Succeeded"}},"entry":"Update succeeded"}
{"timestamp":"2021-08-11T19:19:51.740854617Z","event":{"devLoopEvent":{"iteration":1,"status":"In Progress"}},"entry":"Update initiated"}
{"timestamp":"2021-08-11T19:19:51.744239521Z","event":{"buildEvent":{"artifact":"skaffold-helm","status":"In Progress"}},"entry":"Build started for artifact skaffold-helm"}
{"timestamp":"2021-08-11T19:19:55.757451860Z","event":{"buildEvent":{"artifact":"skaffold-helm","status":"Complete"}},"entry":"Build completed for artifact skaffold-helm"}
{"timestamp":"2021-08-11T19:19:55.757928417Z","event":{"deployEvent":{"status":"In Progress"}},"entry":"Deploy started"}
{"timestamp":"2021-08-11T19:19:56.728808748Z","event":{"deployEvent":{"status":"Complete"}},"entry":"Deploy completed"}
{"timestamp":"2021-08-11T19:19:56.728840707Z","event":{"statusCheckEvent":{"status":"Started"}},"entry":"Status check started"}
{"timestamp":"2021-08-11T19:20:00.823570232Z","event":{"resourceStatusCheckEvent":{"resource":"deployment/skaffold-helm","status":"Succeeded","message":"Succeeded","statusCode":"STATUSCHECK_SUCCESS"}},"entry":"Resource deployment/skaffold-helm status completed successfully"}
{"timestamp":"2021-08-11T19:20:00.823640653Z","event":{"statusCheckEvent":{"status":"Succeeded"}},"entry":"Status check succeeded"}
{"timestamp":"2021-08-11T19:20:00.823857159Z","event":{"devLoopEvent":{"iteration":1,"status":"Succeeded"}},"entry":"Update succeeded"}
`

var invalidEventsFile = `invalid-events-file`

func TestParseEventDuration(t *testing.T) {
	tests := []struct {
		description      string
		eventsFileText   string
		shouldErr        bool
		expectedDevLoops int
	}{
		{
			description:      "valid events file",
			eventsFileText:   validEventsFile,
			expectedDevLoops: 1,
		},
		{
			description:    "invalid events file",
			eventsFileText: invalidEventsFile,
			shouldErr:      true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fp := t.TempFile("TestParseEventDuration-", []byte(test.eventsFileText))
			devLoopTimes, err := ParseEventDuration(context.Background(), fp)
			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				if len(devLoopTimes.InnerBuildTimes) != len(devLoopTimes.InnerDeployTimes) && len(devLoopTimes.InnerBuildTimes) != len(devLoopTimes.InnerStatusCheckTimes) {
					t.Fatalf("expected devLoopTimes arrays to have same lengths")
				}
				t.CheckDeepEqual(len(devLoopTimes.InnerBuildTimes), test.expectedDevLoops)
			}
		})
	}
}
