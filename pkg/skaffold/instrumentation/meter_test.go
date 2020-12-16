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

package instrumentation

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/statik"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestOfflineExportMetrics(t *testing.T) {
	startTime, _ := time.Parse(time.ANSIC, "Mon Jan 2 15:04:05 -0700 MST 2006")
	validMeter := skaffoldMeter{
		Command:        "build",
		BuildArtifacts: 5,
		Version:        "vTest.0",
		Arch:           "test arch",
		OS:             "test os",
		Builders:       map[string]int{"docker": 1, "buildpacks": 1},
		EnumFlags:      map[string]*pflag.Flag{"test": {Name: "test", Shorthand: "t"}},
		StartTime:      startTime,
		Duration:       time.Minute,
	}
	validMeterBytes, _ := json.Marshal(validMeter)
	fs := &testutil.FakeFileSystem{
		Files: map[string][]byte{
			"/keys.json": []byte(`{
				"client_id": "test_id",
				"client_secret": "test_secret",
				"project_id": "test_project",
				"refresh_token": "test_token",
				"type": "authorized_user"
			}`),
		},
	}

	tests := []struct {
		name                string
		meter               skaffoldMeter
		savedMetrics        []byte
		shouldSkip          bool
		shouldFailUnmarshal bool
	}{
		{
			name:       "skips exporting if command is not set",
			shouldSkip: true,
		},
		{
			name:  "saves meter to a new file",
			meter: validMeter,
		},
		{
			name: "meter is appended to previously saved metrics",
			meter: skaffoldMeter{
				Command:      "dev",
				Version:      "vTest.1",
				Arch:         "test arch 2",
				OS:           "test os 2",
				PlatformType: "test platform",
				Deployers:    []string{"test helm", "test kpt"},
				SyncType:     map[string]bool{"manual": true},
				EnumFlags:    map[string]*pflag.Flag{"test_run": {Name: "test_run", Shorthand: "r"}},
				ErrorCode:    proto.StatusCode_BUILD_CANCELLED,
				StartTime:    startTime.Add(time.Hour * 24 * 30),
				Duration:     time.Minute,
			},
			savedMetrics: validMeterBytes,
		},
		{
			name:                "meter does not re-save invalid metrics",
			meter:               validMeter,
			savedMetrics:        []byte("[{\"Command\":\"run\", Invalid\": 10000000000010202301230}]"),
			shouldFailUnmarshal: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.Override(&isOnline, false)
			t.Override(&statik.FS, func() (http.FileSystem, error) { return fs, nil })
			filename := "metrics"
			tmp := t.NewTempDir()
			var savedMetrics []skaffoldMeter
			_ = json.Unmarshal(test.savedMetrics, &savedMetrics)

			if len(test.savedMetrics) > 0 {
				err := ioutil.WriteFile(tmp.Path(filename), test.savedMetrics, 0666)
				if err != nil {
					t.Error(err)
				}
			}
			_ = exportMetrics(context.Background(), tmp.Path(filename), test.meter)

			if test.shouldSkip {
				_, err := os.Stat(filename)
				t.CheckDeepEqual(true, os.IsNotExist(err))
			} else {
				b, _ := ioutil.ReadFile(tmp.Path(filename))
				var actual []skaffoldMeter
				_ = json.Unmarshal(b, &actual)
				expected := append(savedMetrics, test.meter)
				if test.shouldFailUnmarshal {
					expected = []skaffoldMeter{test.meter}
				}
				t.CheckDeepEqual(expected, actual)
			}
		})
	}
}
