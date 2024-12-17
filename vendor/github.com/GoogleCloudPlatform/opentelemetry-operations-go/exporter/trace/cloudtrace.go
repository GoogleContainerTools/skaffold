// Copyright 2019 OpenTelemetry Authors
// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package trace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	traceapi "cloud.google.com/go/trace/apiv2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// Option is function type that is passed to the exporter initialization function.
type Option func(*options)

// options contains options for configuring the exporter.
type options struct {
	// OnError is the hook to be called when there is
	// an error uploading the stats or tracing data.
	// If no custom hook is set, errors are handled with the
	// OpenTelemetry global handler, which defaults to logging.
	// Optional.
	errorHandler otel.ErrorHandler
	// context allows you to provide a custom context for API calls.
	//
	// This context will be used several times: first, to create Stackdriver
	// trace and metric clients, and then every time a new batch of traces or
	// stats needs to be uploaded.
	//
	// Do not set a timeout on this context. Instead, set the Timeout option.
	//
	// If unset, context.Background() will be used.
	context context.Context
	// mapAttribute maps otel attribute keys to cloud trace attribute keys
	mapAttribute AttributeMapping
	// projectID is the identifier of the Stackdriver
	// project the user is uploading the stats data to.
	// If not set, this will default to your "Application Default Credentials".
	// For details see: https://developers.google.com/accounts/docs/application-default-credentials.
	//
	// It will be used in the project_id label of a Stackdriver monitored
	// resource if the resource does not inherently belong to a specific
	// project, e.g. on-premise resource like k8s_container or generic_task.
	projectID string
	// traceClientOptions are additional options to be passed
	// to the underlying Stackdriver Trace API client.
	// Optional.
	traceClientOptions []option.ClientOption
	// timeout for all API calls. If not set, defaults to 12 seconds.
	timeout time.Duration
	// destinationProjectQuota sets whether the request should use quota from
	// the destination project for the request.
	destinationProjectQuota bool
}

// WithProjectID sets Google Cloud Platform project as projectID.
// Without using this option, it automatically detects the project ID
// from the default credential detection process.
// Please find the detailed order of the default credential detection process on the doc:
// https://godoc.org/golang.org/x/oauth2/google#FindDefaultCredentials
func WithProjectID(projectID string) func(o *options) {
	return func(o *options) {
		o.projectID = projectID
	}
}

// WithDestinationProjectQuota enables per-request usage of the destination
// project's quota. For example, when setting the gcp.project.id resource attribute.
func WithDestinationProjectQuota() func(o *options) {
	return func(o *options) {
		o.destinationProjectQuota = true
	}
}

// WithErrorHandler sets the hook to be called when there is an error
// occurred on uploading the span data to Stackdriver.
// If no custom hook is set, errors are logged.
func WithErrorHandler(handler otel.ErrorHandler) func(o *options) {
	return func(o *options) {
		o.errorHandler = handler
	}
}

// WithTraceClientOptions sets additionial client options for tracing.
func WithTraceClientOptions(opts []option.ClientOption) func(o *options) {
	return func(o *options) {
		o.traceClientOptions = opts
	}
}

// WithContext sets the context that trace exporter and metric exporter
// relies on.
func WithContext(ctx context.Context) func(o *options) {
	return func(o *options) {
		o.context = ctx
	}
}

// WithTimeout sets the timeout for trace exporter and metric exporter
// If unset, it defaults to a 12 second timeout.
func WithTimeout(t time.Duration) func(o *options) {
	return func(o *options) {
		o.timeout = t
	}
}

// AttributeMapping determines how to map from OpenTelemetry span attribute keys to
// cloud trace attribute keys.
type AttributeMapping func(attribute.Key) attribute.Key

// WithAttributeMapping configures how to map OpenTelemetry span attributes
// to google cloud trace span attributes.  By default, it maps to attributes
// that are used prominently in the trace UI.
func WithAttributeMapping(mapping AttributeMapping) func(o *options) {
	return func(o *options) {
		o.mapAttribute = mapping
	}
}

func (o *options) handleError(err error) {
	if o.errorHandler != nil {
		o.errorHandler.Handle(err)
		return
	}
	otel.Handle(err)
}

// defaultTimeout is used as default when timeout is not set in newContextWithTimout.
const defaultTimeout = 12 * time.Second

// Exporter is a trace exporter that uploads data to Stackdriver.
//
// TODO(yoshifumi): add a metrics exporter once the spec definition
// process and the sampler implementation are done.
type Exporter struct {
	traceExporter *traceExporter
}

// New creates a new Exporter thats implements trace.Exporter.
func New(opts ...Option) (*Exporter, error) {
	o := options{
		context:      context.Background(),
		mapAttribute: defaultAttributeMapping,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return newExporterWithOptions(&o)
}

func newExporterWithOptions(o *options) (*Exporter, error) {
	if o.projectID == "" {
		creds, err := google.FindDefaultCredentials(o.context, traceapi.DefaultAuthScopes()...)
		if err != nil {
			return nil, fmt.Errorf("stackdriver: %v", err)
		}
		if creds.ProjectID == "" {
			return nil, errors.New("stackdriver: no project found with application default credentials")
		}
		o.projectID = creds.ProjectID
	}
	te, err := newTraceExporter(o)
	if err != nil {
		return nil, err
	}

	return &Exporter{
		traceExporter: te,
	}, nil
}

func newContextWithTimeout(ctx context.Context, timeout time.Duration) (context.Context, func()) {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// ExportSpans exports a ReadOnlySpan to Stackdriver Trace.
func (e *Exporter) ExportSpans(ctx context.Context, spanData []sdktrace.ReadOnlySpan) error {
	return e.traceExporter.ExportSpans(ctx, spanData)
}

// Shutdown waits for exported data to be uploaded.
//
// For our purposes it closed down the client.
func (e *Exporter) Shutdown(ctx context.Context) error {
	return e.traceExporter.Shutdown(ctx)
}
