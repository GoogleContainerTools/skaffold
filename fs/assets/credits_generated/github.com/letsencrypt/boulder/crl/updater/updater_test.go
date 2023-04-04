package updater

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/jmhodges/clock"
	capb "github.com/letsencrypt/boulder/ca/proto"
	corepb "github.com/letsencrypt/boulder/core/proto"
	cspb "github.com/letsencrypt/boulder/crl/storer/proto"
	"github.com/letsencrypt/boulder/issuance"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/mocks"
	sapb "github.com/letsencrypt/boulder/sa/proto"
	"github.com/letsencrypt/boulder/test"
	"github.com/prometheus/client_golang/prometheus"
)

// fakeGRCC is a fake sapb.StorageAuthority_GetRevokedCertsClient which can be
// populated with some CRL entries or an error for use as the return value of
// a faked GetRevokedCerts call.
type fakeGRCC struct {
	grpc.ClientStream
	entries []*corepb.CRLEntry
	nextIdx int
	err     error
}

func (f *fakeGRCC) Recv() (*corepb.CRLEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.nextIdx < len(f.entries) {
		res := f.entries[f.nextIdx]
		f.nextIdx++
		return res, nil
	}
	return nil, io.EOF
}

// fakeSAC is a fake sapb.StorageAuthorityClient which can be populated with a
// fakeGRCC to be used as the return value for calls to GetRevokedCerts, and a
// fake timestamp to serve as the database's maximum notAfter value.
type fakeSAC struct {
	mocks.StorageAuthorityReadOnly
	grcc        fakeGRCC
	maxNotAfter time.Time
}

func (f *fakeSAC) GetRevokedCerts(ctx context.Context, _ *sapb.GetRevokedCertsRequest, _ ...grpc.CallOption) (sapb.StorageAuthorityReadOnly_GetRevokedCertsClient, error) {
	return &f.grcc, nil
}

func (f *fakeSAC) GetMaxExpiration(_ context.Context, req *emptypb.Empty, _ ...grpc.CallOption) (*timestamppb.Timestamp, error) {
	return timestamppb.New(f.maxNotAfter), nil
}

// fakeGCC is a fake capb.CRLGenerator_GenerateCRLClient which can be
// populated with some CRL entries or an error for use as the return value of
// a faked GenerateCRL call.
type fakeGCC struct {
	grpc.ClientStream
	chunks  [][]byte
	nextIdx int
	sendErr error
	recvErr error
}

func (f *fakeGCC) Send(*capb.GenerateCRLRequest) error {
	return f.sendErr
}

func (f *fakeGCC) CloseSend() error {
	return nil
}

func (f *fakeGCC) Recv() (*capb.GenerateCRLResponse, error) {
	if f.recvErr != nil {
		return nil, f.recvErr
	}
	if f.nextIdx < len(f.chunks) {
		res := f.chunks[f.nextIdx]
		f.nextIdx++
		return &capb.GenerateCRLResponse{Chunk: res}, nil
	}
	return nil, io.EOF
}

// fakeCGC is a fake capb.CRLGeneratorClient which can be populated with a
// fakeGCC to be used as the return value for calls to GenerateCRL.
type fakeCGC struct {
	gcc fakeGCC
}

func (f *fakeCGC) GenerateCRL(ctx context.Context, opts ...grpc.CallOption) (capb.CRLGenerator_GenerateCRLClient, error) {
	return &f.gcc, nil
}

// fakeUCC is a fake cspb.CRLStorer_UploadCRLClient which can be populated with
// an error for use as the return value of a faked UploadCRL call.
type fakeUCC struct {
	grpc.ClientStream
	sendErr error
	recvErr error
}

func (f *fakeUCC) Send(*cspb.UploadCRLRequest) error {
	return f.sendErr
}

func (f *fakeUCC) CloseAndRecv() (*emptypb.Empty, error) {
	if f.recvErr != nil {
		return nil, f.recvErr
	}
	return &emptypb.Empty{}, nil
}

// fakeCSC is a fake cspb.CRLStorerClient which can be populated with a
// fakeUCC for use as the return value for calls to UploadCRL.
type fakeCSC struct {
	ucc fakeUCC
}

func (f *fakeCSC) UploadCRL(ctx context.Context, opts ...grpc.CallOption) (cspb.CRLStorer_UploadCRLClient, error) {
	return &f.ucc, nil
}

func TestTickShard(t *testing.T) {
	e1, err := issuance.LoadCertificate("../../test/hierarchy/int-e1.cert.pem")
	test.AssertNotError(t, err, "loading test issuer")
	r3, err := issuance.LoadCertificate("../../test/hierarchy/int-r3.cert.pem")
	test.AssertNotError(t, err, "loading test issuer")

	sentinelErr := errors.New("oops")

	clk := clock.NewFake()
	clk.Set(time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC))
	cu, err := NewUpdater(
		[]*issuance.Certificate{e1, r3},
		2, 18*time.Hour, 24*time.Hour,
		6*time.Hour, 1*time.Minute, 1,
		&fakeSAC{grcc: fakeGRCC{}, maxNotAfter: clk.Now().Add(90 * 24 * time.Hour)},
		&fakeCGC{gcc: fakeGCC{}},
		&fakeCSC{ucc: fakeUCC{}},
		metrics.NoopRegisterer, blog.NewMock(), clk,
	)
	test.AssertNotError(t, err, "building test crlUpdater")

	testChunks := []chunk{
		{clk.Now(), clk.Now().Add(18 * time.Hour), 0},
	}

	// Ensure that getting no results from the SA still works.
	err = cu.tickShard(context.Background(), cu.clk.Now(), e1.NameID(), 0, testChunks)
	test.AssertNotError(t, err, "empty CRL")
	test.AssertMetricWithLabelsEquals(t, cu.updatedCounter, prometheus.Labels{
		"issuer": "(TEST) Elegant Elephant E1", "result": "success",
	}, 1)
	cu.updatedCounter.Reset()

	// Errors closing the Storer upload stream should bubble up.
	cu.cs = &fakeCSC{ucc: fakeUCC{recvErr: sentinelErr}}
	err = cu.tickShard(context.Background(), cu.clk.Now(), e1.NameID(), 0, testChunks)
	test.AssertError(t, err, "storer error")
	test.AssertContains(t, err.Error(), "closing CRLStorer upload stream")
	test.AssertErrorIs(t, err, sentinelErr)
	test.AssertMetricWithLabelsEquals(t, cu.updatedCounter, prometheus.Labels{
		"issuer": "(TEST) Elegant Elephant E1", "result": "failed",
	}, 1)
	cu.updatedCounter.Reset()

	// Errors sending to the Storer should bubble up sooner.
	cu.cs = &fakeCSC{ucc: fakeUCC{sendErr: sentinelErr}}
	err = cu.tickShard(context.Background(), cu.clk.Now(), e1.NameID(), 0, testChunks)
	test.AssertError(t, err, "storer error")
	test.AssertContains(t, err.Error(), "sending CRLStorer metadata")
	test.AssertErrorIs(t, err, sentinelErr)
	test.AssertMetricWithLabelsEquals(t, cu.updatedCounter, prometheus.Labels{
		"issuer": "(TEST) Elegant Elephant E1", "result": "failed",
	}, 1)
	cu.updatedCounter.Reset()

	// Errors reading from the CA should bubble up sooner.
	cu.ca = &fakeCGC{gcc: fakeGCC{recvErr: sentinelErr}}
	err = cu.tickShard(context.Background(), cu.clk.Now(), e1.NameID(), 0, testChunks)
	test.AssertError(t, err, "CA error")
	test.AssertContains(t, err.Error(), "receiving CRL bytes")
	test.AssertErrorIs(t, err, sentinelErr)
	test.AssertMetricWithLabelsEquals(t, cu.updatedCounter, prometheus.Labels{
		"issuer": "(TEST) Elegant Elephant E1", "result": "failed",
	}, 1)
	cu.updatedCounter.Reset()

	// Errors sending to the CA should bubble up sooner.
	cu.ca = &fakeCGC{gcc: fakeGCC{sendErr: sentinelErr}}
	err = cu.tickShard(context.Background(), cu.clk.Now(), e1.NameID(), 0, testChunks)
	test.AssertError(t, err, "CA error")
	test.AssertContains(t, err.Error(), "sending CA metadata")
	test.AssertErrorIs(t, err, sentinelErr)
	test.AssertMetricWithLabelsEquals(t, cu.updatedCounter, prometheus.Labels{
		"issuer": "(TEST) Elegant Elephant E1", "result": "failed",
	}, 1)
	cu.updatedCounter.Reset()

	// Errors reading from the SA should bubble up soonest.
	cu.sa = &fakeSAC{grcc: fakeGRCC{err: sentinelErr}, maxNotAfter: clk.Now().Add(90 * 24 * time.Hour)}
	err = cu.tickShard(context.Background(), cu.clk.Now(), e1.NameID(), 0, testChunks)
	test.AssertError(t, err, "database error")
	test.AssertContains(t, err.Error(), "retrieving entry from SA")
	test.AssertErrorIs(t, err, sentinelErr)
	test.AssertMetricWithLabelsEquals(t, cu.updatedCounter, prometheus.Labels{
		"issuer": "(TEST) Elegant Elephant E1", "result": "failed",
	}, 1)
	cu.updatedCounter.Reset()
}

func TestTickIssuer(t *testing.T) {
	e1, err := issuance.LoadCertificate("../../test/hierarchy/int-e1.cert.pem")
	test.AssertNotError(t, err, "loading test issuer")
	r3, err := issuance.LoadCertificate("../../test/hierarchy/int-r3.cert.pem")
	test.AssertNotError(t, err, "loading test issuer")

	mockLog := blog.NewMock()
	clk := clock.NewFake()
	clk.Set(time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC))
	cu, err := NewUpdater(
		[]*issuance.Certificate{e1, r3},
		2, 18*time.Hour, 24*time.Hour,
		6*time.Hour, 1*time.Minute, 1,
		&fakeSAC{grcc: fakeGRCC{err: errors.New("db no worky")}, maxNotAfter: clk.Now().Add(90 * 24 * time.Hour)},
		&fakeCGC{gcc: fakeGCC{}},
		&fakeCSC{ucc: fakeUCC{}},
		metrics.NoopRegisterer, mockLog, clk,
	)
	test.AssertNotError(t, err, "building test crlUpdater")

	// An error that affects all shards should have every shard reflected in the
	// combined error message.
	err = cu.tickIssuer(context.Background(), cu.clk.Now(), e1.NameID())
	test.AssertError(t, err, "database error")
	test.AssertContains(t, err.Error(), "2 shards failed")
	test.AssertContains(t, err.Error(), "[0 1]")
	test.AssertEquals(t, len(mockLog.GetAllMatching("Generating CRL failed:")), 2)
	test.AssertMetricWithLabelsEquals(t, cu.tickHistogram, prometheus.Labels{
		"issuer": "(TEST) Elegant Elephant E1", "result": "failed",
	}, 2)
	test.AssertMetricWithLabelsEquals(t, cu.tickHistogram, prometheus.Labels{
		"issuer": "(TEST) Elegant Elephant E1 (Overall)", "result": "failed",
	}, 1)
	cu.tickHistogram.Reset()
}

func TestTick(t *testing.T) {
	e1, err := issuance.LoadCertificate("../../test/hierarchy/int-e1.cert.pem")
	test.AssertNotError(t, err, "loading test issuer")
	r3, err := issuance.LoadCertificate("../../test/hierarchy/int-r3.cert.pem")
	test.AssertNotError(t, err, "loading test issuer")

	mockLog := blog.NewMock()
	clk := clock.NewFake()
	clk.Set(time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC))
	cu, err := NewUpdater(
		[]*issuance.Certificate{e1, r3},
		2, 18*time.Hour, 24*time.Hour,
		6*time.Hour, 1*time.Minute, 1,
		&fakeSAC{grcc: fakeGRCC{err: errors.New("db no worky")}, maxNotAfter: clk.Now().Add(90 * 24 * time.Hour)},
		&fakeCGC{gcc: fakeGCC{}},
		&fakeCSC{ucc: fakeUCC{}},
		metrics.NoopRegisterer, mockLog, clk,
	)
	test.AssertNotError(t, err, "building test crlUpdater")

	// An error that affects all issuers should have every issuer reflected in the
	// combined error message.
	now := cu.clk.Now()
	err = cu.Tick(context.Background(), now)
	test.AssertError(t, err, "database error")
	test.AssertContains(t, err.Error(), "2 issuers failed")
	test.AssertContains(t, err.Error(), "(TEST) Elegant Elephant E1")
	test.AssertContains(t, err.Error(), "(TEST) Radical Rhino R3")
	test.AssertEquals(t, len(mockLog.GetAllMatching("Generating CRL failed:")), 4)
	test.AssertEquals(t, len(mockLog.GetAllMatching("Generating CRLs for issuer failed:")), 2)
	test.AssertMetricWithLabelsEquals(t, cu.tickHistogram, prometheus.Labels{
		"issuer": "(TEST) Elegant Elephant E1 (Overall)", "result": "failed",
	}, 1)
	test.AssertMetricWithLabelsEquals(t, cu.tickHistogram, prometheus.Labels{
		"issuer": "(TEST) Radical Rhino R3 (Overall)", "result": "failed",
	}, 1)
	test.AssertMetricWithLabelsEquals(t, cu.tickHistogram, prometheus.Labels{
		"issuer": "all", "result": "failed",
	}, 1)
	cu.tickHistogram.Reset()
}

func TestGetShardMappings(t *testing.T) {
	// We set atTime to be exactly one day (numShards * shardWidth) after the
	// anchorTime for these tests, so that we know that the index of the first
	// chunk we would normally (i.e. not taking lookback or overshoot into
	// account) care about is 0.
	atTime := anchorTime().Add(24 * time.Hour)

	// When there is no lookback, and the maxNotAfter is exactly as far in the
	// future as the numShards * shardWidth looks, every shard should be mapped to
	// exactly one chunk.
	tcu := crlUpdater{
		numShards:      24,
		shardWidth:     1 * time.Hour,
		sa:             &fakeSAC{maxNotAfter: atTime.Add(23*time.Hour + 30*time.Minute)},
		lookbackPeriod: 0,
	}
	m, err := tcu.getShardMappings(context.Background(), atTime)
	test.AssertNotError(t, err, "getting aligned shards")
	test.AssertEquals(t, len(m), 24)
	for _, s := range m {
		test.AssertEquals(t, len(s), 1)
	}

	// When there is 1.5 hours each of lookback and maxNotAfter overshoot, then
	// there should be four shards which each get two chunks mapped to them.
	tcu = crlUpdater{
		numShards:      24,
		shardWidth:     1 * time.Hour,
		sa:             &fakeSAC{maxNotAfter: atTime.Add(24*time.Hour + 90*time.Minute)},
		lookbackPeriod: 90 * time.Minute,
	}
	m, err = tcu.getShardMappings(context.Background(), atTime)
	test.AssertNotError(t, err, "getting overshoot shards")
	test.AssertEquals(t, len(m), 24)
	for i, s := range m {
		if i == 0 || i == 1 || i == 22 || i == 23 {
			test.AssertEquals(t, len(s), 2)
		} else {
			test.AssertEquals(t, len(s), 1)
		}
	}

	// When there is a massive amount of overshoot, many chunks should be mapped
	// to each shard.
	tcu = crlUpdater{
		numShards:      24,
		shardWidth:     1 * time.Hour,
		sa:             &fakeSAC{maxNotAfter: atTime.Add(90 * 24 * time.Hour)},
		lookbackPeriod: time.Minute,
	}
	m, err = tcu.getShardMappings(context.Background(), atTime)
	test.AssertNotError(t, err, "getting overshoot shards")
	test.AssertEquals(t, len(m), 24)
	for i, s := range m {
		if i == 23 {
			test.AssertEquals(t, len(s), 91)
		} else {
			test.AssertEquals(t, len(s), 90)
		}
	}

	// An arbitrarily-chosen chunk should always end up in the same shard no
	// matter what the current time, lookback, and overshoot are, as long as the
	// number of shards and the shard width remains constant.
	tcu = crlUpdater{
		numShards:      24,
		shardWidth:     1 * time.Hour,
		sa:             &fakeSAC{maxNotAfter: atTime.Add(24 * time.Hour)},
		lookbackPeriod: time.Hour,
	}
	m, err = tcu.getShardMappings(context.Background(), atTime)
	test.AssertNotError(t, err, "getting consistency shards")
	test.AssertEquals(t, m[10][0].start, anchorTime().Add(34*time.Hour))
	tcu.lookbackPeriod = 4 * time.Hour
	m, err = tcu.getShardMappings(context.Background(), atTime)
	test.AssertNotError(t, err, "getting consistency shards")
	test.AssertEquals(t, m[10][0].start, anchorTime().Add(34*time.Hour))
	tcu.sa = &fakeSAC{maxNotAfter: atTime.Add(300 * 24 * time.Hour)}
	m, err = tcu.getShardMappings(context.Background(), atTime)
	test.AssertNotError(t, err, "getting consistency shards")
	test.AssertEquals(t, m[10][0].start, anchorTime().Add(34*time.Hour))
	atTime = atTime.Add(6 * time.Hour)
	m, err = tcu.getShardMappings(context.Background(), atTime)
	test.AssertNotError(t, err, "getting consistency shards")
	test.AssertEquals(t, m[10][0].start, anchorTime().Add(34*time.Hour))
}

func TestGetChunkAtTime(t *testing.T) {
	// Our test updater divides time into chunks 1 day wide, numbered 0 through 9.
	tcu := crlUpdater{
		numShards:  10,
		shardWidth: 24 * time.Hour,
	}

	// The chunk right at the anchor time should have index 0 and start at the
	// anchor time. This also tests behavior when atTime is on a chunk boundary.
	atTime := anchorTime()
	c, err := tcu.getChunkAtTime(atTime)
	test.AssertNotError(t, err, "getting chunk at anchor")
	test.AssertEquals(t, c.idx, 0)
	test.Assert(t, c.start.Equal(atTime), "getting chunk at anchor")
	test.Assert(t, c.end.Equal(atTime.Add(24*time.Hour)), "getting chunk at anchor")

	// The chunk a bit over a year in the future should have index 5.
	atTime = anchorTime().Add(365 * 24 * time.Hour)
	c, err = tcu.getChunkAtTime(atTime.Add(1 * time.Minute))
	test.AssertNotError(t, err, "getting chunk")
	test.AssertEquals(t, c.idx, 5)
	test.Assert(t, c.start.Equal(atTime), "getting chunk")
	test.Assert(t, c.end.Equal(atTime.Add(24*time.Hour)), "getting chunk")

	// A chunk very far in the future should break the math. We have to add to
	// the time twice, since the whole point of "very far in the future" is that
	// it isn't representable by a time.Duration.
	atTime = anchorTime().Add(200 * 365 * 24 * time.Hour).Add(200 * 365 * 24 * time.Hour)
	c, err = tcu.getChunkAtTime(atTime)
	test.AssertError(t, err, "getting far-future chunk")
}
