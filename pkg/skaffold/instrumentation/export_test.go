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
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/GoogleContainerTools/skaffold/v2/fs"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
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
		Command:                      "build",
		BuildArtifacts:               5,
		Version:                      "vTest.0",
		Arch:                         "test arch",
		OS:                           "test os",
		ConfigCount:                  1,
		Deployers:                    []string{"test kubectl"},
		Builders:                     map[string]int{"docker": 1, "buildpacks": 1},
		BuildWithPlatforms:           map[string]int{"docker": 1},
		BuildDependencies:            map[string]int{"docker": 1},
		EnumFlags:                    map[string]string{"test": "test_value"},
		DeployNodePlatforms:          "linux/amd64",
		CliBuildTargetPlatforms:      "linux/amd64;linux/arm64",
		ResolvedBuildTargetPlatforms: []string{"linux/amd64"},
		StartTime:                    startTime,
		Duration:                     time.Minute,
	}
	devMeter := skaffoldMeter{
		Command:                      "dev",
		Version:                      "vTest.1",
		Arch:                         "test arch 2",
		OS:                           "test os 2",
		ConfigCount:                  2,
		PlatformType:                 "test platform 1:test platform 2",
		Deployers:                    []string{"test helm", "test kpt"},
		SyncType:                     map[string]bool{"manual": true},
		EnumFlags:                    map[string]string{"test_run": "test_run_value"},
		Builders:                     map[string]int{"kustomize": 3, "buildpacks": 2},
		BuildWithPlatforms:           map[string]int{"kustomize": 3},
		BuildDependencies:            map[string]int{"docker": 1},
		DeployNodePlatforms:          "linux/amd64",
		CliBuildTargetPlatforms:      "linux/amd64;linux/arm64",
		ResolvedBuildTargetPlatforms: []string{"linux/amd64"},
		HelmReleasesCount:            2,
		DevIterations:                []devIteration{{"sync", 0}, {"build", 400}, {"build", 0}, {"sync", 200}, {"deploy", 0}},
		ResourceFilters:              []resourceFilter{{"schema", "allow"}, {"schema", "deny"}, {"cli-flag", "allow"}, {"cli-flag", "deny"}},
		ErrorCode:                    proto.StatusCode_BUILD_CANCELLED,
		StartTime:                    startTime.Add(time.Hour * 24 * 30),
		Duration:                     time.Minute * 2,
	}
	debugMeter := skaffoldMeter{
		Command:                      "debug",
		Version:                      "vTest.2",
		Arch:                         "test arch 1",
		OS:                           "test os 2",
		ConfigCount:                  2,
		PlatformType:                 "test platform",
		Deployers:                    []string{"test helm", "test kpt"},
		SyncType:                     map[string]bool{"manual": true, "sync": true},
		EnumFlags:                    map[string]string{"test_run": "test_run_value"},
		Builders:                     map[string]int{"jib": 3, "buildpacks": 2},
		DeployNodePlatforms:          "linux/amd64",
		CliBuildTargetPlatforms:      "linux/amd64;linux/arm64",
		ResolvedBuildTargetPlatforms: []string{"linux/amd64"},
		DevIterations:                []devIteration{{"build", 104}, {"build", 0}, {"sync", 0}, {"deploy", 1014}},
		ErrorCode:                    proto.StatusCode_BUILD_CANCELLED,
		StartTime:                    startTime.Add(time.Hour * 24 * 10),
		Duration:                     time.Minute * 4,
	}
	metersBytes, _ := json.Marshal([]skaffoldMeter{buildMeter, devMeter, debugMeter})
	fakeFS := testutil.FakeFileSystem{
		Files: map[string][]byte{
			"assets/secrets_generated/keys.json": []byte(testKey),
		},
	}

	tests := []struct {
		name                string
		meter               skaffoldMeter
		savedMetrics        []byte
		expectedMeter       skaffoldMeter
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
			savedMetrics: metersBytes,
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
			name:     "test creating debug otel metrics",
			meter:    debugMeter,
			isOnline: true,
		},
		{
			name:         "test otel metrics include offline metrics",
			meter:        devMeter,
			savedMetrics: metersBytes,
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

			fs.AssetsFS = fakeFS
			t.Override(&isOnline, test.isOnline)

			if test.isOnline {
				tmpFile, err := os.OpenFile(tmp.Path(openTelFilename), os.O_RDWR|os.O_CREATE, os.ModePerm)
				if err != nil {
					t.Error(err)
				}
				t.Override(&initExporter, func() (sdkmetric.Exporter, error) {
					enc := json.NewEncoder(tmpFile)
					return stdoutmetric.New(stdoutmetric.WithEncoder(enc))
				})
			}
			if len(test.savedMetrics) > 0 {
				json.Unmarshal(test.savedMetrics, &savedMetrics)
				tmp.Write(filename, string(test.savedMetrics))
			}

			_ = exportMetrics(context.Background(), tmp.Path(filename), test.meter)
			b, err := os.ReadFile(tmp.Path(filename))

			if !test.isOnline {
				_ = json.Unmarshal(b, &actual)
				expected := append(savedMetrics, test.meter)
				if test.shouldFailUnmarshal {
					expected = []skaffoldMeter{test.meter}
				}
				t.CheckDeepEqual(expected, actual)
			} else {
				t.CheckDeepEqual(true, os.IsNotExist(err))
				b, err := os.ReadFile(tmp.Path(openTelFilename))
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
				Files: map[string][]byte{"assets/secrets_generated/keys.json": []byte(testKey)},
			},
		},
		{
			name: "key not present returns nil err",
			fileSystem: &testutil.FakeFileSystem{
				Files: map[string][]byte{},
			},
			pusherIsNil: true,
		},
		{
			name: "credentials without project_id returns an error",
			fileSystem: &testutil.FakeFileSystem{
				Files: map[string][]byte{
					"assets/secrets_generated/keys.json": []byte(`{
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
			fs.AssetsFS = test.fileSystem

			p, err := initCloudMonitoringExporterMetrics()

			t.CheckErrorAndDeepEqual(test.shouldError, err, test.pusherIsNil || test.shouldError, p == nil)
		})
	}
}

func TestUserMetricReported(t *testing.T) {
	fakeFS := &testutil.FakeFileSystem{
		Files: map[string][]byte{
			"/secret/keys.json": []byte(testKey),
		},
	}

	tests := []struct {
		name         string
		meter        skaffoldMeter
		expectedUser string
	}{
		{
			name: "test meter with user intellij",
			meter: skaffoldMeter{
				Command: "build",
				Version: "vTest.0",
				Arch:    "test arch",
				OS:      "test os",
				User:    "intellij",
			},
			expectedUser: "intellij",
		},
		{
			name: "test meter with user vsc",
			meter: skaffoldMeter{
				Command: "build",
				Version: "vTest.0",
				Arch:    "test arch",
				OS:      "test os",
				User:    "vsc",
			},
			expectedUser: "vsc",
		},
		{
			name: "test meter with user gcloud",
			meter: skaffoldMeter{
				Command: "build",
				Version: "vTest.0",
				Arch:    "test arch",
				OS:      "test os",
				User:    "gcloud",
			},
			expectedUser: "gcloud",
		},
		{
			name: "test meter with user cloud-deploy",
			meter: skaffoldMeter{
				Command: "build",
				Version: "vTest.0",
				Arch:    "test arch",
				OS:      "test os",
				User:    "cloud-deploy",
			},
			expectedUser: "cloud-deploy",
		},
		{
			name: "test meter with valid user pattern cloud-deploy/staging",
			meter: skaffoldMeter{
				Command: "build",
				Version: "vTest.0",
				Arch:    "test arch",
				OS:      "test os",
				User:    "cloud-deploy/staging",
			},
			expectedUser: "cloud-deploy/staging",
		},
		{
			name: "test meter with invalid user pattern cloud-deploy/",
			meter: skaffoldMeter{
				Command: "build",
				Version: "vTest.0",
				Arch:    "test arch",
				OS:      "test os",
				User:    "cloud-deploy/",
			},
			expectedUser: "",
		},
		{
			name: "test meter with invalid user pattern cloud-deploy|staging",
			meter: skaffoldMeter{
				Command: "build",
				Version: "vTest.0",
				Arch:    "test arch",
				OS:      "test os",
				User:    "cloud-deploy|staging",
			},
			expectedUser: "",
		},
		{
			name: "test meter with no user set",
			meter: skaffoldMeter{
				Command: "build",
				Version: "vTest.0",
				Arch:    "test arch",
				OS:      "test os",
			},
		},
		{
			name: "test meter with user set to any value then allowed",
			meter: skaffoldMeter{
				Command: "build",
				Version: "vTest.0",
				Arch:    "test arch",
				OS:      "test os",
				User:    "random",
			},
			expectedUser: "",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			tmp := t.NewTempDir()
			filename := "metrics"
			openTelFilename := "otel_metrics"

			fs.AssetsFS = fakeFS
			t.Override(&isOnline, true)
			tmpFile, err := os.OpenFile(tmp.Path(openTelFilename), os.O_RDWR|os.O_CREATE, os.ModePerm)
			if err != nil {
				t.Error(err)
			}
			t.Override(&initExporter, func() (sdkmetric.Exporter, error) {
				enc := json.NewEncoder(tmpFile)
				return stdoutmetric.New(stdoutmetric.WithEncoder(enc))
			})

			_ = exportMetrics(context.Background(), tmp.Path(filename), test.meter)

			b, err := os.ReadFile(tmp.Path(openTelFilename))
			t.CheckNoError(err)
			checkUser(t, test.expectedUser, b)
		})
	}
}

// todo refactor
func checkOutput(t *testutil.T, meters []skaffoldMeter, b []byte) {
	osCount := make(map[interface{}]int)
	versionCount := make(map[interface{}]int)
	archCount := make(map[interface{}]int)
	durationCount := make(map[interface{}]int)
	commandCount := make(map[interface{}]int)
	errorCount := make(map[interface{}]int)
	builders := make(map[interface{}]int)
	buildersWithPlatforms := make(map[interface{}]int)
	buildDeps := make(map[interface{}]int)
	helmReleases := make(map[interface{}]int)
	devIterations := make(map[interface{}]int)
	resourceFilters := make(map[interface{}]int)
	deployers := make(map[interface{}]int)
	enumFlags := make(map[interface{}]int)
	platform := make(map[interface{}]int)
	buildPlatforms := make(map[interface{}]int)
	cliPlatforms := make(map[interface{}]int)
	nodePlatforms := make(map[interface{}]int)

	testMaps := []map[interface{}]int{
		platform, osCount, versionCount, archCount, durationCount, commandCount, errorCount, builders, devIterations, resourceFilters, deployers}

	for _, meter := range meters {
		osCount[meter.OS]++
		versionCount[meter.Version]++
		durationCount[fmt.Sprintf("%s:%f", meter.Command, meter.Duration.Seconds())]++
		archCount[meter.Arch]++
		commandCount[meter.Command]++
		errorCount[meter.ErrorCode.String()]++
		platform[meter.PlatformType]++

		for k, v := range meter.EnumFlags {
			n := strings.ReplaceAll(k, "-", "_")
			enumFlags[n+":"+v]++
		}

		if doesBuild.Contains(meter.Command) {
			for k, v := range meter.Builders {
				builders[k] += v
			}
			for k, v := range meter.BuildWithPlatforms {
				buildersWithPlatforms[k] += v
			}
			for k, v := range meter.BuildDependencies {
				buildDeps[k] += v
			}
		}
		if meter.Command == "dev" || meter.Command == "debug" {
			for _, devI := range meter.DevIterations {
				devIterations[devI]++
			}
		}
		if doesDeploy.Contains(meter.Command) {
			for _, d := range meter.Deployers {
				deployers[d]++
			}
		}
		if doesDeploy.Contains(meter.Command) || meter.Command == "render" {
			for _, filterI := range meter.ResourceFilters {
				resourceFilters[filterI]++
			}
		}
	}

	var r Result
	var lines []*line
	err := json.Unmarshal(b, &r)
	if err != nil {
		t.Fatal(err)
	}

	for _, smetric := range r.ScopeMetrics {
		for _, metric := range smetric.Metrics {
			dataPoints := metric.Data.DataPoints
			for _, point := range dataPoints {
				var l = line{Labels: make(map[string]string)}
				l.Name = metric.Name
				l.Count = point.Value
				l.Max = point.Value
				for _, attr := range point.Attributes {
					switch reflect.ValueOf(attr.Value).Kind() {
					case reflect.Int:
						v := strconv.FormatInt(reflect.ValueOf(attr.Value.Value).Int(), 10)
						l.Labels[attr.Key] = v
					case reflect.Float64:
						v := fmt.Sprintf("%f", reflect.ValueOf(attr.Value.Value).Float())
						l.Labels[attr.Key] = v
					default:
						v := reflect.ValueOf(attr.Value.Value).String()
						l.Labels[attr.Key] = v
					}
				}
				lines = append(lines, &l)
			}
		}
	}
	for _, l := range lines {
		fmt.Println(l.Name)
		fmt.Println(l.Labels)
	}

	for _, l := range lines {
		switch l.Name {
		case "launches":
			if v, ok := l.Labels["arch"]; ok {
				archCount[v]--
			}
			if v, ok := l.Labels["os"]; ok {
				osCount[v]--
			}
			if v, ok := l.Labels["version"]; ok {
				versionCount[v]--
			}
			if v, ok := l.Labels["platform_type"]; ok {
				platform[v]--
			}
			e := l.Labels["error"]
			if e == proto.StatusCode_OK.String() {
				errorCount[e]--
			}
		case "launch/duration":
			if v, ok := l.Labels["command"]; ok {
				durationCount[fmt.Sprintf("%s:%f", v, l.value().(float64))]--
			}
		case "artifacts":
			if v, ok := l.Labels["builder"]; ok {
				builders[v] -= int(l.value().(float64)) - 1
			}
		case "artifact-with-platforms":
			if v, ok := l.Labels["builder"]; ok {
				buildersWithPlatforms[v] -= int(l.value().(float64)) - 1
			}
		case "artifact-dependencies":
			if v, ok := l.Labels["builder"]; ok {
				buildDeps[v] -= int(l.value().(float64)) - 1
			}
		case "builders":
			if v, ok := l.Labels["builder"]; ok {
				builders[v]--
			}
		case "deployer":
			if v, ok := l.Labels["deployer"]; ok {
				deployers[v]--
			}
		case "dev/iterations", "debug/iterations":
			if _, ok := l.Labels["error"]; !ok {
				continue
			}
			if _, ok := l.Labels["intent"]; !ok {
				continue
			}
			e := l.Labels["error"]
			devIterations[devIteration{l.Labels["intent"], proto.StatusCode(proto.StatusCode_value[e])}]--
		case "resource-filters":
			if _, ok := l.Labels["source"]; !ok {
				continue
			}
			if _, ok := l.Labels["type"]; !ok {
				continue
			}
			resourceFilters[resourceFilter{l.Labels["source"], l.Labels["type"]}]--
		case "errors":
			if e, ok := l.Labels["error"]; ok {
				errorCount[e]--
			}
		case "flags":
			if _, ok := l.Labels["flag_name"]; !ok {
				continue
			}
			if _, ok := l.Labels["value"]; !ok {
				continue
			}
			enumFlags[l.Labels["flag_name"]+":"+l.Labels["value"]]--
		case "helmReleases":
			if v, ok := l.Labels["helmReleases"]; ok {
				helmReleases[v]++
			}
		case "build-platforms":
			if v, ok := l.Labels["build-platforms"]; ok {
				buildPlatforms[v]++
			}
		case "node-platforms":
			if v, ok := l.Labels["platforms"]; ok {
				nodePlatforms[v]++
			}
		case "cli-platforms":
			if v, ok := l.Labels["cli-platforms"]; ok {
				cliPlatforms[v]++
			}
		default:
			switch {
			case MeteredCommands.Contains(l.Name):
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

func checkUser(t *testutil.T, user string, b []byte) {
	var lines []*line
	json.Unmarshal(b, &lines)
	expectedFound := user != ""
	for _, l := range lines {
		l.initLine()
		if l.Name == "launches" {
			v, ok := l.Labels["user"]
			t.CheckDeepEqual(expectedFound, ok)
			t.CheckDeepEqual(user, v)
			return
		}
	}
}

func TestGetClusterType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "is gke",
			input:    "gke_test",
			expected: "gke",
		},
		{
			name:     "not gke",
			input:    "minikube",
			expected: "others",
		},
		{
			name:     "azure",
			input:    "azure_",
			expected: "others",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, getClusterType(test.input))
		})
	}
}

// Derived from go.opentelemetry.io/otel/exporters/stdout/metric.go
type line struct {
	Name   string      `json:"Name"`
	Min    interface{} `json:"Min,omitempty"`
	Max    interface{} `json:"Max,omitempty"`
	Sum    interface{} `json:"Sum,omitempty"`
	Count  interface{} `json:"Count,omitempty"`
	Labels map[string]string
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
	return l.Max
}

type KeyValue struct {
	Key   string `json:"Key"`
	Value Value  `json:"Value"`
}

type Result struct {
	Resources    []KeyValue     `json:"Resources"`
	ScopeMetrics []ScopeMetrics `json:"ScopeMetrics"`
}

type ScopeMetrics struct {
	Scope   Scope     `json:"Scope"`
	Metrics []Metrics `json:"Metrics"`
}

type Scope struct {
	Name      string
	Version   string
	SchemaURL string
}

type Metrics struct {
	Name string `json:"Name"`
	Data Data   `json:"Data"`
}

type Data struct {
	DataPoints []DataPoint `json:"DataPoints"`
}

type DataPoint struct {
	Attributes []KeyValue  `json:"Attributes"`
	Value      interface{} `json:"Value"`
}
