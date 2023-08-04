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

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/internal"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

var (
	zeroInstrumentKind InstrumentKind
	zeroScope          instrumentation.Scope
)

// InstrumentKind is the identifier of a group of instruments that all
// performing the same function.
type InstrumentKind uint8

const (
	// instrumentKindUndefined is an undefined instrument kind, it should not
	// be used by any initialized type.
	instrumentKindUndefined InstrumentKind = iota // nolint:deadcode,varcheck,unused
	// InstrumentKindCounter identifies a group of instruments that record
	// increasing values synchronously with the code path they are measuring.
	InstrumentKindCounter
	// InstrumentKindUpDownCounter identifies a group of instruments that
	// record increasing and decreasing values synchronously with the code path
	// they are measuring.
	InstrumentKindUpDownCounter
	// InstrumentKindHistogram identifies a group of instruments that record a
	// distribution of values synchronously with the code path they are
	// measuring.
	InstrumentKindHistogram
	// InstrumentKindObservableCounter identifies a group of instruments that
	// record increasing values in an asynchronous callback.
	InstrumentKindObservableCounter
	// InstrumentKindObservableUpDownCounter identifies a group of instruments
	// that record increasing and decreasing values in an asynchronous
	// callback.
	InstrumentKindObservableUpDownCounter
	// InstrumentKindObservableGauge identifies a group of instruments that
	// record current values in an asynchronous callback.
	InstrumentKindObservableGauge
)

type nonComparable [0]func() // nolint: unused  // This is indeed used.

// Instrument describes properties an instrument is created with.
type Instrument struct {
	// Name is the human-readable identifier of the instrument.
	Name string
	// Description describes the purpose of the instrument.
	Description string
	// Kind defines the functional group of the instrument.
	Kind InstrumentKind
	// Unit is the unit of measurement recorded by the instrument.
	Unit string
	// Scope identifies the instrumentation that created the instrument.
	Scope instrumentation.Scope

	// Ensure forward compatibility if non-comparable fields need to be added.
	nonComparable // nolint: unused
}

// empty returns if all fields of i are their zero-value.
func (i Instrument) empty() bool {
	return i.Name == "" &&
		i.Description == "" &&
		i.Kind == zeroInstrumentKind &&
		i.Unit == "" &&
		i.Scope == zeroScope
}

// matches returns whether all the non-zero-value fields of i match the
// corresponding fields of other. If i is empty it will match all other, and
// true will always be returned.
func (i Instrument) matches(other Instrument) bool {
	return i.matchesName(other) &&
		i.matchesDescription(other) &&
		i.matchesKind(other) &&
		i.matchesUnit(other) &&
		i.matchesScope(other)
}

// matchesName returns true if the Name of i is "" or it equals the Name of
// other, otherwise false.
func (i Instrument) matchesName(other Instrument) bool {
	return i.Name == "" || i.Name == other.Name
}

// matchesDescription returns true if the Description of i is "" or it equals
// the Description of other, otherwise false.
func (i Instrument) matchesDescription(other Instrument) bool {
	return i.Description == "" || i.Description == other.Description
}

// matchesKind returns true if the Kind of i is its zero-value or it equals the
// Kind of other, otherwise false.
func (i Instrument) matchesKind(other Instrument) bool {
	return i.Kind == zeroInstrumentKind || i.Kind == other.Kind
}

// matchesUnit returns true if the Unit of i is its zero-value or it equals the
// Unit of other, otherwise false.
func (i Instrument) matchesUnit(other Instrument) bool {
	return i.Unit == "" || i.Unit == other.Unit
}

// matchesScope returns true if the Scope of i is its zero-value or it equals
// the Scope of other, otherwise false.
func (i Instrument) matchesScope(other Instrument) bool {
	return (i.Scope.Name == "" || i.Scope.Name == other.Scope.Name) &&
		(i.Scope.Version == "" || i.Scope.Version == other.Scope.Version) &&
		(i.Scope.SchemaURL == "" || i.Scope.SchemaURL == other.Scope.SchemaURL)
}

// Stream describes the stream of data an instrument produces.
type Stream struct {
	// Name is the human-readable identifier of the stream.
	Name string
	// Description describes the purpose of the data.
	Description string
	// Unit is the unit of measurement recorded.
	Unit string
	// Aggregation the stream uses for an instrument.
	Aggregation aggregation.Aggregation
	// AttributeFilter applied to all attributes recorded for an instrument.
	AttributeFilter attribute.Filter
}

// streamID are the identifying properties of a stream.
type streamID struct {
	// Name is the name of the stream.
	Name string
	// Description is the description of the stream.
	Description string
	// Unit is the unit of the stream.
	Unit string
	// Aggregation is the aggregation data type of the stream.
	Aggregation string
	// Monotonic is the monotonicity of an instruments data type. This field is
	// not used for all data types, so a zero value needs to be understood in the
	// context of Aggregation.
	Monotonic bool
	// Temporality is the temporality of a stream's data type. This field is
	// not used by some data types.
	Temporality metricdata.Temporality
	// Number is the number type of the stream.
	Number string
}

type instrumentImpl[N int64 | float64] struct {
	instrument.Synchronous

	aggregators []internal.Aggregator[N]
}

var _ instrument.Float64Counter = (*instrumentImpl[float64])(nil)
var _ instrument.Float64UpDownCounter = (*instrumentImpl[float64])(nil)
var _ instrument.Float64Histogram = (*instrumentImpl[float64])(nil)
var _ instrument.Int64Counter = (*instrumentImpl[int64])(nil)
var _ instrument.Int64UpDownCounter = (*instrumentImpl[int64])(nil)
var _ instrument.Int64Histogram = (*instrumentImpl[int64])(nil)

func (i *instrumentImpl[N]) Add(ctx context.Context, val N, attrs ...attribute.KeyValue) {
	i.aggregate(ctx, val, attrs)
}

func (i *instrumentImpl[N]) Record(ctx context.Context, val N, attrs ...attribute.KeyValue) {
	i.aggregate(ctx, val, attrs)
}

func (i *instrumentImpl[N]) aggregate(ctx context.Context, val N, attrs []attribute.KeyValue) {
	if err := ctx.Err(); err != nil {
		return
	}
	for _, agg := range i.aggregators {
		agg.Aggregate(val, attribute.NewSet(attrs...))
	}
}

// observablID is a comparable unique identifier of an observable.
type observablID[N int64 | float64] struct {
	name        string
	description string
	kind        InstrumentKind
	unit        string
	scope       instrumentation.Scope
}

type float64Observable struct {
	instrument.Float64Observable
	*observable[float64]
}

var _ instrument.Float64ObservableCounter = float64Observable{}
var _ instrument.Float64ObservableUpDownCounter = float64Observable{}
var _ instrument.Float64ObservableGauge = float64Observable{}

func newFloat64Observable(scope instrumentation.Scope, kind InstrumentKind, name, desc, u string, agg []internal.Aggregator[float64]) float64Observable {
	return float64Observable{
		observable: newObservable[float64](scope, kind, name, desc, u, agg),
	}
}

type int64Observable struct {
	instrument.Int64Observable
	*observable[int64]
}

var _ instrument.Int64ObservableCounter = int64Observable{}
var _ instrument.Int64ObservableUpDownCounter = int64Observable{}
var _ instrument.Int64ObservableGauge = int64Observable{}

func newInt64Observable(scope instrumentation.Scope, kind InstrumentKind, name, desc, u string, agg []internal.Aggregator[int64]) int64Observable {
	return int64Observable{
		observable: newObservable[int64](scope, kind, name, desc, u, agg),
	}
}

type observable[N int64 | float64] struct {
	instrument.Asynchronous
	observablID[N]

	aggregators []internal.Aggregator[N]
}

func newObservable[N int64 | float64](scope instrumentation.Scope, kind InstrumentKind, name, desc, u string, agg []internal.Aggregator[N]) *observable[N] {
	return &observable[N]{
		observablID: observablID[N]{
			name:        name,
			description: desc,
			kind:        kind,
			unit:        u,
			scope:       scope,
		},
		aggregators: agg,
	}
}

// observe records the val for the set of attrs.
func (o *observable[N]) observe(val N, attrs []attribute.KeyValue) {
	for _, agg := range o.aggregators {
		agg.Aggregate(val, attribute.NewSet(attrs...))
	}
}

var errEmptyAgg = errors.New("no aggregators for observable instrument")

// registerable returns an error if the observable o should not be registered,
// and nil if it should. An errEmptyAgg error is returned if o is effecively a
// no-op because it does not have any aggregators. Also, an error is returned
// if scope defines a Meter other than the one o was created by.
func (o *observable[N]) registerable(scope instrumentation.Scope) error {
	if len(o.aggregators) == 0 {
		return errEmptyAgg
	}
	if scope != o.scope {
		return fmt.Errorf(
			"invalid registration: observable %q from Meter %q, registered with Meter %q",
			o.name,
			o.scope.Name,
			scope.Name,
		)
	}
	return nil
}
