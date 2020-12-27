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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"

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

func TestExportMetrics(t *testing.T) {
	startTime, _ := time.Parse(time.ANSIC, "Mon Jan 2 15:04:05 -0700 MST 2006")
	buildMeter := skaffoldMeter{
		Command:        "build",
		BuildArtifacts: 5,
		Version:        "vTest.0",
		Arch:           "test arch",
		OS:             "test os",
		Deployers:      []string{"test kubectl"},
		Builders:       map[string]int{"docker": 1, "buildpacks": 1},
		EnumFlags:      map[string]string{"test": "test_value"},
		StartTime:      startTime,
		Duration:       time.Minute,
	}
	devMeter := skaffoldMeter{
		Command:       "dev",
		Version:       "vTest.1",
		Arch:          "test arch 2",
		OS:            "test os 2",
		PlatformType:  "test platform",
		Deployers:     []string{"test helm", "test kpt"},
		SyncType:      map[string]bool{"manual": true},
		EnumFlags:     map[string]string{"test_run": "test_run_value"},
		Builders:      map[string]int{"kustomize": 3, "buildpacks": 2},
		DevIterations: []devIteration{{"sync", 0}, {"build", 400}, {"build", 0}, {"sync", 200}, {"deploy", 0}},
		ErrorCode:     proto.StatusCode_BUILD_CANCELLED,
		StartTime:     startTime.Add(time.Hour * 24 * 30),
		Duration:      time.Minute * 2,
	}
	devBuildBytes, _ := json.Marshal([]skaffoldMeter{buildMeter, devMeter})
	fs := &testutil.FakeFileSystem{
		Files: map[string][]byte{
			"/keys.json": []byte(testKey),
		},
	}

	tests := []struct {
		name                string
		meter               skaffoldMeter
		savedMetrics        []byte
		shouldFailUnmarshal bool
		isOnline            bool
	}{
		{
			name:  "saves meter to a new file",
			meter: buildMeter,
		},
		{
			name:         "meter is appended to previously saved metrics",
			meter:        devMeter,
			savedMetrics: devBuildBytes,
		},
		{
			name:                "meter does not re-save invalid metrics",
			meter:               buildMeter,
			savedMetrics:        []byte("[{\"Command\":\"run\", Invalid\": 10000000000010202301230}]"),
			shouldFailUnmarshal: true,
		},
		{
			name:     "test creating builder otel metrics",
			meter:    buildMeter,
			isOnline: true,
		},
		{
			name:     "test creating dev otel metrics",
			meter:    devMeter,
			isOnline: true,
		},
		{
			name:         "test otel metrics include offline metrics",
			meter:        devMeter,
			savedMetrics: devBuildBytes,
			isOnline:     true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			var actual []skaffoldMeter
			var savedMetrics []skaffoldMeter

			tmp := t.NewTempDir()
			filename := "metrics"
			openTelFilename := "otel_metrics"

			t.Override(&statik.FS, func() (http.FileSystem, error) { return fs, nil })
			t.Override(&isOnline, test.isOnline)

			if test.isOnline {
				tmpFile, err := os.OpenFile(tmp.Path(openTelFilename), os.O_RDWR|os.O_CREATE, os.ModePerm)
				if err != nil {
					t.Error(err)
				}
				t.Override(&initExporter, func() (*push.Controller, error) {
					return stdout.InstallNewPipeline([]stdout.Option{
						stdout.WithQuantiles([]float64{0.5}),
						stdout.WithPrettyPrint(),
						stdout.WithWriter(tmpFile),
					}, nil)
				})
			}
			if len(test.savedMetrics) > 0 {
				json.Unmarshal(test.savedMetrics, &savedMetrics)
				tmp.Write(filename, string(test.savedMetrics))
			}

			_ = exportMetrics(context.Background(), tmp.Path(filename), test.meter)
			b, err := ioutil.ReadFile(tmp.Path(filename))

			if !test.isOnline {
				_ = json.Unmarshal(b, &actual)
				expected := append(savedMetrics, test.meter)
				if test.shouldFailUnmarshal {
					expected = []skaffoldMeter{test.meter}
				}
				t.CheckDeepEqual(expected, actual)
			} else {
				t.CheckDeepEqual(true, os.IsNotExist(err))
				b, err := ioutil.ReadFile(tmp.Path(openTelFilename))
				t.CheckError(false, err)
				checkOutput(t, append(savedMetrics, test.meter), b)
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

func checkOutput(t *testutil.T, meters []skaffoldMeter, b []byte) {
	osCount := make(map[interface{}]int)
	versionCount := make(map[interface{}]int)
	archCount := make(map[interface{}]int)
	durationCount := make(map[interface{}]int)
	commandCount := make(map[interface{}]int)
	errorCount := make(map[interface{}]int)
	builders := make(map[interface{}]int)
	devIterations := make(map[interface{}]int)
	deployers := make(map[interface{}]int)

	testMaps := []map[interface{}]int{
		osCount, versionCount, archCount, durationCount, commandCount, errorCount, builders, devIterations, deployers}

	for _, meter := range meters {
		osCount[meter.OS]++
		versionCount[meter.Version]++
		durationCount[fmt.Sprintf("%s:%f", meter.Command, meter.Duration.Seconds())]++
		archCount[meter.Arch]++
		commandCount[meter.Command]++
		errorCount[meter.ErrorCode]++

		if doesBuild.Contains(meter.Command) {
			for k, v := range meter.Builders {
				builders[k] += v
			}
		}
		if meter.Command == "dev" {
			for _, devI := range meter.DevIterations {
				devIterations[devI]++
			}
		}
		if doesDeploy.Contains(meter.Command) {
			for _, d := range meter.Deployers {
				deployers[d]++
			}
		}
	}

	var lines []*line
	json.Unmarshal(b, &lines)

	for _, l := range lines {
		l.initLine()
		switch l.Name {
		case "launches":
			archCount[l.Labels["arch"]]--
			osCount[l.Labels["os"]]--
			versionCount[l.Labels["version"]]--
			e, _ := strconv.Atoi(l.Labels["error"])
			if e == int(proto.StatusCode_OK) {
				errorCount[proto.StatusCode(e)]--
			}
		case "launch/duration":
			durationCount[fmt.Sprintf("%s:%f", l.Labels["command"], l.value().(float64))]--
		case "artifacts":
			builders[l.Labels["builder"]] -= int(l.value().(float64)) - 1
		case "builders":
			builders[l.Labels["builder"]]--
		case "deployer":
			deployers[l.Labels["deployer"]]--
		case "dev/iterations":
			e, _ := strconv.Atoi(l.Labels["error"])
			devIterations[devIteration{l.Labels["intent"], proto.StatusCode(e)}]--
		case "errors":
			e, _ := strconv.Atoi(l.Labels["error"])
			errorCount[proto.StatusCode(e)]--
		default:
			switch {
			case meteredCommands.Contains(l.Name):
				commandCount[l.Name]--
			default:
				t.Error("unexpected metric with name", l.Name)
			}
		}
	}

	for _, m := range testMaps {
		for n, v := range m {
			t.Logf("Checking %s", n)
			t.CheckDeepEqual(0, v)
		}
	}
}

// Derived from go.opentelemetry.io/otel/exporters/stdout/metric.go
type line struct {
	Name      string      `json:"Name"`
	Count     interface{} `json:"Count,omitempty"`
	Quantiles []quantile  `json:"Quantiles,omitempty"`
	Labels    map[string]string
}

type quantile struct {
	Quantile interface{} `json:"Quantile"`
	Value    interface{} `json:"Value"`
}

func (l *line) initLine() {
	l.Labels = make(map[string]string)
	leftBracket := strings.Index(l.Name, "{")
	rightBracket := strings.Index(l.Name, "}")

	labels := strings.Split(l.Name[leftBracket+1:rightBracket], ",")[1:]
	for _, lbl := range labels {
		ll := strings.Split(lbl, "=")
		l.Labels[ll[0]] = ll[1]
	}
	l.Name = l.Name[:leftBracket]
}

func (l *line) value() interface{} {
	return l.Quantiles[0].Value
}
