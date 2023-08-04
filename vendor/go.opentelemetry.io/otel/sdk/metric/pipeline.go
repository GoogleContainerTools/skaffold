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
	"container/list"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/internal/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/internal"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

var (
	errCreatingAggregators     = errors.New("could not create all aggregators")
	errIncompatibleAggregation = errors.New("incompatible aggregation")
	errUnknownAggregation      = errors.New("unrecognized aggregation")
	errUnknownTemporality      = errors.New("unrecognized temporality")
)

type aggregator interface {
	Aggregation() metricdata.Aggregation
}

// instrumentSync is a synchronization point between a pipeline and an
// instrument's Aggregators.
type instrumentSync struct {
	name        string
	description string
	unit        string
	aggregator  aggregator
}

func newPipeline(res *resource.Resource, reader Reader, views []View) *pipeline {
	if res == nil {
		res = resource.Empty()
	}
	return &pipeline{
		resource:     res,
		reader:       reader,
		views:        views,
		aggregations: make(map[instrumentation.Scope][]instrumentSync),
	}
}

// pipeline connects all of the instruments created by a meter provider to a Reader.
// This is the object that will be `Reader.register()` when a meter provider is created.
//
// As instruments are created the instrument should be checked if it exists in the
// views of a the Reader, and if so each aggregator should be added to the pipeline.
type pipeline struct {
	resource *resource.Resource

	reader Reader
	views  []View

	sync.Mutex
	aggregations   map[instrumentation.Scope][]instrumentSync
	callbacks      []func(context.Context) error
	multiCallbacks list.List
}

// addSync adds the instrumentSync to pipeline p with scope. This method is not
// idempotent. Duplicate calls will result in duplicate additions, it is the
// callers responsibility to ensure this is called with unique values.
func (p *pipeline) addSync(scope instrumentation.Scope, iSync instrumentSync) {
	p.Lock()
	defer p.Unlock()
	if p.aggregations == nil {
		p.aggregations = map[instrumentation.Scope][]instrumentSync{
			scope: {iSync},
		}
		return
	}
	p.aggregations[scope] = append(p.aggregations[scope], iSync)
}

// addCallback registers a single instrument callback to be run when
// `produce()` is called.
func (p *pipeline) addCallback(cback func(context.Context) error) {
	p.Lock()
	defer p.Unlock()
	p.callbacks = append(p.callbacks, cback)
}

type multiCallback func(context.Context) error

// addMultiCallback registers a multi-instrument callback to be run when
// `produce()` is called.
func (p *pipeline) addMultiCallback(c multiCallback) (unregister func()) {
	p.Lock()
	defer p.Unlock()
	e := p.multiCallbacks.PushBack(c)
	return func() {
		p.Lock()
		p.multiCallbacks.Remove(e)
		p.Unlock()
	}
}

// produce returns aggregated metrics from a single collection.
//
// This method is safe to call concurrently.
func (p *pipeline) produce(ctx context.Context) (metricdata.ResourceMetrics, error) {
	p.Lock()
	defer p.Unlock()

	var errs multierror
	for _, c := range p.callbacks {
		// TODO make the callbacks parallel. ( #3034 )
		if err := c(ctx); err != nil {
			errs.append(err)
		}
		if err := ctx.Err(); err != nil {
			return metricdata.ResourceMetrics{}, err
		}
	}
	for e := p.multiCallbacks.Front(); e != nil; e = e.Next() {
		// TODO make the callbacks parallel. ( #3034 )
		f := e.Value.(multiCallback)
		if err := f(ctx); err != nil {
			errs.append(err)
		}
		if err := ctx.Err(); err != nil {
			// This means the context expired before we finished running callbacks.
			return metricdata.ResourceMetrics{}, err
		}
	}

	sm := make([]metricdata.ScopeMetrics, 0, len(p.aggregations))
	for scope, instruments := range p.aggregations {
		metrics := make([]metricdata.Metrics, 0, len(instruments))
		for _, inst := range instruments {
			data := inst.aggregator.Aggregation()
			if data != nil {
				metrics = append(metrics, metricdata.Metrics{
					Name:        inst.name,
					Description: inst.description,
					Unit:        inst.unit,
					Data:        data,
				})
			}
		}
		if len(metrics) > 0 {
			sm = append(sm, metricdata.ScopeMetrics{
				Scope:   scope,
				Metrics: metrics,
			})
		}
	}

	return metricdata.ResourceMetrics{
		Resource:     p.resource,
		ScopeMetrics: sm,
	}, errs.errorOrNil()
}

// inserter facilitates inserting of new instruments from a single scope into a
// pipeline.
type inserter[N int64 | float64] struct {
	// aggregators is a cache that holds Aggregators inserted into the
	// underlying reader pipeline. This cache ensures no duplicate Aggregators
	// are inserted into the reader pipeline and if a new request during an
	// instrument creation asks for the same Aggregator the same instance is
	// returned.
	aggregators *cache[streamID, aggVal[N]]

	// views is a cache that holds instrument identifiers for all the
	// instruments a Meter has created, it is provided from the Meter that owns
	// this inserter. This cache ensures during the creation of instruments
	// with the same name but different options (e.g. description, unit) a
	// warning message is logged.
	views *cache[string, streamID]

	pipeline *pipeline
}

func newInserter[N int64 | float64](p *pipeline, vc *cache[string, streamID]) *inserter[N] {
	if vc == nil {
		vc = &cache[string, streamID]{}
	}
	return &inserter[N]{
		aggregators: &cache[streamID, aggVal[N]]{},
		views:       vc,
		pipeline:    p,
	}
}

// Instrument inserts the instrument inst with instUnit into a pipeline. All
// views the pipeline contains are matched against, and any matching view that
// creates a unique Aggregator will be inserted into the pipeline and included
// in the returned slice.
//
// The returned Aggregators are ensured to be deduplicated and unique. If
// another view in another pipeline that is cached by this inserter's cache has
// already inserted the same Aggregator for the same instrument, that
// Aggregator instance is returned.
//
// If another instrument has already been inserted by this inserter, or any
// other using the same cache, and it conflicts with the instrument being
// inserted in this call, an Aggregator matching the arguments will still be
// returned but an Info level log message will also be logged to the OTel
// global logger.
//
// If the passed instrument would result in an incompatible Aggregator, an
// error is returned and that Aggregator is not inserted or returned.
//
// If an instrument is determined to use a Drop aggregation, that instrument is
// not inserted nor returned.
func (i *inserter[N]) Instrument(inst Instrument) ([]internal.Aggregator[N], error) {
	var (
		matched bool
		aggs    []internal.Aggregator[N]
	)

	errs := &multierror{wrapped: errCreatingAggregators}
	// The cache will return the same Aggregator instance. Use this fact to
	// compare pointer addresses to deduplicate Aggregators.
	seen := make(map[internal.Aggregator[N]]struct{})
	for _, v := range i.pipeline.views {
		stream, match := v(inst)
		if !match {
			continue
		}
		matched = true

		agg, err := i.cachedAggregator(inst.Scope, inst.Kind, stream)
		if err != nil {
			errs.append(err)
		}
		if agg == nil { // Drop aggregator.
			continue
		}
		if _, ok := seen[agg]; ok {
			// This aggregator has already been added.
			continue
		}
		seen[agg] = struct{}{}
		aggs = append(aggs, agg)
	}

	if matched {
		return aggs, errs.errorOrNil()
	}

	// Apply implicit default view if no explicit matched.
	stream := Stream{
		Name:        inst.Name,
		Description: inst.Description,
		Unit:        inst.Unit,
	}
	agg, err := i.cachedAggregator(inst.Scope, inst.Kind, stream)
	if err != nil {
		errs.append(err)
	}
	if agg != nil {
		// Ensured to have not seen given matched was false.
		aggs = append(aggs, agg)
	}
	return aggs, errs.errorOrNil()
}

// aggVal is the cached value in an aggregators cache.
type aggVal[N int64 | float64] struct {
	Aggregator internal.Aggregator[N]
	Err        error
}

// cachedAggregator returns the appropriate Aggregator for an instrument
// configuration. If the exact instrument has been created within the
// inst.Scope, that Aggregator instance will be returned. Otherwise, a new
// computed Aggregator will be cached and returned.
//
// If the instrument configuration conflicts with an instrument that has
// already been created (e.g. description, unit, data type) a warning will be
// logged at the "Info" level with the global OTel logger. A valid new
// Aggregator for the instrument configuration will still be returned without
// an error.
//
// If the instrument defines an unknown or incompatible aggregation, an error
// is returned.
func (i *inserter[N]) cachedAggregator(scope instrumentation.Scope, kind InstrumentKind, stream Stream) (internal.Aggregator[N], error) {
	switch stream.Aggregation.(type) {
	case nil, aggregation.Default:
		// Undefined, nil, means to use the default from the reader.
		stream.Aggregation = i.pipeline.reader.aggregation(kind)
	}

	if err := isAggregatorCompatible(kind, stream.Aggregation); err != nil {
		return nil, fmt.Errorf(
			"creating aggregator with instrumentKind: %d, aggregation %v: %w",
			kind, stream.Aggregation, err,
		)
	}

	id := i.streamID(kind, stream)
	// If there is a conflict, the specification says the view should
	// still be applied and a warning should be logged.
	i.logConflict(id)
	cv := i.aggregators.Lookup(id, func() aggVal[N] {
		agg, err := i.aggregator(stream.Aggregation, kind, id.Temporality, id.Monotonic)
		if err != nil {
			return aggVal[N]{nil, err}
		}
		if agg == nil { // Drop aggregator.
			return aggVal[N]{nil, nil}
		}
		if stream.AttributeFilter != nil {
			agg = internal.NewFilter(agg, stream.AttributeFilter)
		}

		i.pipeline.addSync(scope, instrumentSync{
			name:        stream.Name,
			description: stream.Description,
			unit:        stream.Unit,
			aggregator:  agg,
		})
		return aggVal[N]{agg, err}
	})
	return cv.Aggregator, cv.Err
}

// logConflict validates if an instrument with the same name as id has already
// been created. If that instrument conflicts with id, a warning is logged.
func (i *inserter[N]) logConflict(id streamID) {
	existing := i.views.Lookup(id.Name, func() streamID { return id })
	if id == existing {
		return
	}

	global.Info(
		"duplicate metric stream definitions",
		"names", fmt.Sprintf("%q, %q", existing.Name, id.Name),
		"descriptions", fmt.Sprintf("%q, %q", existing.Description, id.Description),
		"units", fmt.Sprintf("%s, %s", existing.Unit, id.Unit),
		"numbers", fmt.Sprintf("%s, %s", existing.Number, id.Number),
		"aggregations", fmt.Sprintf("%s, %s", existing.Aggregation, id.Aggregation),
		"monotonics", fmt.Sprintf("%t, %t", existing.Monotonic, id.Monotonic),
		"temporalities", fmt.Sprintf("%s, %s", existing.Temporality.String(), id.Temporality.String()),
	)
}

func (i *inserter[N]) streamID(kind InstrumentKind, stream Stream) streamID {
	var zero N
	id := streamID{
		Name:        stream.Name,
		Description: stream.Description,
		Unit:        stream.Unit,
		Aggregation: fmt.Sprintf("%T", stream.Aggregation),
		Temporality: i.pipeline.reader.temporality(kind),
		Number:      fmt.Sprintf("%T", zero),
	}

	switch kind {
	case InstrumentKindObservableCounter, InstrumentKindCounter, InstrumentKindHistogram:
		id.Monotonic = true
	}

	return id
}

// aggregator returns a new Aggregator matching agg, kind, temporality, and
// monotonic. If the agg is unknown or temporality is invalid, an error is
// returned.
func (i *inserter[N]) aggregator(agg aggregation.Aggregation, kind InstrumentKind, temporality metricdata.Temporality, monotonic bool) (internal.Aggregator[N], error) {
	switch a := agg.(type) {
	case aggregation.Drop:
		return nil, nil
	case aggregation.LastValue:
		return internal.NewLastValue[N](), nil
	case aggregation.Sum:
		switch kind {
		case InstrumentKindObservableCounter, InstrumentKindObservableUpDownCounter:
			// Asynchronous counters and up-down-counters are defined to record
			// the absolute value of the count:
			// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/metrics/api.md#asynchronous-counter-creation
			switch temporality {
			case metricdata.CumulativeTemporality:
				return internal.NewPrecomputedCumulativeSum[N](monotonic), nil
			case metricdata.DeltaTemporality:
				return internal.NewPrecomputedDeltaSum[N](monotonic), nil
			default:
				return nil, fmt.Errorf("%w: %s(%d)", errUnknownTemporality, temporality.String(), temporality)
			}
		}

		switch temporality {
		case metricdata.CumulativeTemporality:
			return internal.NewCumulativeSum[N](monotonic), nil
		case metricdata.DeltaTemporality:
			return internal.NewDeltaSum[N](monotonic), nil
		default:
			return nil, fmt.Errorf("%w: %s(%d)", errUnknownTemporality, temporality.String(), temporality)
		}
	case aggregation.ExplicitBucketHistogram:
		switch temporality {
		case metricdata.CumulativeTemporality:
			return internal.NewCumulativeHistogram[N](a), nil
		case metricdata.DeltaTemporality:
			return internal.NewDeltaHistogram[N](a), nil
		default:
			return nil, fmt.Errorf("%w: %s(%d)", errUnknownTemporality, temporality.String(), temporality)
		}
	}
	return nil, errUnknownAggregation
}

// isAggregatorCompatible checks if the aggregation can be used by the instrument.
// Current compatibility:
//
// | Instrument Kind          | Drop | LastValue | Sum | Histogram | Exponential Histogram |
// |--------------------------|------|-----------|-----|-----------|-----------------------|
// | Counter                  | X    |           | X   | X         | X                     |
// | UpDownCounter            | X    |           | X   |           |                       |
// | Histogram                | X    |           | X   | X         | X                     |
// | Observable Counter       | X    |           | X   |           |                       |
// | Observable UpDownCounter | X    |           | X   |           |                       |
// | Observable Gauge         | X    | X         |     |           |                       |.
func isAggregatorCompatible(kind InstrumentKind, agg aggregation.Aggregation) error {
	switch agg.(type) {
	case aggregation.ExplicitBucketHistogram:
		if kind == InstrumentKindCounter || kind == InstrumentKindHistogram {
			return nil
		}
		// TODO: review need for aggregation check after
		// https://github.com/open-telemetry/opentelemetry-specification/issues/2710
		return errIncompatibleAggregation
	case aggregation.Sum:
		switch kind {
		case InstrumentKindObservableCounter, InstrumentKindObservableUpDownCounter, InstrumentKindCounter, InstrumentKindHistogram, InstrumentKindUpDownCounter:
			return nil
		default:
			// TODO: review need for aggregation check after
			// https://github.com/open-telemetry/opentelemetry-specification/issues/2710
			return errIncompatibleAggregation
		}
	case aggregation.LastValue:
		if kind == InstrumentKindObservableGauge {
			return nil
		}
		// TODO: review need for aggregation check after
		// https://github.com/open-telemetry/opentelemetry-specification/issues/2710
		return errIncompatibleAggregation
	case aggregation.Drop:
		return nil
	default:
		// This is used passed checking for default, it should be an error at this point.
		return fmt.Errorf("%w: %v", errUnknownAggregation, agg)
	}
}

// pipelines is the group of pipelines connecting Readers with instrument
// measurement.
type pipelines []*pipeline

func newPipelines(res *resource.Resource, readers []Reader, views []View) pipelines {
	pipes := make([]*pipeline, 0, len(readers))
	for _, r := range readers {
		p := &pipeline{
			resource: res,
			reader:   r,
			views:    views,
		}
		r.register(p)
		pipes = append(pipes, p)
	}
	return pipes
}

func (p pipelines) registerCallback(cback func(context.Context) error) {
	for _, pipe := range p {
		pipe.addCallback(cback)
	}
}

func (p pipelines) registerMultiCallback(c multiCallback) metric.Registration {
	unregs := make([]func(), len(p))
	for i, pipe := range p {
		unregs[i] = pipe.addMultiCallback(c)
	}
	return unregisterFuncs(unregs)
}

type unregisterFuncs []func()

func (u unregisterFuncs) Unregister() error {
	for _, f := range u {
		f()
	}
	return nil
}

// resolver facilitates resolving Aggregators an instrument needs to aggregate
// measurements with while updating all pipelines that need to pull from those
// aggregations.
type resolver[N int64 | float64] struct {
	inserters []*inserter[N]
}

func newResolver[N int64 | float64](p pipelines, vc *cache[string, streamID]) resolver[N] {
	in := make([]*inserter[N], len(p))
	for i := range in {
		in[i] = newInserter[N](p[i], vc)
	}
	return resolver[N]{in}
}

// Aggregators returns the Aggregators that must be updated by the instrument
// defined by key.
func (r resolver[N]) Aggregators(id Instrument) ([]internal.Aggregator[N], error) {
	var aggs []internal.Aggregator[N]

	errs := &multierror{}
	for _, i := range r.inserters {
		a, err := i.Instrument(id)
		if err != nil {
			errs.append(err)
		}
		aggs = append(aggs, a...)
	}
	return aggs, errs.errorOrNil()
}

type multierror struct {
	wrapped error
	errors  []string
}

func (m *multierror) errorOrNil() error {
	if len(m.errors) == 0 {
		return nil
	}
	if m.wrapped == nil {
		return errors.New(strings.Join(m.errors, "; "))
	}
	return fmt.Errorf("%w: %s", m.wrapped, strings.Join(m.errors, "; "))
}

func (m *multierror) append(err error) {
	m.errors = append(m.errors, err.Error())
}
