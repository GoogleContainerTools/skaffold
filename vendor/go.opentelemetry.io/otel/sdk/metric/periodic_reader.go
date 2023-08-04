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

package metric // import "go.opentelemetry.io/otel/sdk/metric"

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/internal/global"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// Default periodic reader timing.
const (
	defaultTimeout  = time.Millisecond * 30000
	defaultInterval = time.Millisecond * 60000
)

// periodicReaderConfig contains configuration options for a PeriodicReader.
type periodicReaderConfig struct {
	interval time.Duration
	timeout  time.Duration
}

// newPeriodicReaderConfig returns a periodicReaderConfig configured with
// options.
func newPeriodicReaderConfig(options []PeriodicReaderOption) periodicReaderConfig {
	c := periodicReaderConfig{
		interval: envDuration(envInterval, defaultInterval),
		timeout:  envDuration(envTimeout, defaultTimeout),
	}
	for _, o := range options {
		c = o.applyPeriodic(c)
	}
	return c
}

// PeriodicReaderOption applies a configuration option value to a PeriodicReader.
type PeriodicReaderOption interface {
	applyPeriodic(periodicReaderConfig) periodicReaderConfig
}

// periodicReaderOptionFunc applies a set of options to a periodicReaderConfig.
type periodicReaderOptionFunc func(periodicReaderConfig) periodicReaderConfig

// applyPeriodic returns a periodicReaderConfig with option(s) applied.
func (o periodicReaderOptionFunc) applyPeriodic(conf periodicReaderConfig) periodicReaderConfig {
	return o(conf)
}

// WithTimeout configures the time a PeriodicReader waits for an export to
// complete before canceling it.
//
// This option overrides any value set for the
// OTEL_METRIC_EXPORT_TIMEOUT environment variable.
//
// If this option is not used or d is less than or equal to zero, 30 seconds
// is used as the default.
func WithTimeout(d time.Duration) PeriodicReaderOption {
	return periodicReaderOptionFunc(func(conf periodicReaderConfig) periodicReaderConfig {
		if d <= 0 {
			return conf
		}
		conf.timeout = d
		return conf
	})
}

// WithInterval configures the intervening time between exports for a
// PeriodicReader.
//
// This option overrides any value set for the
// OTEL_METRIC_EXPORT_INTERVAL environment variable.
//
// If this option is not used or d is less than or equal to zero, 60 seconds
// is used as the default.
func WithInterval(d time.Duration) PeriodicReaderOption {
	return periodicReaderOptionFunc(func(conf periodicReaderConfig) periodicReaderConfig {
		if d <= 0 {
			return conf
		}
		conf.interval = d
		return conf
	})
}

// NewPeriodicReader returns a Reader that collects and exports metric data to
// the exporter at a defined interval. By default, the returned Reader will
// collect and export data every 60 seconds, and will cancel export attempts
// that exceed 30 seconds. The export time is not counted towards the interval
// between attempts.
//
// The Collect method of the returned Reader continues to gather and return
// metric data to the user. It will not automatically send that data to the
// exporter. That is left to the user to accomplish.
func NewPeriodicReader(exporter Exporter, options ...PeriodicReaderOption) Reader {
	conf := newPeriodicReaderConfig(options)
	ctx, cancel := context.WithCancel(context.Background())
	r := &periodicReader{
		timeout:  conf.timeout,
		exporter: exporter,
		flushCh:  make(chan chan error),
		cancel:   cancel,
		done:     make(chan struct{}),
	}
	r.externalProducers.Store([]Producer{})

	go func() {
		defer func() { close(r.done) }()
		r.run(ctx, conf.interval)
	}()

	return r
}

// periodicReader is a Reader that continuously collects and exports metric
// data at a set interval.
type periodicReader struct {
	sdkProducer atomic.Value

	mu                sync.Mutex
	isShutdown        bool
	externalProducers atomic.Value

	timeout  time.Duration
	exporter Exporter
	flushCh  chan chan error

	done         chan struct{}
	cancel       context.CancelFunc
	shutdownOnce sync.Once
}

// Compile time check the periodicReader implements Reader and is comparable.
var _ = map[Reader]struct{}{&periodicReader{}: {}}

// newTicker allows testing override.
var newTicker = time.NewTicker

// run continuously collects and exports metric data at the specified
// interval. This will run until ctx is canceled or times out.
func (r *periodicReader) run(ctx context.Context, interval time.Duration) {
	ticker := newTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := r.collectAndExport(ctx)
			if err != nil {
				otel.Handle(err)
			}
		case errCh := <-r.flushCh:
			errCh <- r.collectAndExport(ctx)
			ticker.Reset(interval)
		case <-ctx.Done():
			return
		}
	}
}

// register registers p as the producer of this reader.
func (r *periodicReader) register(p sdkProducer) {
	// Only register once. If producer is already set, do nothing.
	if !r.sdkProducer.CompareAndSwap(nil, produceHolder{produce: p.produce}) {
		msg := "did not register periodic reader"
		global.Error(errDuplicateRegister, msg)
	}
}

// RegisterProducer registers p as an external Producer of this reader.
func (r *periodicReader) RegisterProducer(p Producer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isShutdown {
		return
	}
	currentProducers := r.externalProducers.Load().([]Producer)
	newProducers := []Producer{}
	newProducers = append(newProducers, currentProducers...)
	newProducers = append(newProducers, p)
	r.externalProducers.Store(newProducers)
}

// temporality reports the Temporality for the instrument kind provided.
func (r *periodicReader) temporality(kind InstrumentKind) metricdata.Temporality {
	return r.exporter.Temporality(kind)
}

// aggregation returns what Aggregation to use for kind.
func (r *periodicReader) aggregation(kind InstrumentKind) aggregation.Aggregation { // nolint:revive  // import-shadow for method scoped by type.
	return r.exporter.Aggregation(kind)
}

// collectAndExport gather all metric data related to the periodicReader r from
// the SDK and exports it with r's exporter.
func (r *periodicReader) collectAndExport(ctx context.Context) error {
	// TODO (#3047): Use a sync.Pool or persistent pointer instead of allocating rm every Collect.
	rm := metricdata.ResourceMetrics{}
	err := r.Collect(ctx, &rm)
	if err == nil {
		err = r.export(ctx, rm)
	}
	return err
}

// Collect gathers and returns all metric data related to the Reader from
// the SDK and other Producers and stores the result in rm. The returned metric
// data is not exported to the configured exporter, it is left to the caller to
// handle that if desired.
//
// An error is returned if this is called after Shutdown. An error is return if rm is nil.
func (r *periodicReader) Collect(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	if rm == nil {
		return errors.New("periodic reader: *metricdata.ResourceMetrics is nil")
	}
	// TODO (#3047): When collect is updated to accept output as param, pass rm.
	rmTemp, err := r.collect(ctx, r.sdkProducer.Load())
	*rm = rmTemp
	return err
}

// collect unwraps p as a produceHolder and returns its produce results.
func (r *periodicReader) collect(ctx context.Context, p interface{}) (metricdata.ResourceMetrics, error) {
	if p == nil {
		return metricdata.ResourceMetrics{}, ErrReaderNotRegistered
	}

	ph, ok := p.(produceHolder)
	if !ok {
		// The atomic.Value is entirely in the periodicReader's control so
		// this should never happen. In the unforeseen case that this does
		// happen, return an error instead of panicking so a users code does
		// not halt in the processes.
		err := fmt.Errorf("periodic reader: invalid producer: %T", p)
		return metricdata.ResourceMetrics{}, err
	}

	rm, err := ph.produce(ctx)
	if err != nil {
		return metricdata.ResourceMetrics{}, err
	}
	var errs []error
	for _, producer := range r.externalProducers.Load().([]Producer) {
		externalMetrics, err := producer.Produce(ctx)
		if err != nil {
			errs = append(errs, err)
		}
		rm.ScopeMetrics = append(rm.ScopeMetrics, externalMetrics...)
	}
	return rm, unifyErrors(errs)
}

// export exports metric data m using r's exporter.
func (r *periodicReader) export(ctx context.Context, m metricdata.ResourceMetrics) error {
	c, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.exporter.Export(c, m)
}

// ForceFlush flushes pending telemetry.
func (r *periodicReader) ForceFlush(ctx context.Context) error {
	errCh := make(chan error, 1)
	select {
	case r.flushCh <- errCh:
		select {
		case err := <-errCh:
			if err != nil {
				return err
			}
			close(errCh)
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-r.done:
		return ErrReaderShutdown
	case <-ctx.Done():
		return ctx.Err()
	}
	return r.exporter.ForceFlush(ctx)
}

// Shutdown flushes pending telemetry and then stops the export pipeline.
func (r *periodicReader) Shutdown(ctx context.Context) error {
	err := ErrReaderShutdown
	r.shutdownOnce.Do(func() {
		// Stop the run loop.
		r.cancel()
		<-r.done

		// Any future call to Collect will now return ErrReaderShutdown.
		ph := r.sdkProducer.Swap(produceHolder{
			produce: shutdownProducer{}.produce,
		})

		if ph != nil { // Reader was registered.
			// Flush pending telemetry.
			var m metricdata.ResourceMetrics
			m, err = r.collect(ctx, ph)
			if err == nil {
				err = r.export(ctx, m)
			}
		}

		sErr := r.exporter.Shutdown(ctx)
		if err == nil || err == ErrReaderShutdown {
			err = sErr
		}

		r.mu.Lock()
		defer r.mu.Unlock()
		r.isShutdown = true
		// release references to Producer(s)
		r.externalProducers.Store([]Producer{})
	})
	return err
}
