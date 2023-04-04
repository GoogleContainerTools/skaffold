package nonce

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/letsencrypt/boulder/metrics"
	noncepb "github.com/letsencrypt/boulder/nonce/proto"
	"github.com/letsencrypt/boulder/test"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestValidNonce(t *testing.T) {
	ns, err := NewNonceService(metrics.NoopRegisterer, 0, "")
	test.AssertNotError(t, err, "Could not create nonce service")
	n, err := ns.Nonce()
	test.AssertNotError(t, err, "Could not create nonce")
	test.Assert(t, ns.Valid(n), fmt.Sprintf("Did not recognize fresh nonce %s", n))
}

func TestAlreadyUsed(t *testing.T) {
	ns, err := NewNonceService(metrics.NoopRegisterer, 0, "")
	test.AssertNotError(t, err, "Could not create nonce service")
	n, err := ns.Nonce()
	test.AssertNotError(t, err, "Could not create nonce")
	test.Assert(t, ns.Valid(n), "Did not recognize fresh nonce")
	test.Assert(t, !ns.Valid(n), "Recognized the same nonce twice")
}

func TestRejectMalformed(t *testing.T) {
	ns, err := NewNonceService(metrics.NoopRegisterer, 0, "")
	test.AssertNotError(t, err, "Could not create nonce service")
	n, err := ns.Nonce()
	test.AssertNotError(t, err, "Could not create nonce")
	test.Assert(t, !ns.Valid("asdf"+n), "Accepted an invalid nonce")
}

func TestRejectShort(t *testing.T) {
	ns, err := NewNonceService(metrics.NoopRegisterer, 0, "")
	test.AssertNotError(t, err, "Could not create nonce service")
	test.Assert(t, !ns.Valid("aGkK"), "Accepted an invalid nonce")
}

func TestRejectUnknown(t *testing.T) {
	ns1, err := NewNonceService(metrics.NoopRegisterer, 0, "")
	test.AssertNotError(t, err, "Could not create nonce service")
	ns2, err := NewNonceService(metrics.NoopRegisterer, 0, "")
	test.AssertNotError(t, err, "Could not create nonce service")

	n, err := ns1.Nonce()
	test.AssertNotError(t, err, "Could not create nonce")
	test.Assert(t, !ns2.Valid(n), "Accepted a foreign nonce")
}

func TestRejectTooLate(t *testing.T) {
	ns, err := NewNonceService(metrics.NoopRegisterer, 0, "")
	test.AssertNotError(t, err, "Could not create nonce service")

	ns.latest = 2
	n, err := ns.Nonce()
	test.AssertNotError(t, err, "Could not create nonce")
	ns.latest = 1
	test.Assert(t, !ns.Valid(n), "Accepted a nonce with a too-high counter")
}

func TestRejectTooEarly(t *testing.T) {
	ns, err := NewNonceService(metrics.NoopRegisterer, 0, "")
	test.AssertNotError(t, err, "Could not create nonce service")

	n0, err := ns.Nonce()
	test.AssertNotError(t, err, "Could not create nonce")

	for i := 0; i < ns.maxUsed; i++ {
		n, err := ns.Nonce()
		test.AssertNotError(t, err, "Could not create nonce")
		if !ns.Valid(n) {
			t.Errorf("generated invalid nonce")
		}
	}

	n1, err := ns.Nonce()
	test.AssertNotError(t, err, "Could not create nonce")
	n2, err := ns.Nonce()
	test.AssertNotError(t, err, "Could not create nonce")
	n3, err := ns.Nonce()
	test.AssertNotError(t, err, "Could not create nonce")

	test.Assert(t, ns.Valid(n3), "Rejected a valid nonce")
	test.Assert(t, ns.Valid(n2), "Rejected a valid nonce")
	test.Assert(t, ns.Valid(n1), "Rejected a valid nonce")
	test.Assert(t, !ns.Valid(n0), "Accepted a nonce that we should have forgotten")
}

func BenchmarkNonces(b *testing.B) {
	ns, err := NewNonceService(metrics.NoopRegisterer, 0, "")
	if err != nil {
		b.Fatal("creating nonce service", err)
	}

	for i := 0; i < ns.maxUsed; i++ {
		n, err := ns.Nonce()
		if err != nil {
			b.Fatal("noncing", err)
		}
		if !ns.Valid(n) {
			b.Fatal("generated invalid nonce")
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n, err := ns.Nonce()
			if err != nil {
				b.Fatal("noncing", err)
			}
			if !ns.Valid(n) {
				b.Fatal("generated invalid nonce")
			}
		}
	})
}

func TestNoncePrefixing(t *testing.T) {
	ns, err := NewNonceService(metrics.NoopRegisterer, 0, "zinc")
	test.AssertNotError(t, err, "Could not create nonce service")

	n, err := ns.Nonce()
	test.AssertNotError(t, err, "Could not create nonce")
	test.Assert(t, ns.Valid(n), "Valid nonce rejected")

	n, err = ns.Nonce()
	test.AssertNotError(t, err, "Could not create nonce")
	n = n[1:]
	test.Assert(t, !ns.Valid(n), "Valid nonce with incorrect prefix accepted")

	n, err = ns.Nonce()
	test.AssertNotError(t, err, "Could not create nonce")
	test.Assert(t, !ns.Valid(n[6:]), "Valid nonce without prefix accepted")
}

type malleableNonceClient struct {
	redeem func(ctx context.Context, in *noncepb.NonceMessage, opts ...grpc.CallOption) (*noncepb.ValidMessage, error)
}

func (mnc *malleableNonceClient) Redeem(ctx context.Context, in *noncepb.NonceMessage, opts ...grpc.CallOption) (*noncepb.ValidMessage, error) {
	return mnc.redeem(ctx, in, opts...)
}

func (mnc *malleableNonceClient) Nonce(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*noncepb.NonceMessage, error) {
	return nil, errors.New("unimplemented")
}

func TestRemoteRedeem(t *testing.T) {
	valid, err := RemoteRedeem(context.Background(), nil, "q")
	test.AssertNotError(t, err, "RemoteRedeem failed")
	test.Assert(t, !valid, "RemoteRedeem accepted an invalid nonce")
	valid, err = RemoteRedeem(context.Background(), nil, "")
	test.AssertNotError(t, err, "RemoteRedeem failed")
	test.Assert(t, !valid, "RemoteRedeem accepted an empty nonce")

	prefixMap := map[string]Redeemer{
		"abcd": &malleableNonceClient{
			redeem: func(ctx context.Context, in *noncepb.NonceMessage, opts ...grpc.CallOption) (*noncepb.ValidMessage, error) {
				return nil, errors.New("wrong one!")
			},
		},
		"wxyz": &malleableNonceClient{
			redeem: func(ctx context.Context, in *noncepb.NonceMessage, opts ...grpc.CallOption) (*noncepb.ValidMessage, error) {
				return &noncepb.ValidMessage{Valid: false}, nil
			},
		},
	}
	// Attempt to redeem a nonce with a prefix not in the prefix map, expect return false, nil
	valid, err = RemoteRedeem(context.Background(), prefixMap, "asddCQEC")
	test.AssertNotError(t, err, "RemoteRedeem failed")
	test.Assert(t, !valid, "RemoteRedeem accepted nonce not in prefix map")

	// Attempt to redeem a nonce with a prefix in the prefix map, remote returns error
	// expect false, err
	_, err = RemoteRedeem(context.Background(), prefixMap, "abcdbeef")
	test.AssertError(t, err, "RemoteRedeem didn't return error when remote did")

	// Attempt to redeem a nonce with a prefix in the prefix map, remote returns valid
	// expect true, nil
	valid, err = RemoteRedeem(context.Background(), prefixMap, "wxyzdead")
	test.AssertNotError(t, err, "RemoteRedeem failed")
	test.Assert(t, !valid, "RemoteRedeem didn't honor remote result")

	// Attempt to redeem a nonce with a prefix in the prefix map, remote returns invalid
	// expect false, nil
	prefixMap["wxyz"] = &malleableNonceClient{
		redeem: func(ctx context.Context, in *noncepb.NonceMessage, opts ...grpc.CallOption) (*noncepb.ValidMessage, error) {
			return &noncepb.ValidMessage{Valid: true}, nil
		},
	}
	valid, err = RemoteRedeem(context.Background(), prefixMap, "wxyzdead")
	test.AssertNotError(t, err, "RemoteRedeem failed")
	test.Assert(t, valid, "RemoteRedeem didn't honor remote result")
}

func TestNoncePrefixValidation(t *testing.T) {
	_, err := NewNonceService(metrics.NoopRegisterer, 0, "hey")
	test.AssertError(t, err, "NewNonceService didn't fail with short prefix")
	_, err = NewNonceService(metrics.NoopRegisterer, 0, "hey!")
	test.AssertError(t, err, "NewNonceService didn't fail with invalid base64")
	_, err = NewNonceService(metrics.NoopRegisterer, 0, "heyy")
	test.AssertNotError(t, err, "NewNonceService failed with valid nonce prefix")
}

func TestDerivePrefix(t *testing.T) {
	prefix := DerivePrefix("192.168.1.1:8080", "3b8c758dd85e113ea340ce0b3a99f389d40a308548af94d1730a7692c1874f1f")
	test.AssertEquals(t, prefix, "P9qQaK4o")
}
