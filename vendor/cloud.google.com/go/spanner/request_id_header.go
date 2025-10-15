// Copyright 2024 Google LLC
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

package spanner

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/googleapis/gax-go/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// randIDForProcess is a strongly randomly generated value derived
// from a uint64, and in the range [0, maxUint64].
var randIDForProcess string

func init() {
	bigMaxInt64, _ := new(big.Int).SetString(fmt.Sprintf("%d", uint64(math.MaxUint64)), 10)
	if g, w := bigMaxInt64.Uint64(), uint64(math.MaxUint64); g != w {
		panic(fmt.Sprintf("mismatch in randIDForProcess.maxUint64:\n\tGot:  %d\n\tWant: %d", g, w))
	}
	r64, err := rand.Int(rand.Reader, bigMaxInt64)
	if err != nil {
		panic(err)
	}
	randIDForProcess = fmt.Sprintf("%016x", r64.Uint64())
}

// Please bump this version whenever this implementation
// executes on the plans of a new specification.
const xSpannerRequestIDVersion uint8 = 1

const xSpannerRequestIDHeader = "x-goog-spanner-request-id"
const xSpannerRequestIDSpanAttr = "x_goog_spanner_request_id"

// optsWithNextRequestID bundles priors with a new header "x-goog-spanner-request-id"
func (g *grpcSpannerClient) optsWithNextRequestID(priors []gax.CallOption) []gax.CallOption {
	return append(priors, &retryerWithRequestID{g})
}

func (g *grpcSpannerClient) prepareRequestIDTrackers(clientID int, channelID uint64, nthRequest *atomic.Uint32) {
	g.id = clientID // The ID derived from the SpannerClient.
	g.channelID = channelID
	g.nthRequest = nthRequest
}

// retryerWithRequestID is a gax.CallOption that injects "x-goog-spanner-request-id"
// into every RPC, and it appropriately increments the RPC's ordinal number per retry.
type retryerWithRequestID struct {
	gsc *grpcSpannerClient
}

var _ gax.CallOption = (*retryerWithRequestID)(nil)

func (g *grpcSpannerClient) appendRequestIDToGRPCOptions(priors []grpc.CallOption, nthRequest, attempt uint32) []grpc.CallOption {
	// Each value should be added in Decimal, unpadded.
	requestID := fmt.Sprintf("%d.%s.%d.%d.%d.%d", xSpannerRequestIDVersion, randIDForProcess, g.id, g.channelID, nthRequest, attempt)
	md := metadata.MD{xSpannerRequestIDHeader: []string{requestID}}
	return append(priors, grpc.Header(&md))
}

type requestID string

// augmentErrorWithRequestID introspects error converting it to an *.Error and
// attaching the subject requestID, unless it is one of the following:
// * nil
// * context.Canceled
// * io.EOF
// * iterator.Done
// of which in this case, the original error will be attached as is, since those
// are sentinel errors used to break sensitive conditions like ending iterations.
func (r requestID) augmentErrorWithRequestID(err error) error {
	if err == nil {
		return nil
	}

	switch err {
	case iterator.Done, io.EOF, context.Canceled:
		return err

	default:
		potentialCommit := errors.Is(err, context.DeadlineExceeded)
		if code := status.Code(err); code == codes.DeadlineExceeded {
			potentialCommit = true
		}
		sErr := toSpannerErrorWithCommitInfo(err, potentialCommit)
		if sErr == nil {
			return err
		}

		spErr := sErr.(*Error)
		spErr.RequestID = string(r)
		return spErr
	}
}

func gRPCCallOptionsToRequestID(opts []grpc.CallOption) (md metadata.MD, reqID requestID, found bool) {
	for _, opt := range opts {
		hdrOpt, ok := opt.(grpc.HeaderCallOption)
		if !ok {
			continue
		}

		metadata := hdrOpt.HeaderAddr
		reqIDs := metadata.Get(xSpannerRequestIDHeader)
		if len(reqIDs) != 0 && len(reqIDs[0]) != 0 {
			md = *metadata
			reqID = requestID(reqIDs[0])
			found = true
			break
		}
	}
	return
}

func (wr *requestIDHeaderInjector) interceptUnary(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// It is imperative to search for the requestID before the call
	// because gRPC's internals will consume the headers.
	_, reqID, foundRequestID := gRPCCallOptionsToRequestID(opts)
	if foundRequestID {
		ctx = metadata.AppendToOutgoingContext(ctx, xSpannerRequestIDHeader, string(reqID))

		// Associate the requestId as an attribute on the span in the current context.
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.KeyValue{
			Key:   xSpannerRequestIDSpanAttr,
			Value: attribute.StringValue(string(reqID)),
		})
	}

	err := invoker(ctx, method, req, reply, cc, opts...)
	if !foundRequestID {
		return err
	}
	return reqID.augmentErrorWithRequestID(err)
}

type requestIDErrWrappingClientStream struct {
	grpc.ClientStream
	reqID requestID
}

func (rew *requestIDErrWrappingClientStream) processFromOutgoingContext(err error) error {
	if err == nil {
		return nil
	}
	return rew.reqID.augmentErrorWithRequestID(err)
}

func (rew *requestIDErrWrappingClientStream) SendMsg(msg any) error {
	err := rew.ClientStream.SendMsg(msg)
	return rew.processFromOutgoingContext(err)
}

func (rew *requestIDErrWrappingClientStream) RecvMsg(msg any) error {
	err := rew.ClientStream.RecvMsg(msg)
	return rew.processFromOutgoingContext(err)
}

var _ grpc.ClientStream = (*requestIDErrWrappingClientStream)(nil)

type requestIDHeaderInjector int

func (wr *requestIDHeaderInjector) interceptStream(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	// It is imperative to search for the requestID before the call
	// because gRPC's internals will consume the headers.
	_, reqID, foundRequestID := gRPCCallOptionsToRequestID(opts)
	if foundRequestID {
		ctx = metadata.AppendToOutgoingContext(ctx, xSpannerRequestIDHeader, string(reqID))

		// Associate the requestId as an attribute on the span in the current context.
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.KeyValue{
			Key:   xSpannerRequestIDSpanAttr,
			Value: attribute.StringValue(string(reqID)),
		})
	}

	cs, err := streamer(ctx, desc, cc, method, opts...)
	if !foundRequestID {
		return cs, err
	}
	wcs := &requestIDErrWrappingClientStream{cs, reqID}
	if err == nil {
		return wcs, nil
	}

	return wcs, reqID.augmentErrorWithRequestID(err)
}

func (wr *retryerWithRequestID) Resolve(cs *gax.CallSettings) {
	nthRequest := wr.gsc.nextNthRequest()
	attempt := uint32(1)
	// Inject the first request-id header.
	// Note: after every gax.Invoke call, all the gRPC option headers are cleared out
	// and nullified, but yet cs.GRPC still contains a reference to the inserted *metadata.MD
	// just that it got cleared out and nullified. However, for retries we need to retain control
	// of the entry to re-insert the updated request-id on every call, hence why we are creating
	// and retaining a pointer reference to the metadata and shall be re-inserting the header value
	// on every retry.
	md := new(metadata.MD)
	wr.generateAndInsertRequestID(md, nthRequest, attempt)
	// Insert our grpc.CallOption that'll be updated by reference on every retry attempt.
	cs.GRPC = append(cs.GRPC, grpc.Header(md))

	if cs.Retry == nil {
		// If there was no retry manager, our journey has ended.
		return
	}

	originalRetryer := cs.Retry()
	newRetryer := func() gax.Retryer {
		return (wrapRetryFn)(func(err error) (pause time.Duration, shouldRetry bool) {
			attempt++
			wr.generateAndInsertRequestID(md, nthRequest, attempt)
			return originalRetryer.Retry(err)
		})
	}
	cs.Retry = newRetryer
}

func (wr *retryerWithRequestID) generateAndInsertRequestID(md *metadata.MD, nthRequest, attempt uint32) {
	wr.gsc.generateAndInsertRequestID(md, nthRequest, attempt)
}

func (gsc *grpcSpannerClient) generateAndInsertRequestID(md *metadata.MD, nthRequest, attempt uint32) {
	// Google Engineering has requested that each value be added in Decimal unpadded.
	// Should we have a standardized endianness: Little Endian or Big Endian?
	reqID := fmt.Sprintf("%d.%s.%d.%d.%d.%d", xSpannerRequestIDVersion, randIDForProcess, gsc.id, gsc.channelID, nthRequest, attempt)
	if *md == nil {
		*md = metadata.MD{}
	}
	md.Set(xSpannerRequestIDHeader, reqID)
}

type wrapRetryFn func(err error) (time.Duration, bool)

var _ gax.Retryer = (wrapRetryFn)(nil)

func (fn wrapRetryFn) Retry(err error) (time.Duration, bool) {
	return fn(err)
}

func (g *grpcSpannerClient) nextNthRequest() uint32 {
	return g.nthRequest.Add(1)
}

type requestIDWrap struct {
	md         *metadata.MD
	nthRequest uint32
	gsc        *grpcSpannerClient
}

func (gsc *grpcSpannerClient) generateRequestIDHeaderInjector() *requestIDWrap {
	// Setup and track x-goog-request-id.
	md := new(metadata.MD)
	return &requestIDWrap{md: md, nthRequest: gsc.nextNthRequest(), gsc: gsc}
}

func (riw *requestIDWrap) withNextRetryAttempt(attempt uint32) gax.CallOption {
	riw.gsc.generateAndInsertRequestID(riw.md, riw.nthRequest, attempt)
	// If no gRPC stream is available, try to initiate one.
	return gax.WithGRPCOptions(grpc.Header(riw.md))
}
