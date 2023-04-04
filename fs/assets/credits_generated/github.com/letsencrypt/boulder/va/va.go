package va

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/honeycombio/beeline-go"
	"github.com/jmhodges/clock"
	"github.com/letsencrypt/boulder/bdns"
	"github.com/letsencrypt/boulder/canceled"
	"github.com/letsencrypt/boulder/core"
	berrors "github.com/letsencrypt/boulder/errors"
	"github.com/letsencrypt/boulder/features"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	"github.com/letsencrypt/boulder/identifier"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/probs"
	vapb "github.com/letsencrypt/boulder/va/proto"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// badTLSHeader contains the string 'HTTP /' which is returned when
	// we try to talk TLS to a server that only talks HTTP
	badTLSHeader = []byte{0x48, 0x54, 0x54, 0x50, 0x2f}
	// h2SettingsFrameErrRegex is a regex against a net/http error indicating
	// a malformed HTTP response that matches the initial SETTINGS frame of an
	// HTTP/2 connection. This happens when a server configures HTTP/2 on port
	// :80, failing HTTP-01 challenges.
	//
	// The regex first matches the error string prefix and then matches the raw
	// bytes of an arbitrarily sized HTTP/2 SETTINGS frame:
	//   0x00 0x00 0x?? 0x04 0x00 0x00 0x00 0x00
	//
	// The third byte is variable and indicates the frame size. Typically
	// this will be 0x12.
	// The 0x04 in the fourth byte indicates that the frame is SETTINGS type.
	//
	// See:
	//   * https://tools.ietf.org/html/rfc7540#section-4.1
	//   * https://tools.ietf.org/html/rfc7540#section-6.5
	//
	// NOTE(@cpu): Using a regex is a hack but unfortunately for this case
	// http.Client.Do() will return a url.Error err that wraps
	// a errors.ErrorString instance. There isn't much else to do with one of
	// those except match the encoded byte string with a regex. :-X
	//
	// NOTE(@cpu): The first component of this regex is optional to avoid an
	// integration test flake. In some (fairly rare) conditions the malformed
	// response error will be returned simply as a http.badStringError without
	// the broken transport prefix. Most of the time the error is returned with
	// a transport connection error prefix.
	h2SettingsFrameErrRegex = regexp.MustCompile(`(?:net\/http\: HTTP\/1\.x transport connection broken: )?malformed HTTP response \"\\x00\\x00\\x[a-f0-9]{2}\\x04\\x00\\x00\\x00\\x00\\x00.*"`)
)

// RemoteVA wraps the vapb.VAClient interface and adds a field containing the
// address of the remote gRPC server since the underlying gRPC client doesn't
// provide a way to extract this metadata which is useful for debugging gRPC
// connection issues.
type RemoteVA struct {
	vapb.VAClient
	Address string
}

type vaMetrics struct {
	validationTime                      *prometheus.HistogramVec
	localValidationTime                 *prometheus.HistogramVec
	remoteValidationTime                *prometheus.HistogramVec
	remoteValidationFailures            prometheus.Counter
	prospectiveRemoteValidationFailures prometheus.Counter
	tlsALPNOIDCounter                   *prometheus.CounterVec
	http01Fallbacks                     prometheus.Counter
	http01Redirects                     prometheus.Counter
	caaCounter                          *prometheus.CounterVec
	ipv4FallbackCounter                 prometheus.Counter
}

func initMetrics(stats prometheus.Registerer) *vaMetrics {
	validationTime := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "validation_time",
			Help:    "Total time taken to validate a challenge and aggregate results",
			Buckets: metrics.InternetFacingBuckets,
		},
		[]string{"type", "result", "problem_type"})
	stats.MustRegister(validationTime)
	localValidationTime := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "local_validation_time",
			Help:    "Time taken to locally validate a challenge",
			Buckets: metrics.InternetFacingBuckets,
		},
		[]string{"type", "result"})
	stats.MustRegister(localValidationTime)
	remoteValidationTime := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "remote_validation_time",
			Help:    "Time taken to remotely validate a challenge",
			Buckets: metrics.InternetFacingBuckets,
		},
		[]string{"type", "result"})
	stats.MustRegister(remoteValidationTime)
	remoteValidationFailures := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "remote_validation_failures",
			Help: "Number of validations failed due to remote VAs returning failure when consensus is enforced",
		})
	stats.MustRegister(remoteValidationFailures)
	prospectiveRemoteValidationFailures := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "prospective_remote_validation_failures",
			Help: "Number of validations that would have failed due to remote VAs returning failure if consesus were enforced",
		})
	stats.MustRegister(prospectiveRemoteValidationFailures)
	tlsALPNOIDCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tls_alpn_oid_usage",
			Help: "Number of TLS ALPN validations using either of the two OIDs",
		},
		[]string{"oid"},
	)
	stats.MustRegister(tlsALPNOIDCounter)
	http01Fallbacks := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "http01_fallbacks",
			Help: "Number of IPv6 to IPv4 HTTP-01 fallback requests made",
		})
	stats.MustRegister(http01Fallbacks)
	http01Redirects := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "http01_redirects",
			Help: "Number of HTTP-01 redirects followed",
		})
	stats.MustRegister(http01Redirects)
	caaCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "caa_sets_processed",
		Help: "A counter of CAA sets processed labelled by result",
	}, []string{"result"})
	stats.MustRegister(caaCounter)
	ipv4FallbackCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tls_alpn_ipv4_fallback",
		Help: "A counter of IPv4 fallbacks during TLS ALPN validation",
	})
	stats.MustRegister(ipv4FallbackCounter)

	return &vaMetrics{
		validationTime:                      validationTime,
		remoteValidationTime:                remoteValidationTime,
		localValidationTime:                 localValidationTime,
		remoteValidationFailures:            remoteValidationFailures,
		prospectiveRemoteValidationFailures: prospectiveRemoteValidationFailures,
		tlsALPNOIDCounter:                   tlsALPNOIDCounter,
		http01Fallbacks:                     http01Fallbacks,
		http01Redirects:                     http01Redirects,
		caaCounter:                          caaCounter,
		ipv4FallbackCounter:                 ipv4FallbackCounter,
	}
}

// PortConfig specifies what ports the VA should call to on the remote
// host when performing its checks.
type portConfig struct {
	HTTPPort  int
	HTTPSPort int
	TLSPort   int
}

// newDefaultPortConfig is a constructor which returns a portConfig with default
// settings.
//
// CABF BRs section 1.6.1: Authorized Ports: One of the following ports: 80
// (http), 443 (https), 25 (smtp), 22 (ssh).
//
// RFC 8555 section 8.3: Dereference the URL using an HTTP GET request. This
// request MUST be sent to TCP port 80 on the HTTP server.
//
// RFC 8737 section 3: The ACME server initiates a TLS connection to the chosen
// IP address. This connection MUST use TCP port 443.
func newDefaultPortConfig() *portConfig {
	return &portConfig{
		HTTPPort:  80,
		HTTPSPort: 443,
		TLSPort:   443,
	}
}

// ValidationAuthorityImpl represents a VA
type ValidationAuthorityImpl struct {
	vapb.UnimplementedVAServer
	vapb.UnimplementedCAAServer
	log                blog.Logger
	dnsClient          bdns.Client
	issuerDomain       string
	httpPort           int
	httpsPort          int
	tlsPort            int
	userAgent          string
	clk                clock.Clock
	remoteVAs          []RemoteVA
	maxRemoteFailures  int
	accountURIPrefixes []string
	singleDialTimeout  time.Duration

	metrics *vaMetrics
}

// NewValidationAuthorityImpl constructs a new VA
func NewValidationAuthorityImpl(
	resolver bdns.Client,
	remoteVAs []RemoteVA,
	maxRemoteFailures int,
	userAgent string,
	issuerDomain string,
	stats prometheus.Registerer,
	clk clock.Clock,
	logger blog.Logger,
	accountURIPrefixes []string,
) (*ValidationAuthorityImpl, error) {

	if features.Enabled(features.CAAAccountURI) && len(accountURIPrefixes) == 0 {
		return nil, errors.New("no account URI prefixes configured")
	}

	pc := newDefaultPortConfig()

	va := &ValidationAuthorityImpl{
		log:                logger,
		dnsClient:          resolver,
		issuerDomain:       issuerDomain,
		httpPort:           pc.HTTPPort,
		httpsPort:          pc.HTTPSPort,
		tlsPort:            pc.TLSPort,
		userAgent:          userAgent,
		clk:                clk,
		metrics:            initMetrics(stats),
		remoteVAs:          remoteVAs,
		maxRemoteFailures:  maxRemoteFailures,
		accountURIPrefixes: accountURIPrefixes,
		// singleDialTimeout specifies how long an individual `DialContext` operation may take
		// before timing out. This timeout ignores the base RPC timeout and is strictly
		// used for the DialContext operations that take place during an
		// HTTP-01 challenge validation.
		singleDialTimeout: 10 * time.Second,
	}

	return va, nil
}

// Used for audit logging
type verificationRequestEvent struct {
	ID                string         `json:",omitempty"`
	Requester         int64          `json:",omitempty"`
	Hostname          string         `json:",omitempty"`
	Challenge         core.Challenge `json:",omitempty"`
	ValidationLatency float64
	Error             string `json:",omitempty"`
}

// ipError is an error type used to pass though the IP address of the remote
// host when an error occurs during HTTP-01 and TLS-ALPN domain validation.
type ipError struct {
	ip  net.IP
	err error
}

// Unwrap returns the underlying error.
func (i ipError) Unwrap() error {
	return i.err
}

// Error returns a string representation of the error.
func (i ipError) Error() string {
	return fmt.Sprintf("%s: %s", i.ip, i.err)
}

// detailedError returns a ProblemDetails corresponding to an error
// that occurred during HTTP-01 or TLS-ALPN domain validation. Specifically it
// tries to unwrap known Go error types and present something a little more
// meaningful. It additionally handles `berrors.ConnectionFailure` errors by
// passing through the detailed message.
func detailedError(err error) *probs.ProblemDetails {
	var ipErr ipError
	if errors.As(err, &ipErr) {
		detailedErr := detailedError(ipErr.err)
		if ipErr.ip == nil {
			// This should never happen.
			return detailedErr
		}
		// Prefix the error message with the IP address of the remote host.
		detailedErr.Detail = fmt.Sprintf("%s: %s", ipErr.ip, detailedErr.Detail)
		return detailedErr
	}
	// net/http wraps net.OpError in a url.Error. Unwrap them.
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		prob := detailedError(urlErr.Err)
		prob.Detail = fmt.Sprintf("Fetching %s: %s", urlErr.URL, prob.Detail)
		return prob
	}

	var tlsErr tls.RecordHeaderError
	if errors.As(err, &tlsErr) && bytes.Equal(tlsErr.RecordHeader[:], badTLSHeader) {
		return probs.Malformed("Server only speaks HTTP, not TLS")
	}

	var netOpErr *net.OpError
	if errors.As(err, &netOpErr) {
		if fmt.Sprintf("%T", netOpErr.Err) == "tls.alert" {
			// All the tls.alert error strings are reasonable to hand back to a
			// user. Confirmed against Go 1.8.
			return probs.TLSError(netOpErr.Error())
		} else if netOpErr.Timeout() && netOpErr.Op == "dial" {
			return probs.ConnectionFailure("Timeout during connect (likely firewall problem)")
		} else if netOpErr.Timeout() {
			return probs.ConnectionFailure(fmt.Sprintf("Timeout during %s (your server may be slow or overloaded)", netOpErr.Op))
		}
	}
	var syscallErr *os.SyscallError
	if errors.As(err, &syscallErr) {
		switch syscallErr.Err {
		case syscall.ECONNREFUSED:
			return probs.ConnectionFailure("Connection refused")
		case syscall.ENETUNREACH:
			return probs.ConnectionFailure("Network unreachable")
		case syscall.ECONNRESET:
			return probs.ConnectionFailure("Connection reset by peer")
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return probs.ConnectionFailure("Timeout after connect (your server may be slow or overloaded)")
	}
	if errors.Is(err, berrors.ConnectionFailure) {
		return probs.ConnectionFailure(err.Error())
	}
	if errors.Is(err, berrors.Unauthorized) {
		return probs.Unauthorized(err.Error())
	}
	if errors.Is(err, berrors.DNS) {
		return probs.DNS(err.Error())
	}

	if h2SettingsFrameErrRegex.MatchString(err.Error()) {
		return probs.ConnectionFailure("Server is speaking HTTP/2 over HTTP")
	}
	return probs.ConnectionFailure("Error getting validation data")
}

// validate performs a challenge validation and, in parallel,
// checks CAA and GSB for the identifier. If any of those steps fails, it
// returns a ProblemDetails plus the validation records created during the
// validation attempt.
func (va *ValidationAuthorityImpl) validate(
	ctx context.Context,
	identifier identifier.ACMEIdentifier,
	regid int64,
	challenge core.Challenge,
) ([]core.ValidationRecord, *probs.ProblemDetails) {

	// If the identifier is a wildcard domain we need to validate the base
	// domain by removing the "*." wildcard prefix. We create a separate
	// `baseIdentifier` here before starting the `va.checkCAA` goroutine with the
	// `identifier` to avoid a data race.
	baseIdentifier := identifier
	if strings.HasPrefix(identifier.Value, "*.") {
		baseIdentifier.Value = strings.TrimPrefix(identifier.Value, "*.")
	}

	// va.checkCAA accepts wildcard identifiers and handles them appropriately so
	// we can dispatch `checkCAA` with the provided `identifier` instead of
	// `baseIdentifier`
	ch := make(chan *probs.ProblemDetails, 1)
	go func() {
		params := &caaParams{
			accountURIID:     regid,
			validationMethod: challenge.Type,
		}
		ch <- va.checkCAA(ctx, identifier, params)
	}()

	// TODO(#1292): send into another goroutine
	validationRecords, prob := va.validateChallenge(ctx, baseIdentifier, challenge)
	if prob != nil {
		// The ProblemDetails will be serialized through gRPC, which requires UTF-8.
		// It will also later be serialized in JSON, which defaults to UTF-8. Make
		// sure it is UTF-8 clean now.
		prob = filterProblemDetails(prob)

		return validationRecords, prob
	}

	for i := 0; i < cap(ch); i++ {
		if extraProblem := <-ch; extraProblem != nil {
			return validationRecords, extraProblem
		}
	}
	return validationRecords, nil
}

func (va *ValidationAuthorityImpl) validateChallenge(ctx context.Context, identifier identifier.ACMEIdentifier, challenge core.Challenge) ([]core.ValidationRecord, *probs.ProblemDetails) {
	err := challenge.CheckConsistencyForValidation()
	if err != nil {
		return nil, probs.Malformed("Challenge failed consistency check: %s", err)
	}
	switch challenge.Type {
	case core.ChallengeTypeHTTP01:
		return va.validateHTTP01(ctx, identifier, challenge)
	case core.ChallengeTypeDNS01:
		return va.validateDNS01(ctx, identifier, challenge)
	case core.ChallengeTypeTLSALPN01:
		return va.validateTLSALPN01(ctx, identifier, challenge)
	}
	return nil, probs.Malformed("invalid challenge type %s", challenge.Type)
}

// performRemoteValidation calls `PerformValidation` for each of the configured
// remoteVAs in a random order. The provided `results` chan should have an equal
// size to the number of remote VAs. The validations will be performed in
// separate go-routines. If the result `error` from a remote
// `PerformValidation` RPC is nil or a nil `ProblemDetails` instance it is
// written directly to the `results` chan. If the err is a cancelled error it is
// treated as a nil error. Otherwise the error/problem is written to the results
// channel as-is.
func (va *ValidationAuthorityImpl) performRemoteValidation(
	ctx context.Context,
	req *vapb.PerformValidationRequest,
	results chan *remoteValidationResult) {
	for _, i := range rand.Perm(len(va.remoteVAs)) {
		remoteVA := va.remoteVAs[i]
		go func(rva RemoteVA) {
			result := &remoteValidationResult{
				VAHostname: rva.Address,
			}
			res, err := rva.PerformValidation(ctx, req)
			if err != nil && canceled.Is(err) {
				// If the non-nil err was a canceled error, ignore it. That's fine: it
				// just means we cancelled the remote VA request before it was
				// finished because we didn't care about its result. Don't log to avoid
				// spamming the logs.
				result.Problem = probs.ServerInternal("Remote PerformValidation RPC canceled")
			} else if err != nil {
				// This is a real error, not just a problem with the validation.
				va.log.Errf("Remote VA %q.PerformValidation failed: %s", rva.Address, err)
				result.Problem = probs.ServerInternal("Remote PerformValidation RPC failed")
			} else if res.Problems != nil {
				prob, err := bgrpc.PBToProblemDetails(res.Problems)
				if err != nil {
					va.log.Infof("Remote VA %q.PerformValidation returned malformed problem: %s", rva.Address, err)
					result.Problem = probs.ServerInternal(
						fmt.Sprintf("Remote PerformValidation RPC returned malformed result: %s", err))
				} else {
					va.log.Infof("Remote VA %q.PerformValidation returned problem: %s", rva.Address, prob)
					result.Problem = prob
				}
			}
			results <- result
		}(remoteVA)
	}
}

// processRemoteResults evaluates a primary VA result, and a channel of remote
// VA problems to produce a single overall validation result based on configured
// feature flags. The overall result is calculated based on the VA's configured
// `maxRemoteFailures` value.
//
// If the `MultiVAFullResults` feature is enabled then `processRemoteResults`
// will expect to read a result from the `remoteErrors` channel for each VA and
// will not produce an overall result until all remote VAs have responded. In
// this case `logRemoteFailureDifferentials` will also be called to describe the
// differential between the primary and all of the remote VAs.
//
// If the `MultiVAFullResults` feature flag is not enabled then
// `processRemoteResults` will potentially return before all remote VAs have had
// a chance to respond. This happens if the success or failure threshold is met.
// This doesn't allow for logging the differential between the primary and
// remote VAs but is more performant.
func (va *ValidationAuthorityImpl) processRemoteResults(
	domain string,
	acctID int64,
	challengeType string,
	primaryResult *probs.ProblemDetails,
	remoteResultsChan chan *remoteValidationResult,
	numRemoteVAs int) *probs.ProblemDetails {

	state := "failure"
	start := va.clk.Now()

	defer func() {
		va.metrics.remoteValidationTime.With(prometheus.Labels{
			"type":   challengeType,
			"result": state,
		}).Observe(va.clk.Since(start).Seconds())
	}()

	required := numRemoteVAs - va.maxRemoteFailures
	good := 0
	bad := 0

	var remoteResults []*remoteValidationResult
	var firstProb *probs.ProblemDetails
	// Due to channel behavior this could block indefinitely and we rely on gRPC
	// honoring the context deadline used in client calls to prevent that from
	// happening.
	for result := range remoteResultsChan {
		// Add the result to the slice
		remoteResults = append(remoteResults, result)
		if result.Problem == nil {
			good++
		} else {
			bad++
		}

		// Store the first non-nil problem to return later (if `MultiVAFullResults`
		// is enabled).
		if firstProb == nil && result.Problem != nil {
			firstProb = result.Problem
		}

		// If MultiVAFullResults isn't enabled then return early whenever the
		// success or failure threshold is met.
		if !features.Enabled(features.MultiVAFullResults) {
			if good >= required {
				state = "success"
				return nil
			} else if bad > va.maxRemoteFailures {
				modifiedProblem := *result.Problem
				modifiedProblem.Detail = "During secondary validation: " + firstProb.Detail
				return &modifiedProblem
			}
		}

		// If we haven't returned early because of MultiVAFullResults being enabled
		// we need to break the loop once all of the VAs have returned a result.
		if len(remoteResults) == numRemoteVAs {
			break
		}
	}

	// If we are using `features.MultiVAFullResults` then we haven't returned
	// early and can now log the differential between what the primary VA saw and
	// what all of the remote VAs saw.
	va.logRemoteValidationDifferentials(
		domain,
		acctID,
		challengeType,
		primaryResult,
		remoteResults)

	// Based on the threshold of good/bad return nil or a problem.
	if good >= required {
		state = "success"
		return nil
	} else if bad > va.maxRemoteFailures {
		modifiedProblem := *firstProb
		modifiedProblem.Detail = "During secondary validation: " + firstProb.Detail
		return &modifiedProblem
	}

	// This condition should not occur - it indicates the good/bad counts didn't
	// meet either the required threshold or the maxRemoteFailures threshold.
	return probs.ServerInternal("Too few remote PerformValidation RPC results")
}

// logRemoteValidationDifferentials is called by `processRemoteResults` when the
// `MultiVAFullResults` feature flag is enabled. It produces a JSON log line
// that contains the primary VA result and the results each remote VA returned.
func (va *ValidationAuthorityImpl) logRemoteValidationDifferentials(
	domain string,
	acctID int64,
	challengeType string,
	primaryResult *probs.ProblemDetails,
	remoteResults []*remoteValidationResult) {

	var successes []*remoteValidationResult
	var failures []*remoteValidationResult

	allEqual := true
	for _, result := range remoteResults {
		if result.Problem != primaryResult {
			allEqual = false
		}
		if result.Problem == nil {
			successes = append(successes, result)
		} else {
			failures = append(failures, result)
		}
	}
	if allEqual {
		// There's no point logging a differential line if the primary VA and remote
		// VAs all agree.
		return
	}

	// If the primary result was OK and there were more failures than the allowed
	// threshold increment a stat that indicates this overall validation will have
	// failed if features.EnforceMultiVA is enabled.
	if primaryResult == nil && len(failures) > va.maxRemoteFailures {
		va.metrics.prospectiveRemoteValidationFailures.Inc()
	}

	logOb := struct {
		Domain          string
		AccountID       int64
		ChallengeType   string
		PrimaryResult   *probs.ProblemDetails
		RemoteSuccesses int
		RemoteFailures  []*remoteValidationResult
	}{
		Domain:          domain,
		AccountID:       acctID,
		ChallengeType:   challengeType,
		PrimaryResult:   primaryResult,
		RemoteSuccesses: len(successes),
		RemoteFailures:  failures,
	}

	logJSON, err := json.Marshal(logOb)
	if err != nil {
		// log a warning - a marshaling failure isn't expected given the data and
		// isn't critical enough to break validation for by returning an error to
		// the caller.
		va.log.Warningf("Could not marshal log object in "+
			"logRemoteValidationDifferentials: %s", err)
		return
	}

	va.log.Infof("remoteVADifferentials JSON=%s", string(logJSON))
}

// remoteValidationResult is a struct that combines a problem details instance
// (that may be nil) with the remote VA hostname that produced it.
type remoteValidationResult struct {
	VAHostname string
	Problem    *probs.ProblemDetails
}

// PerformValidation validates the challenge for the domain in the request.
// The returned result will always contain a list of validation records, even
// when it also contains a problem.
func (va *ValidationAuthorityImpl) PerformValidation(ctx context.Context, req *vapb.PerformValidationRequest) (*vapb.ValidationResult, error) {
	if core.IsAnyNilOrZero(req, req.Domain, req.Challenge, req.Authz) {
		return nil, berrors.InternalServerError("Incomplete validation request")
	}
	logEvent := verificationRequestEvent{
		ID:        req.Authz.Id,
		Requester: req.Authz.RegID,
		Hostname:  req.Domain,
	}
	beeline.AddFieldToTrace(ctx, "authz.id", req.Authz.Id)
	beeline.AddFieldToTrace(ctx, "acct.id", req.Authz.RegID)
	beeline.AddFieldToTrace(ctx, "authz.dnsname", req.Domain)
	vStart := va.clk.Now()

	var remoteResults chan *remoteValidationResult
	if remoteVACount := len(va.remoteVAs); remoteVACount > 0 {
		remoteResults = make(chan *remoteValidationResult, remoteVACount)
		go va.performRemoteValidation(ctx, req, remoteResults)
	}

	challenge, err := bgrpc.PBToChallenge(req.Challenge)
	if err != nil {
		return nil, probs.ServerInternal("Challenge failed to deserialize")
	}

	records, prob := va.validate(ctx, identifier.DNSIdentifier(req.Domain), req.Authz.RegID, challenge)
	challenge.ValidationRecord = records
	localValidationLatency := time.Since(vStart)

	// Check for malformed ValidationRecords
	if !challenge.RecordsSane() && prob == nil {
		prob = probs.ServerInternal("Records for validation failed sanity check")
	}

	var problemType string
	if prob != nil {
		problemType = string(prob.Type)
		challenge.Status = core.StatusInvalid
		challenge.Error = prob
		logEvent.Error = prob.Error()
		beeline.AddFieldToTrace(ctx, "challenge.error", prob.Error())
	} else if remoteResults != nil {
		if !features.Enabled(features.EnforceMultiVA) && features.Enabled(features.MultiVAFullResults) {
			// If we're not going to enforce multi VA but we are logging the
			// differentials then collect and log the remote results in a separate go
			// routine to avoid blocking the primary VA.
			go func() {
				_ = va.processRemoteResults(
					req.Domain,
					req.Authz.RegID,
					string(challenge.Type),
					prob,
					remoteResults,
					len(va.remoteVAs))
			}()
			// Since prob was nil and we're not enforcing the results from
			// `processRemoteResults` set the challenge status to valid so the
			// validationTime metrics increment has the correct result label.
			challenge.Status = core.StatusValid
		} else if features.Enabled(features.EnforceMultiVA) {
			remoteProb := va.processRemoteResults(
				req.Domain,
				req.Authz.RegID,
				string(challenge.Type),
				prob,
				remoteResults,
				len(va.remoteVAs))

			// If the remote result was a non-nil problem then fail the validation
			if remoteProb != nil {
				prob = remoteProb
				challenge.Status = core.StatusInvalid
				challenge.Error = remoteProb
				logEvent.Error = remoteProb.Error()
				beeline.AddFieldToTrace(ctx, "challenge.error", remoteProb.Error())
				va.log.Infof("Validation failed due to remote failures: identifier=%v err=%s",
					req.Domain, remoteProb)
				va.metrics.remoteValidationFailures.Inc()
			} else {
				challenge.Status = core.StatusValid
			}
		}
	} else {
		challenge.Status = core.StatusValid
	}

	logEvent.Challenge = challenge
	beeline.AddFieldToTrace(ctx, "challenge.type", challenge.Type)
	beeline.AddFieldToTrace(ctx, "challenge.status", challenge.Status)

	validationLatency := time.Since(vStart)
	logEvent.ValidationLatency = validationLatency.Round(time.Millisecond).Seconds()

	va.metrics.localValidationTime.With(prometheus.Labels{
		"type":   string(challenge.Type),
		"result": string(challenge.Status),
	}).Observe(localValidationLatency.Seconds())
	va.metrics.validationTime.With(prometheus.Labels{
		"type":         string(challenge.Type),
		"result":       string(challenge.Status),
		"problem_type": problemType,
	}).Observe(validationLatency.Seconds())

	va.log.AuditObject("Validation result", logEvent)

	return bgrpc.ValidationResultToPB(records, prob)
}
