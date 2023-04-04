package va

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/jmhodges/clock"
	"github.com/letsencrypt/boulder/bdns"
	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/identifier"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/probs"
	"github.com/letsencrypt/boulder/test"
	"github.com/prometheus/client_golang/prometheus"
)

func dnsChallenge() core.Challenge {
	return createChallenge(core.ChallengeTypeDNS01)
}

func TestDNSValidationEmpty(t *testing.T) {
	va, _ := setup(nil, 0, "", nil)

	// This test calls PerformValidation directly, because that is where the
	// metrics checked below are incremented.
	req := createValidationRequest("empty-txts.com", core.ChallengeTypeDNS01)
	res, _ := va.PerformValidation(context.Background(), req)
	test.AssertEquals(t, res.Problems.ProblemType, "unauthorized")
	test.AssertEquals(t, res.Problems.Detail, "No TXT record found at _acme-challenge.empty-txts.com")

	test.AssertMetricWithLabelsEquals(t, va.metrics.validationTime, prometheus.Labels{
		"type":         "dns-01",
		"result":       "invalid",
		"problem_type": "unauthorized",
	}, 1)
}

func TestDNSValidationWrong(t *testing.T) {
	va, _ := setup(nil, 0, "", nil)

	_, prob := va.validateDNS01(context.Background(), dnsi("wrong-dns01.com"), dnsChallenge())
	if prob == nil {
		t.Fatalf("Successful DNS validation with wrong TXT record")
	}
	test.AssertEquals(t, prob.Error(), "unauthorized :: Incorrect TXT record \"a\" found at _acme-challenge.wrong-dns01.com")
}

func TestDNSValidationWrongMany(t *testing.T) {
	va, _ := setup(nil, 0, "", nil)

	_, prob := va.validateDNS01(context.Background(), dnsi("wrong-many-dns01.com"), dnsChallenge())
	if prob == nil {
		t.Fatalf("Successful DNS validation with wrong TXT record")
	}
	test.AssertEquals(t, prob.Error(), "unauthorized :: Incorrect TXT record \"a\" (and 4 more) found at _acme-challenge.wrong-many-dns01.com")
}

func TestDNSValidationWrongLong(t *testing.T) {
	va, _ := setup(nil, 0, "", nil)

	_, prob := va.validateDNS01(context.Background(), dnsi("long-dns01.com"), dnsChallenge())
	if prob == nil {
		t.Fatalf("Successful DNS validation with wrong TXT record")
	}
	test.AssertEquals(t, prob.Error(), "unauthorized :: Incorrect TXT record \"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa...\" found at _acme-challenge.long-dns01.com")
}

func TestDNSValidationFailure(t *testing.T) {
	va, _ := setup(nil, 0, "", nil)

	_, prob := va.validateDNS01(ctx, dnsi("localhost"), dnsChallenge())

	test.AssertEquals(t, prob.Type, probs.UnauthorizedProblem)
}

func TestDNSValidationInvalid(t *testing.T) {
	var notDNS = identifier.ACMEIdentifier{
		Type:  identifier.IdentifierType("iris"),
		Value: "790DB180-A274-47A4-855F-31C428CB1072",
	}

	va, _ := setup(nil, 0, "", nil)

	_, prob := va.validateDNS01(ctx, notDNS, dnsChallenge())

	test.AssertEquals(t, prob.Type, probs.MalformedProblem)
}

func TestDNSValidationNotSane(t *testing.T) {
	va, _ := setup(nil, 0, "", nil)

	chall := dnsChallenge()
	chall.Token = ""
	_, prob := va.validateChallenge(ctx, dnsi("localhost"), chall)
	if prob.Type != probs.MalformedProblem {
		t.Errorf("Got wrong error type: expected %s, got %s",
			prob.Type, probs.MalformedProblem)
	}
	if !strings.Contains(prob.Error(), "Challenge failed consistency check:") {
		t.Errorf("Got wrong error: %s", prob.Error())
	}

	chall.Token = "yfCBb-bRTLz8Wd1C0lTUQK3qlKj3-t2tYGwx5Hj7r_"
	_, prob = va.validateChallenge(ctx, dnsi("localhost"), chall)
	if prob.Type != probs.MalformedProblem {
		t.Errorf("Got wrong error type: expected %s, got %s",
			prob.Type, probs.MalformedProblem)
	}
	if !strings.Contains(prob.Error(), "Challenge failed consistency check:") {
		t.Errorf("Got wrong error: %s", prob.Error())
	}

	chall.ProvidedKeyAuthorization = "a"
	_, prob = va.validateChallenge(ctx, dnsi("localhost"), chall)
	if prob.Type != probs.MalformedProblem {
		t.Errorf("Got wrong error type: expected %s, got %s",
			prob.Type, probs.MalformedProblem)
	}
	if !strings.Contains(prob.Error(), "Challenge failed consistency check:") {
		t.Errorf("Got wrong error: %s", prob.Error())
	}

}

func TestDNSValidationServFail(t *testing.T) {
	va, _ := setup(nil, 0, "", nil)

	_, prob := va.validateChallenge(ctx, dnsi("servfail.com"), dnsChallenge())

	test.AssertEquals(t, prob.Type, probs.DNSProblem)
}

func TestDNSValidationNoServer(t *testing.T) {
	va, log := setup(nil, 0, "", nil)
	staticProvider, err := bdns.NewStaticProvider([]string{})
	test.AssertNotError(t, err, "Couldn't make new static provider")

	va.dnsClient = bdns.NewTest(
		time.Second*5,
		staticProvider,
		metrics.NoopRegisterer,
		clock.New(),
		1,
		log)

	_, prob := va.validateChallenge(ctx, dnsi("localhost"), dnsChallenge())

	test.AssertEquals(t, prob.Type, probs.DNSProblem)
}

func TestDNSValidationOK(t *testing.T) {
	va, _ := setup(nil, 0, "", nil)

	_, prob := va.validateChallenge(ctx, dnsi("good-dns01.com"), dnsChallenge())

	test.Assert(t, prob == nil, "Should be valid.")
}

func TestDNSValidationNoAuthorityOK(t *testing.T) {
	va, _ := setup(nil, 0, "", nil)

	_, prob := va.validateChallenge(ctx, dnsi("no-authority-dns01.com"), dnsChallenge())

	test.Assert(t, prob == nil, "Should be valid.")
}

func TestAvailableAddresses(t *testing.T) {
	v6a := net.ParseIP("::1")
	v6b := net.ParseIP("2001:db8::2:1") // 2001:DB8 is reserved for docs (RFC 3849)
	v4a := net.ParseIP("127.0.0.1")
	v4b := net.ParseIP("192.0.2.1") // 192.0.2.0/24 is reserved for docs (RFC 5737)

	testcases := []struct {
		input []net.IP
		v4    []net.IP
		v6    []net.IP
	}{
		// An empty validation record
		{
			[]net.IP{},
			[]net.IP{},
			[]net.IP{},
		},
		// A validation record with one IPv4 address
		{
			[]net.IP{v4a},
			[]net.IP{v4a},
			[]net.IP{},
		},
		// A dual homed record with an IPv4 and IPv6 address
		{
			[]net.IP{v4a, v6a},
			[]net.IP{v4a},
			[]net.IP{v6a},
		},
		// The same as above but with the v4/v6 order flipped
		{
			[]net.IP{v6a, v4a},
			[]net.IP{v4a},
			[]net.IP{v6a},
		},
		// A validation record with just IPv6 addresses
		{
			[]net.IP{v6a, v6b},
			[]net.IP{},
			[]net.IP{v6a, v6b},
		},
		// A validation record with interleaved IPv4/IPv6 records
		{
			[]net.IP{v6a, v4a, v6b, v4b},
			[]net.IP{v4a, v4b},
			[]net.IP{v6a, v6b},
		},
	}

	for _, tc := range testcases {
		// Split the input record into v4/v6 addresses
		v4result, v6result := availableAddresses(tc.input)

		// Test that we got the right number of v4 results
		test.Assert(t, len(tc.v4) == len(v4result),
			fmt.Sprintf("Wrong # of IPv4 results: expected %d, got %d", len(tc.v4), len(v4result)))

		// Check that all of the v4 results match expected values
		for i, v4addr := range tc.v4 {
			test.Assert(t, v4addr.String() == v4result[i].String(),
				fmt.Sprintf("Wrong v4 result index %d: expected %q got %q", i, v4addr.String(), v4result[i].String()))
		}

		// Test that we got the right number of v6 results
		test.Assert(t, len(tc.v6) == len(v6result),
			fmt.Sprintf("Wrong # of IPv6 results: expected %d, got %d", len(tc.v6), len(v6result)))

		// Check that all of the v6 results match expected values
		for i, v6addr := range tc.v6 {
			test.Assert(t, v6addr.String() == v6result[i].String(),
				fmt.Sprintf("Wrong v6 result index %d: expected %q got %q", i, v6addr.String(), v6result[i].String()))
		}
	}
}
