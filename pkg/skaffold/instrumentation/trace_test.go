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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestInitCloudTrace(t *testing.T) {
	tests := []struct {
		shouldError        bool
		traceProviderIsNil bool
		isConcurrentTrace  bool
		name               string
		traceEnvVar        string
		parentSpans        []string
		childSpans         []string
	}{
		{
			name:        "SKAFFOLD_TRACE=stdout, verify spans output to stdout and spans are sequential",
			traceEnvVar: "stdout",
			parentSpans: []string{"SequentialSpanOne", "SequentialSpanTwo"},
		},
		{
			name:              "SKAFFOLD_TRACE=stdout, verify spans output to stdout and spans are concurrent",
			traceEnvVar:       "stdout",
			parentSpans:       []string{"ConcurrentSpanOne", "ConcurrentSpanTwo"},
			isConcurrentTrace: true,
		},
		{
			name:              "SKAFFOLD_TRACE=stdout, verify spans output to stdout and parent/child relationship exists spans",
			traceEnvVar:       "stdout",
			parentSpans:       []string{"ParentSpanOne"},
			childSpans:        []string{"ChildSpanOne"},
			isConcurrentTrace: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			if len(test.traceEnvVar) > 0 {
				t.Setenv("SKAFFOLD_TRACE", test.traceEnvVar)
			}
			var b bytes.Buffer
			func() {
				ctx := context.Background()
				tp, _, err := InitTraceFromEnvVar(WithWriter(&b))
				t.CheckErrorAndDeepEqual(test.shouldError, err, test.traceProviderIsNil || test.shouldError, tp == nil)
				defer func() { _ = TracerShutdown(ctx) }()

				for _, pName := range test.parentSpans {
					ctx, endTrace := StartTrace(ctx, pName)
					for _, cName := range test.childSpans {
						_, endTrace := StartTrace(ctx, cName)
						if test.isConcurrentTrace {
							defer endTrace()
						} else {
							endTrace()
						}
					}
					if test.isConcurrentTrace {
						defer endTrace()
					} else {
						endTrace()
					}
					time.Sleep(1 * time.Millisecond)
				}
			}()
			if len(test.parentSpans) > 0 {
				var spans SpanArray
				r := bytes.NewReader(b.Bytes())
				decoder := json.NewDecoder(r)
				for {
					var span Span
					if err := decoder.Decode(&span); err != nil {
						// Break when there are no more documents to decode
						if err != io.EOF {
							t.Fatal(err)
						}
						break
					}
					spans = append(spans, span)
				}
				t.CheckTrue(len(spans) == len(test.parentSpans)+len(test.childSpans))
				for i := range spans {
					if strings.Contains(spans[i].Name, "Parent") {
						t.CheckTrue(spans[i].Childspancount > 0)
					}
					if strings.Contains(spans[i].Name, "Child") {
						// 0000000000000000 value for Spanid means parent does not exist for a span.  Should be set to parent Spanid if
						// span is a child span
						t.CheckTrue(spans[i].Parent.Spanid != "0000000000000000")
					}

					if i == 0 {
						continue
					} // skipping first span for comparing spans for sequential/concurrent tests
					lastEndtime, err := time.Parse(time.RFC3339, spans[i-1].Endtime)
					if err != nil {
						t.Errorf("unexpected error occurred parsing trace span Endtime %v: %v", b.String(), err)
					}
					startime, err := time.Parse(time.RFC3339, spans[i].Starttime)
					if err != nil {
						t.Errorf("unexpected error occurred parsing trace span Endtime %v: %v", b.String(), err)
					}
					if test.isConcurrentTrace {
						t.CheckTrue(!lastEndtime.Before(startime))
					} else {
						// sequential ordering of traces
						t.CheckTrue(lastEndtime.Before(startime))
					}
				}
			}
		})
	}
}

type SpanArray []Span

type Span struct {
	Spancontext              Spancontext            `json:"SpanContext"`
	Parent                   Parent                 `json:"Parent"`
	Spankind                 int                    `json:"SpanKind"`
	Name                     string                 `json:"Name"`
	Starttime                string                 `json:"StartTime"`
	Endtime                  string                 `json:"EndTime"`
	Attributes               interface{}            `json:"Attributes"`
	Messageevents            interface{}            `json:"MessageEvents"`
	Links                    interface{}            `json:"Links"`
	Statuscode               Status                 `json:"Status"`
	Statusmessage            string                 `json:"StatusMessage"`
	Droppedattributecount    int                    `json:"DroppedAttributeCount"`
	Droppedmessageeventcount int                    `json:"DroppedMessageEventCount"`
	Droppedlinkcount         int                    `json:"DroppedLinkCount"`
	Childspancount           int                    `json:"ChildSpanCount"`
	Resource                 []Resource             `json:"Resource"`
	Instrumentationlibrary   Instrumentationlibrary `json:"InstrumentationLibrary"`
}

type Status struct {
	Code        string `json:"Code"`
	Description string `json:"Description"`
}
type Spancontext struct {
	Traceid    string      `json:"TraceID"`
	Spanid     string      `json:"SpanID"`
	Traceflags string      `json:"TraceFlags"`
	Tracestate interface{} `json:"TraceState"`
	Remote     bool        `json:"Remote"`
}
type Parent struct {
	Traceid    string      `json:"TraceID"`
	Spanid     string      `json:"SpanID"`
	Traceflags string      `json:"TraceFlags"`
	Tracestate interface{} `json:"TraceState"`
	Remote     bool        `json:"Remote"`
}
type Value struct {
	Type  string      `json:"Type"`
	Value interface{} `json:"Value"`
}
type Resource struct {
	Key   string `json:"Key"`
	Value Value  `json:"Value"`
}
type Instrumentationlibrary struct {
	Name    string `json:"Name"`
	Version string `json:"Version"`
}
