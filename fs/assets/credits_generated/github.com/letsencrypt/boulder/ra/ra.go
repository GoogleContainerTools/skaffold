package ra

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/honeycombio/beeline-go"
	"github.com/jmhodges/clock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/weppos/publicsuffix-go/publicsuffix"
	"golang.org/x/crypto/ocsp"
	grpc "google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.in/go-jose/go-jose.v2"

	"github.com/letsencrypt/boulder/akamai"
	akamaipb "github.com/letsencrypt/boulder/akamai/proto"
	capb "github.com/letsencrypt/boulder/ca/proto"
	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	csrlib "github.com/letsencrypt/boulder/csr"
	"github.com/letsencrypt/boulder/ctpolicy"
	berrors "github.com/letsencrypt/boulder/errors"
	"github.com/letsencrypt/boulder/features"
	"github.com/letsencrypt/boulder/goodkey"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	"github.com/letsencrypt/boulder/identifier"
	"github.com/letsencrypt/boulder/issuance"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/policy"
	"github.com/letsencrypt/boulder/probs"
	pubpb "github.com/letsencrypt/boulder/publisher/proto"
	rapb "github.com/letsencrypt/boulder/ra/proto"
	"github.com/letsencrypt/boulder/ratelimit"
	"github.com/letsencrypt/boulder/reloader"
	"github.com/letsencrypt/boulder/revocation"
	sapb "github.com/letsencrypt/boulder/sa/proto"
	vapb "github.com/letsencrypt/boulder/va/proto"
	"github.com/letsencrypt/boulder/web"
)

var (
	errIncompleteGRPCRequest  = errors.New("incomplete gRPC request message")
	errIncompleteGRPCResponse = errors.New("incomplete gRPC response message")

	// caaRecheckDuration is the amount of time after a CAA check that we will
	// recheck the CAA records for a domain. Per Baseline Requirements, we must
	// recheck CAA records within 8 hours of issuance. We set this to 7 hours to
	// stay on the safe side.
	caaRecheckDuration = -7 * time.Hour
)

type caaChecker interface {
	IsCAAValid(
		ctx context.Context,
		in *vapb.IsCAAValidRequest,
		opts ...grpc.CallOption,
	) (*vapb.IsCAAValidResponse, error)
}

// RegistrationAuthorityImpl defines an RA.
//
// NOTE: All of the fields in RegistrationAuthorityImpl need to be
// populated, or there is a risk of panic.
type RegistrationAuthorityImpl struct {
	rapb.UnimplementedRegistrationAuthorityServer
	CA        capb.CertificateAuthorityClient
	OCSP      capb.OCSPGeneratorClient
	VA        vapb.VAClient
	SA        sapb.StorageAuthorityClient
	PA        core.PolicyAuthority
	publisher pubpb.PublisherClient
	caa       caaChecker

	clk       clock.Clock
	log       blog.Logger
	keyPolicy goodkey.KeyPolicy
	// How long before a newly created authorization expires.
	authorizationLifetime        time.Duration
	pendingAuthorizationLifetime time.Duration
	rlPolicies                   ratelimit.Limits
	maxContactsPerReg            int
	maxNames                     int
	reuseValidAuthz              bool
	orderLifetime                time.Duration

	issuersByNameID map[issuance.IssuerNameID]*issuance.Certificate
	issuersByID     map[issuance.IssuerID]*issuance.Certificate
	purger          akamaipb.AkamaiPurgerClient

	ctpolicy *ctpolicy.CTPolicy

	ctpolicyResults             *prometheus.HistogramVec
	rateLimitCounter            *prometheus.CounterVec
	revocationReasonCounter     *prometheus.CounterVec
	namesPerCert                *prometheus.HistogramVec
	newRegCounter               prometheus.Counter
	reusedValidAuthzCounter     prometheus.Counter
	recheckCAACounter           prometheus.Counter
	newCertCounter              prometheus.Counter
	recheckCAAUsedAuthzLifetime prometheus.Counter
	authzAges                   prometheus.Histogram
	orderAges                   prometheus.Histogram
}

// NewRegistrationAuthorityImpl constructs a new RA object.
func NewRegistrationAuthorityImpl(
	clk clock.Clock,
	logger blog.Logger,
	stats prometheus.Registerer,
	maxContactsPerReg int,
	keyPolicy goodkey.KeyPolicy,
	maxNames int,
	reuseValidAuthz bool,
	authorizationLifetime time.Duration,
	pendingAuthorizationLifetime time.Duration,
	pubc pubpb.PublisherClient,
	caaClient caaChecker,
	orderLifetime time.Duration,
	ctp *ctpolicy.CTPolicy,
	purger akamaipb.AkamaiPurgerClient,
	issuers []*issuance.Certificate,
) *RegistrationAuthorityImpl {
	ctpolicyResults := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ctpolicy_results",
			Help:    "Histogram of latencies of ctpolicy.GetSCTs calls with success/failure/deadlineExceeded labels",
			Buckets: metrics.InternetFacingBuckets,
		},
		[]string{"result"},
	)
	stats.MustRegister(ctpolicyResults)

	namesPerCert := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "names_per_cert",
			Help: "Histogram of the number of SANs in requested and issued certificates",
			// The namesPerCert buckets are chosen based on the current Let's Encrypt
			// limit of 100 SANs per certificate.
			Buckets: []float64{1, 5, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
		},
		// Type label value is either "requested" or "issued".
		[]string{"type"},
	)
	stats.MustRegister(namesPerCert)

	rateLimitCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ra_ratelimits",
		Help: "A counter of RA ratelimit checks labelled by type and pass/exceed",
	}, []string{"limit", "result"})
	stats.MustRegister(rateLimitCounter)

	newRegCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "new_registrations",
		Help: "A counter of new registrations",
	})
	stats.MustRegister(newRegCounter)

	reusedValidAuthzCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "reused_valid_authz",
		Help: "A counter of reused valid authorizations",
	})
	stats.MustRegister(reusedValidAuthzCounter)

	recheckCAACounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "recheck_caa",
		Help: "A counter of CAA rechecks",
	})
	stats.MustRegister(recheckCAACounter)

	recheckCAAUsedAuthzLifetime := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "recheck_caa_used_authz_lifetime",
		Help: "A counter times the old codepath was used for CAA recheck time",
	})
	stats.MustRegister(recheckCAAUsedAuthzLifetime)

	newCertCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "new_certificates",
		Help: "A counter of new certificates",
	})
	stats.MustRegister(newCertCounter)

	revocationReasonCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "revocation_reason",
		Help: "A counter of certificate revocation reasons",
	}, []string{"reason"})
	stats.MustRegister(revocationReasonCounter)

	authzAges := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "authz_ages",
		Help: "Histogram of ages of Authorization objects when they're attached to a new Order",
		// authzAges keeps track of how old, in seconds, authorizations were at
		// the time that we attached them to an order. We give it a non-standard
		// bucket distribution so that the leftmost (closest to zero) bucket can be
		// used exclusively for brand-new (i.e. not reused) authzs. Our buckets are:
		// one nanosecond, one second, one minute, one hour, 7 hours (our CAA reuse
		// time), 1 day, 2 days, 7 days, 30 days, +inf (should be empty).
		Buckets: []float64{0.000000001, 1, 60, 3600, 25200, 86400, 172800, 604800, 2592000, 7776000},
	})
	stats.MustRegister(authzAges)

	orderAges := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "order_ages",
		Help: "Histogram of ages of Order objects when they're Finalized",
		// Orders currently have a max age of 7 days (168hrs), so our buckets are:
		// 1 second, 10 seconds, 1 minute, 10 minutes, 1 hour, 10 hours, 1 day,
		// 7 days, +inf.
		Buckets: []float64{1, 10, 60, 600, 3600, 36000, 86400, 172800},
	})
	stats.MustRegister(orderAges)

	issuersByNameID := make(map[issuance.IssuerNameID]*issuance.Certificate)
	issuersByID := make(map[issuance.IssuerID]*issuance.Certificate)
	for _, issuer := range issuers {
		issuersByNameID[issuer.NameID()] = issuer
		issuersByID[issuer.ID()] = issuer
	}

	ra := &RegistrationAuthorityImpl{
		clk:                          clk,
		log:                          logger,
		authorizationLifetime:        authorizationLifetime,
		pendingAuthorizationLifetime: pendingAuthorizationLifetime,
		rlPolicies:                   ratelimit.New(),
		maxContactsPerReg:            maxContactsPerReg,
		keyPolicy:                    keyPolicy,
		maxNames:                     maxNames,
		// TODO(#2734): Remove reuseValidAuthz hardcoding.
		reuseValidAuthz:             true,
		publisher:                   pubc,
		caa:                         caaClient,
		orderLifetime:               orderLifetime,
		ctpolicy:                    ctp,
		ctpolicyResults:             ctpolicyResults,
		purger:                      purger,
		issuersByNameID:             issuersByNameID,
		issuersByID:                 issuersByID,
		namesPerCert:                namesPerCert,
		rateLimitCounter:            rateLimitCounter,
		newRegCounter:               newRegCounter,
		reusedValidAuthzCounter:     reusedValidAuthzCounter,
		recheckCAACounter:           recheckCAACounter,
		newCertCounter:              newCertCounter,
		revocationReasonCounter:     revocationReasonCounter,
		recheckCAAUsedAuthzLifetime: recheckCAAUsedAuthzLifetime,
		authzAges:                   authzAges,
		orderAges:                   orderAges,
	}
	return ra
}

func (ra *RegistrationAuthorityImpl) SetRateLimitPoliciesFile(filename string) error {
	_, err := reloader.New(filename, ra.rlPolicies.LoadPolicies, ra.log)
	if err != nil {
		return err
	}

	return nil
}

// certificateRequestAuthz is a struct for holding information about a valid
// authz referenced during a certificateRequestEvent. It holds both the
// authorization ID and the challenge type that made the authorization valid. We
// specifically include the challenge type that solved the authorization to make
// some common analysis easier.
type certificateRequestAuthz struct {
	ID            string
	ChallengeType core.AcmeChallenge
}

// certificateRequestEvent is a struct for holding information that is logged as
// JSON to the audit log as the result of an issuance event.
type certificateRequestEvent struct {
	ID string `json:",omitempty"`
	// Requester is the associated account ID
	Requester int64 `json:",omitempty"`
	// OrderID is the associated order ID (may be empty for an ACME v1 issuance)
	OrderID int64 `json:",omitempty"`
	// SerialNumber is the string representation of the issued certificate's
	// serial number
	SerialNumber string `json:",omitempty"`
	// VerifiedFields are required by the baseline requirements and are always
	// a static value for Boulder.
	VerifiedFields []string `json:",omitempty"`
	// CommonName is the subject common name from the issued cert
	CommonName string `json:",omitempty"`
	// Names are the DNS SAN entries from the issued cert
	Names []string `json:",omitempty"`
	// NotBefore is the starting timestamp of the issued cert's validity period
	NotBefore time.Time `json:",omitempty"`
	// NotAfter is the ending timestamp of the issued cert's validity period
	NotAfter time.Time `json:",omitempty"`
	// RequestTime and ResponseTime are for tracking elapsed time during issuance
	RequestTime  time.Time `json:",omitempty"`
	ResponseTime time.Time `json:",omitempty"`
	// Error contains any encountered errors
	Error string `json:",omitempty"`
	// Authorizations is a map of identifier names to certificateRequestAuthz
	// objects. It can be used to understand how the names in a certificate
	// request were authorized.
	Authorizations map[string]certificateRequestAuthz
}

// certificateRevocationEvent is a struct for holding information that is logged
// as JSON to the audit log as the result of a revocation event.
type certificateRevocationEvent struct {
	ID string `json:",omitempty"`
	// SerialNumber is the string representation of the revoked certificate's
	// serial number.
	SerialNumber string `json:",omitempty"`
	// Reason is the integer representing the revocation reason used.
	Reason int64 `json:",omitempty"`
	// Method is the way in which revocation was requested.
	// It will be one of the strings: "applicant", "subscriber", "control", "key", or "admin".
	Method string `json:",omitempty"`
	// RequesterID is the account ID of the requester.
	// Will be zero for admin revocations.
	RequesterID int64 `json:",omitempty"`
	// AdminName is the name of the admin requester.
	// Will be zero for subscriber revocations.
	AdminName string `json:",omitempty"`
	// Error contains any error encountered during revocation.
	Error string `json:",omitempty"`
}

// finalizationCAACheckEvent is a struct for holding information logged as JSON
// to the info log as the result of an issuance event. It is logged when the RA
// performs the final CAA check of a certificate finalization request.
type finalizationCAACheckEvent struct {
	// Requester is the associated account ID.
	Requester int64 `json:",omitempty"`
	// Reused is a count of Authz where the original CAA check was performed in
	// the last 7 hours.
	Reused int `json:",omitempty"`
	// Rechecked is a count of Authz where a new CAA check was performed because
	// the original check was older than 7 hours.
	Rechecked int `json:",omitempty"`
}

// noRegistrationID is used for the regID parameter to GetThreshold when no
// registration-based overrides are necessary.
const noRegistrationID = -1

// registrationCounter is a type to abstract the use of `CountRegistrationsByIP`
// or `CountRegistrationsByIPRange` SA methods.
type registrationCounter func(context.Context, *sapb.CountRegistrationsByIPRequest, ...grpc.CallOption) (*sapb.Count, error)

// checkRegistrationIPLimit checks a specific registraton limit by using the
// provided registrationCounter function to determine if the limit has been
// exceeded for a given IP or IP range
func (ra *RegistrationAuthorityImpl) checkRegistrationIPLimit(ctx context.Context, limit ratelimit.RateLimitPolicy, ip net.IP, counter registrationCounter) error {
	if !limit.Enabled() {
		return nil
	}

	now := ra.clk.Now()
	count, err := counter(ctx, &sapb.CountRegistrationsByIPRequest{
		Ip: ip,
		Range: &sapb.Range{
			Earliest: limit.WindowBegin(now).UnixNano(),
			Latest:   now.UnixNano(),
		},
	})
	if err != nil {
		return err
	}

	if count.Count >= limit.GetThreshold(ip.String(), noRegistrationID) {
		return berrors.RegistrationsPerIPError(0, "too many registrations for this IP")
	}

	return nil
}

// checkRegistrationLimits enforces the RegistrationsPerIP and
// RegistrationsPerIPRange limits
func (ra *RegistrationAuthorityImpl) checkRegistrationLimits(ctx context.Context, ip net.IP) error {
	// Check the registrations per IP limit using the CountRegistrationsByIP SA
	// function that matches IP addresses exactly
	exactRegLimit := ra.rlPolicies.RegistrationsPerIP()
	err := ra.checkRegistrationIPLimit(ctx, exactRegLimit, ip, ra.SA.CountRegistrationsByIP)
	if err != nil {
		ra.rateLimitCounter.WithLabelValues("registrations_by_ip", "exceeded").Inc()
		ra.log.Infof("Rate limit exceeded, RegistrationsByIP, IP: %s", ip)
		return err
	}
	ra.rateLimitCounter.WithLabelValues("registrations_by_ip", "pass").Inc()

	// We only apply the fuzzy reg limit to IPv6 addresses.
	// Per https://golang.org/pkg/net/#IP.To4 "If ip is not an IPv4 address, To4
	// returns nil"
	if ip.To4() != nil {
		return nil
	}

	// Check the registrations per IP range limit using the
	// CountRegistrationsByIPRange SA function that fuzzy-matches IPv6 addresses
	// within a larger address range
	fuzzyRegLimit := ra.rlPolicies.RegistrationsPerIPRange()
	err = ra.checkRegistrationIPLimit(ctx, fuzzyRegLimit, ip, ra.SA.CountRegistrationsByIPRange)
	if err != nil {
		ra.rateLimitCounter.WithLabelValues("registrations_by_ip_range", "exceeded").Inc()
		ra.log.Infof("Rate limit exceeded, RegistrationsByIPRange, IP: %s", ip)
		// For the fuzzyRegLimit we use a new error message that specifically
		// mentions that the limit being exceeded is applied to a *range* of IPs
		return berrors.RateLimitError(0, "too many registrations for this IP range")
	}
	ra.rateLimitCounter.WithLabelValues("registrations_by_ip_range", "pass").Inc()

	return nil
}

// NewRegistration constructs a new Registration from a request.
func (ra *RegistrationAuthorityImpl) NewRegistration(ctx context.Context, request *corepb.Registration) (*corepb.Registration, error) {
	// Error if the request is nil, there is no account key or IP address
	if request == nil || len(request.Key) == 0 || len(request.InitialIP) == 0 {
		return nil, errIncompleteGRPCRequest
	}

	// Check if account key is acceptable for use.
	var key jose.JSONWebKey
	err := key.UnmarshalJSON(request.Key)
	if err != nil {
		return nil, berrors.InternalServerError("failed to unmarshal account key: %s", err.Error())
	}
	err = ra.keyPolicy.GoodKey(ctx, key.Key)
	if err != nil {
		return nil, berrors.MalformedError("invalid public key: %s", err.Error())
	}

	// Check IP address rate limits.
	var ipAddr net.IP
	err = ipAddr.UnmarshalText(request.InitialIP)
	if err != nil {
		return nil, berrors.InternalServerError("failed to unmarshal ip address: %s", err.Error())
	}
	err = ra.checkRegistrationLimits(ctx, ipAddr)
	if err != nil {
		return nil, err
	}

	// Check that contacts conform to our expectations.
	err = validateContactsPresent(request.Contact, request.ContactsPresent)
	if err != nil {
		return nil, err
	}
	err = ra.validateContacts(request.Contact)
	if err != nil {
		return nil, err
	}

	// Don't populate ID or CreatedAt because those will be set by the SA.
	req := &corepb.Registration{
		Key:             request.Key,
		Contact:         request.Contact,
		ContactsPresent: request.ContactsPresent,
		Agreement:       request.Agreement,
		InitialIP:       request.InitialIP,
		Status:          string(core.StatusValid),
	}

	// Store the registration object, then return the version that got stored.
	res, err := ra.SA.NewRegistration(ctx, req)
	if err != nil {
		return nil, err
	}

	ra.newRegCounter.Inc()
	return res, nil
}

// validateContacts checks the provided list of contacts, returning an error if
// any are not acceptable. Unacceptable contacts lists include:
// * An empty list
// * A list has more than maxContactsPerReg contacts
// * A list containing an empty contact
// * A list containing a contact that does not parse as a URL
// * A list containing a contact that has a URL scheme other than mailto
// * A list containing a mailto contact that contains hfields
// * A list containing a contact that has non-ascii characters
// * A list containing a contact that doesn't pass `policy.ValidEmail`
func (ra *RegistrationAuthorityImpl) validateContacts(contacts []string) error {
	if len(contacts) == 0 {
		return nil // Nothing to validate
	}
	if ra.maxContactsPerReg > 0 && len(contacts) > ra.maxContactsPerReg {
		return berrors.MalformedError(
			"too many contacts provided: %d > %d",
			len(contacts),
			ra.maxContactsPerReg,
		)
	}

	for _, contact := range contacts {
		if contact == "" {
			return berrors.InvalidEmailError("empty contact")
		}
		parsed, err := url.Parse(contact)
		if err != nil {
			return berrors.InvalidEmailError("invalid contact")
		}
		if parsed.Scheme != "mailto" {
			return berrors.InvalidEmailError("contact method %q is not supported", parsed.Scheme)
		}
		if parsed.RawQuery != "" || contact[len(contact)-1] == '?' {
			return berrors.InvalidEmailError("contact email %q contains a question mark", contact)
		}
		if parsed.Fragment != "" || contact[len(contact)-1] == '#' {
			return berrors.InvalidEmailError("contact email %q contains a '#'", contact)
		}
		if !core.IsASCII(contact) {
			return berrors.InvalidEmailError(
				"contact email [%q] contains non-ASCII characters",
				contact,
			)
		}
		err = policy.ValidEmail(parsed.Opaque)
		if err != nil {
			return err
		}
	}

	// NOTE(@cpu): For historical reasons (</3) we store ACME account contact
	// information de-normalized in a fixed size `contact` field on the
	// `registrations` table. At the time of writing this field is VARCHAR(191)
	// That means the largest marshalled JSON value we can store is 191 bytes.
	const maxContactBytes = 191
	if jsonBytes, err := json.Marshal(contacts); err != nil {
		// This shouldn't happen with a simple []string but if it does we want the
		// error to be logged internally but served as a 500 to the user so we
		// return a bare error and not a berror here.
		return fmt.Errorf("failed to marshal reg.Contact to JSON: %#v", contacts)
	} else if len(jsonBytes) >= maxContactBytes {
		return berrors.InvalidEmailError(
			"too many/too long contact(s). Please use shorter or fewer email addresses")
	}

	return nil
}

func (ra *RegistrationAuthorityImpl) checkPendingAuthorizationLimit(ctx context.Context, regID int64) error {
	limit := ra.rlPolicies.PendingAuthorizationsPerAccount()
	if limit.Enabled() {
		// This rate limit's threshold can only be overridden on a per-regID basis,
		// not based on any other key.
		threshold := limit.GetThreshold("", regID)
		if threshold == -1 {
			return nil
		}
		countPB, err := ra.SA.CountPendingAuthorizations2(ctx, &sapb.RegistrationID{
			Id: regID,
		})
		if err != nil {
			return err
		}
		if countPB.Count >= threshold {
			ra.rateLimitCounter.WithLabelValues("pending_authorizations_by_registration_id", "exceeded").Inc()
			ra.log.Infof("Rate limit exceeded, PendingAuthorizationsByRegID, regID: %d", regID)
			return berrors.RateLimitError(0, "too many currently pending authorizations: %d", countPB.Count)
		}
		ra.rateLimitCounter.WithLabelValues("pending_authorizations_by_registration_id", "pass").Inc()
	}
	return nil
}

// checkInvalidAuthorizationLimits checks the failed validation limit for each
// of the provided hostnames. It returns the first error.
func (ra *RegistrationAuthorityImpl) checkInvalidAuthorizationLimits(ctx context.Context, regID int64, hostnames []string) error {
	results := make(chan error, len(hostnames))
	for _, hostname := range hostnames {
		go func(hostname string) {
			results <- ra.checkInvalidAuthorizationLimit(ctx, regID, hostname)
		}(hostname)
	}
	// We don't have to wait for all of the goroutines to finish because there's
	// enough capacity in the chan for them all to write their result even if
	// nothing is reading off the chan anymore.
	for i := 0; i < len(hostnames); i++ {
		err := <-results
		if err != nil {
			return err
		}
	}
	return nil
}

func (ra *RegistrationAuthorityImpl) checkInvalidAuthorizationLimit(ctx context.Context, regID int64, hostname string) error {
	limit := ra.rlPolicies.InvalidAuthorizationsPerAccount()
	if !limit.Enabled() {
		return nil
	}
	latest := ra.clk.Now().Add(ra.pendingAuthorizationLifetime)
	earliest := latest.Add(-limit.Window.Duration)
	req := &sapb.CountInvalidAuthorizationsRequest{
		RegistrationID: regID,
		Hostname:       hostname,
		Range: &sapb.Range{
			Earliest: earliest.UnixNano(),
			Latest:   latest.UnixNano(),
		},
	}
	count, err := ra.SA.CountInvalidAuthorizations2(ctx, req)
	if err != nil {
		return err
	}
	// Most rate limits have a key for overrides, but there is no meaningful key
	// here.
	noKey := ""
	if count.Count >= limit.GetThreshold(noKey, regID) {
		ra.log.Infof("Rate limit exceeded, InvalidAuthorizationsByRegID, regID: %d", regID)
		return berrors.FailedValidationError(0, "too many failed authorizations recently")
	}
	return nil
}

// checkNewOrdersPerAccountLimit enforces the rlPolicies `NewOrdersPerAccount`
// rate limit. This rate limit ensures a client can not create more than the
// specified threshold of new orders within the specified time window.
func (ra *RegistrationAuthorityImpl) checkNewOrdersPerAccountLimit(ctx context.Context, acctID int64) error {
	limit := ra.rlPolicies.NewOrdersPerAccount()
	if !limit.Enabled() {
		return nil
	}
	now := ra.clk.Now()
	count, err := ra.SA.CountOrders(ctx, &sapb.CountOrdersRequest{
		AccountID: acctID,
		Range: &sapb.Range{
			Earliest: now.Add(-limit.Window.Duration).UnixNano(),
			Latest:   now.UnixNano(),
		},
	})
	if err != nil {
		return err
	}
	// There is no meaningful override key to use for this rate limit
	noKey := ""
	if count.Count >= limit.GetThreshold(noKey, acctID) {
		ra.rateLimitCounter.WithLabelValues("new_order_by_registration_id", "exceeded").Inc()
		return berrors.RateLimitError(0, "too many new orders recently")
	}
	ra.rateLimitCounter.WithLabelValues("new_order_by_registration_id", "pass").Inc()
	return nil
}

// matchesCSR tests the contents of a generated certificate to make sure
// that the PublicKey, CommonName, and DNSNames match those provided in
// the CSR that was used to generate the certificate. It also checks the
// following fields for:
//   - notBefore is not more than 24 hours ago
//   - BasicConstraintsValid is true
//   - IsCA is false
//   - ExtKeyUsage only contains ExtKeyUsageServerAuth & ExtKeyUsageClientAuth
//   - Subject only contains CommonName & Names
func (ra *RegistrationAuthorityImpl) matchesCSR(parsedCertificate *x509.Certificate, csr *x509.CertificateRequest) error {
	// Check issued certificate matches what was expected from the CSR
	hostNames := make([]string, len(csr.DNSNames))
	copy(hostNames, csr.DNSNames)
	if len(csr.Subject.CommonName) > 0 {
		hostNames = append(hostNames, csr.Subject.CommonName)
	}
	hostNames = core.UniqueLowerNames(hostNames)

	if !core.KeyDigestEquals(parsedCertificate.PublicKey, csr.PublicKey) {
		return berrors.InternalServerError("generated certificate public key doesn't match CSR public key")
	}
	if parsedCertificate.Subject.CommonName != strings.ToLower(csr.Subject.CommonName) {
		return berrors.InternalServerError("generated certificate CommonName doesn't match CSR CommonName")
	}
	// Sort both slices of names before comparison.
	parsedNames := parsedCertificate.DNSNames
	sort.Strings(parsedNames)
	sort.Strings(hostNames)
	if !reflect.DeepEqual(parsedNames, hostNames) {
		return berrors.InternalServerError("generated certificate DNSNames don't match CSR DNSNames")
	}
	if !reflect.DeepEqual(parsedCertificate.IPAddresses, csr.IPAddresses) {
		return berrors.InternalServerError("generated certificate IPAddresses don't match CSR IPAddresses")
	}
	if !reflect.DeepEqual(parsedCertificate.EmailAddresses, csr.EmailAddresses) {
		return berrors.InternalServerError("generated certificate EmailAddresses don't match CSR EmailAddresses")
	}
	if len(parsedCertificate.Subject.Country) > 0 || len(parsedCertificate.Subject.Organization) > 0 ||
		len(parsedCertificate.Subject.OrganizationalUnit) > 0 || len(parsedCertificate.Subject.Locality) > 0 ||
		len(parsedCertificate.Subject.Province) > 0 || len(parsedCertificate.Subject.StreetAddress) > 0 ||
		len(parsedCertificate.Subject.PostalCode) > 0 {
		return berrors.InternalServerError("generated certificate Subject contains fields other than CommonName, or SerialNumber")
	}
	now := ra.clk.Now()
	if now.Sub(parsedCertificate.NotBefore) > time.Hour*24 {
		return berrors.InternalServerError("generated certificate is back dated %s", now.Sub(parsedCertificate.NotBefore))
	}
	if !parsedCertificate.BasicConstraintsValid {
		return berrors.InternalServerError("generated certificate doesn't have basic constraints set")
	}
	if parsedCertificate.IsCA {
		return berrors.InternalServerError("generated certificate can sign other certificates")
	}
	if !reflect.DeepEqual(parsedCertificate.ExtKeyUsage, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}) {
		return berrors.InternalServerError("generated certificate doesn't have correct key usage extensions")
	}

	return nil
}

// checkOrderAuthorizations verifies that a provided set of names associated
// with a specific order and account has all of the required valid, unexpired
// authorizations to proceed with issuance. It returns the authorizations that
// satisfied the set of names or it returns an error. If it returns an error, it
// will be of type BoulderError.
func (ra *RegistrationAuthorityImpl) checkOrderAuthorizations(
	ctx context.Context,
	names []string,
	acctID accountID,
	orderID orderID) (map[string]*core.Authorization, error) {
	// Get all of the valid authorizations for this account/order
	req := &sapb.GetValidOrderAuthorizationsRequest{
		Id:     int64(orderID),
		AcctID: int64(acctID),
	}
	authzMapPB, err := ra.SA.GetValidOrderAuthorizations2(ctx, req)
	if err != nil {
		return nil, berrors.InternalServerError("error in GetValidOrderAuthorizations: %s", err)
	}
	authzs, err := bgrpc.PBToAuthzMap(authzMapPB)
	if err != nil {
		return nil, err
	}

	// Ensure the names from the CSR are free of duplicates & lowercased.
	names = core.UniqueLowerNames(names)

	// Check the authorizations to ensure validity for the names required.
	err = ra.checkAuthorizationsCAA(ctx, int64(acctID), names, authzs, ra.clk.Now())
	if err != nil {
		return nil, err
	}

	// Check the challenges themselves too.
	for _, authz := range authzs {
		err = ra.PA.CheckAuthz(authz)
		if err != nil {
			return nil, err
		}
	}

	return authzs, nil
}

// validatedBefore checks if a given authorization's challenge was
// validated before a given time. Returns a bool.
func validatedBefore(authz *core.Authorization, caaRecheckTime time.Time) (bool, error) {
	numChallenges := len(authz.Challenges)
	if numChallenges != 1 {
		return false, fmt.Errorf("authorization has incorrect number of challenges. 1 expected, %d found for: id %s", numChallenges, authz.ID)
	}
	if authz.Challenges[0].Validated == nil {
		return false, fmt.Errorf("authorization's challenge has no validated timestamp for: id %s", authz.ID)
	}
	return authz.Challenges[0].Validated.Before(caaRecheckTime), nil
}

// checkAuthorizationsCAA implements the common logic of validating a set of
// authorizations against a set of names that is used by both
// `checkAuthorizations` and `checkOrderAuthorizations`. If required CAA will be
// rechecked for authorizations that are too old.
// If it returns an error, it will be of type BoulderError.
func (ra *RegistrationAuthorityImpl) checkAuthorizationsCAA(
	ctx context.Context,
	acctID int64,
	names []string,
	authzs map[string]*core.Authorization,
	now time.Time) error {
	// badNames contains the names that were unauthorized
	var badNames []string
	// recheckAuthzs is a list of authorizations that must have their CAA records rechecked
	var recheckAuthzs []*core.Authorization

	// Per Baseline Requirements, CAA must be checked within 8 hours of
	// issuance. CAA is checked when an authorization is validated, so as
	// long as that was less than 8 hours ago, we're fine. We recheck if
	// that was more than 7 hours ago, to be on the safe side. We can
	// check to see if the authorized challenge `AttemptedAt`
	// (`Validated`) value from the database is before our caaRecheckTime.
	// Set the recheck time to 7 hours ago.
	caaRecheckAfter := now.Add(caaRecheckDuration)

	// Set a CAA recheck time based on the assumption of a 30 day authz
	// lifetime. This has been deprecated in favor of a new check based
	// off the Validated time stored in the database, but we want to check
	// both for a time and increment a stat if this code path is hit for
	// compliance safety.
	caaRecheckTime := now.Add(ra.authorizationLifetime).Add(caaRecheckDuration)

	for _, name := range names {
		authz := authzs[name]
		if authz == nil {
			badNames = append(badNames, name)
		} else if authz.Expires == nil {
			return berrors.InternalServerError("found an authorization with a nil Expires field: id %s", authz.ID)
		} else if authz.Expires.Before(now) {
			badNames = append(badNames, name)
		} else if staleCAA, err := validatedBefore(authz, caaRecheckAfter); err != nil {
			return berrors.InternalServerError(err.Error())
		} else if staleCAA {
			// Ensure that CAA is rechecked for this name
			recheckAuthzs = append(recheckAuthzs, authz)
		} else if authz.Expires.Before(caaRecheckTime) {
			// Ensure that CAA is rechecked for this name
			recheckAuthzs = append(recheckAuthzs, authz)
			// This codepath should not be used, but is here as a safety
			// net until the new codepath is proven. Increment metric if
			// it is used.
			ra.recheckCAAUsedAuthzLifetime.Add(1)
		}
	}

	if len(recheckAuthzs) > 0 {
		err := ra.recheckCAA(ctx, recheckAuthzs)
		if err != nil {
			return err
		}
	}

	if len(badNames) > 0 {
		return berrors.UnauthorizedError(
			"authorizations for these names not found or expired: %s",
			strings.Join(badNames, ", "),
		)
	}

	caaEvent := &finalizationCAACheckEvent{
		Requester: acctID,
		Reused:    len(authzs) - len(recheckAuthzs),
		Rechecked: len(recheckAuthzs),
	}
	ra.log.InfoObject("FinalizationCaaCheck", caaEvent)

	return nil
}

// recheckCAA accepts a list of of names that need to have their CAA records
// rechecked because their associated authorizations are sufficiently old and
// performs the CAA checks required for each. If any of the rechecks fail an
// error is returned.
func (ra *RegistrationAuthorityImpl) recheckCAA(ctx context.Context, authzs []*core.Authorization) error {
	ra.recheckCAACounter.Add(float64(len(authzs)))

	type authzCAAResult struct {
		authz *core.Authorization
		err   error
	}
	ch := make(chan authzCAAResult, len(authzs))
	for _, authz := range authzs {
		go func(authz *core.Authorization) {
			name := authz.Identifier.Value

			// If an authorization has multiple valid challenges,
			// the type of the first valid challenge is used for
			// the purposes of CAA rechecking.
			var method string
			for _, challenge := range authz.Challenges {
				if challenge.Status == core.StatusValid {
					method = string(challenge.Type)
					break
				}
			}
			if method == "" {
				ch <- authzCAAResult{
					authz: authz,
					err: berrors.InternalServerError(
						"Internal error determining validation method for authorization ID %v (%v)",
						authz.ID, name),
				}
				return
			}

			resp, err := ra.caa.IsCAAValid(ctx, &vapb.IsCAAValidRequest{
				Domain:           name,
				ValidationMethod: method,
				AccountURIID:     authz.RegistrationID,
			})
			if err != nil {
				ra.log.AuditErrf("Rechecking CAA: %s", err)
				err = berrors.InternalServerError(
					"Internal error rechecking CAA for authorization ID %v (%v)",
					authz.ID, name,
				)
			} else if resp.Problem != nil {
				err = berrors.CAAError(resp.Problem.Detail)
			}
			ch <- authzCAAResult{
				authz: authz,
				err:   err,
			}
		}(authz)
	}
	var subErrors []berrors.SubBoulderError
	// Read a recheckResult for each authz from the results channel
	for i := 0; i < len(authzs); i++ {
		recheckResult := <-ch
		// If the result had a CAA boulder error, construct a suberror with the
		// identifier from the authorization that was checked.
		err := recheckResult.err
		if err != nil {
			var bErr *berrors.BoulderError
			if errors.As(err, &bErr) && bErr.Type == berrors.CAA {
				subErrors = append(subErrors, berrors.SubBoulderError{
					Identifier:   recheckResult.authz.Identifier,
					BoulderError: bErr})
			} else {
				return err
			}
		}
	}
	if len(subErrors) > 0 {
		var detail string
		// If there was only one error, then use it as the top level error that is
		// returned.
		if len(subErrors) == 1 {
			return subErrors[0].BoulderError
		}
		detail = fmt.Sprintf(
			"Rechecking CAA for %q and %d more identifiers failed. "+
				"Refer to sub-problems for more information",
			subErrors[0].Identifier.Value,
			len(subErrors)-1)
		return (&berrors.BoulderError{
			Type:   berrors.CAA,
			Detail: detail,
		}).WithSubErrors(subErrors)
	}
	return nil
}

// failOrder marks an order as failed by setting the problem details field of
// the order & persisting it through the SA. If an error occurs doing this we
// log it and don't modify the input order. There aren't any alternatives if we
// can't add the error to the order. This function MUST only be called when we
// are already returning an error for another reason.
func (ra *RegistrationAuthorityImpl) failOrder(
	ctx context.Context,
	order *corepb.Order,
	prob *probs.ProblemDetails) {

	// Convert the problem to a protobuf problem for the *corepb.Order field
	pbProb, err := bgrpc.ProblemDetailsToPB(prob)
	if err != nil {
		ra.log.AuditErrf("Could not convert order error problem to PB: %q", err)
		return
	}

	// Assign the protobuf problem to the field and save it via the SA
	order.Error = pbProb
	_, err = ra.SA.SetOrderError(ctx, &sapb.SetOrderErrorRequest{
		Id:    order.Id,
		Error: order.Error,
	})
	if err != nil {
		ra.log.AuditErrf("Could not persist order error: %q", err)
	}
}

// To help minimize the chance that an accountID would be used as an order ID
// (or vice versa) when calling functions that use both we define internal
// `accountID` and `orderID` types so that callers must explicitly cast.
type accountID int64
type orderID int64

// FinalizeOrder accepts a request to finalize an order object and, if possible,
// issues a certificate to satisfy the order. If an order does not have valid,
// unexpired authorizations for all of its associated names an error is
// returned. Similarly we vet that all of the names in the order are acceptable
// based on current policy and return an error if the order can't be fulfilled.
// If successful the order will be returned in processing status for the client
// to poll while awaiting finalization to occur.
func (ra *RegistrationAuthorityImpl) FinalizeOrder(ctx context.Context, req *rapb.FinalizeOrderRequest) (*corepb.Order, error) {
	// Step 1: Set up logging/tracing and validate the Order
	if req == nil || req.Order == nil || len(req.Csr) == 0 {
		return nil, errIncompleteGRPCRequest
	}

	logEvent := certificateRequestEvent{
		ID:          core.NewToken(),
		OrderID:     req.Order.Id,
		Requester:   req.Order.RegistrationID,
		RequestTime: ra.clk.Now(),
	}
	beeline.AddFieldToTrace(ctx, "issuance.id", logEvent.ID)
	beeline.AddFieldToTrace(ctx, "order.id", req.Order.Id)
	beeline.AddFieldToTrace(ctx, "acct.id", req.Order.RegistrationID)

	csr, err := ra.validateFinalizeRequest(ctx, req, &logEvent)
	if err != nil {
		return nil, err
	}

	// Observe the age of this order, so we know how quickly most clients complete
	// issuance flows.
	ra.orderAges.Observe(ra.clk.Since(time.Unix(0, req.Order.Created)).Seconds())

	// Step 2: Set the Order to Processing status
	//
	// We do this separately from the issuance process itself so that, when we
	// switch to doing issuance asynchronously, we aren't lying to the client
	// when we say that their order is already Processing.
	//
	// NOTE(@cpu): After this point any errors that are encountered must update
	// the state of the order to invalid by setting the order's error field.
	// Otherwise the order will be "stuck" in processing state. It can not be
	// finalized because it isn't pending, but we aren't going to process it
	// further because we already did and encountered an error.
	_, err = ra.SA.SetOrderProcessing(ctx, &sapb.OrderRequest{Id: req.Order.Id})
	if err != nil {
		// Fail the order with a server internal error - we weren't able to set the
		// status to processing and that's unexpected & weird.
		ra.failOrder(ctx, req.Order, probs.ServerInternal("Error setting order processing"))
		return nil, err
	}

	// Step 3: Issue the Certificate
	cert, err := ra.issueCertificateInner(
		ctx, csr, accountID(req.Order.RegistrationID), orderID(req.Order.Id))

	// Step 4: Fail the order if necessary, and update metrics and log fields
	var result string
	order := req.Order
	if err != nil {
		// The problem is computed using `web.ProblemDetailsForError`, the same
		// function the WFE uses to convert between `berrors` and problems. This
		// will turn normal expected berrors like berrors.UnauthorizedError into the
		// correct `urn:ietf:params:acme:error:unauthorized` problem while not
		// letting anything like a server internal error through with sensitive
		// info.
		ra.failOrder(ctx, req.Order, web.ProblemDetailsForError(err, "Error finalizing order"))

		// Update the order status locally since the SA doesn't return the updated
		// order itself after setting the status
		order.Status = string(core.StatusInvalid)

		logEvent.Error = err.Error()
		beeline.AddFieldToTrace(ctx, "issuance.error", err)
		result = "error"
	} else {
		// Update the order status locally since the SA doesn't return the updated
		// order itself after setting the status
		order.CertificateSerial = core.SerialToString(cert.SerialNumber)
		order.Status = string(core.StatusValid)

		ra.namesPerCert.With(
			prometheus.Labels{"type": "issued"},
		).Observe(float64(len(order.Names)))

		ra.newCertCounter.Inc()

		logEvent.SerialNumber = core.SerialToString(cert.SerialNumber)
		beeline.AddFieldToTrace(ctx, "cert.serial", core.SerialToString(cert.SerialNumber))
		logEvent.CommonName = cert.Subject.CommonName
		beeline.AddFieldToTrace(ctx, "cert.common_name", cert.Subject.CommonName)
		logEvent.Names = cert.DNSNames
		beeline.AddFieldToTrace(ctx, "cert.dns_names", cert.DNSNames)
		logEvent.NotBefore = cert.NotBefore
		beeline.AddFieldToTrace(ctx, "cert.not_before", cert.NotBefore)
		logEvent.NotAfter = cert.NotAfter
		beeline.AddFieldToTrace(ctx, "cert.not_after", cert.NotAfter)

		result = "successful"
	}

	logEvent.ResponseTime = ra.clk.Now()
	ra.log.AuditObject(fmt.Sprintf("Certificate request - %s", result), logEvent)

	// Return both the order and the error: if issueCertificateInner worked, then
	// err will be nil; if it didn't, then we'll propagate that error upwards.
	return order, err
}

// validateFinalizeRequest checks that a FinalizeOrder request is fully correct
// and ready for issuance.
func (ra *RegistrationAuthorityImpl) validateFinalizeRequest(
	ctx context.Context,
	req *rapb.FinalizeOrderRequest,
	logEvent *certificateRequestEvent) (*x509.CertificateRequest, error) {
	if req.Order.Id <= 0 {
		return nil, berrors.MalformedError("invalid order ID: %d", req.Order.Id)
	}

	if req.Order.RegistrationID <= 0 {
		return nil, berrors.MalformedError("invalid account ID: %d", req.Order.RegistrationID)
	}

	if core.AcmeStatus(req.Order.Status) != core.StatusReady {
		return nil, berrors.OrderNotReadyError(
			"Order's status (%q) is not acceptable for finalization",
			req.Order.Status)
	}

	// There should never be an order with 0 names at the stage, but we check to
	// be on the safe side, throwing an internal server error if this assumption
	// is ever violated.
	if len(req.Order.Names) == 0 {
		return nil, berrors.InternalServerError("Order has no associated names")
	}

	// Parse the CSR from the request
	csr, err := x509.ParseCertificateRequest(req.Csr)
	if err != nil {
		return nil, berrors.BadCSRError("unable to parse CSR: %s", err.Error())
	}

	err = csrlib.VerifyCSR(ctx, csr, ra.maxNames, &ra.keyPolicy, ra.PA)
	if err != nil {
		// VerifyCSR returns berror instances that can be passed through as-is
		// without wrapping.
		return nil, err
	}

	beeline.AddFieldToTrace(ctx, "csr.cn", csr.Subject.CommonName)
	beeline.AddFieldToTrace(ctx, "csr.dnsnames", csr.DNSNames)

	// Dedupe, lowercase and sort both the names from the CSR and the names in the
	// order.
	csrNames := core.UniqueLowerNames(csr.DNSNames)
	orderNames := core.UniqueLowerNames(req.Order.Names)

	// Immediately reject the request if the number of names differ
	if len(orderNames) != len(csrNames) {
		return nil, berrors.UnauthorizedError("Order includes different number of names than CSR specifies")
	}

	// Check that the order names and the CSR names are an exact match
	for i, name := range orderNames {
		if name != csrNames[i] {
			return nil, berrors.UnauthorizedError("CSR is missing Order domain %q", name)
		}
	}

	// Get the originating account for use in the next check.
	regPB, err := ra.SA.GetRegistration(ctx, &sapb.RegistrationID{Id: req.Order.RegistrationID})
	if err != nil {
		return nil, err
	}

	account, err := bgrpc.PbToRegistration(regPB)
	if err != nil {
		return nil, err
	}

	// Make sure they're not using their account key as the certificate key too.
	if core.KeyDigestEquals(csr.PublicKey, account.Key) {
		return nil, berrors.MalformedError("certificate public key must be different than account key")
	}

	// Double-check that all authorizations on this order are also associated with
	// the same account as the order itself.
	authzs, err := ra.checkOrderAuthorizations(ctx, csrNames, accountID(req.Order.RegistrationID), orderID(req.Order.Id))
	if err != nil {
		// Pass through the error without wrapping it because the called functions
		// return BoulderError and we don't want to lose the type.
		return nil, err
	}

	// Collect up a certificateRequestAuthz that stores the ID and challenge type
	// of each of the valid authorizations we used for this issuance.
	logEventAuthzs := make(map[string]certificateRequestAuthz, len(csrNames))
	for name, authz := range authzs {
		// No need to check for error here because we know this same call just
		// succeeded inside ra.checkOrderAuthorizations
		solvedByChallengeType, _ := authz.SolvedBy()
		logEventAuthzs[name] = certificateRequestAuthz{
			ID:            authz.ID,
			ChallengeType: solvedByChallengeType,
		}
	}
	logEvent.Authorizations = logEventAuthzs

	// Mark that we verified the CN and SANs
	logEvent.VerifiedFields = []string{"subject.commonName", "subjectAltName"}

	return csr, nil
}

// issueCertificateInner handles the heavy lifting aspects of certificate
// issuance.
//
// This function is responsible for ensuring that we never try to issue a final
// certificate twice for the same precertificate, because that has the potential
// to create certificates with duplicate serials. For instance, this could
// happen if final certificates were created with different sets of SCTs. This
// function accomplishes that by bailing on issuance if there is any error in
// IssueCertificateForPrecertificate; there are no retries, and serials are
// generated in IssuePrecertificate, so serials with errors are dropped and
// never have final certificates issued for them (because there is a possibility
// that the certificate was actually issued but there was an error returning
// it).
func (ra *RegistrationAuthorityImpl) issueCertificateInner(
	ctx context.Context,
	csr *x509.CertificateRequest,
	acctID accountID,
	oID orderID) (*x509.Certificate, error) {
	// wrapError adds a prefix to an error. If the error is a boulder error then
	// the problem detail is updated with the prefix. Otherwise a new error is
	// returned with the message prefixed using `fmt.Errorf`
	wrapError := func(e error, prefix string) error {
		if berr, ok := e.(*berrors.BoulderError); ok {
			berr.Detail = fmt.Sprintf("%s: %s", prefix, berr.Detail)
			return berr
		}
		return fmt.Errorf("%s: %s", prefix, e)
	}

	issueReq := &capb.IssueCertificateRequest{
		Csr:            csr.Raw,
		RegistrationID: int64(acctID),
		OrderID:        int64(oID),
	}
	precert, err := ra.CA.IssuePrecertificate(ctx, issueReq)
	if err != nil {
		return nil, wrapError(err, "issuing precertificate")
	}

	parsedPrecert, err := x509.ParseCertificate(precert.DER)
	if err != nil {
		return nil, wrapError(err, "parsing precertificate")
	}

	scts, err := ra.getSCTs(ctx, precert.DER, parsedPrecert.NotAfter)
	if err != nil {
		return nil, wrapError(err, "getting SCTs")
	}

	cert, err := ra.CA.IssueCertificateForPrecertificate(ctx, &capb.IssueCertificateForPrecertificateRequest{
		DER:            precert.DER,
		SCTs:           scts,
		RegistrationID: int64(acctID),
		OrderID:        int64(oID),
	})
	if err != nil {
		return nil, wrapError(err, "issuing certificate for precertificate")
	}

	parsedCertificate, err := x509.ParseCertificate(cert.Der)
	if err != nil {
		return nil, wrapError(err, "parsing final certificate")
	}

	// Asynchronously submit the final certificate to any configured logs
	go ra.ctpolicy.SubmitFinalCert(cert.Der, parsedCertificate.NotAfter)

	// TODO(#6587): Make this error case Very Alarming
	err = ra.matchesCSR(parsedCertificate, csr)
	if err != nil {
		return nil, err
	}

	_, err = ra.SA.FinalizeOrder(ctx, &sapb.FinalizeOrderRequest{
		Id:                int64(oID),
		CertificateSerial: core.SerialToString(parsedCertificate.SerialNumber),
	})
	if err != nil {
		return nil, wrapError(err, "persisting finalized order")
	}

	return parsedCertificate, nil
}

func (ra *RegistrationAuthorityImpl) getSCTs(ctx context.Context, cert []byte, expiration time.Time) (core.SCTDERs, error) {
	started := ra.clk.Now()
	scts, err := ra.ctpolicy.GetSCTs(ctx, cert, expiration)
	took := ra.clk.Since(started)
	// The final cert has already been issued so actually return it to the
	// user even if this fails since we aren't actually doing anything with
	// the SCTs yet.
	if err != nil {
		state := "failure"
		if err == context.DeadlineExceeded {
			state = "deadlineExceeded"
			// Convert the error to a missingSCTsError to communicate the timeout,
			// otherwise it will be a generic serverInternalError
			err = berrors.MissingSCTsError(err.Error())
		}
		ra.log.Warningf("ctpolicy.GetSCTs failed: %s", err)
		ra.ctpolicyResults.With(prometheus.Labels{"result": state}).Observe(took.Seconds())
		return nil, err
	}
	ra.ctpolicyResults.With(prometheus.Labels{"result": "success"}).Observe(took.Seconds())
	return scts, nil
}

// domainsForRateLimiting transforms a list of FQDNs into a list of eTLD+1's
// for the purpose of rate limiting. It also de-duplicates the output
// domains. Exact public suffix matches are included.
func domainsForRateLimiting(names []string) []string {
	var domains []string
	for _, name := range names {
		domain, err := publicsuffix.Domain(name)
		if err != nil {
			// The only possible errors are:
			// (1) publicsuffix.Domain is giving garbage values
			// (2) the public suffix is the domain itself
			// We assume 2 and include the original name in the result.
			domains = append(domains, name)
		} else {
			domains = append(domains, domain)
		}
	}
	return core.UniqueLowerNames(domains)
}

// enforceNameCounts uses the provided count RPC to find a count of certificates
// for each of the names. If the count for any of the names exceeds the limit
// for the given registration then the names out of policy are returned to be
// used for a rate limit error.
func (ra *RegistrationAuthorityImpl) enforceNameCounts(ctx context.Context, names []string, limit ratelimit.RateLimitPolicy, regID int64) ([]string, time.Time, error) {
	now := ra.clk.Now()
	req := &sapb.CountCertificatesByNamesRequest{
		Names: names,
		Range: &sapb.Range{
			Earliest: limit.WindowBegin(now).UnixNano(),
			Latest:   now.UnixNano(),
		},
	}

	response, err := ra.SA.CountCertificatesByNames(ctx, req)
	if err != nil {
		return nil, time.Time{}, err
	}

	if len(response.Counts) == 0 {
		return nil, time.Time{}, errIncompleteGRPCResponse
	}

	var badNames []string
	// Find the names that have counts at or over the threshold. Range
	// over the names slice input to ensure the order of badNames will
	// return the badNames in the same order they were input.
	for _, name := range names {
		if response.Counts[name] >= limit.GetThreshold(name, regID) {
			badNames = append(badNames, name)
		}
	}
	return badNames, response.Earliest.AsTime(), nil
}

func (ra *RegistrationAuthorityImpl) checkCertificatesPerNameLimit(ctx context.Context, names []string, limit ratelimit.RateLimitPolicy, regID int64) error {
	// check if there is already an existing certificate for
	// the exact name set we are issuing for. If so bypass the
	// the certificatesPerName limit.
	exists, err := ra.SA.FQDNSetExists(ctx, &sapb.FQDNSetExistsRequest{Domains: names})
	if err != nil {
		return fmt.Errorf("checking renewal exemption for %q: %s", names, err)
	}
	if exists.Exists {
		ra.rateLimitCounter.WithLabelValues("certificates_for_domain", "FQDN set bypass").Inc()
		return nil
	}

	tldNames := domainsForRateLimiting(names)
	namesOutOfLimit, earliest, err := ra.enforceNameCounts(ctx, tldNames, limit, regID)
	if err != nil {
		return fmt.Errorf("checking certificates per name limit for %q: %s",
			names, err)
	}

	if len(namesOutOfLimit) > 0 {
		// check if there is already an existing certificate for
		// the exact name set we are issuing for. If so bypass the
		// the certificatesPerName limit.
		exists, err := ra.SA.FQDNSetExists(ctx, &sapb.FQDNSetExistsRequest{Domains: names})
		if err != nil {
			return fmt.Errorf("checking renewal exemption for %q: %s", names, err)
		}
		if exists.Exists {
			ra.rateLimitCounter.WithLabelValues("certificates_for_domain", "FQDN set bypass").Inc()
			return nil
		}

		// Determine the amount of time until the earliest event would fall out
		// of the window.
		retryAfter := earliest.Add(limit.Window.Duration).Sub(ra.clk.Now())
		retryString := earliest.Add(limit.Window.Duration).Format(time.RFC3339)

		ra.log.Infof("Rate limit exceeded, CertificatesForDomain, regID: %d, domains: %s", regID, strings.Join(namesOutOfLimit, ", "))
		ra.rateLimitCounter.WithLabelValues("certificates_for_domain", "exceeded").Inc()
		if len(namesOutOfLimit) > 1 {
			var subErrors []berrors.SubBoulderError
			for _, name := range namesOutOfLimit {
				subErrors = append(subErrors, berrors.SubBoulderError{
					Identifier:   identifier.DNSIdentifier(name),
					BoulderError: berrors.RateLimitError(retryAfter, "too many certificates already issued. Retry after %s", retryString).(*berrors.BoulderError),
				})
			}
			return berrors.RateLimitError(retryAfter, "too many certificates already issued for multiple names (%q and %d others). Retry after %s", namesOutOfLimit[0], len(namesOutOfLimit), retryString).(*berrors.BoulderError).WithSubErrors(subErrors)
		}
		return berrors.RateLimitError(retryAfter, "too many certificates already issued for %q. Retry after %s", namesOutOfLimit[0], retryString)
	}
	ra.rateLimitCounter.WithLabelValues("certificates_for_domain", "pass").Inc()

	return nil
}

func (ra *RegistrationAuthorityImpl) checkCertificatesPerFQDNSetLimit(ctx context.Context, names []string, limit ratelimit.RateLimitPolicy, regID int64) error {
	names = core.UniqueLowerNames(names)
	threshold := limit.GetThreshold(strings.Join(names, ","), regID)
	if threshold <= 0 {
		// No limit configured.
		return nil
	}

	prevIssuances, err := ra.SA.FQDNSetTimestampsForWindow(ctx, &sapb.CountFQDNSetsRequest{
		Domains: names,
		Window:  limit.Window.Duration.Nanoseconds(),
	})
	if err != nil {
		return fmt.Errorf("checking duplicate certificate limit for %q: %s", names, err)
	}

	if int64(len(prevIssuances.Timestamps)) < threshold {
		// Issuance in window is below the threshold, no need to limit.
		return nil
	} else {
		// Evaluate the rate limit using a leaky bucket algorithm. The bucket
		// has a capacity of threshold and is refilled at a rate of 1 token per
		// limit.Window/threshold from the time of each issuance timestamp.
		now := ra.clk.Now()
		nsPerToken := limit.Window.Nanoseconds() / threshold
		for i, timestamp := range prevIssuances.Timestamps {
			tokensGeneratedSince := now.Add(-time.Duration(int64(i+1) * nsPerToken))
			if time.Unix(0, timestamp).Before(tokensGeneratedSince) {
				// We know `i+1` tokens were generated since `tokenGeneratedSince`,
				// and only `i` certificates were issued, so there's room to allow
				// for an additional issuance.
				return nil
			}
		}
		retryTime := time.Unix(0, prevIssuances.Timestamps[0]).Add(time.Duration(nsPerToken))
		retryAfter := retryTime.Sub(now)
		return berrors.DuplicateCertificateError(
			retryAfter,
			"too many certificates (%d) already issued for this exact set of domains in the last %.0f hours: %s, retry after %s",
			threshold, limit.Window.Duration.Hours(), strings.Join(names, ","), retryTime.Format(time.RFC3339),
		)
	}
}

func (ra *RegistrationAuthorityImpl) checkLimits(ctx context.Context, names []string, regID int64) error {
	certNameLimits := ra.rlPolicies.CertificatesPerName()
	if certNameLimits.Enabled() {
		err := ra.checkCertificatesPerNameLimit(ctx, names, certNameLimits, regID)
		if err != nil {
			return err
		}
	}

	fqdnFastLimits := ra.rlPolicies.CertificatesPerFQDNSetFast()
	if fqdnFastLimits.Enabled() {
		err := ra.checkCertificatesPerFQDNSetLimit(ctx, names, fqdnFastLimits, regID)
		if err != nil {
			return err
		}
	}

	fqdnLimits := ra.rlPolicies.CertificatesPerFQDNSet()
	if fqdnLimits.Enabled() {
		err := ra.checkCertificatesPerFQDNSetLimit(ctx, names, fqdnLimits, regID)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateRegistration updates an existing Registration with new values. Caller
// is responsible for making sure that update.Key is only different from base.Key
// if it is being called from the WFE key change endpoint.
// TODO(#5554): Split this into separate methods for updating Contacts vs Key.
func (ra *RegistrationAuthorityImpl) UpdateRegistration(ctx context.Context, req *rapb.UpdateRegistrationRequest) (*corepb.Registration, error) {
	// Error if the request is nil, there is no account key or IP address
	if req.Base == nil || len(req.Base.Key) == 0 || len(req.Base.InitialIP) == 0 || req.Base.Id == 0 {
		return nil, errIncompleteGRPCRequest
	}

	err := validateContactsPresent(req.Base.Contact, req.Base.ContactsPresent)
	if err != nil {
		return nil, err
	}
	err = validateContactsPresent(req.Update.Contact, req.Update.ContactsPresent)
	if err != nil {
		return nil, err
	}
	err = ra.validateContacts(req.Update.Contact)
	if err != nil {
		return nil, err
	}

	update, changed := mergeUpdate(req.Base, req.Update)
	if !changed {
		// If merging the update didn't actually change the base then our work is
		// done, we can return before calling ra.SA.UpdateRegistration since there's
		// nothing for the SA to do
		return req.Base, nil
	}

	_, err = ra.SA.UpdateRegistration(ctx, update)
	if err != nil {
		// berrors.InternalServerError since the user-data was validated before being
		// passed to the SA.
		err = berrors.InternalServerError("Could not update registration: %s", err)
		return nil, err
	}

	return update, nil
}

func contactsEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// If there is an existing contact slice and it has the same length as the
	// new contact slice we need to look at each contact to determine if there
	// is a change being made. Use `sort.Strings` here to ensure a consistent
	// comparison
	sort.Strings(a)
	sort.Strings(b)
	for i := 0; i < len(b); i++ {
		// If the contact's string representation differs at any index they aren't
		// equal
		if a[i] != b[i] {
			return false
		}
	}

	// They are equal!
	return true
}

// MergeUpdate returns a new corepb.Registration with the majority of its fields
// copies from the base Registration, and a subset (Contact, Agreement, and Key)
// copied from the update Registration. It also returns a boolean indicating
// whether or not this operation resulted in a Registration which differs from
// the base.
func mergeUpdate(base *corepb.Registration, update *corepb.Registration) (*corepb.Registration, bool) {
	var changed bool

	// Start by copying all of the fields.
	res := &corepb.Registration{
		Id:              base.Id,
		Key:             base.Key,
		Contact:         base.Contact,
		ContactsPresent: base.ContactsPresent,
		Agreement:       base.Agreement,
		InitialIP:       base.InitialIP,
		CreatedAt:       base.CreatedAt,
		Status:          base.Status,
	}

	// Note: we allow update.Contact to overwrite base.Contact even if the former
	// is empty in order to allow users to remove the contact associated with
	// a registration. If the update has ContactsPresent set to false, then we
	// know it is not attempting to update the contacts field.
	if update.ContactsPresent && !contactsEqual(base.Contact, update.Contact) {
		res.Contact = update.Contact
		res.ContactsPresent = update.ContactsPresent
		changed = true
	}

	if len(update.Agreement) > 0 && update.Agreement != base.Agreement {
		res.Agreement = update.Agreement
		changed = true
	}

	if len(update.Key) > 0 {
		if len(update.Key) != len(base.Key) {
			res.Key = update.Key
			changed = true
		} else {
			for i := 0; i < len(base.Key); i++ {
				if update.Key[i] != base.Key[i] {
					res.Key = update.Key
					changed = true
					break
				}
			}
		}
	}

	return res, changed
}

// recordValidation records an authorization validation event,
// it should only be used on v2 style authorizations.
func (ra *RegistrationAuthorityImpl) recordValidation(ctx context.Context, authID string, authExpires *time.Time, challenge *core.Challenge) error {
	authzID, err := strconv.ParseInt(authID, 10, 64)
	if err != nil {
		return err
	}
	var expires int64
	if challenge.Status == core.StatusInvalid {
		expires = authExpires.UnixNano()
	} else {
		expires = ra.clk.Now().Add(ra.authorizationLifetime).UnixNano()
	}
	vr, err := bgrpc.ValidationResultToPB(challenge.ValidationRecord, challenge.Error)
	if err != nil {
		return err
	}
	var validated int64
	if challenge.Validated != nil {
		validated = challenge.Validated.UTC().UnixNano()
	}
	_, err = ra.SA.FinalizeAuthorization2(ctx, &sapb.FinalizeAuthorizationRequest{
		Id:                authzID,
		Status:            string(challenge.Status),
		Expires:           expires,
		Attempted:         string(challenge.Type),
		AttemptedAt:       validated,
		ValidationRecords: vr.Records,
		ValidationError:   vr.Problems,
	})
	if err != nil {
		return err
	}
	return nil
}

// PerformValidation initiates validation for a specific challenge associated
// with the given base authorization. The authorization and challenge are
// updated based on the results.
func (ra *RegistrationAuthorityImpl) PerformValidation(
	ctx context.Context,
	req *rapb.PerformValidationRequest) (*corepb.Authorization, error) {

	// Clock for start of PerformValidation.
	vStart := ra.clk.Now()

	if req.Authz == nil || req.Authz.Id == "" || req.Authz.Identifier == "" || req.Authz.Status == "" || req.Authz.Expires == 0 {
		return nil, errIncompleteGRPCRequest
	}

	authz, err := bgrpc.PBToAuthz(req.Authz)
	if err != nil {
		return nil, err
	}

	// Refuse to update expired authorizations
	if authz.Expires == nil || authz.Expires.Before(ra.clk.Now()) {
		return nil, berrors.MalformedError("expired authorization")
	}

	challIndex := int(req.ChallengeIndex)
	if challIndex >= len(authz.Challenges) {
		return nil,
			berrors.MalformedError("invalid challenge index '%d'", challIndex)
	}

	ch := &authz.Challenges[challIndex]

	// This challenge type may have been disabled since the challenge was created.
	if !ra.PA.ChallengeTypeEnabled(ch.Type) {
		return nil, berrors.MalformedError("challenge type %q no longer allowed", ch.Type)
	}

	// When configured with `reuseValidAuthz` we can expect some clients to try
	// and update a challenge for an authorization that is already valid. In this
	// case we don't need to process the challenge update. It wouldn't be helpful,
	// the overall authorization is already good! We increment a stat for this
	// case and return early.
	if ra.reuseValidAuthz && authz.Status == core.StatusValid {
		return req.Authz, nil
	}

	if authz.Status != core.StatusPending {
		return nil, berrors.MalformedError("authorization must be pending")
	}

	// Look up the account key for this authorization
	regPB, err := ra.SA.GetRegistration(ctx, &sapb.RegistrationID{Id: authz.RegistrationID})
	if err != nil {
		return nil, berrors.InternalServerError(err.Error())
	}
	reg, err := bgrpc.PbToRegistration(regPB)
	if err != nil {
		return nil, berrors.InternalServerError(err.Error())
	}

	// Compute the key authorization field based on the registration key
	expectedKeyAuthorization, err := ch.ExpectedKeyAuthorization(reg.Key)
	if err != nil {
		return nil, berrors.InternalServerError("could not compute expected key authorization value")
	}

	// Populate the ProvidedKeyAuthorization such that the VA can confirm the
	// expected vs actual without needing the registration key. Historically this
	// was done with the value from the challenge response and so the field name
	// is called "ProvidedKeyAuthorization", in reality this is just
	// "KeyAuthorization".
	// TODO(@cpu): Rename ProvidedKeyAuthorization to KeyAuthorization
	ch.ProvidedKeyAuthorization = expectedKeyAuthorization

	// Double check before sending to VA
	if cErr := ch.CheckConsistencyForValidation(); cErr != nil {
		return nil, berrors.MalformedError(cErr.Error())
	}

	// Dispatch to the VA for service
	vaCtx := context.Background()
	go func(authz core.Authorization) {
		// We will mutate challenges later in this goroutine to change status and
		// add error, but we also return a copy of authz immediately. To avoid a
		// data race, make a copy of the challenges slice here for mutation.
		challenges := make([]core.Challenge, len(authz.Challenges))
		copy(challenges, authz.Challenges)
		authz.Challenges = challenges
		chall, _ := bgrpc.ChallengeToPB(authz.Challenges[challIndex])

		req := vapb.PerformValidationRequest{
			Domain:    authz.Identifier.Value,
			Challenge: chall,
			Authz: &vapb.AuthzMeta{
				Id:    authz.ID,
				RegID: authz.RegistrationID,
			},
		}
		res, err := ra.VA.PerformValidation(vaCtx, &req)

		challenge := &authz.Challenges[challIndex]
		var prob *probs.ProblemDetails

		if err != nil {
			prob = probs.ServerInternal("Could not communicate with VA")
			ra.log.AuditErrf("Could not communicate with VA: %s", err)
		} else {
			if res.Problems != nil {
				prob, err = bgrpc.PBToProblemDetails(res.Problems)
				if err != nil {
					prob = probs.ServerInternal("Could not communicate with VA")
					ra.log.AuditErrf("Could not communicate with VA: %s", err)
				}
			}

			// Save the updated records
			records := make([]core.ValidationRecord, len(res.Records))
			for i, r := range res.Records {
				records[i], err = bgrpc.PBToValidationRecord(r)
				if err != nil {
					prob = probs.ServerInternal("Records for validation corrupt")
				}
			}
			challenge.ValidationRecord = records
		}

		if !challenge.RecordsSane() && prob == nil {
			prob = probs.ServerInternal("Records for validation failed sanity check")
		}

		if prob != nil {
			challenge.Status = core.StatusInvalid
			challenge.Error = prob
		} else {
			challenge.Status = core.StatusValid
		}
		challenge.Validated = &vStart
		authz.Challenges[challIndex] = *challenge

		err = ra.recordValidation(vaCtx, authz.ID, authz.Expires, challenge)
		if err != nil {
			ra.log.AuditErrf("Could not record updated validation: regID=[%d] authzID=[%s] err=[%s]",
				authz.RegistrationID, authz.ID, err)
		}
	}(authz)
	return bgrpc.AuthzToPB(authz)
}

// revokeCertificate generates a revoked OCSP response for the certificate with
// the given serial and issuer and stores that response in the database.
// TODO(#5152) make the issuerID argument an issuance.IssuerNameID
func (ra *RegistrationAuthorityImpl) revokeCertificate(ctx context.Context, serial *big.Int, issuerID int64, reason revocation.Reason) error {
	serialString := core.SerialToString(serial)
	revokedAt := ra.clk.Now().UnixNano()

	var ocspResponse []byte
	if !features.Enabled(features.ROCSPStage7) {
		ocspResponsePB, err := ra.OCSP.GenerateOCSP(ctx, &capb.GenerateOCSPRequest{
			Serial:    serialString,
			IssuerID:  issuerID,
			Status:    string(core.OCSPStatusRevoked),
			Reason:    int32(reason),
			RevokedAt: revokedAt,
		})
		if err != nil {
			return err
		}
		ocspResponse = ocspResponsePB.Response
	}

	_, err := ra.SA.RevokeCertificate(ctx, &sapb.RevokeCertificateRequest{
		Serial:   serialString,
		Reason:   int64(reason),
		Date:     revokedAt,
		Response: ocspResponse,
		IssuerID: issuerID,
	})
	if err != nil {
		return err
	}

	ra.revocationReasonCounter.WithLabelValues(revocation.ReasonToString[reason]).Inc()
	return nil
}

// updateRevocationForKeyCompromise generates a revoked OCSP response for the
// already-revoked certificate with the given serial and issuer, and stores that
// response in the database. This only works for certificates that were
// previously revoked for a reason other than keyCompromise, and which are now
// being updated to keyCompromise instead.
// TODO(#5152) make the issuerID argument an issuance.IssuerNameID
func (ra *RegistrationAuthorityImpl) updateRevocationForKeyCompromise(ctx context.Context, serial *big.Int, issuerID int64) error {
	serialString := core.SerialToString(serial)
	thisUpdate := ra.clk.Now().UnixNano()

	status, err := ra.SA.GetCertificateStatus(ctx, &sapb.Serial{Serial: serialString})
	if err != nil {
		return berrors.NotFoundError("unable to confirm that serial %q was ever issued: %s", serialString, err)
	}

	if status.Status != string(core.OCSPStatusRevoked) {
		// Internal server error, because we shouldn't be in the function at all
		// unless the cert was already revoked.
		return fmt.Errorf("unable to re-revoke serial %q which is not currently revoked", serialString)
	}
	if status.RevokedReason == ocsp.KeyCompromise {
		return berrors.AlreadyRevokedError("unable to re-revoke serial %q which is already revoked for keyCompromise", serialString)
	}

	// The new OCSP response has to be back-dated to the original date.
	var ocspResponse []byte
	if !features.Enabled(features.ROCSPStage7) {
		ocspResponsePB, err := ra.OCSP.GenerateOCSP(ctx, &capb.GenerateOCSPRequest{
			Serial:    serialString,
			IssuerID:  issuerID,
			Status:    string(core.OCSPStatusRevoked),
			Reason:    int32(ocsp.KeyCompromise),
			RevokedAt: status.RevokedDate,
		})
		if err != nil {
			return err
		}
		ocspResponse = ocspResponsePB.Response
	}
	_, err = ra.SA.UpdateRevokedCertificate(ctx, &sapb.RevokeCertificateRequest{
		Serial:   serialString,
		Reason:   int64(ocsp.KeyCompromise),
		Date:     thisUpdate,
		Backdate: status.RevokedDate,
		Response: ocspResponse,
		IssuerID: issuerID,
	})
	if err != nil {
		return err
	}

	ra.revocationReasonCounter.WithLabelValues(revocation.ReasonToString[ocsp.KeyCompromise]).Inc()
	return nil
}

// purgeOCSPCache makes a request to akamai-purger to purge the cache entries
// for the given certificate.
// TODO(#5152) make the issuerID argument an issuance.IssuerNameID
func (ra *RegistrationAuthorityImpl) purgeOCSPCache(ctx context.Context, cert *x509.Certificate, issuerID int64) error {
	issuer, ok := ra.issuersByNameID[issuance.IssuerNameID(issuerID)]
	if !ok {
		// TODO(#5152): Remove this fallback (which only gets used when revoking by
		// serial, so the issuer ID had to be read from the db).
		issuer, ok = ra.issuersByID[issuance.IssuerID(issuerID)]
		if !ok {
			return fmt.Errorf("unable to identify issuer of cert with serial %q", core.SerialToString(cert.SerialNumber))
		}
	}

	purgeURLs, err := akamai.GeneratePurgeURLs(cert, issuer.Certificate)
	if err != nil {
		return err
	}

	_, err = ra.purger.Purge(ctx, &akamaipb.PurgeRequest{Urls: purgeURLs})
	if err != nil {
		return err
	}

	return nil
}

// RevokeCertByApplicant revokes the certificate in question. It allows any
// revocation reason from (0, 1, 3, 4, 5, 9), because Subscribers are allowed to
// request any revocation reason for their own certificates. However, if the
// requesting RegID is an account which has authorizations for all names in the
// cert but is *not* the original subscriber, it overrides the revocation reason
// to be 5 (cessationOfOperation), because that code is used to cover instances
// where "the certificate subscriber no longer owns the domain names in the
// certificate". It does not add the key to the blocked keys list, even if
// reason 1 (keyCompromise) is requested, as it does not demonstrate said
// compromise. It attempts to purge the certificate from the Akamai cache, but
// it does not hard-fail if doing so is not successful, because the cache will
// drop the old OCSP response in less than 24 hours anyway.
func (ra *RegistrationAuthorityImpl) RevokeCertByApplicant(ctx context.Context, req *rapb.RevokeCertByApplicantRequest) (*emptypb.Empty, error) {
	if req == nil || req.Cert == nil || req.RegID == 0 {
		return nil, errIncompleteGRPCRequest
	}

	if _, present := revocation.UserAllowedReasons[revocation.Reason(req.Code)]; !present {
		return nil, berrors.BadRevocationReasonError(req.Code)
	}

	cert, err := x509.ParseCertificate(req.Cert)
	if err != nil {
		return nil, err
	}

	serialString := core.SerialToString(cert.SerialNumber)

	logEvent := certificateRevocationEvent{
		ID:           core.NewToken(),
		SerialNumber: serialString,
		Reason:       req.Code,
		Method:       "applicant",
		RequesterID:  req.RegID,
	}

	// Below this point, do not re-declare `err` (i.e. type `err :=`) in a
	// nested scope. Doing so will create a new `err` variable that is not
	// captured by this closure.
	defer func() {
		if err != nil {
			logEvent.Error = err.Error()
		}
		ra.log.AuditObject("Revocation request:", logEvent)
	}()

	metadata, err := ra.SA.GetSerialMetadata(ctx, &sapb.Serial{Serial: serialString})
	if err != nil {
		return nil, err
	}

	if req.RegID == metadata.RegistrationID {
		// The requester is the original subscriber. They can revoke for any reason.
		logEvent.Method = "subscriber"
	} else {
		// The requester is a different account. We need to confirm that they have
		// authorizations for all names in the cert.
		logEvent.Method = "control"

		var authzMapPB *sapb.Authorizations
		authzMapPB, err = ra.SA.GetValidAuthorizations2(ctx, &sapb.GetValidAuthorizationsRequest{
			RegistrationID: req.RegID,
			Domains:        cert.DNSNames,
			Now:            ra.clk.Now().UnixNano(),
		})
		if err != nil {
			return nil, err
		}

		m := make(map[string]struct{})
		for _, authz := range authzMapPB.Authz {
			m[authz.Domain] = struct{}{}
		}
		for _, name := range cert.DNSNames {
			if _, present := m[name]; !present {
				return nil, berrors.UnauthorizedError("requester does not control all names in cert with serial %q", serialString)
			}
		}

		// Applicants who are not the original Subscriber are not allowed to
		// revoke for any reason other than cessationOfOperation, which covers
		// circumstances where "the certificate subscriber no longer owns the
		// domain names in the certificate". Override the reason code to match.
		req.Code = ocsp.CessationOfOperation
		logEvent.Reason = req.Code
	}

	issuerID := issuance.GetIssuerNameID(cert)
	err = ra.revokeCertificate(
		ctx,
		cert.SerialNumber,
		int64(issuerID),
		revocation.Reason(req.Code),
	)
	if err != nil {
		return nil, err
	}

	// TODO(#5979): Check this error when it can't simply be due to a full queue.
	_ = ra.purgeOCSPCache(ctx, cert, int64(issuerID))

	return &emptypb.Empty{}, nil
}

// RevokeCertByKey revokes the certificate in question. It always uses
// reason code 1 (keyCompromise). It ensures that they public key is added to
// the blocked keys list, even if revocation otherwise fails. It attempts to
// purge the certificate from the Akamai cache, but it does not hard-fail if
// doing so is not successful, because the cache will drop the old OCSP response
// in less than 24 hours anyway.
func (ra *RegistrationAuthorityImpl) RevokeCertByKey(ctx context.Context, req *rapb.RevokeCertByKeyRequest) (*emptypb.Empty, error) {
	if req == nil || req.Cert == nil {
		return nil, errIncompleteGRPCRequest
	}

	cert, err := x509.ParseCertificate(req.Cert)
	if err != nil {
		return nil, err
	}

	issuerID := issuance.GetIssuerNameID(cert)

	logEvent := certificateRevocationEvent{
		ID:           core.NewToken(),
		SerialNumber: core.SerialToString(cert.SerialNumber),
		Reason:       ocsp.KeyCompromise,
		Method:       "key",
		RequesterID:  0,
	}

	// Below this point, do not re-declare `err` (i.e. type `err :=`) in a
	// nested scope. Doing so will create a new `err` variable that is not
	// captured by this closure.
	defer func() {
		if err != nil {
			logEvent.Error = err.Error()
		}
		ra.log.AuditObject("Revocation request:", logEvent)
	}()

	// We revoke the cert before adding it to the blocked keys list, to avoid a
	// race between this and the bad-key-revoker. But we don't check the error on
	// from this operation until after we add to the blocked keys list, since that
	// add needs to happen no matter what.
	revokeErr := ra.revokeCertificate(
		ctx,
		cert.SerialNumber,
		int64(issuerID),
		revocation.Reason(ocsp.KeyCompromise),
	)

	// Now add the public key to the blocked keys list, and report the error if
	// there is one. It's okay to error out here because failing to add the key
	// to the blocked keys list is a worse failure than failing to revoke in the
	// first place, because it means that bad-key-revoker won't revoke the cert
	// anyway.
	var digest core.Sha256Digest
	digest, err = core.KeyDigest(cert.PublicKey)
	if err != nil {
		return nil, err
	}
	_, err = ra.SA.AddBlockedKey(ctx, &sapb.AddBlockedKeyRequest{
		KeyHash: digest[:],
		Added:   ra.clk.Now().UnixNano(),
		Source:  "API",
	})
	if err != nil {
		return nil, err
	}

	// Finally check the error from revocation itself. If it was an AlreadyRevoked
	// error, try to re-revoke the cert, in case it is revoked for a reason other
	// than keyCompromise.
	err = revokeErr
	if err != nil {
		// Error out if the error was anything other than AlreadyRevoked. Otherwise
		// try re-revocation.
		if !errors.Is(err, berrors.AlreadyRevoked) {
			return nil, err
		}
		err = ra.updateRevocationForKeyCompromise(ctx, cert.SerialNumber, int64(issuerID))
		if err != nil {
			return nil, err
		}
	}

	// TODO(#5979): Check this error when it can't simply be due to a full queue.
	_ = ra.purgeOCSPCache(ctx, cert, int64(issuerID))

	return &emptypb.Empty{}, nil
}

// AdministrativelyRevokeCertificate terminates trust in the certificate
// provided and does not require the registration ID of the requester since this
// method is only called from the admin-revoker tool. It trusts that the admin
// is doing the right thing, so if the requested reason is keyCompromise, it
// blocks the key from future issuance even though compromise has not been
// demonstrated here. It purges the certificate from the Akamai cache, and
// returns an error if that purge fails, since this method may be called late
// in the BRs-mandated revocation timeframe.
func (ra *RegistrationAuthorityImpl) AdministrativelyRevokeCertificate(ctx context.Context, req *rapb.AdministrativelyRevokeCertificateRequest) (*emptypb.Empty, error) {
	if req == nil || req.AdminName == "" {
		return nil, errIncompleteGRPCRequest
	}
	if req.Cert == nil && req.Serial == "" {
		return nil, errIncompleteGRPCRequest
	}

	reasonCode := revocation.Reason(req.Code)
	if reasonCode == ocsp.KeyCompromise && req.Cert == nil && !req.SkipBlockKey {
		return nil, fmt.Errorf("cannot revoke and block for KeyCompromise by serial alone")
	}
	if req.SkipBlockKey && reasonCode != ocsp.KeyCompromise {
		return nil, fmt.Errorf("cannot skip key blocking for reasons other than KeyCompromise")
	}

	if _, present := revocation.AdminAllowedReasons[reasonCode]; !present {
		return nil, fmt.Errorf("cannot revoke for reason %d", reasonCode)
	}

	// If we don't have a real cert, we create a fake cert (containing just the
	// serial number, which is all we need) and look up the IssuerID from the db.
	// We could instead look up and parse the certificate itself, but we avoid
	// that in case we are administratively revoking the certificate because it is
	// so badly malformed that it can't be parsed.
	var cert *x509.Certificate
	var issuerID int64 // TODO(#5152) make this an issuance.IssuerNameID
	var err error
	if req.Cert == nil {
		serial, err := core.StringToSerial(req.Serial)
		if err != nil {
			return nil, err
		}

		cert = &x509.Certificate{
			SerialNumber: serial,
		}

		status, err := ra.SA.GetCertificateStatus(ctx, &sapb.Serial{Serial: req.Serial})
		if err != nil {
			return nil, fmt.Errorf("unable to confirm that serial %q was ever issued: %w", serial, err)
		}
		issuerID = status.IssuerID
	} else {
		cert, err = x509.ParseCertificate(req.Cert)
		if err != nil {
			return nil, err
		}
		issuerID = int64(issuance.GetIssuerNameID(cert))
	}

	logEvent := certificateRevocationEvent{
		ID:           core.NewToken(),
		Method:       "key",
		AdminName:    req.AdminName,
		SerialNumber: core.SerialToString(cert.SerialNumber),
	}

	// Below this point, do not re-declare `err` (i.e. type `err :=`) in a
	// nested scope. Doing so will create a new `err` variable that is not
	// captured by this closure.
	defer func() {
		if err != nil {
			logEvent.Error = err.Error()
		}
		ra.log.AuditObject("Revocation request:", logEvent)
	}()

	err = ra.revokeCertificate(ctx, cert.SerialNumber, issuerID, revocation.Reason(req.Code))
	if err != nil {
		if req.Code == ocsp.KeyCompromise && errors.Is(err, berrors.AlreadyRevoked) {
			err = ra.updateRevocationForKeyCompromise(ctx, cert.SerialNumber, issuerID)
			if err != nil {
				return nil, err
			}
		}
		return nil, err
	}

	if req.Code == ocsp.KeyCompromise && !req.SkipBlockKey {
		var digest core.Sha256Digest
		digest, err = core.KeyDigest(cert.PublicKey)
		if err != nil {
			return nil, err
		}
		_, err = ra.SA.AddBlockedKey(ctx, &sapb.AddBlockedKeyRequest{
			KeyHash: digest[:],
			Added:   ra.clk.Now().UnixNano(),
			Source:  "admin-revoker",
			Comment: fmt.Sprintf("revoked by %s", req.AdminName),
		})
		if err != nil {
			return nil, err
		}
	}

	err = ra.purgeOCSPCache(ctx, cert, issuerID)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// DeactivateRegistration deactivates a valid registration
func (ra *RegistrationAuthorityImpl) DeactivateRegistration(ctx context.Context, reg *corepb.Registration) (*emptypb.Empty, error) {
	if reg == nil || reg.Id == 0 {
		return nil, errIncompleteGRPCRequest
	}
	if reg.Status != string(core.StatusValid) {
		return nil, berrors.MalformedError("only valid registrations can be deactivated")
	}
	_, err := ra.SA.DeactivateRegistration(ctx, &sapb.RegistrationID{Id: reg.Id})
	if err != nil {
		return nil, berrors.InternalServerError(err.Error())
	}
	return &emptypb.Empty{}, nil
}

// DeactivateAuthorization deactivates a currently valid authorization
func (ra *RegistrationAuthorityImpl) DeactivateAuthorization(ctx context.Context, req *corepb.Authorization) (*emptypb.Empty, error) {
	if req == nil || req.Id == "" || req.Status == "" {
		return nil, errIncompleteGRPCRequest
	}
	authzID, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, err
	}
	if _, err := ra.SA.DeactivateAuthorization2(ctx, &sapb.AuthorizationID2{Id: authzID}); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// checkOrderNames validates that the RA's policy authority allows issuing for
// each of the names in an order. If any of the names are unacceptable a
// malformed or rejectedIdentifier error with suberrors for each rejected
// identifier is returned.
func (ra *RegistrationAuthorityImpl) checkOrderNames(names []string) error {
	idents := make([]identifier.ACMEIdentifier, len(names))
	for i, name := range names {
		idents[i] = identifier.DNSIdentifier(name)
	}
	err := ra.PA.WillingToIssueWildcards(idents)
	if err != nil {
		return err
	}
	return nil
}

// GenerateOCSP looks up a certificate's status, then requests a signed OCSP
// response for it from the CA. If the certificate status is not available
// or the certificate is expired, it returns berrors.NotFoundError.
// This does not write back the result to the SA or any other storage.
func (ra *RegistrationAuthorityImpl) GenerateOCSP(ctx context.Context, req *rapb.GenerateOCSPRequest) (*capb.OCSPResponse, error) {
	status, err := ra.SA.GetCertificateStatus(ctx, &sapb.Serial{Serial: req.Serial})
	if err != nil {
		return nil, err
	}

	notAfter := time.Unix(0, status.NotAfter).UTC()
	if ra.clk.Now().After(notAfter) {
		return nil, berrors.NotFoundError("certificate is expired")
	}

	return ra.OCSP.GenerateOCSP(ctx, &capb.GenerateOCSPRequest{
		Serial:    req.Serial,
		Status:    status.Status,
		Reason:    int32(status.RevokedReason),
		RevokedAt: status.RevokedDate,
		IssuerID:  status.IssuerID,
	})
}

// NewOrder creates a new order object
func (ra *RegistrationAuthorityImpl) NewOrder(ctx context.Context, req *rapb.NewOrderRequest) (*corepb.Order, error) {
	if req == nil || req.RegistrationID == 0 {
		return nil, errIncompleteGRPCRequest
	}

	newOrder := &sapb.NewOrderRequest{
		RegistrationID: req.RegistrationID,
		Names:          core.UniqueLowerNames(req.Names),
	}

	if len(newOrder.Names) > ra.maxNames {
		return nil, berrors.MalformedError(
			"Order cannot contain more than %d DNS names", ra.maxNames)
	}

	// Validate that our policy allows issuing for each of the names in the order
	err := ra.checkOrderNames(newOrder.Names)
	if err != nil {
		return nil, err
	}

	err = wildcardOverlap(newOrder.Names)
	if err != nil {
		return nil, err
	}

	// See if there is an existing unexpired pending (or ready) order that can be reused
	// for this account
	existingOrder, err := ra.SA.GetOrderForNames(ctx, &sapb.GetOrderForNamesRequest{
		AcctID: newOrder.RegistrationID,
		Names:  newOrder.Names,
	})
	// If there was an error and it wasn't an acceptable "NotFound" error, return
	// immediately
	if err != nil && !errors.Is(err, berrors.NotFound) {
		return nil, err
	}

	// If there was an order, make sure it has expected fields and return it
	// Error if an incomplete order is returned.
	if existingOrder != nil {
		// Check to see if the expected fields of the existing order are set.
		if existingOrder.Id == 0 || existingOrder.Created == 0 || existingOrder.Status == "" || existingOrder.RegistrationID == 0 || existingOrder.Expires == 0 || len(existingOrder.Names) == 0 {
			return nil, errIncompleteGRPCResponse
		}
		return existingOrder, nil
	}

	// Check if there is rate limit space for a new order within the current window
	err = ra.checkNewOrdersPerAccountLimit(ctx, newOrder.RegistrationID)
	if err != nil {
		return nil, err
	}
	// Check if there is rate limit space for issuing a certificate for the new
	// order's names. If there isn't then it doesn't make sense to allow creating
	// an order - it will just fail when finalization checks the same limits.
	err = ra.checkLimits(ctx, newOrder.Names, newOrder.RegistrationID)
	if err != nil {
		return nil, err
	}
	err = ra.checkInvalidAuthorizationLimits(ctx, newOrder.RegistrationID, newOrder.Names)
	if err != nil {
		return nil, err
	}

	// An order's lifetime is effectively bound by the shortest remaining lifetime
	// of its associated authorizations. For that reason it would be Uncool if
	// `sa.GetAuthorizations` returned an authorization that was very close to
	// expiry. The resulting pending order that references it would itself end up
	// expiring very soon.
	// To prevent this we only return authorizations that are at least 1 day away
	// from expiring.
	authzExpiryCutoff := ra.clk.Now().AddDate(0, 0, 1).UnixNano()

	getAuthReq := &sapb.GetAuthorizationsRequest{
		RegistrationID: newOrder.RegistrationID,
		Now:            authzExpiryCutoff,
		Domains:        newOrder.Names,
	}
	existingAuthz, err := ra.SA.GetAuthorizations2(ctx, getAuthReq)
	if err != nil {
		return nil, err
	}

	// Collect up the authorizations we found into a map keyed by the domains the
	// authorizations correspond to
	nameToExistingAuthz := make(map[string]*corepb.Authorization, len(newOrder.Names))
	for _, v := range existingAuthz.Authz {
		// Don't reuse a valid authorization if the reuseValidAuthz flag is
		// disabled.
		if v.Authz.Status == string(core.StatusValid) && !ra.reuseValidAuthz {
			continue
		}
		nameToExistingAuthz[v.Domain] = v.Authz
	}

	// For each of the names in the order, if there is an acceptable
	// existing authz, append it to the order to reuse it. Otherwise track
	// that there is a missing authz for that name.
	var missingAuthzNames []string
	for _, name := range newOrder.Names {
		// If there isn't an existing authz, note that its missing and continue
		if _, exists := nameToExistingAuthz[name]; !exists {
			missingAuthzNames = append(missingAuthzNames, name)
			continue
		}
		authz := nameToExistingAuthz[name]
		// If the identifier is a wildcard and the existing authz only has one
		// DNS-01 type challenge we can reuse it. In theory we will
		// never get back an authorization for a domain with a wildcard prefix
		// that doesn't meet this criteria from SA.GetAuthorizations but we verify
		// again to be safe.
		if strings.HasPrefix(name, "*.") &&
			len(authz.Challenges) == 1 && core.AcmeChallenge(authz.Challenges[0].Type) == core.ChallengeTypeDNS01 {
			authzID, err := strconv.ParseInt(authz.Id, 10, 64)
			if err != nil {
				return nil, err
			}
			newOrder.V2Authorizations = append(newOrder.V2Authorizations, authzID)
			ra.authzAges.Observe((time.Unix(0, authz.Expires).Sub(ra.clk.Now()) - ra.authorizationLifetime).Seconds())
			continue
		} else if !strings.HasPrefix(name, "*.") {
			// If the identifier isn't a wildcard, we can reuse any authz
			authzID, err := strconv.ParseInt(authz.Id, 10, 64)
			if err != nil {
				return nil, err
			}
			newOrder.V2Authorizations = append(newOrder.V2Authorizations, authzID)
			ra.authzAges.Observe((time.Unix(0, authz.Expires).Sub(ra.clk.Now()) - ra.authorizationLifetime).Seconds())
			continue
		}

		// Delete the authz from the nameToExistingAuthz map since we are not reusing it.
		delete(nameToExistingAuthz, name)
		// If we reached this point then the existing authz was not acceptable for
		// reuse and we need to mark the name as requiring a new pending authz
		missingAuthzNames = append(missingAuthzNames, name)
	}
	ra.reusedValidAuthzCounter.Add(float64(len(newOrder.V2Authorizations)))

	// If the order isn't fully authorized we need to check that the client has
	// rate limit room for more pending authorizations
	if len(missingAuthzNames) > 0 {
		err := ra.checkPendingAuthorizationLimit(ctx, newOrder.RegistrationID)
		if err != nil {
			return nil, err
		}
	}

	// Loop through each of the names missing authzs and create a new pending
	// authorization for each.
	var newAuthzs []*corepb.Authorization
	for _, name := range missingAuthzNames {
		pb, err := ra.createPendingAuthz(newOrder.RegistrationID, identifier.ACMEIdentifier{
			Type:  identifier.DNS,
			Value: name,
		})
		if err != nil {
			return nil, err
		}
		newAuthzs = append(newAuthzs, pb)
		ra.authzAges.Observe(0)
	}

	// Start with the order's own expiry as the minExpiry. We only care
	// about authz expiries that are sooner than the order's expiry
	minExpiry := ra.clk.Now().Add(ra.orderLifetime)

	// Check the reused authorizations to see if any have an expiry before the
	// minExpiry (the order's lifetime)
	for _, authz := range nameToExistingAuthz {
		// An authz without an expiry is an unexpected internal server event
		if authz.Expires == 0 {
			return nil, berrors.InternalServerError(
				"SA.GetAuthorizations returned an authz (%s) with zero expiry",
				authz.Id)
		}
		// If the reused authorization expires before the minExpiry, it's expiry
		// is the new minExpiry.
		authzExpiry := time.Unix(0, authz.Expires)
		if authzExpiry.Before(minExpiry) {
			minExpiry = authzExpiry
		}
	}
	// If the newly created pending authz's have an expiry closer than the
	// minExpiry the minExpiry is the pending authz expiry.
	if len(newAuthzs) > 0 {
		newPendingAuthzExpires := ra.clk.Now().Add(ra.pendingAuthorizationLifetime)
		if newPendingAuthzExpires.Before(minExpiry) {
			minExpiry = newPendingAuthzExpires
		}
	}
	// Set the order's expiry to the minimum expiry. The db doesn't store
	// sub-second values, so truncate here.
	newOrder.Expires = minExpiry.Truncate(time.Second).UnixNano()

	newOrderAndAuthzsReq := &sapb.NewOrderAndAuthzsRequest{
		NewOrder:  newOrder,
		NewAuthzs: newAuthzs,
	}
	storedOrder, err := ra.SA.NewOrderAndAuthzs(ctx, newOrderAndAuthzsReq)
	if err != nil {
		return nil, err
	}
	if storedOrder.Id == 0 || storedOrder.Created == 0 || storedOrder.Status == "" || storedOrder.RegistrationID == 0 || storedOrder.Expires == 0 || len(storedOrder.Names) == 0 {
		return nil, errIncompleteGRPCResponse
	}

	// Note how many names are being requested in this certificate order.
	ra.namesPerCert.With(prometheus.Labels{"type": "requested"}).Observe(float64(len(storedOrder.Names)))

	return storedOrder, nil
}

// createPendingAuthz checks that a name is allowed for issuance and creates the
// necessary challenges for it and puts this and all of the relevant information
// into a corepb.Authorization for transmission to the SA to be stored
func (ra *RegistrationAuthorityImpl) createPendingAuthz(reg int64, identifier identifier.ACMEIdentifier) (*corepb.Authorization, error) {
	authz := &corepb.Authorization{
		Identifier:     identifier.Value,
		RegistrationID: reg,
		Status:         string(core.StatusPending),
		Expires:        ra.clk.Now().Add(ra.pendingAuthorizationLifetime).Truncate(time.Second).UnixNano(),
	}

	// Create challenges. The WFE will update them with URIs before sending them out.
	challenges, err := ra.PA.ChallengesFor(identifier)
	if err != nil {
		// The only time ChallengesFor errors it is a fatal configuration error
		// where challenges required by policy for an identifier are not enabled. We
		// want to treat this as an internal server error.
		return nil, berrors.InternalServerError(err.Error())
	}
	// Check each challenge for sanity.
	for _, challenge := range challenges {
		err := challenge.CheckConsistencyForClientOffer()
		if err != nil {
			// berrors.InternalServerError because we generated these challenges, they should
			// be OK.
			err = berrors.InternalServerError("challenge didn't pass sanity check: %+v", challenge)
			return nil, err
		}
		challPB, err := bgrpc.ChallengeToPB(challenge)
		if err != nil {
			return nil, err
		}
		authz.Challenges = append(authz.Challenges, challPB)
	}
	return authz, nil
}

// wildcardOverlap takes a slice of domain names and returns an error if any of
// them is a non-wildcard FQDN that overlaps with a wildcard domain in the map.
func wildcardOverlap(dnsNames []string) error {
	nameMap := make(map[string]bool, len(dnsNames))
	for _, v := range dnsNames {
		nameMap[v] = true
	}
	for name := range nameMap {
		if name[0] == '*' {
			continue
		}
		labels := strings.Split(name, ".")
		labels[0] = "*"
		if nameMap[strings.Join(labels, ".")] {
			return berrors.MalformedError(
				"Domain name %q is redundant with a wildcard domain in the same request. Remove one or the other from the certificate request.", name)
		}
	}
	return nil
}

// validateContactsPresent will return an error if the contacts []string
// len is greater than zero and the contactsPresent bool is false. We
// don't care about any other cases. If the length of the contacts is zero
// and contactsPresent is true, it seems like a mismatch but we have to
// assume that the client is requesting to update the contacts field with
// by removing the existing contacts value so we don't want to return an
// error here.
func validateContactsPresent(contacts []string, contactsPresent bool) error {
	if len(contacts) > 0 && !contactsPresent {
		return berrors.InternalServerError("account contacts present but contactsPresent false")
	}
	return nil
}
