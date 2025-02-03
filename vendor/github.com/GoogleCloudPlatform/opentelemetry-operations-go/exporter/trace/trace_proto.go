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
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"cloud.google.com/go/trace/apiv2/tracepb"
	codepb "google.golang.org/genproto/googleapis/rpc/code"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping"
)

const (
	maxAnnotationEventsPerSpan = 32
	// TODO(ymotongpoo): uncomment this after gRPC trace get supported.
	// maxMessageEventsPerSpan    = 128.
	maxAttributeStringValue = 256
	maxNumLinks             = 128
	agentLabel              = "g.co/agent"

	// Attributes recorded on the span for the requests.
	// Only trace exporters will need them.
	hostAttribute       = "http.host"
	methodAttribute     = "http.method"
	pathAttribute       = "http.path"
	urlAttribute        = "http.url"
	userAgentAttribute  = "http.user_agent"
	statusCodeAttribute = "http.status_code"
	serviceAttribute    = "service.name"

	labelHTTPHost       = `/http/host`
	labelHTTPMethod     = `/http/method`
	labelHTTPStatusCode = `/http/status_code`
	labelHTTPPath       = `/http/path`
	labelHTTPUserAgent  = `/http/user_agent`

	instrumentationScopeNameAttribute    = "otel.scope.name"
	instrumentationScopeVersionAttribute = "otel.scope.version"
)

var userAgent = fmt.Sprintf("opentelemetry-go %s; google-cloud-trace-exporter %s", otel.Version(), Version())

// Adapters for using resourcemapping library.
type attrs struct {
	Attrs []attribute.KeyValue
}

func (a *attrs) GetString(key string) (string, bool) {
	for _, kv := range a.Attrs {
		if kv.Key == attribute.Key(key) {
			return kv.Value.AsString(), true
		}
	}
	return "", false
}

// If there are duplicate keys present in the list of attributes,
// then the first value found for the key is preserved.
func attributeWithLabelsFromResources(sd sdktrace.ReadOnlySpan) []attribute.KeyValue {
	attributes := sd.Attributes()
	if sd.Resource().Len() == 0 {
		return attributes
	}
	uniqueAttrs := make(map[attribute.Key]bool, len(sd.Attributes()))
	// Span Attributes take precedence
	for _, attr := range sd.Attributes() {
		uniqueAttrs[attr.Key] = true
	}
	// Raw resource attributes are next.
	for _, attr := range sd.Resource().Attributes() {
		if uniqueAttrs[attr.Key] {
			continue // skip resource attributes which conflict with span attributes
		}
		uniqueAttrs[attr.Key] = true
		attributes = append(attributes, attr)
	}
	// Instrumentation Scope attributes come next.
	if !uniqueAttrs[instrumentationScopeNameAttribute] {
		uniqueAttrs[instrumentationScopeNameAttribute] = true
		scopeNameAttrs := attribute.String(instrumentationScopeNameAttribute, sd.InstrumentationScope().Name)
		attributes = append(attributes, scopeNameAttrs)
	}
	if !uniqueAttrs[instrumentationScopeVersionAttribute] && strings.Compare("", sd.InstrumentationScope().Version) != 0 {
		uniqueAttrs[instrumentationScopeVersionAttribute] = true
		scopeVersionAttrs := attribute.String(instrumentationScopeVersionAttribute, sd.InstrumentationScope().Version)
		attributes = append(attributes, scopeVersionAttrs)
	}

	// Monitored resource attributes (`g.co/r/{resource_type}/{resource_label}`) come next.
	gceResource := resourcemapping.ResourceAttributesToMonitoringMonitoredResource(&attrs{
		Attrs: sd.Resource().Attributes(),
	})
	for key, value := range gceResource.Labels {
		name := fmt.Sprintf("g.co/r/%v/%v", gceResource.Type, key)
		attributes = append(attributes, attribute.String(name, value))
	}
	return attributes
}

func (e *traceExporter) protoFromReadOnlySpan(s sdktrace.ReadOnlySpan) (*tracepb.Span, string) {
	if s == nil {
		return nil, ""
	}

	traceIDString := s.SpanContext().TraceID().String()
	spanIDString := s.SpanContext().SpanID().String()
	projectID := e.projectID
	// override project ID with gcp.project.id, if present
	attrs := s.Resource().Attributes()
	for _, attr := range attrs {
		if attr.Key == resourcemapping.ProjectIDAttributeKey {
			projectID = attr.Value.AsString()
			break
		}
	}

	sp := &tracepb.Span{
		Name:                    "projects/" + projectID + "/traces/" + traceIDString + "/spans/" + spanIDString,
		SpanId:                  spanIDString,
		DisplayName:             trunc(s.Name(), 128),
		StartTime:               timestampProto(s.StartTime()),
		EndTime:                 timestampProto(s.EndTime()),
		SameProcessAsParentSpan: &wrapperspb.BoolValue{Value: !s.Parent().IsRemote()},
		SpanKind:                convertSpanKind(s.SpanKind()),
	}
	if s.Parent().SpanID() != s.SpanContext().SpanID() && s.Parent().SpanID().IsValid() {
		sp.ParentSpanId = s.Parent().SpanID().String()
	}
	switch s.Status().Code {
	case codes.Ok:
		sp.Status = &statuspb.Status{Code: int32(codepb.Code_OK)}
	case codes.Unset:
		// Don't set status code.
	case codes.Error:
		sp.Status = &statuspb.Status{Code: int32(codepb.Code_UNKNOWN), Message: s.Status().Description}
	default:
		sp.Status = &statuspb.Status{Code: int32(codepb.Code_UNKNOWN)}
	}

	attributes := attributeWithLabelsFromResources(s)
	e.copyAttributes(&sp.Attributes, attributes)
	// NOTE(ymotongpoo): omitting copyMonitoringReesourceAttributes()

	var annotations, droppedAnnotationsCount int
	es := s.Events()
	for i, ev := range es {
		if annotations >= maxAnnotationEventsPerSpan {
			droppedAnnotationsCount = len(es) - i
			break
		}
		annotation := &tracepb.Span_TimeEvent_Annotation{Description: trunc(ev.Name, maxAttributeStringValue)}
		e.copyAttributes(&annotation.Attributes, ev.Attributes)
		event := &tracepb.Span_TimeEvent{
			Time:  timestampProto(ev.Time),
			Value: &tracepb.Span_TimeEvent_Annotation_{Annotation: annotation},
		}
		annotations++
		if sp.TimeEvents == nil {
			sp.TimeEvents = &tracepb.Span_TimeEvents{}
		}
		sp.TimeEvents.TimeEvent = append(sp.TimeEvents.TimeEvent, event)
	}

	if sp.Attributes == nil {
		sp.Attributes = &tracepb.Span_Attributes{
			AttributeMap: make(map[string]*tracepb.AttributeValue),
		}
	}

	// Only set the agent label if it is not already set. That enables the
	// OpenTelemery service/collector to set the agent label based on the library that
	// sent the span to the service.
	// TODO(jsuereth): This scenario is highly unlikely.  This would require vanilla OTLP
	// sources of tracess to be setting "g.co/agent" labels on spans.  We should confirm
	// and remove/update this code.
	if _, hasAgent := sp.Attributes.AttributeMap[agentLabel]; !hasAgent {
		sp.Attributes.AttributeMap[agentLabel] = &tracepb.AttributeValue{
			Value: &tracepb.AttributeValue_StringValue{
				StringValue: trunc(userAgent, maxAttributeStringValue),
			},
		}
	}

	// TODO(ymotongpoo): add implementations for Span_TimeEvent_MessageEvent_
	// once OTel finish implementations for gRPC.

	if droppedAnnotationsCount != 0 {
		if sp.TimeEvents == nil {
			sp.TimeEvents = &tracepb.Span_TimeEvents{}
		}
		sp.TimeEvents.DroppedAnnotationsCount = clip32(droppedAnnotationsCount)
	}

	sp.Links = e.linksProtoFromLinks(s.Links())

	return sp, projectID
}

// Converts OTel span links to Cloud Trace links proto in order. If there are
// more than maxNumLinks links, the first maxNumLinks will be taken and the rest
// dropped.
func (e *traceExporter) linksProtoFromLinks(links []sdktrace.Link) *tracepb.Span_Links {
	numLinks := len(links)
	if numLinks == 0 {
		return nil
	}

	linksPb := &tracepb.Span_Links{}
	numLinksToKeep := numLinks
	if numLinksToKeep > maxNumLinks {
		numLinksToKeep = maxNumLinks
	}

	for _, link := range links[:numLinksToKeep] {
		linkPb := &tracepb.Span_Link{
			TraceId: link.SpanContext.TraceID().String(),
			SpanId:  link.SpanContext.SpanID().String(),
			Type:    tracepb.Span_Link_TYPE_UNSPECIFIED,
		}
		e.copyAttributes(&linkPb.Attributes, link.Attributes)
		linksPb.Link = append(linksPb.Link, linkPb)
	}
	linksPb.DroppedLinksCount = clip32(numLinks - numLinksToKeep)

	return linksPb
}

// timestampProto creates a timestamp proto for a time.Time.
func timestampProto(t time.Time) *timestamppb.Timestamp {
	return &timestamppb.Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}

// copyAttributes copies a map of attributes to a proto map field.
// It creates the map if it is nil.
func (e *traceExporter) copyAttributes(out **tracepb.Span_Attributes, in []attribute.KeyValue) {
	if len(in) == 0 {
		return
	}
	if *out == nil {
		*out = &tracepb.Span_Attributes{}
	}
	if (*out).AttributeMap == nil {
		(*out).AttributeMap = make(map[string]*tracepb.AttributeValue)
	}
	var dropped int32
	for _, kv := range in {
		av := attributeValue(kv)
		if av == nil {
			continue
		}
		key := e.o.mapAttribute(kv.Key)
		if len(key) > 128 {
			dropped++
			continue
		}
		(*out).AttributeMap[string(key)] = av
	}
	(*out).DroppedAttributesCount = dropped
}

// defaultAttributeMapping maps attributes to trace attributes which are
// used by cloud trace for prominent UI functions, and keeps all others.
func defaultAttributeMapping(k attribute.Key) attribute.Key {
	switch k {
	case pathAttribute:
		return labelHTTPPath
	case hostAttribute:
		return labelHTTPHost
	case methodAttribute:
		return labelHTTPMethod
	case userAgentAttribute:
		return labelHTTPUserAgent
	case statusCodeAttribute:
		return labelHTTPStatusCode
	}
	return k
}

func attributeValue(keyValue attribute.KeyValue) *tracepb.AttributeValue {
	v := keyValue.Value
	switch v.Type() {
	case attribute.BOOL:
		return &tracepb.AttributeValue{
			Value: &tracepb.AttributeValue_BoolValue{BoolValue: v.AsBool()},
		}
	case attribute.INT64:
		return &tracepb.AttributeValue{
			Value: &tracepb.AttributeValue_IntValue{IntValue: v.AsInt64()},
		}
	case attribute.FLOAT64:
		// TODO: set double value if Google Cloud Trace support it in the future.
		return &tracepb.AttributeValue{
			Value: &tracepb.AttributeValue_StringValue{
				StringValue: trunc(strconv.FormatFloat(v.AsFloat64(), 'f', -1, 64),
					maxAttributeStringValue)},
		}
	case attribute.STRING:
		return &tracepb.AttributeValue{
			Value: &tracepb.AttributeValue_StringValue{StringValue: trunc(v.AsString(), maxAttributeStringValue)},
		}
	}
	return nil
}

// trunc returns a TruncatableString truncated to the given limit.
func trunc(s string, limit int) *tracepb.TruncatableString {
	if len(s) > limit {
		b := []byte(s[:limit])
		for {
			r, size := utf8.DecodeLastRune(b)
			if r != utf8.RuneError || size != 1 {
				break
			}
			b = b[:len(b)-1]
		}
		return &tracepb.TruncatableString{
			Value:              string(b),
			TruncatedByteCount: clip32(len(s) - len(b)),
		}
	}
	return &tracepb.TruncatableString{
		Value:              s,
		TruncatedByteCount: 0,
	}
}

// clip32 clips an int to the range of an int32.
func clip32(x int) int32 {
	if x < math.MinInt32 {
		return math.MinInt32
	}
	if x > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(x)
}

func convertSpanKind(kind trace.SpanKind) tracepb.Span_SpanKind {
	switch kind {
	case trace.SpanKindUnspecified, trace.SpanKindInternal:
		// SpanKindUnspecified is an unspecified SpanKind and is not a
		// valid SpanKind. SpanKindUnspecified should be replaced with
		// SpanKindInternal if it is received.
		return tracepb.Span_INTERNAL
	case trace.SpanKindServer:
		return tracepb.Span_SERVER
	case trace.SpanKindClient:
		return tracepb.Span_CLIENT
	case trace.SpanKindProducer:
		return tracepb.Span_PRODUCER
	case trace.SpanKindConsumer:
		return tracepb.Span_CONSUMER
	default:
		return tracepb.Span_INTERNAL
	}
}
