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

package instrumentation

import (
	"context"

	"github.com/sirupsen/logrus"

	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

// Init initializes the skaffold metrics and trace tooling built on top of open-telemetry (otel)
func Init(configs []*latestV2.SkaffoldConfig, user string, opts ...TraceExporterOption) {
	InitMeterFromConfig(configs, user)
	InitTraceFromEnvVar()
}

func ShutdownAndFlush(ctx context.Context, exitCode int) {
	if err := ExportMetrics(exitCode); err != nil {
		logrus.Debugf("error exporting metrics %v", err)
	}
	if err := TracerShutdown(ctx); err != nil {
		logrus.Debugf("error shutting down tracer %v", err)
	}
}
