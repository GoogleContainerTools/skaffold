// Copyright The OpenTelemetry Authors
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

// Package stdoutmetric provides an exporter for OpenTelemetry metric
// telemetry.
//
// The exporter is intended to be used for testing and debugging, it is not
// meant for production use. Additionally, it does not provide an interchange
// format for OpenTelemetry that is supported with any stability or
// compatibility guarantees. If these are needed features, please use the OTLP
// exporter instead.
package stdoutmetric // import "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
