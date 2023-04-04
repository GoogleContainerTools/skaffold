package ctpolicy

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/ctpolicy/loglist"
	berrors "github.com/letsencrypt/boulder/errors"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	pubpb "github.com/letsencrypt/boulder/publisher/proto"
	"github.com/letsencrypt/boulder/test"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

type mockPub struct{}

func (mp *mockPub) SubmitToSingleCTWithResult(_ context.Context, _ *pubpb.Request, _ ...grpc.CallOption) (*pubpb.Result, error) {
	return &pubpb.Result{Sct: []byte{0}}, nil
}

type mockFailPub struct{}

func (mp *mockFailPub) SubmitToSingleCTWithResult(_ context.Context, _ *pubpb.Request, _ ...grpc.CallOption) (*pubpb.Result, error) {
	return nil, errors.New("BAD")
}

type mockSlowPub struct{}

func (mp *mockSlowPub) SubmitToSingleCTWithResult(ctx context.Context, _ *pubpb.Request, _ ...grpc.CallOption) (*pubpb.Result, error) {
	<-ctx.Done()
	return nil, errors.New("timed out")
}

func TestGetSCTs(t *testing.T) {
	expired, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()
	missingSCTErr := berrors.MissingSCTs
	testCases := []struct {
		name       string
		mock       pubpb.PublisherClient
		groups     loglist.List
		ctx        context.Context
		result     core.SCTDERs
		expectErr  string
		berrorType *berrors.ErrorType
	}{
		{
			name: "basic success case",
			mock: &mockPub{},
			groups: loglist.List{
				"OperA": {
					"LogA1": {Url: "UrlA1", Key: "KeyA1"},
					"LogA2": {Url: "UrlA2", Key: "KeyA2"},
				},
				"OperB": {
					"LogB1": {Url: "UrlB1", Key: "KeyB1"},
				},
				"OperC": {
					"LogC1": {Url: "UrlC1", Key: "KeyC1"},
				},
			},
			ctx:    context.Background(),
			result: core.SCTDERs{[]byte{0}, []byte{0}},
		},
		{
			name: "basic failure case",
			mock: &mockFailPub{},
			groups: loglist.List{
				"OperA": {
					"LogA1": {Url: "UrlA1", Key: "KeyA1"},
					"LogA2": {Url: "UrlA2", Key: "KeyA2"},
				},
				"OperB": {
					"LogB1": {Url: "UrlB1", Key: "KeyB1"},
				},
				"OperC": {
					"LogC1": {Url: "UrlC1", Key: "KeyC1"},
				},
			},
			ctx:        context.Background(),
			expectErr:  "failed to get 2 SCTs, got 3 error(s)",
			berrorType: &missingSCTErr,
		},
		{
			name: "parent context timeout failure case",
			mock: &mockSlowPub{},
			groups: loglist.List{
				"OperA": {
					"LogA1": {Url: "UrlA1", Key: "KeyA1"},
					"LogA2": {Url: "UrlA2", Key: "KeyA2"},
				},
				"OperB": {
					"LogB1": {Url: "UrlB1", Key: "KeyB1"},
				},
				"OperC": {
					"LogC1": {Url: "UrlC1", Key: "KeyC1"},
				},
			},
			ctx:        expired,
			expectErr:  "failed to get 2 SCTs before ctx finished",
			berrorType: &missingSCTErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctp := New(tc.mock, tc.groups, nil, nil, 0, blog.NewMock(), metrics.NoopRegisterer)
			ret, err := ctp.GetSCTs(tc.ctx, []byte{0}, time.Time{})
			if tc.result != nil {
				test.AssertDeepEquals(t, ret, tc.result)
			} else if tc.expectErr != "" {
				if !strings.Contains(err.Error(), tc.expectErr) {
					t.Errorf("Error %q did not match expected %q", err, tc.expectErr)
				}
				if tc.berrorType != nil {
					test.AssertErrorIs(t, err, *tc.berrorType)
				}
			}
		})
	}
}

type mockFailOnePub struct {
	badURL string
}

func (mp *mockFailOnePub) SubmitToSingleCTWithResult(_ context.Context, req *pubpb.Request, _ ...grpc.CallOption) (*pubpb.Result, error) {
	if req.LogURL == mp.badURL {
		return nil, errors.New("BAD")
	}
	return &pubpb.Result{Sct: []byte{0}}, nil
}

func TestGetSCTsMetrics(t *testing.T) {
	ctp := New(&mockFailOnePub{badURL: "UrlA1"}, loglist.List{
		"OperA": {
			"LogA1": {Url: "UrlA1", Key: "KeyA1"},
		},
		"OperB": {
			"LogB1": {Url: "UrlB1", Key: "KeyB1"},
		},
		"OperC": {
			"LogC1": {Url: "UrlC1", Key: "KeyC1"},
		},
	}, nil, nil, 0, blog.NewMock(), metrics.NoopRegisterer)
	_, err := ctp.GetSCTs(context.Background(), []byte{0}, time.Time{})
	test.AssertNotError(t, err, "GetSCTs failed")
	test.AssertMetricWithLabelsEquals(t, ctp.winnerCounter, prometheus.Labels{"url": "UrlB1", "result": succeeded}, 1)
	test.AssertMetricWithLabelsEquals(t, ctp.winnerCounter, prometheus.Labels{"url": "UrlC1", "result": succeeded}, 1)
}

func TestGetSCTsFailMetrics(t *testing.T) {
	// Ensure the proper metrics are incremented when GetSCTs fails.
	ctp := New(&mockFailOnePub{badURL: "UrlA1"}, loglist.List{
		"OperA": {
			"LogA1": {Url: "UrlA1", Key: "KeyA1"},
		},
	}, nil, nil, 0, blog.NewMock(), metrics.NoopRegisterer)
	_, err := ctp.GetSCTs(context.Background(), []byte{0}, time.Time{})
	test.AssertError(t, err, "GetSCTs should have failed")
	test.AssertErrorIs(t, err, berrors.MissingSCTs)
	test.AssertMetricWithLabelsEquals(t, ctp.winnerCounter, prometheus.Labels{"url": "UrlA1", "result": failed}, 1)

	// Ensure the proper metrics are incremented when GetSCTs times out.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ctp = New(&mockSlowPub{}, loglist.List{
		"OperA": {
			"LogA1": {Url: "UrlA1", Key: "KeyA1"},
		},
	}, nil, nil, 0, blog.NewMock(), metrics.NoopRegisterer)
	_, err = ctp.GetSCTs(ctx, []byte{0}, time.Time{})
	test.AssertError(t, err, "GetSCTs should have timed out")
	test.AssertErrorIs(t, err, berrors.MissingSCTs)
	test.AssertContains(t, err.Error(), context.DeadlineExceeded.Error())
	test.AssertMetricWithLabelsEquals(t, ctp.winnerCounter, prometheus.Labels{"url": "UrlA1", "result": failed}, 1)
}
