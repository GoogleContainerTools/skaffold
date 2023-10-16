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
	"fmt"
	"os"
	"runtime"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

var traceEnabled bool
var traceInitOnce sync.Once

var tracerProvider trace.TracerProvider
var tracerShutdown func(context.Context) error = func(context.Context) error { return nil }
var tracerInitErr error

// InitTraceFromEnvVar initializes the singleton skaffold tracer from the SKAFFOLD_TRACE env variable.
// The code here is a wrapper around the opentelemetry(otel) trace libs for usability
// When SKAFFOLD_TRACE is set, this will setup the proper tracer provider (& exporter),
// configures otel to use this tracer provider and saves the  tracer provider shutdown function
// to be used globally so that it can be run before skaffold exits.
func InitTraceFromEnvVar(opts ...TraceExporterOption) (trace.TracerProvider, func(context.Context) error, error) {
	traceInitOnce.Do(func() {
		_, skaffTraceEnv := os.LookupEnv("SKAFFOLD_TRACE")
		_, otelTraceExporterEnv := os.LookupEnv("OTEL_TRACES_EXPORTER")
		if skaffTraceEnv || otelTraceExporterEnv {
			traceEnabled = true
		}
		if traceEnabled {
			tp, shutdown, err := initTraceExporter(opts...)
			tracerInitErr = err
			if err == nil && skaffTraceEnv { // if only OTEL_TRACES_EXPORTER, tp set automatically
				otel.SetTracerProvider(tp)
				tracerProvider = tp
				tracerShutdown = shutdown
			}
		}
	})
	if tracerInitErr != nil {
		log.Entry(context.TODO()).Debugf("error initializing tracing: %v", tracerInitErr)
	}
	return tracerProvider, tracerShutdown, tracerInitErr
}

// TracerShutdown is a function used to flush all current running spans and make sure they are exported.  This should be called once
// at the end of a skaffold run to properly shutdown and export all spans for the singleton.
func TracerShutdown(ctx context.Context) error {
	traceInitOnce = sync.Once{}
	return tracerShutdown(ctx)
}

// StartTrace uses the otel trace provider to export timing spans (with optional attributes) to the chosen exporter
// via the value set in SKAFFOLD_TRACE.  Tracing is done via metadata stored in a context.Context. This means that
// to properly get parent/child traces, callers should use the returned context for subsequent calls in skaffold.
// The returned function should be called to end the trace span, for example this can be done with
// the form:  _, endTrace = StartTrace...; defer endTrace()
func StartTrace(ctx context.Context, name string, attributes ...map[string]string) (context.Context, func(options ...trace.SpanEndOption)) {
	if traceEnabled {
		_, file, ln, _ := runtime.Caller(1)
		tracer := otel.Tracer(file)
		ctx, span := tracer.Start(ctx, name)
		for _, attrs := range attributes {
			for k, v := range attrs {
				span.SetAttributes(attribute.Key(k).String(v))
			}
		}
		// currently Cloud Trace doesn't show the package in the UI, hack to get package information in Cloud Trace
		span.SetAttributes(attribute.Key("source_file").String(fmt.Sprintf("%s:%d", file, ln)))
		return ctx, span.End
	}
	return ctx, func(options ...trace.SpanEndOption) {}
}

// TraceEndError adds an "error" attribute with value err.Error() to a span during it's end/shutdown callback
// This fnx is intended to used with the StartTrace callback - "endTrace" when an error occurs during the code path
// of trace, ex: endTrace(instrumentation.TraceEndError(err)); return nil, err
func TraceEndError(err error) trace.SpanEndOption {
	if traceEnabled {
		return trace.WithStackTrace(true)
	}
	return nil
}

// AddAttributesToCurrentSpanFromContext adds the attributes from the input map to the span pulled from the current context.
// This is useful when additional attributes should be added to a parent span but the span object is not directly accessible.
func AddAttributesToCurrentSpanFromContext(ctx context.Context, attrs map[string]string) {
	if traceEnabled {
		span := trace.SpanFromContext(ctx)
		for k, v := range attrs {
			span.SetAttributes(attribute.Key(k).String(v))
		}
	}
}

// PII stub function tracking trace attributes that have PII in them.  Currently no trace information is uploaded so
// PII values are not an issue but if in the future they are uploaded this will need to properly strip PII
func PII(s string) string {
	// TODO(#5885) add functionality
	// currently this is a stub for tracking PII attributes
	return s
}
