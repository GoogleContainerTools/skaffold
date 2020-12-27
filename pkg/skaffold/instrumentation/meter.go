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
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	mexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	"github.com/mitchellh/go-homedir"
	"github.com/rakyll/statik/fs"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"google.golang.org/api/option"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/statik"

	//  import embedded secret for uploading metrics
	_ "github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/secret/statik"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yamltags"
	"github.com/GoogleContainerTools/skaffold/proto"
)

type skaffoldMeter struct {
	ExitCode       int
	BuildArtifacts int
	Command        string
	Version        string
	OS             string
	Arch           string
	PlatformType   string
	Deployers      []string
	EnumFlags      map[string]string
	Builders       map[string]int
	SyncType       map[string]bool
	DevIterations  []devIteration
	StartTime      time.Time
	Duration       time.Duration
	ErrorCode      proto.StatusCode
}

type devIteration struct {
	Intent    string
	ErrorCode proto.StatusCode
}

var (
	meter = skaffoldMeter{
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		EnumFlags:     map[string]string{},
		Builders:      map[string]int{},
		SyncType:      map[string]bool{},
		DevIterations: []devIteration{},
		StartTime:     time.Now(),
		Version:       version.Get().Version,
		ExitCode:      0,
		ErrorCode:     proto.StatusCode_OK,
	}
	shouldExportMetrics = os.Getenv("SKAFFOLD_EXPORT_METRICS") == "true"
	meteredCommands     = util.NewStringSet()
	doesBuild           = util.NewStringSet()
	doesDeploy          = util.NewStringSet()
	initExporter        = initCloudMonitoringExporterMetrics
	isOnline            bool
)

func init() {
	meteredCommands.Insert("build", "delete", "deploy", "dev", "debug", "filter", "generate_pipeline", "render", "run", "test")
	doesBuild.Insert("build", "render", "dev", "debug", "run")
	doesDeploy.Insert("deploy", "dev", "debug", "run")
	go func() {
		if shouldExportMetrics {
			r, err := http.Get("http://clients3.google.com/generate_204")
			if err == nil {
				r.Body.Close()
				isOnline = true
			}
		}
	}()
}

func InitMeterFromConfig(configs []*latest.SkaffoldConfig) {
	meter.PlatformType = yamltags.GetYamlTag(configs[0].Build.BuildType) // TODO: support multiple build types in events.
	for _, config := range configs {
		for _, artifact := range config.Pipeline.Build.Artifacts {
			meter.Builders[yamltags.GetYamlTag(artifact.ArtifactType)]++
			if artifact.Sync != nil {
				meter.SyncType[yamltags.GetYamlTag(artifact.Sync)] = true
			}
		}
		meter.Deployers = append(meter.Deployers, yamltags.GetYamlTags(config.Deploy.DeployType)...)
		meter.BuildArtifacts += len(config.Pipeline.Build.Artifacts)
	}
}

func SetCommand(cmd string) {
	if meteredCommands.Contains(cmd) {
		meter.Command = cmd
	}
}

func SetErrorCode(errorCode proto.StatusCode) {
	meter.ErrorCode = errorCode
}

func AddDevIteration(intent string) {
	meter.DevIterations = append(meter.DevIterations, devIteration{Intent: intent})
}

func AddDevIterationErr(i int, errorCode proto.StatusCode) {
	if len(meter.DevIterations) == i {
		meter.DevIterations = append(meter.DevIterations, devIteration{Intent: "error"})
	}
	meter.DevIterations[i].ErrorCode = errorCode
}

func AddFlag(flag *flag.Flag) {
	if flag.Changed {
		meter.EnumFlags[flag.Name] = flag.Value.String()
	}
}

func ExportMetrics(exitCode int) error {
	if !shouldExportMetrics || meter.Command == "" {
		return nil
	}
	home, err := homedir.Dir()
	if err != nil {
		return fmt.Errorf("retrieving home directory: %w", err)
	}
	meter.ExitCode = exitCode
	meter.Duration = time.Since(meter.StartTime)
	return exportMetrics(context.Background(),
		filepath.Join(home, constants.DefaultSkaffoldDir, constants.DefaultMetricFile),
		meter)
}

func exportMetrics(ctx context.Context, filename string, meter skaffoldMeter) error {
	logrus.Debug("exporting metrics")
	p, err := initExporter()
	if p == nil {
		return err
	}

	b, err := ioutil.ReadFile(filename)
	fileExists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var meters []skaffoldMeter
	err = json.Unmarshal(b, &meters)
	if err != nil {
		meters = []skaffoldMeter{}
	}
	meters = append(meters, meter)
	if !isOnline {
		b, _ = json.Marshal(meters)
		return ioutil.WriteFile(filename, b, 0666)
	}

	start := time.Now()
	p.Start()
	for _, m := range meters {
		createMetrics(ctx, m)
	}
	p.Stop()
	logrus.Debugf("metrics uploading complete in %s", time.Since(start).String())

	if fileExists {
		return os.Remove(filename)
	}
	return nil
}

type creds struct {
	ProjectID string `json:"project_id"`
}

func initCloudMonitoringExporterMetrics() (*push.Controller, error) {
	statikFS, err := statik.FS()
	if err != nil {
		return nil, err
	}
	b, err := fs.ReadFile(statikFS, "/keys.json")
	if err != nil {
		// No keys have been set in this version so do not attempt to write metrics
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var c creds
	err = json.Unmarshal(b, &c)
	if c.ProjectID == "" || err != nil {
		return nil, fmt.Errorf("no project id found in metrics credentials")
	}

	formatter := func(desc *metric.Descriptor) string {
		return fmt.Sprintf("custom.googleapis.com/skaffold/%s", desc.Name())
	}

	return mexporter.InstallNewPipeline(
		[]mexporter.Option{
			mexporter.WithProjectID(c.ProjectID),
			mexporter.WithMetricDescriptorTypeFormatter(formatter),
			mexporter.WithMonitoringClientOptions(option.WithCredentialsJSON(b)),
			mexporter.WithOnError(func(err error) {
				logrus.Debugf("Error with metrics: %v", err)
			}),
		},
	)
}

func createMetrics(ctx context.Context, meter skaffoldMeter) {
	// There is a minimum 10 second interval that metrics are allowed to upload to Cloud monitoring
	// A metric is uniquely identified by the metric name and the labels and corresponding values
	// This random number is used as a label to differentiate the metrics per user so if two users
	// run `skaffold build` at the same time they will both have their metrics recorded
	randLabel := label.String("randomizer", strconv.Itoa(rand.Intn(75000)))

	m := global.Meter("skaffold")
	labels := []label.KeyValue{
		label.String("version", meter.Version),
		label.String("os", meter.OS),
		label.String("arch", meter.Arch),
		label.String("command", meter.Command),
		label.String("error", strconv.Itoa(int(meter.ErrorCode))),
		randLabel,
	}

	runCounter := metric.Must(m).NewInt64ValueRecorder("launches", metric.WithDescription("Skaffold Invocations"))
	runCounter.Record(ctx, 1, labels...)

	durationRecorder := metric.Must(m).NewFloat64ValueRecorder("launch/duration",
		metric.WithDescription("durations of skaffold commands in seconds"))
	durationRecorder.Record(ctx, meter.Duration.Seconds(), labels...)
	if meter.Command != "" {
		commandMetrics(ctx, meter, m, randLabel)
		if doesBuild.Contains(meter.Command) {
			builderMetrics(ctx, meter, m, randLabel)
		}
		if doesDeploy.Contains(meter.Command) {
			deployerMetrics(ctx, meter, m, randLabel)
		}
	}

	if meter.ErrorCode != 0 {
		errorMetrics(ctx, meter, m, randLabel)
	}
}

func commandMetrics(ctx context.Context, meter skaffoldMeter, m metric.Meter, randLabel label.KeyValue) {
	commandCounter := metric.Must(m).NewInt64ValueRecorder(meter.Command,
		metric.WithDescription(fmt.Sprintf("Number of times %s is used", meter.Command)))
	labels := []label.KeyValue{
		label.String("error", strconv.Itoa(int(meter.ErrorCode))),
		randLabel,
	}
	commandCounter.Record(ctx, 1, labels...)

	if meter.Command == "dev" {
		iterationCounter := metric.Must(m).NewInt64ValueRecorder("dev/iterations",
			metric.WithDescription("Number of iterations in a dev session"))

		counts := make(map[string]map[proto.StatusCode]int)

		for _, iteration := range meter.DevIterations {
			if _, ok := counts[iteration.Intent]; !ok {
				counts[iteration.Intent] = make(map[proto.StatusCode]int)
			}
			m := counts[iteration.Intent]
			m[iteration.ErrorCode]++
		}
		for intention, errorCounts := range counts {
			for errorCode, count := range errorCounts {
				iterationCounter.Record(ctx, int64(count),
					label.String("intent", intention),
					label.String("error", strconv.Itoa(int(errorCode))),
					randLabel)
			}
		}
	}
}

func deployerMetrics(ctx context.Context, meter skaffoldMeter, m metric.Meter, randLabel label.KeyValue) {
	deployerCounter := metric.Must(m).NewInt64ValueRecorder("deployer", metric.WithDescription("Deployers used"))
	for _, deployer := range meter.Deployers {
		deployerCounter.Record(ctx, 1, randLabel, label.String("deployer", deployer))
	}
}

func builderMetrics(ctx context.Context, meter skaffoldMeter, m metric.Meter, randLabel label.KeyValue) {
	builderCounter := metric.Must(m).NewInt64ValueRecorder("builders", metric.WithDescription("Builders used"))
	artifactCounter := metric.Must(m).NewInt64ValueRecorder("artifacts", metric.WithDescription("Number of artifacts used"))
	for builder, count := range meter.Builders {
		bLabel := label.String("builder", builder)
		builderCounter.Record(ctx, 1, bLabel, randLabel)
		artifactCounter.Record(ctx, int64(count), bLabel, randLabel)
	}
}

func errorMetrics(ctx context.Context, meter skaffoldMeter, m metric.Meter, randLabel label.KeyValue) {
	errCounter := metric.Must(m).NewInt64ValueRecorder("errors", metric.WithDescription("Skaffold errors"))
	errCounter.Record(ctx, 1, label.String("error", strconv.Itoa(int(meter.ErrorCode))), randLabel)

	commandLabel := label.String("command", meter.Command)

	switch meter.ErrorCode {
	case proto.StatusCode_UNKNOWN_ERROR:
		unknownErrCounter := metric.Must(m).NewInt64ValueRecorder("errors/unknown", metric.WithDescription("Unknown Skaffold Errors"))
		unknownErrCounter.Record(ctx, 1, randLabel)
	case proto.StatusCode_DEPLOY_UNKNOWN:
		unknownCounter := metric.Must(m).NewInt64ValueRecorder("deploy/unknown", metric.WithDescription("Unknown deploy Skaffold Errors"))
		unknownCounter.Record(ctx, 1, commandLabel, randLabel)
	case proto.StatusCode_BUILD_UNKNOWN:
		unknownCounter := metric.Must(m).NewInt64ValueRecorder("build/unknown", metric.WithDescription("Unknown build Skaffold Errors"))
		unknownCounter.Record(ctx, 1, commandLabel, randLabel)
	}
}
