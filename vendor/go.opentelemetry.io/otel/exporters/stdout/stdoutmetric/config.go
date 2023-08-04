// Copyright The OpenTelemetry Authors
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

package stdoutmetric // import "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"

import (
	"encoding/json"
	"os"

	"go.opentelemetry.io/otel/internal/global"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
)

// config contains options for the exporter.
type config struct {
	encoder             *encoderHolder
	temporalitySelector metric.TemporalitySelector
	aggregationSelector metric.AggregationSelector
}

// newConfig creates a validated config configured with options.
func newConfig(options ...Option) config {
	cfg := config{}
	for _, opt := range options {
		cfg = opt.apply(cfg)
	}

	if cfg.encoder == nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "\t")
		cfg.encoder = &encoderHolder{encoder: enc}
	}

	if cfg.temporalitySelector == nil {
		cfg.temporalitySelector = metric.DefaultTemporalitySelector
	}

	if cfg.aggregationSelector == nil {
		cfg.aggregationSelector = metric.DefaultAggregationSelector
	}

	return cfg
}

// Option sets exporter option values.
type Option interface {
	apply(config) config
}

type optionFunc func(config) config

func (o optionFunc) apply(c config) config {
	return o(c)
}

// WithEncoder sets the exporter to use encoder to encode all the metric
// data-types to an output.
func WithEncoder(encoder Encoder) Option {
	return optionFunc(func(c config) config {
		if encoder != nil {
			c.encoder = &encoderHolder{encoder: encoder}
		}
		return c
	})
}

// WithTemporalitySelector sets the TemporalitySelector the exporter will use
// to determine the Temporality of an instrument based on its kind. If this
// option is not used, the exporter will use the DefaultTemporalitySelector
// from the go.opentelemetry.io/otel/sdk/metric package.
func WithTemporalitySelector(selector metric.TemporalitySelector) Option {
	return temporalitySelectorOption{selector: selector}
}

type temporalitySelectorOption struct {
	selector metric.TemporalitySelector
}

func (t temporalitySelectorOption) apply(c config) config {
	c.temporalitySelector = t.selector
	return c
}

// WithAggregationSelector sets the AggregationSelector the exporter will use
// to determine the aggregation to use for an instrument based on its kind. If
// this option is not used, the exporter will use the
// DefaultAggregationSelector from the go.opentelemetry.io/otel/sdk/metric
// package or the aggregation explicitly passed for a view matching an
// instrument.
func WithAggregationSelector(selector metric.AggregationSelector) Option {
	// Deep copy and validate before using.
	wrapped := func(ik metric.InstrumentKind) aggregation.Aggregation {
		a := selector(ik)
		cpA := a.Copy()
		if err := cpA.Err(); err != nil {
			cpA = metric.DefaultAggregationSelector(ik)
			global.Error(
				err, "using default aggregation instead",
				"aggregation", a,
				"replacement", cpA,
			)
		}
		return cpA
	}

	return aggregationSelectorOption{selector: wrapped}
}

type aggregationSelectorOption struct {
	selector metric.AggregationSelector
}

func (t aggregationSelectorOption) apply(c config) config {
	c.aggregationSelector = t.selector
	return c
}
