package va

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/jmhodges/clock"
	"github.com/letsencrypt/boulder/bdns"
	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	"github.com/letsencrypt/boulder/features"
	"github.com/letsencrypt/boulder/identifier"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/probs"
	"github.com/letsencrypt/boulder/test"
	vapb "github.com/letsencrypt/boulder/va/proto"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"gopkg.in/go-jose/go-jose.v2"
)

var expectedToken = "LoqXcYV8q5ONbJQxbmR7SCTNo3tiAXDfowyjxAjEuX0"
var expectedKeyAuthorization = "LoqXcYV8q5ONbJQxbmR7SCTNo3tiAXDfowyjxAjEuX0.9jg46WB3rR_AHD-EBXdN7cBkH1WOu0tA3M9fm21mqTI"

func bigIntFromB64(b64 string) *big.Int {
	bytes, _ := base64.URLEncoding.DecodeString(b64)
	x := big.NewInt(0)
	x.SetBytes(bytes)
	return x
}

func intFromB64(b64 string) int {
	return int(bigIntFromB64(b64).Int64())
}

var n = bigIntFromB64("n4EPtAOCc9AlkeQHPzHStgAbgs7bTZLwUBZdR8_KuKPEHLd4rHVTeT-O-XV2jRojdNhxJWTDvNd7nqQ0VEiZQHz_AJmSCpMaJMRBSFKrKb2wqVwGU_NsYOYL-QtiWN2lbzcEe6XC0dApr5ydQLrHqkHHig3RBordaZ6Aj-oBHqFEHYpPe7Tpe-OfVfHd1E6cS6M1FZcD1NNLYD5lFHpPI9bTwJlsde3uhGqC0ZCuEHg8lhzwOHrtIQbS0FVbb9k3-tVTU4fg_3L_vniUFAKwuCLqKnS2BYwdq_mzSnbLY7h_qixoR7jig3__kRhuaxwUkRz5iaiQkqgc5gHdrNP5zw==")
var e = intFromB64("AQAB")
var d = bigIntFromB64("bWUC9B-EFRIo8kpGfh0ZuyGPvMNKvYWNtB_ikiH9k20eT-O1q_I78eiZkpXxXQ0UTEs2LsNRS-8uJbvQ-A1irkwMSMkK1J3XTGgdrhCku9gRldY7sNA_AKZGh-Q661_42rINLRCe8W-nZ34ui_qOfkLnK9QWDDqpaIsA-bMwWWSDFu2MUBYwkHTMEzLYGqOe04noqeq1hExBTHBOBdkMXiuFhUq1BU6l-DqEiWxqg82sXt2h-LMnT3046AOYJoRioz75tSUQfGCshWTBnP5uDjd18kKhyv07lhfSJdrPdM5Plyl21hsFf4L_mHCuoFau7gdsPfHPxxjVOcOpBrQzwQ==")
var p = bigIntFromB64("uKE2dh-cTf6ERF4k4e_jy78GfPYUIaUyoSSJuBzp3Cubk3OCqs6grT8bR_cu0Dm1MZwWmtdqDyI95HrUeq3MP15vMMON8lHTeZu2lmKvwqW7anV5UzhM1iZ7z4yMkuUwFWoBvyY898EXvRD-hdqRxHlSqAZ192zB3pVFJ0s7pFc=")
var q = bigIntFromB64("uKE2dh-cTf6ERF4k4e_jy78GfPYUIaUyoSSJuBzp3Cubk3OCqs6grT8bR_cu0Dm1MZwWmtdqDyI95HrUeq3MP15vMMON8lHTeZu2lmKvwqW7anV5UzhM1iZ7z4yMkuUwFWoBvyY898EXvRD-hdqRxHlSqAZ192zB3pVFJ0s7pFc=")

var TheKey = rsa.PrivateKey{
	PublicKey: rsa.PublicKey{N: n, E: e},
	D:         d,
	Primes:    []*big.Int{p, q},
}

var accountKey = &jose.JSONWebKey{Key: TheKey.Public()}

// Return an ACME DNS identifier for the given hostname
func dnsi(hostname string) identifier.ACMEIdentifier {
	return identifier.DNSIdentifier(hostname)
}

var ctx context.Context

func TestMain(m *testing.M) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
	ret := m.Run()
	cancel()
	os.Exit(ret)
}

var accountURIPrefixes = []string{"http://boulder.service.consul:4000/acme/reg/"}

func createValidationRequest(domain string, challengeType core.AcmeChallenge) *vapb.PerformValidationRequest {
	return &vapb.PerformValidationRequest{
		Domain: domain,
		Challenge: &corepb.Challenge{
			Type:              string(challengeType),
			Status:            string(core.StatusPending),
			Token:             expectedToken,
			Validationrecords: nil,
			KeyAuthorization:  expectedKeyAuthorization,
		},
		Authz: &vapb.AuthzMeta{
			Id:    "",
			RegID: 1,
		},
	}
}

func createChallenge(challengeType core.AcmeChallenge) core.Challenge {
	return core.Challenge{
		Type:                     challengeType,
		Status:                   core.StatusPending,
		Token:                    expectedToken,
		ValidationRecord:         []core.ValidationRecord{},
		ProvidedKeyAuthorization: expectedKeyAuthorization,
	}
}

// setChallengeToken sets the token value, and sets the ProvidedKeyAuthorization
// to match.
func setChallengeToken(ch *core.Challenge, token string) {
	ch.Token = token
	ch.ProvidedKeyAuthorization = token + ".9jg46WB3rR_AHD-EBXdN7cBkH1WOu0tA3M9fm21mqTI"
}

func setup(srv *httptest.Server, maxRemoteFailures int, userAgent string, remoteVAs []RemoteVA) (*ValidationAuthorityImpl, *blog.Mock) {
	features.Reset()
	fc := clock.NewFake()

	logger := blog.NewMock()

	if userAgent == "" {
		userAgent = "user agent 1.0"
	}

	va, err := NewValidationAuthorityImpl(
		&bdns.MockClient{Log: logger},
		nil,
		maxRemoteFailures,
		userAgent,
		"letsencrypt.org",
		metrics.NoopRegisterer,
		fc,
		logger,
		accountURIPrefixes,
	)

	// Adjusting industry regulated ACME challenge port settings is fine during
	// testing
	if srv != nil {
		port := getPort(srv)
		va.httpPort = port
		va.tlsPort = port
	}

	if err != nil {
		panic(fmt.Sprintf("Failed to create validation authority: %v", err))
	}
	if remoteVAs != nil {
		va.remoteVAs = remoteVAs
	}
	return va, logger
}

func setupRemote(srv *httptest.Server, userAgent string) vapb.VAClient {
	innerVA, _ := setup(srv, 0, userAgent, nil)
	return &localRemoteVA{remote: *innerVA}
}

type multiSrv struct {
	*httptest.Server

	mu         sync.Mutex
	allowedUAs map[string]bool
}

func (s *multiSrv) setAllowedUAs(allowedUAs map[string]bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allowedUAs = allowedUAs
}

const slowRemoteSleepMillis = 1000

func httpMultiSrv(t *testing.T, token string, allowedUAs map[string]bool) *multiSrv {
	t.Helper()
	m := http.NewServeMux()

	server := httptest.NewUnstartedServer(m)
	ms := &multiSrv{server, sync.Mutex{}, allowedUAs}

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.UserAgent() == "slow remote" {
			time.Sleep(slowRemoteSleepMillis)
		}
		ms.mu.Lock()
		defer ms.mu.Unlock()
		if ms.allowedUAs[r.UserAgent()] {
			ch := core.Challenge{Token: token}
			keyAuthz, _ := ch.ExpectedKeyAuthorization(accountKey)
			fmt.Fprint(w, keyAuthz, "\n\r \t")
		} else {
			fmt.Fprint(w, "???")
		}
	})

	ms.Start()
	return ms
}

// cancelledVA is a mock that always returns context.Canceled for
// PerformValidation calls
type cancelledVA struct{}

func (v cancelledVA) PerformValidation(_ context.Context, _ *vapb.PerformValidationRequest, _ ...grpc.CallOption) (*vapb.ValidationResult, error) {
	return nil, context.Canceled
}

// brokenRemoteVA is a mock for the vapb.VAClient interface mocked to
// always return errors.
type brokenRemoteVA struct{}

// errBrokenRemoteVA is the error returned by a brokenRemoteVA's
// PerformValidation and IsSafeDomain functions.
var errBrokenRemoteVA = errors.New("brokenRemoteVA is broken")

// PerformValidation returns errBrokenRemoteVA unconditionally
func (b brokenRemoteVA) PerformValidation(_ context.Context, _ *vapb.PerformValidationRequest, _ ...grpc.CallOption) (*vapb.ValidationResult, error) {
	return nil, errBrokenRemoteVA
}

// localRemoteVA is a wrapper which fulfills the VAClient interface, but then
// forwards requests directly to its inner ValidationAuthorityImpl rather than
// over the network. This lets a local in-memory mock VA act like a remote VA.
type localRemoteVA struct {
	remote ValidationAuthorityImpl
}

func (lrva localRemoteVA) PerformValidation(ctx context.Context, req *vapb.PerformValidationRequest, _ ...grpc.CallOption) (*vapb.ValidationResult, error) {
	return lrva.remote.PerformValidation(ctx, req)
}

func TestValidateMalformedChallenge(t *testing.T) {
	va, _ := setup(nil, 0, "", nil)

	_, prob := va.validateChallenge(ctx, dnsi("example.com"), createChallenge("fake-type-01"))

	test.AssertEquals(t, prob.Type, probs.MalformedProblem)
}

func TestPerformValidationInvalid(t *testing.T) {
	va, _ := setup(nil, 0, "", nil)

	req := createValidationRequest("foo.com", core.ChallengeTypeDNS01)
	res, _ := va.PerformValidation(context.Background(), req)
	test.Assert(t, res.Problems != nil, "validation succeeded")

	test.AssertMetricWithLabelsEquals(t, va.metrics.validationTime, prometheus.Labels{
		"type":         "dns-01",
		"result":       "invalid",
		"problem_type": "unauthorized",
	}, 1)
}

func TestPerformValidationValid(t *testing.T) {
	va, mockLog := setup(nil, 0, "", nil)

	// create a challenge with well known token
	req := createValidationRequest("good-dns01.com", core.ChallengeTypeDNS01)
	res, _ := va.PerformValidation(context.Background(), req)
	test.Assert(t, res.Problems == nil, fmt.Sprintf("validation failed: %#v", res.Problems))

	test.AssertMetricWithLabelsEquals(t, va.metrics.validationTime, prometheus.Labels{
		"type":         "dns-01",
		"result":       "valid",
		"problem_type": "",
	}, 1)
	resultLog := mockLog.GetAllMatching(`Validation result`)
	if len(resultLog) != 1 {
		t.Fatalf("Wrong number of matching lines for 'Validation result'")
	}
	if !strings.Contains(resultLog[0], `"Hostname":"good-dns01.com"`) {
		t.Error("PerformValidation didn't log validation hostname.")
	}
}

// TestPerformValidationWildcard tests that the VA properly strips the `*.`
// prefix from a wildcard name provided to the PerformValidation function.
func TestPerformValidationWildcard(t *testing.T) {
	va, mockLog := setup(nil, 0, "", nil)

	// create a challenge with well known token
	req := createValidationRequest("*.good-dns01.com", core.ChallengeTypeDNS01)
	// perform a validation for a wildcard name
	res, _ := va.PerformValidation(context.Background(), req)
	test.Assert(t, res.Problems == nil, fmt.Sprintf("validation failed: %#v", res.Problems))

	test.AssertMetricWithLabelsEquals(t, va.metrics.validationTime, prometheus.Labels{
		"type":         "dns-01",
		"result":       "valid",
		"problem_type": "",
	}, 1)
	resultLog := mockLog.GetAllMatching(`Validation result`)
	if len(resultLog) != 1 {
		t.Fatalf("Wrong number of matching lines for 'Validation result'")
	}

	// We expect that the top level Hostname reflect the wildcard name
	if !strings.Contains(resultLog[0], `"Hostname":"*.good-dns01.com"`) {
		t.Errorf("PerformValidation didn't log correct validation hostname.")
	}
	// We expect that the ValidationRecord contain the correct non-wildcard
	// hostname that was validated
	if !strings.Contains(resultLog[0], `"hostname":"good-dns01.com"`) {
		t.Errorf("PerformValidation didn't log correct validation record hostname.")
	}
}

func TestMultiVA(t *testing.T) {
	// Create a new challenge to use for the httpSrv
	req := createValidationRequest("localhost", core.ChallengeTypeHTTP01)

	const (
		remoteUA1 = "remote 1"
		remoteUA2 = "remote 2"
		localUA   = "local 1"
	)
	allowedUAs := map[string]bool{
		localUA:   true,
		remoteUA1: true,
		remoteUA2: true,
	}

	// Create an IPv4 test server
	ms := httpMultiSrv(t, expectedToken, allowedUAs)
	defer ms.Close()

	remoteVA1 := setupRemote(ms.Server, remoteUA1)
	remoteVA2 := setupRemote(ms.Server, remoteUA2)

	remoteVAs := []RemoteVA{
		{remoteVA1, remoteUA1},
		{remoteVA2, remoteUA2},
	}

	enforceMultiVA := map[string]bool{
		"EnforceMultiVA": true,
	}
	enforceMultiVAFullResults := map[string]bool{
		"EnforceMultiVA":     true,
		"MultiVAFullResults": true,
	}
	noEnforceMultiVA := map[string]bool{
		"EnforceMultiVA": false,
	}
	noEnforceMultiVAFullResults := map[string]bool{
		"EnforceMultiVA":     false,
		"MultiVAFullResults": true,
	}

	unauthorized := probs.Unauthorized(fmt.Sprintf(
		`The key authorization file from the server did not match this challenge %q != "???"`,
		expectedKeyAuthorization))

	expectedInternalErrLine := fmt.Sprintf(
		`ERR: \[AUDIT\] Remote VA "broken".PerformValidation failed: %s`,
		errBrokenRemoteVA.Error())

	testCases := []struct {
		Name         string
		RemoteVAs    []RemoteVA
		AllowedUAs   map[string]bool
		Features     map[string]bool
		ExpectedProb *probs.ProblemDetails
		ExpectedLog  string
	}{
		{
			// With local and both remote VAs working there should be no problem.
			Name:       "Local and remote VAs OK, enforce multi VA",
			RemoteVAs:  remoteVAs,
			AllowedUAs: allowedUAs,
			Features:   enforceMultiVA,
		},
		{
			// Ditto if multi VA enforcement is disabled
			Name:       "Local and remote VAs OK, no enforce multi VA",
			RemoteVAs:  remoteVAs,
			AllowedUAs: allowedUAs,
			Features:   noEnforceMultiVA,
		},
		{
			// If the local VA fails everything should fail
			Name:         "Local VA bad, remote VAs OK, no enforce multi VA",
			RemoteVAs:    remoteVAs,
			AllowedUAs:   map[string]bool{remoteUA1: true, remoteUA2: true},
			Features:     noEnforceMultiVA,
			ExpectedProb: unauthorized,
		},
		{
			// Ditto when enforcing remote VA
			Name:         "Local VA bad, remote VAs OK, enforce multi VA",
			RemoteVAs:    remoteVAs,
			AllowedUAs:   map[string]bool{remoteUA1: true, remoteUA2: true},
			Features:     enforceMultiVA,
			ExpectedProb: unauthorized,
		},
		{
			// If a remote VA fails with an internal err it should fail when enforcing multi VA
			Name: "Local VA ok, remote VA internal err, enforce multi VA",
			RemoteVAs: []RemoteVA{
				{remoteVA1, remoteUA1},
				{&brokenRemoteVA{}, "broken"},
			},
			AllowedUAs:   allowedUAs,
			Features:     enforceMultiVA,
			ExpectedProb: probs.ServerInternal("During secondary validation: Remote PerformValidation RPC failed"),
			// The real failure cause should be logged
			ExpectedLog: expectedInternalErrLine,
		},
		{
			// If a remote VA fails with an internal err it should not fail when not
			// enforcing multi VA
			Name: "Local VA ok, remote VA internal err, no enforce multi VA",
			RemoteVAs: []RemoteVA{
				{remoteVA1, remoteUA1},
				{&brokenRemoteVA{}, "broken"},
			},
			AllowedUAs: allowedUAs,
			Features:   noEnforceMultiVA,
			// Like above, the real failure cause will be logged eventually, but that
			// will happen asynchronously. It's not guaranteed to happen before the
			// test case exits, so we don't check for it here.
		},
		{
			// With only one working remote VA there should *not* be a validation
			// failure when not enforcing multi VA.
			Name:       "Local VA and one remote VA OK, no enforce multi VA",
			RemoteVAs:  remoteVAs,
			AllowedUAs: map[string]bool{localUA: true, remoteUA2: true},
			Features:   noEnforceMultiVA,
		},
		{
			// With only one working remote VA there should be a validation failure
			// when enforcing multi VA.
			Name:       "Local VA and one remote VA OK, enforce multi VA",
			RemoteVAs:  remoteVAs,
			AllowedUAs: map[string]bool{localUA: true, remoteUA2: true},
			Features:   enforceMultiVA,
			ExpectedProb: probs.Unauthorized(fmt.Sprintf(
				`During secondary validation: The key authorization file from the server did not match this challenge %q != "???"`,
				expectedKeyAuthorization)),
		},
		{
			// When enforcing multi-VA, any cancellations are a problem.
			Name: "Local VA and one remote VA OK, one cancelled VA, enforce multi VA",
			RemoteVAs: []RemoteVA{
				{remoteVA1, remoteUA1},
				{cancelledVA{}, remoteUA2},
			},
			AllowedUAs:   allowedUAs,
			Features:     enforceMultiVA,
			ExpectedProb: probs.ServerInternal("During secondary validation: Remote PerformValidation RPC canceled"),
		},
		{
			// When enforcing multi-VA, any cancellations are a problem.
			Name: "Local VA OK, two cancelled remote VAs, enforce multi VA",
			RemoteVAs: []RemoteVA{
				{cancelledVA{}, remoteUA1},
				{cancelledVA{}, remoteUA2},
			},
			AllowedUAs:   allowedUAs,
			Features:     enforceMultiVA,
			ExpectedProb: probs.ServerInternal("During secondary validation: Remote PerformValidation RPC canceled"),
		},
		{
			// With the local and remote VAs seeing diff problems and the full results
			// feature flag on but multi VA enforcement off we expect
			// no problem.
			Name:       "Local and remote VA differential, full results, no enforce multi VA",
			RemoteVAs:  remoteVAs,
			AllowedUAs: map[string]bool{localUA: true},
			Features:   noEnforceMultiVAFullResults,
		},
		{
			// With the local and remote VAs seeing diff problems and the full results
			// feature flag on and multi VA enforcement on we expect a problem.
			Name:       "Local and remote VA differential, full results, enforce multi VA",
			RemoteVAs:  remoteVAs,
			AllowedUAs: map[string]bool{localUA: true},
			Features:   enforceMultiVAFullResults,
			ExpectedProb: probs.Unauthorized(fmt.Sprintf(
				`During secondary validation: The key authorization file from the server did not match this challenge %q != "???"`,
				expectedKeyAuthorization)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Configure the test server with the testcase allowed UAs.
			ms.setAllowedUAs(tc.AllowedUAs)

			// Configure a primary VA with testcase remote VAs.
			localVA, mockLog := setup(ms.Server, 0, localUA, tc.RemoteVAs)

			if tc.Features != nil {
				err := features.Set(tc.Features)
				test.AssertNotError(t, err, "Failed to set feature flags")
				defer features.Reset()
			}

			// Perform all validations
			res, _ := localVA.PerformValidation(ctx, req)
			if res.Problems == nil && tc.ExpectedProb != nil {
				t.Errorf("expected prob %v, got nil", tc.ExpectedProb)
			} else if res.Problems != nil && tc.ExpectedProb == nil {
				t.Errorf("expected no prob, got %v", res.Problems)
			} else if res.Problems != nil && tc.ExpectedProb != nil {
				// That result should match expected.
				test.AssertEquals(t, res.Problems.ProblemType, string(tc.ExpectedProb.Type))
				test.AssertEquals(t, res.Problems.Detail, tc.ExpectedProb.Detail)
			}

			if tc.ExpectedLog != "" {
				lines := mockLog.GetAllMatching(tc.ExpectedLog)
				if len(lines) != 1 {
					t.Fatalf("Got log %v; expected %q", mockLog.GetAll(), tc.ExpectedLog)
				}
			}
		})
	}
}

func TestMultiVAEarlyReturn(t *testing.T) {
	const (
		remoteUA1 = "remote 1"
		remoteUA2 = "slow remote"
		localUA   = "local 1"
	)
	allowedUAs := map[string]bool{
		localUA:   true,
		remoteUA1: false, // forbid UA 1 to provoke early return
		remoteUA2: true,
	}

	ms := httpMultiSrv(t, expectedToken, allowedUAs)
	defer ms.Close()

	remoteVA1 := setupRemote(ms.Server, remoteUA1)
	remoteVA2 := setupRemote(ms.Server, remoteUA2)

	remoteVAs := []RemoteVA{
		{remoteVA1, remoteUA1},
		{remoteVA2, remoteUA2},
	}

	// Create a local test VA with the two remote VAs
	localVA, mockLog := setup(ms.Server, 0, localUA, remoteVAs)

	testCases := []struct {
		Name        string
		EarlyReturn bool
	}{
		{
			Name: "One slow remote VA, no early return",
		},
		{
			Name:        "One slow remote VA, early return",
			EarlyReturn: true,
		},
	}

	earlyReturnFeatures := map[string]bool{
		"EnforceMultiVA":     true,
		"MultiVAFullResults": false,
	}
	noEarlyReturnFeatures := map[string]bool{
		"EnforceMultiVA":     true,
		"MultiVAFullResults": true,
	}

	req := createValidationRequest("localhost", core.ChallengeTypeHTTP01)
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			mockLog.Clear()

			var err error
			if tc.EarlyReturn {
				err = features.Set(earlyReturnFeatures)
			} else {
				err = features.Set(noEarlyReturnFeatures)
			}
			test.AssertNotError(t, err, "Failed to set MultiVAFullResults feature flag")
			defer features.Reset()

			start := time.Now()

			// Perform all validations
			res, _ := localVA.PerformValidation(ctx, req)
			// It should always fail
			if res.Problems == nil {
				t.Error("expected prob from PerformValidation, got nil")
			}

			elapsed := time.Since(start).Round(time.Millisecond).Milliseconds()

			// The slow UA should sleep for `slowRemoteSleepMillis`. In the early return
			// case the first remote VA should fail the overall validation and a prob
			// should be returned quickly (i.e. in less than half of `slowRemoteSleepMillis`).
			// In the non-early return case we don't expect a problem until
			// `slowRemoteSleepMillis`.
			if tc.EarlyReturn && elapsed > slowRemoteSleepMillis/2 {
				t.Errorf(
					"Expected an early return from PerformValidation in < %d ms, took %d ms",
					slowRemoteSleepMillis/2, elapsed)
			}
		})
	}
}

func TestMultiVAPolicy(t *testing.T) {
	const (
		remoteUA1 = "remote 1"
		remoteUA2 = "remote 2"
		localUA   = "local 1"
	)
	// Forbid both remote UAs to ensure that multi-va fails
	allowedUAs := map[string]bool{
		localUA:   true,
		remoteUA1: false,
		remoteUA2: false,
	}

	ms := httpMultiSrv(t, expectedToken, allowedUAs)
	defer ms.Close()

	remoteVA1 := setupRemote(ms.Server, remoteUA1)
	remoteVA2 := setupRemote(ms.Server, remoteUA2)

	remoteVAs := []RemoteVA{
		{remoteVA1, remoteUA1},
		{remoteVA2, remoteUA2},
	}

	// Create a local test VA with the two remote VAs
	localVA, _ := setup(ms.Server, 0, localUA, remoteVAs)

	// Ensure multi VA enforcement is enabled, don't wait for full multi VA
	// results.
	err := features.Set(map[string]bool{
		"EnforceMultiVA":     true,
		"MultiVAFullResults": false,
	})
	test.AssertNotError(t, err, "setting feature flags")
	defer features.Reset()

	// Perform validation for a domain not in the disabledDomains list
	req := createValidationRequest("letsencrypt.org", core.ChallengeTypeHTTP01)
	res, _ := localVA.PerformValidation(ctx, req)
	// It should fail
	if res.Problems == nil {
		t.Error("expected prob from PerformValidation, got nil")
	}
}

func TestDetailedError(t *testing.T) {
	cases := []struct {
		err      error
		ip       net.IP
		expected string
	}{
		{
			err: ipError{
				ip: net.ParseIP("192.168.1.1"),
				err: &net.OpError{
					Op:  "dial",
					Net: "tcp",
					Err: &os.SyscallError{
						Syscall: "getsockopt",
						Err:     syscall.ECONNREFUSED,
					},
				},
			},
			expected: "192.168.1.1: Connection refused",
		},
		{
			err: &net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: &os.SyscallError{
					Syscall: "getsockopt",
					Err:     syscall.ECONNREFUSED,
				},
			},
			expected: "Connection refused",
		},
		{
			err: &net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: &os.SyscallError{
					Syscall: "getsockopt",
					Err:     syscall.ECONNRESET,
				},
			},
			ip:       nil,
			expected: "Connection reset by peer",
		},
	}
	for _, tc := range cases {
		actual := detailedError(tc.err).Detail
		if actual != tc.expected {
			t.Errorf("Wrong detail for %v. Got %q, expected %q", tc.err, actual, tc.expected)
		}
	}
}

func TestLogRemoteValidationDifferentials(t *testing.T) {
	// Create some remote VAs
	remoteVA1 := setupRemote(nil, "remote 1")
	remoteVA2 := setupRemote(nil, "remote 2")
	remoteVA3 := setupRemote(nil, "remote 3")
	remoteVAs := []RemoteVA{
		{remoteVA1, "remote 1"},
		{remoteVA2, "remote 2"},
		{remoteVA3, "remote 3"},
	}

	// Set up a local VA that allows a max of 2 remote failures.
	localVA, mockLog := setup(nil, 2, "local 1", remoteVAs)

	egProbA := probs.DNS("root DNS servers closed at 4:30pm")
	egProbB := probs.OrderNotReady("please take a number")

	testCases := []struct {
		name          string
		primaryResult *probs.ProblemDetails
		remoteProbs   []*remoteValidationResult
		expectedLog   string
	}{
		{
			name:          "remote and primary results equal (all nil)",
			primaryResult: nil,
			remoteProbs: []*remoteValidationResult{
				{Problem: nil, VAHostname: "remoteA"},
				{Problem: nil, VAHostname: "remoteB"},
				{Problem: nil, VAHostname: "remoteC"},
			},
		},
		{
			name:          "remote and primary results equal (not nil)",
			primaryResult: egProbA,
			remoteProbs: []*remoteValidationResult{
				{Problem: egProbA, VAHostname: "remoteA"},
				{Problem: egProbA, VAHostname: "remoteB"},
				{Problem: egProbA, VAHostname: "remoteC"},
			},
		},
		{
			name:          "remote and primary differ (primary nil)",
			primaryResult: nil,
			remoteProbs: []*remoteValidationResult{
				{Problem: egProbA, VAHostname: "remoteA"},
				{Problem: nil, VAHostname: "remoteB"},
				{Problem: egProbB, VAHostname: "remoteC"},
			},
			expectedLog: `INFO: remoteVADifferentials JSON={"Domain":"example.com","AccountID":1999,"ChallengeType":"blorpus-01","PrimaryResult":null,"RemoteSuccesses":1,"RemoteFailures":[{"VAHostname":"remoteA","Problem":{"type":"dns","detail":"root DNS servers closed at 4:30pm","status":400}},{"VAHostname":"remoteC","Problem":{"type":"orderNotReady","detail":"please take a number","status":403}}]}`,
		},
		{
			name:          "remote and primary differ (primary not nil)",
			primaryResult: egProbA,
			remoteProbs: []*remoteValidationResult{
				{Problem: nil, VAHostname: "remoteA"},
				{Problem: egProbB, VAHostname: "remoteB"},
				{Problem: nil, VAHostname: "remoteC"},
			},
			expectedLog: `INFO: remoteVADifferentials JSON={"Domain":"example.com","AccountID":1999,"ChallengeType":"blorpus-01","PrimaryResult":{"type":"dns","detail":"root DNS servers closed at 4:30pm","status":400},"RemoteSuccesses":2,"RemoteFailures":[{"VAHostname":"remoteB","Problem":{"type":"orderNotReady","detail":"please take a number","status":403}}]}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockLog.Clear()

			localVA.logRemoteValidationDifferentials(
				"example.com", 1999, "blorpus-01", tc.primaryResult, tc.remoteProbs)

			lines := mockLog.GetAllMatching("remoteVADifferentials JSON=.*")
			if tc.expectedLog != "" {
				test.AssertEquals(t, len(lines), 1)
				test.AssertEquals(t, lines[0], tc.expectedLog)
			} else {
				test.AssertEquals(t, len(lines), 0)
			}
		})
	}
}
