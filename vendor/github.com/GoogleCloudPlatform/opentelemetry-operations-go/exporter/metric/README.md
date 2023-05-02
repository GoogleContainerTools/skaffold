# OpenTelemetry Google Cloud Monitoring Exporter

[![Docs](https://godoc.org/github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric?status.svg)](https://pkg.go.dev/github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric)
[![Apache License][license-image]][license-url]

OpenTelemetry Google Cloud Monitoring Exporter allow the user to send collected metrics to Google Cloud.

[Google Cloud Monitoring](https://cloud.google.com/monitoring) provides visibility into the performance, uptime, and overall health of cloud-powered applications. It collects metrics, events, and metadata from Google Cloud, Amazon Web Services, hosted uptime probes, application instrumentation, and a variety of common application components including Cassandra, Nginx, Apache Web Server, Elasticsearch, and many others. Operations ingests that data and generates insights via dashboards, charts, and alerts. Cloud Monitoring alerting helps you collaborate by integrating with Slack, PagerDuty, and more.

## Setup

Google Cloud Monitoring is a managed service provided by Google Cloud Platform. Google Cloud Monitoring requires to set up "Workspace" in advance. The guide to create a new Workspace is available on [the official document](https://cloud.google.com/monitoring/workspaces/create).

## Usage

After you import the metric exporter package, then register the exporter to the application, and start sending metrics. If you are running in a GCP environment, the exporter will automatically authenticate using the environment's service account. If not, you will need to follow the instructions in [Authentication](#Authentication).

```go
package main

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/sdk/metric"

	mexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
)

func main() {
	exporter, err := mexporter.New()
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}
	// initialize a MeterProvider with that periodically exports to the GCP exporter.
	provider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
	)
	ctx := context.Background()
	defer provider.Shutdown(ctx)

	// Start meter
	meter := provider.Meter("github.com/GoogleCloudPlatform/opentelemetry-operations-go/example/metric")

	counter, err := meter.SyncInt64().Counter("counter-foo")
	if err != nil {
		log.Fatalf("Failed to create counter: %v", err)
	}
	labels := []label.KeyValue{
		label.Key("key").String("value"),
	}
	counter.Add(ctx, 123, labels...)
}
```

Note that, as of version 0.2.1, `ValueObserver` and `ValueRecorder` are aggregated to `LastValue`, and other metric kinds are to `Sum`. This behaviour should be change once [Views API](https://github.com/open-telemetry/oteps/pull/89) is introduced to the specification.

## Authentication

The Google Cloud Monitoring exporter depends upon [`google.FindDefaultCredentials`](https://pkg.go.dev/golang.org/x/oauth2/google?tab=doc#FindDefaultCredentials), so the service account is automatically detected by default, but also the custom credential file (so called `service_account_key.json`) can be detected with specific conditions. Quoting from the document of `google.FindDefaultCredentials`:

* A JSON file whose path is specified by the `GOOGLE_APPLICATION_CREDENTIALS` environment variable.
* A JSON file in a location known to the gcloud command-line tool. On Windows, this is `%APPDATA%/gcloud/application_default_credentials.json`. On other systems, `$HOME/.config/gcloud/application_default_credentials.json`.

When running code locally, you may need to specify a Google Project ID in addition to `GOOGLE_APPLICATION_CREDENTIALS`. This is best done using an environment variable (e.g. `GOOGLE_CLOUD_PROJECT`) and the `metric.WithProjectID` method, e.g.:

```golang
projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
opts := []mexporter.Option{
    mexporter.WithProjectID(projectID),
}
```

## Useful links

* For more information on OpenTelemetry, visit: https://opentelemetry.io/
* For more about OpenTelemetry Go, visit: https://github.com/open-telemetry/opentelemetry-go
* Learn more about Google Cloud Monitoring at https://cloud.google.com/monitoring

[license-url]: https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/blob/main/LICENSE
[license-image]: https://img.shields.io/badge/license-Apache_2.0-green.svg?style=flat
