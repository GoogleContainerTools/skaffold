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

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/statik"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var testKey = `{
	"client_id": "test_id",
	"client_secret": "test_secret",
	"project_id": "test_project",
	"refresh_token": "test_token",
	"type": "authorized_user"
}`

func TestOfflineExportMetrics(t *testing.T) {
	startTime, _ := time.Parse(time.ANSIC, "Mon Jan 2 15:04:05 -0700 MST 2006")
	validMeter := skaffoldMeter{
		Command:        "build",
		BuildArtifacts: 5,
		Version:        "vTest.0",
		Arch:           "test arch",
		OS:             "test os",
		Builders:       map[string]int{"docker": 1, "buildpacks": 1},
		EnumFlags:      map[string]string{"test": "test_value"},
		StartTime:      startTime,
		Duration:       time.Minute,
	}
	validMeterBytes, _ := json.Marshal(validMeter)
	fs := &testutil.FakeFileSystem{
		Files: map[string][]byte{
			"/keys.json": []byte(testKey),
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
				EnumFlags:    map[string]string{"test_run": "test_run_value"},
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

func TestInitCloudMonitoring(t *testing.T) {
	tests := []struct {
		name        string
		fileSystem  *testutil.FakeFileSystem
		pusherIsNil bool
		shouldError bool
	}{
		{
			name: "if key present pusher is not nil",
			fileSystem: &testutil.FakeFileSystem{
				Files: map[string][]byte{"/keys.json": []byte(testKey)},
			},
		},
		{
			name: "key not present returns nill err",
			fileSystem: &testutil.FakeFileSystem{
				Files: map[string][]byte{},
			},
			pusherIsNil: true,
		},
		{
			name: "credentials without project_id returns an error",
			fileSystem: &testutil.FakeFileSystem{
				Files: map[string][]byte{
					"/keys.json": []byte(`{
						"client_id": "test_id",
						"client_secret": "test_secret",
						"refresh_token": "test_token",
						"type": "authorized_user"
					}`,
					)},
			},
			shouldError: true,
			pusherIsNil: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.Override(&statik.FS, func() (http.FileSystem, error) { return test.fileSystem, nil })

			p, err := initCloudMonitoringExporterMetrics()

			t.CheckErrorAndDeepEqual(test.shouldError, err, test.pusherIsNil || test.shouldError, p == nil)
		})
	}
}
