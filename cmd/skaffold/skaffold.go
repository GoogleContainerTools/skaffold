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

package main

import (
	"context"
	"errors"
	"os"

	"cloud.google.com/go/profiler"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

func main() {

	if _, ok := os.LookupEnv("SKAFFOLD_PROFILER"); ok {
		err := profiler.Start(profiler.Config{
			Service:              os.Getenv("SKAFFOLD_PROFILER_SERVICE"),
			NoHeapProfiling:      true,
			NoAllocProfiling:     true,
			NoGoroutineProfiling: true,
			DebugLogging:         true,
			// ProjectID must be set if not running on GCP.
			ProjectID:      os.Getenv("SKAFFOLD_PROFILER_PROJECT"),
			ServiceVersion: version.Get().Version,
		})
		if err != nil {
			log.Entry(context.TODO()).Fatalf("failed to start the profiler: %v", err)
		}
	}
	var code int
	if err := app.Run(os.Stdout, os.Stderr); err != nil && !errors.Is(err, context.Canceled) {
		// ignore cancelled errors
		code = app.ExitCode(err)
	}
	instrumentation.ShutdownAndFlush(context.Background(), code)
	os.Exit(code)
}
