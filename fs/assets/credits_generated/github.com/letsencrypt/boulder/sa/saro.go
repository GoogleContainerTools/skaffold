package sa

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jmhodges/clock"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	jose "gopkg.in/go-jose/go-jose.v2"

	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	"github.com/letsencrypt/boulder/db"
	berrors "github.com/letsencrypt/boulder/errors"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	"github.com/letsencrypt/boulder/identifier"
	blog "github.com/letsencrypt/boulder/log"
	sapb "github.com/letsencrypt/boulder/sa/proto"
)

var (
	validIncidentTableRegexp = regexp.MustCompile(`^incident_[0-9a-zA-Z_]{1,100}$`)
)

type certCountFunc func(db db.Selector, domain string, timeRange *sapb.Range) (int64, time.Time, error)

// SQLStorageAuthorityRO defines a read-only subset of a Storage Authority
type SQLStorageAuthorityRO struct {
	sapb.UnimplementedStorageAuthorityReadOnlyServer

	dbReadOnlyMap  *db.WrappedMap
	dbIncidentsMap *db.WrappedMap

	// For RPCs that generate multiple, parallelizable SQL queries, this is the
	// max parallelism they will use (to avoid consuming too many MariaDB
	// threads).
	parallelismPerRPC int

	// We use function types here so we can mock out this internal function in
	// unittests.
	countCertificatesByName certCountFunc

	clk clock.Clock
	log blog.Logger
}

// NewSQLStorageAuthorityRO provides persistence using a SQL backend for
// Boulder. It will modify the given gorp.DbMap by adding relevant tables.
func NewSQLStorageAuthorityRO(
	dbReadOnlyMap *db.WrappedMap,
	dbIncidentsMap *db.WrappedMap,
	parallelismPerRPC int,
	clk clock.Clock,
	logger blog.Logger,
) (*SQLStorageAuthorityRO, error) {
	ssaro := &SQLStorageAuthorityRO{
		dbReadOnlyMap:     dbReadOnlyMap,
		dbIncidentsMap:    dbIncidentsMap,
		parallelismPerRPC: parallelismPerRPC,
		clk:               clk,
		log:               logger,
	}

	ssaro.countCertificatesByName = ssaro.countCertificates

	return ssaro, nil
}

// GetRegistration obtains a Registration by ID
func (ssa *SQLStorageAuthorityRO) GetRegistration(ctx context.Context, req *sapb.RegistrationID) (*corepb.Registration, error) {
	if req == nil || req.Id == 0 {
		return nil, errIncompleteRequest
	}

	const query = "WHERE id = ?"
	model, err := selectRegistration(ssa.dbReadOnlyMap.WithContext(ctx), query, req.Id)
	if err != nil {
		if db.IsNoRows(err) {
			return nil, berrors.NotFoundError("registration with ID '%d' not found", req.Id)
		}
		return nil, err
	}

	return registrationModelToPb(model)
}

func (ssa *SQLStorageAuthority) GetRegistration(ctx context.Context, req *sapb.RegistrationID) (*corepb.Registration, error) {
	return ssa.SQLStorageAuthorityRO.GetRegistration(ctx, req)
}

// GetRegistrationByKey obtains a Registration by JWK
func (ssa *SQLStorageAuthorityRO) GetRegistrationByKey(ctx context.Context, req *sapb.JSONWebKey) (*corepb.Registration, error) {
	if req == nil || len(req.Jwk) == 0 {
		return nil, errIncompleteRequest
	}

	var jwk jose.JSONWebKey
	err := jwk.UnmarshalJSON(req.Jwk)
	if err != nil {
		return nil, err
	}

	const query = "WHERE jwk_sha256 = ?"
	sha, err := core.KeyDigestB64(jwk.Key)
	if err != nil {
		return nil, err
	}
	model, err := selectRegistration(ssa.dbReadOnlyMap.WithContext(ctx), query, sha)
	if err != nil {
		if db.IsNoRows(err) {
			return nil, berrors.NotFoundError("no registrations with public key sha256 %q", sha)
		}
		return nil, err
	}

	return registrationModelToPb(model)
}

func (ssa *SQLStorageAuthority) GetRegistrationByKey(ctx context.Context, req *sapb.JSONWebKey) (*corepb.Registration, error) {
	return ssa.SQLStorageAuthorityRO.GetRegistrationByKey(ctx, req)
}

// incrementIP returns a copy of `ip` incremented at a bit index `index`,
// or in other words the first IP of the next highest subnet given a mask of
// length `index`.
// In order to easily account for overflow, we treat ip as a big.Int and add to
// it. If the increment overflows the max size of a net.IP, return the highest
// possible net.IP.
func incrementIP(ip net.IP, index int) net.IP {
	bigInt := new(big.Int)
	bigInt.SetBytes([]byte(ip))
	incr := new(big.Int).Lsh(big.NewInt(1), 128-uint(index))
	bigInt.Add(bigInt, incr)
	// bigInt.Bytes can be shorter than 16 bytes, so stick it into a
	// full-sized net.IP.
	resultBytes := bigInt.Bytes()
	if len(resultBytes) > 16 {
		return net.ParseIP("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")
	}
	result := make(net.IP, 16)
	copy(result[16-len(resultBytes):], resultBytes)
	return result
}

// ipRange returns a range of IP addresses suitable for querying MySQL for the
// purpose of rate limiting using a range that is inclusive on the lower end and
// exclusive at the higher end. If ip is an IPv4 address, it returns that address,
// plus the one immediately higher than it. If ip is an IPv6 address, it applies
// a /48 mask to it and returns the lowest IP in the resulting network, and the
// first IP outside of the resulting network.
func ipRange(ip net.IP) (net.IP, net.IP) {
	ip = ip.To16()
	// For IPv6, match on a certain subnet range, since one person can commonly
	// have an entire /48 to themselves.
	maskLength := 48
	// For IPv4 addresses, do a match on exact address, so begin = ip and end =
	// next higher IP.
	if ip.To4() != nil {
		maskLength = 128
	}

	mask := net.CIDRMask(maskLength, 128)
	begin := ip.Mask(mask)
	end := incrementIP(begin, maskLength)

	return begin, end
}

// CountRegistrationsByIP returns the number of registrations created in the
// time range for a single IP address.
func (ssa *SQLStorageAuthorityRO) CountRegistrationsByIP(ctx context.Context, req *sapb.CountRegistrationsByIPRequest) (*sapb.Count, error) {
	if len(req.Ip) == 0 || req.Range.Earliest == 0 || req.Range.Latest == 0 {
		return nil, errIncompleteRequest
	}

	var count int64
	err := ssa.dbReadOnlyMap.WithContext(ctx).SelectOne(
		&count,
		`SELECT COUNT(*) FROM registrations
		 WHERE
		 initialIP = :ip AND
		 :earliest < createdAt AND
		 createdAt <= :latest`,
		map[string]interface{}{
			"ip":       req.Ip,
			"earliest": time.Unix(0, req.Range.Earliest),
			"latest":   time.Unix(0, req.Range.Latest),
		})
	if err != nil {
		return nil, err
	}
	return &sapb.Count{Count: count}, nil
}

func (ssa *SQLStorageAuthority) CountRegistrationsByIP(ctx context.Context, req *sapb.CountRegistrationsByIPRequest) (*sapb.Count, error) {
	return ssa.SQLStorageAuthorityRO.CountRegistrationsByIP(ctx, req)
}

// CountRegistrationsByIPRange returns the number of registrations created in
// the time range in an IP range. For IPv4 addresses, that range is limited to
// the single IP. For IPv6 addresses, that range is a /48, since it's not
// uncommon for one person to have a /48 to themselves.
func (ssa *SQLStorageAuthorityRO) CountRegistrationsByIPRange(ctx context.Context, req *sapb.CountRegistrationsByIPRequest) (*sapb.Count, error) {
	if len(req.Ip) == 0 || req.Range.Earliest == 0 || req.Range.Latest == 0 {
		return nil, errIncompleteRequest
	}

	var count int64
	beginIP, endIP := ipRange(req.Ip)
	err := ssa.dbReadOnlyMap.WithContext(ctx).SelectOne(
		&count,
		`SELECT COUNT(*) FROM registrations
		 WHERE
		 :beginIP <= initialIP AND
		 initialIP < :endIP AND
		 :earliest < createdAt AND
		 createdAt <= :latest`,
		map[string]interface{}{
			"earliest": time.Unix(0, req.Range.Earliest),
			"latest":   time.Unix(0, req.Range.Latest),
			"beginIP":  beginIP,
			"endIP":    endIP,
		})
	if err != nil {
		return nil, err
	}
	return &sapb.Count{Count: count}, nil
}

func (ssa *SQLStorageAuthority) CountRegistrationsByIPRange(ctx context.Context, req *sapb.CountRegistrationsByIPRequest) (*sapb.Count, error) {
	return ssa.SQLStorageAuthorityRO.CountRegistrationsByIPRange(ctx, req)
}

// CountCertificatesByNames counts, for each input domain, the number of
// certificates issued in the given time range for that domain and its
// subdomains. It returns a map from domains to counts and a timestamp. The map
// of domains to counts is guaranteed to contain an entry for each input domain,
// so long as err is nil. The timestamp is the earliest time a certificate was
// issued for any of the domains during the provided range of time. Queries will
// be run in parallel. If any of them error, only one error will be returned.
func (ssa *SQLStorageAuthorityRO) CountCertificatesByNames(ctx context.Context, req *sapb.CountCertificatesByNamesRequest) (*sapb.CountByNames, error) {
	if len(req.Names) == 0 || req.Range.Earliest == 0 || req.Range.Latest == 0 {
		return nil, errIncompleteRequest
	}

	work := make(chan string, len(req.Names))
	type result struct {
		err      error
		count    int64
		earliest time.Time
		domain   string
	}
	results := make(chan result, len(req.Names))
	for _, domain := range req.Names {
		work <- domain
	}
	close(work)
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// We may perform up to 100 queries, depending on what's in the certificate
	// request. Parallelize them so we don't hit our timeout, but limit the
	// parallelism so we don't consume too many threads on the database.
	for i := 0; i < ssa.parallelismPerRPC; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for domain := range work {
				select {
				case <-ctx.Done():
					results <- result{err: ctx.Err()}
					return
				default:
				}
				count, earliest, err := ssa.countCertificatesByName(ssa.dbReadOnlyMap.WithContext(ctx), domain, req.Range)
				if err != nil {
					results <- result{err: err}
					// Skip any further work
					cancel()
					return
				}
				results <- result{
					count:    count,
					earliest: earliest,
					domain:   domain,
				}
			}
		}()
	}
	wg.Wait()
	close(results)

	// Set earliest to the latest possible time, so that we can find the
	// earliest certificate in the results.
	earliest := timestamppb.New(time.Unix(0, req.Range.Latest))
	counts := make(map[string]int64)
	for r := range results {
		if r.err != nil {
			return nil, r.err
		}
		counts[r.domain] = r.count
		if !r.earliest.IsZero() && r.earliest.Before(earliest.AsTime()) {
			earliest = timestamppb.New(r.earliest)
		}
	}

	// If we didn't find any certificates in the range, earliest should be set
	// to a zero value.
	if len(counts) == 0 {
		earliest = &timestamppb.Timestamp{}
	}
	return &sapb.CountByNames{Counts: counts, Earliest: earliest}, nil
}

func (ssa *SQLStorageAuthority) CountCertificatesByNames(ctx context.Context, req *sapb.CountCertificatesByNamesRequest) (*sapb.CountByNames, error) {
	return ssa.SQLStorageAuthorityRO.CountCertificatesByNames(ctx, req)
}

func ReverseName(domain string) string {
	labels := strings.Split(domain, ".")
	for i, j := 0, len(labels)-1; i < j; i, j = i+1, j-1 {
		labels[i], labels[j] = labels[j], labels[i]
	}
	return strings.Join(labels, ".")
}

// GetCertificate takes a serial number and returns the corresponding
// certificate, or error if it does not exist.
func (ssa *SQLStorageAuthorityRO) GetCertificate(ctx context.Context, req *sapb.Serial) (*corepb.Certificate, error) {
	if req == nil || req.Serial == "" {
		return nil, errIncompleteRequest
	}
	if !core.ValidSerial(req.Serial) {
		return nil, fmt.Errorf("invalid certificate serial %s", req.Serial)
	}

	cert, err := SelectCertificate(ssa.dbReadOnlyMap.WithContext(ctx), req.Serial)
	if db.IsNoRows(err) {
		return nil, berrors.NotFoundError("certificate with serial %q not found", req.Serial)
	}
	if err != nil {
		return nil, err
	}
	return bgrpc.CertToPB(cert), nil
}

func (ssa *SQLStorageAuthority) GetCertificate(ctx context.Context, req *sapb.Serial) (*corepb.Certificate, error) {
	return ssa.SQLStorageAuthorityRO.GetCertificate(ctx, req)
}

// GetCertificateStatus takes a hexadecimal string representing the full 128-bit serial
// number of a certificate and returns data about that certificate's current
// validity.
func (ssa *SQLStorageAuthorityRO) GetCertificateStatus(ctx context.Context, req *sapb.Serial) (*corepb.CertificateStatus, error) {
	if req.Serial == "" {
		return nil, errIncompleteRequest
	}
	if !core.ValidSerial(req.Serial) {
		err := fmt.Errorf("invalid certificate serial %s", req.Serial)
		return nil, err
	}

	certStatus, err := SelectCertificateStatus(ssa.dbReadOnlyMap.WithContext(ctx), req.Serial)
	if db.IsNoRows(err) {
		return nil, berrors.NotFoundError("certificate status with serial %q not found", req.Serial)
	}
	if err != nil {
		return nil, err
	}

	return bgrpc.CertStatusToPB(certStatus), nil
}

func (ssa *SQLStorageAuthority) GetCertificateStatus(ctx context.Context, req *sapb.Serial) (*corepb.CertificateStatus, error) {
	return ssa.SQLStorageAuthorityRO.GetCertificateStatus(ctx, req)
}

// GetRevocationStatus takes a hexadecimal string representing the full serial
// number of a certificate and returns a minimal set of data about that cert's
// current validity.
func (ssa *SQLStorageAuthorityRO) GetRevocationStatus(ctx context.Context, req *sapb.Serial) (*sapb.RevocationStatus, error) {
	if req.Serial == "" {
		return nil, errIncompleteRequest
	}
	if !core.ValidSerial(req.Serial) {
		return nil, fmt.Errorf("invalid certificate serial %s", req.Serial)
	}

	status, err := SelectRevocationStatus(ssa.dbReadOnlyMap.WithContext(ctx), req.Serial)
	if err != nil {
		if db.IsNoRows(err) {
			return nil, berrors.NotFoundError("certificate status with serial %q not found", req.Serial)
		}
		return nil, err
	}

	return status, nil
}

func (ssa *SQLStorageAuthority) GetRevocationStatus(ctx context.Context, req *sapb.Serial) (*sapb.RevocationStatus, error) {
	return ssa.SQLStorageAuthorityRO.GetRevocationStatus(ctx, req)
}

func (ssa *SQLStorageAuthorityRO) CountOrders(ctx context.Context, req *sapb.CountOrdersRequest) (*sapb.Count, error) {
	if req.AccountID == 0 || req.Range.Earliest == 0 || req.Range.Latest == 0 {
		return nil, errIncompleteRequest
	}

	return countNewOrders(ssa.dbReadOnlyMap.WithContext(ctx), req)
}

func (ssa *SQLStorageAuthority) CountOrders(ctx context.Context, req *sapb.CountOrdersRequest) (*sapb.Count, error) {
	return ssa.SQLStorageAuthorityRO.CountOrders(ctx, req)
}

// CountFQDNSets counts the total number of issuances, for a set of domains,
// that occurred during a given window of time.
func (ssa *SQLStorageAuthorityRO) CountFQDNSets(ctx context.Context, req *sapb.CountFQDNSetsRequest) (*sapb.Count, error) {
	if req.Window == 0 || len(req.Domains) == 0 {
		return nil, errIncompleteRequest
	}

	var count int64
	err := ssa.dbReadOnlyMap.WithContext(ctx).SelectOne(
		&count,
		`SELECT COUNT(*) FROM fqdnSets
		WHERE setHash = ?
		AND issued > ?`,
		HashNames(req.Domains),
		ssa.clk.Now().Add(-time.Duration(req.Window)),
	)
	return &sapb.Count{Count: count}, err
}

func (ssa *SQLStorageAuthority) CountFQDNSets(ctx context.Context, req *sapb.CountFQDNSetsRequest) (*sapb.Count, error) {
	return ssa.SQLStorageAuthorityRO.CountFQDNSets(ctx, req)
}

// FQDNSetTimestampsForWindow returns the issuance timestamps for each
// certificate, issued for a set of domains, during a given window of time,
// starting from the most recent issuance.
func (ssa *SQLStorageAuthorityRO) FQDNSetTimestampsForWindow(ctx context.Context, req *sapb.CountFQDNSetsRequest) (*sapb.Timestamps, error) {
	if req.Window == 0 || len(req.Domains) == 0 {
		return nil, errIncompleteRequest
	}
	type row struct {
		Issued time.Time
	}
	var rows []row
	_, err := ssa.dbReadOnlyMap.WithContext(ctx).Select(
		&rows,
		`SELECT issued FROM fqdnSets 
		WHERE setHash = ?
		AND issued > ?
		ORDER BY issued DESC`,
		HashNames(req.Domains),
		ssa.clk.Now().Add(-time.Duration(req.Window)),
	)
	if err != nil {
		return nil, err
	}

	var results []int64
	for _, i := range rows {
		results = append(results, i.Issued.UnixNano())
	}
	return &sapb.Timestamps{Timestamps: results}, nil
}

func (ssa *SQLStorageAuthority) FQDNSetTimestampsForWindow(ctx context.Context, req *sapb.CountFQDNSetsRequest) (*sapb.Timestamps, error) {
	return ssa.SQLStorageAuthorityRO.FQDNSetTimestampsForWindow(ctx, req)
}

// FQDNSetExists returns a bool indicating if one or more FQDN sets |names|
// exists in the database
func (ssa *SQLStorageAuthorityRO) FQDNSetExists(ctx context.Context, req *sapb.FQDNSetExistsRequest) (*sapb.Exists, error) {
	if len(req.Domains) == 0 {
		return nil, errIncompleteRequest
	}
	exists, err := ssa.checkFQDNSetExists(ssa.dbReadOnlyMap.WithContext(ctx).SelectOne, req.Domains)
	if err != nil {
		return nil, err
	}
	return &sapb.Exists{Exists: exists}, nil
}

func (ssa *SQLStorageAuthority) FQDNSetExists(ctx context.Context, req *sapb.FQDNSetExistsRequest) (*sapb.Exists, error) {
	return ssa.SQLStorageAuthorityRO.FQDNSetExists(ctx, req)
}

// oneSelectorFunc is a func type that matches both gorp.Transaction.SelectOne
// and gorp.DbMap.SelectOne.
type oneSelectorFunc func(holder interface{}, query string, args ...interface{}) error

// checkFQDNSetExists uses the given oneSelectorFunc to check whether an fqdnSet
// for the given names exists.
func (ssa *SQLStorageAuthorityRO) checkFQDNSetExists(selector oneSelectorFunc, names []string) (bool, error) {
	namehash := HashNames(names)
	var exists bool
	err := selector(
		&exists,
		`SELECT EXISTS (SELECT id FROM fqdnSets WHERE setHash = ? LIMIT 1)`,
		namehash,
	)
	return exists, err
}

// PreviousCertificateExists returns true iff there was at least one certificate
// issued with the provided domain name, and the most recent such certificate
// was issued by the provided registration ID. This method is currently only
// used to determine if a certificate has previously been issued for a given
// domain name in order to determine if validations should be allowed during
// the v1 API shutoff.
// TODO(#5816): Consider removing this method, as it has no callers.
func (ssa *SQLStorageAuthorityRO) PreviousCertificateExists(ctx context.Context, req *sapb.PreviousCertificateExistsRequest) (*sapb.Exists, error) {
	if req.Domain == "" || req.RegID == 0 {
		return nil, errIncompleteRequest
	}

	exists := &sapb.Exists{Exists: true}
	notExists := &sapb.Exists{Exists: false}

	// Find the most recently issued certificate containing this domain name.
	var serial string
	err := ssa.dbReadOnlyMap.WithContext(ctx).SelectOne(
		&serial,
		`SELECT serial FROM issuedNames
		WHERE reversedName = ?
		ORDER BY notBefore DESC
		LIMIT 1`,
		ReverseName(req.Domain),
	)
	if err != nil {
		if db.IsNoRows(err) {
			return notExists, nil
		}
		return nil, err
	}

	// Check whether that certificate was issued to the specified account.
	var count int
	err = ssa.dbReadOnlyMap.WithContext(ctx).SelectOne(
		&count,
		`SELECT COUNT(*) FROM certificates
		WHERE serial = ?
		AND registrationID = ?`,
		serial,
		req.RegID,
	)
	if err != nil {
		// If no rows found, that means the certificate we found in issuedNames wasn't
		// issued by the registration ID we are checking right now, but is not an
		// error.
		if db.IsNoRows(err) {
			return notExists, nil
		}
		return nil, err
	}
	if count > 0 {
		return exists, nil
	}
	return notExists, nil
}

func (ssa *SQLStorageAuthority) PreviousCertificateExists(ctx context.Context, req *sapb.PreviousCertificateExistsRequest) (*sapb.Exists, error) {
	return ssa.SQLStorageAuthorityRO.PreviousCertificateExists(ctx, req)
}

// GetOrder is used to retrieve an already existing order object
func (ssa *SQLStorageAuthorityRO) GetOrder(ctx context.Context, req *sapb.OrderRequest) (*corepb.Order, error) {
	if req == nil || req.Id == 0 {
		return nil, errIncompleteRequest
	}

	output, err := db.WithTransaction(ctx, ssa.dbReadOnlyMap, func(txWithCtx db.Executor) (interface{}, error) {
		omObj, err := txWithCtx.Get(orderModel{}, req.Id)
		if err != nil {
			if db.IsNoRows(err) {
				return nil, berrors.NotFoundError("no order found for ID %d", req.Id)
			}
			return nil, err
		}
		if omObj == nil {
			return nil, berrors.NotFoundError("no order found for ID %d", req.Id)
		}

		order, err := modelToOrder(omObj.(*orderModel))
		if err != nil {
			return nil, err
		}

		orderExp := time.Unix(0, order.Expires)
		if orderExp.Before(ssa.clk.Now()) {
			return nil, berrors.NotFoundError("no order found for ID %d", req.Id)
		}

		v2AuthzIDs, err := authzForOrder(txWithCtx, order.Id)
		if err != nil {
			return nil, err
		}
		order.V2Authorizations = v2AuthzIDs

		names, err := namesForOrder(txWithCtx, order.Id)
		if err != nil {
			return nil, err
		}
		// The requested names are stored reversed to improve indexing performance. We
		// need to reverse the reversed names here before giving them back to the
		// caller.
		reversedNames := make([]string, len(names))
		for i, n := range names {
			reversedNames[i] = ReverseName(n)
		}
		order.Names = reversedNames

		// Calculate the status for the order
		status, err := statusForOrder(txWithCtx, order, ssa.clk.Now())
		if err != nil {
			return nil, err
		}
		order.Status = status

		return order, nil
	})
	if err != nil {
		return nil, err
	}

	order, ok := output.(*corepb.Order)
	if !ok {
		return nil, fmt.Errorf("casting error in GetOrder")
	}

	return order, nil
}

func (ssa *SQLStorageAuthority) GetOrder(ctx context.Context, req *sapb.OrderRequest) (*corepb.Order, error) {
	return ssa.SQLStorageAuthorityRO.GetOrder(ctx, req)
}

// GetOrderForNames tries to find a **pending** or **ready** order with the
// exact set of names requested, associated with the given accountID. Only
// unexpired orders are considered. If no order meeting these requirements is
// found a nil corepb.Order pointer is returned.
func (ssa *SQLStorageAuthorityRO) GetOrderForNames(ctx context.Context, req *sapb.GetOrderForNamesRequest) (*corepb.Order, error) {

	if req.AcctID == 0 || len(req.Names) == 0 {
		return nil, errIncompleteRequest
	}

	// Hash the names requested for lookup in the orderFqdnSets table
	fqdnHash := HashNames(req.Names)

	// Find a possibly-suitable order. We don't include the account ID or order
	// status in this query because there's no index that includes those, so
	// including them could require the DB to scan extra rows.
	// Instead, we select one unexpired order that matches the fqdnSet. If
	// that order doesn't match the account ID or status we need, just return
	// nothing. We use `ORDER BY expires ASC` because the index on
	// (setHash, expires) is in ASC order. DESC would be slightly nicer from a
	// user experience perspective but would be slow when there are many entries
	// to sort.
	// This approach works fine because in most cases there's only one account
	// issuing for a given name. If there are other accounts issuing for the same
	// name, it just means order reuse happens less often.
	var result struct {
		OrderID        int64
		RegistrationID int64
	}
	var err error
	err = ssa.dbReadOnlyMap.WithContext(ctx).SelectOne(&result, `
					SELECT orderID, registrationID
					FROM orderFqdnSets
					WHERE setHash = ?
					AND expires > ?
					ORDER BY expires ASC
					LIMIT 1`,
		fqdnHash, ssa.clk.Now())

	if db.IsNoRows(err) {
		return nil, berrors.NotFoundError("no order matching request found")
	} else if err != nil {
		return nil, err
	}

	if result.RegistrationID != req.AcctID {
		return nil, berrors.NotFoundError("no order matching request found")
	}

	// Get the order
	order, err := ssa.GetOrder(ctx, &sapb.OrderRequest{Id: result.OrderID})
	if err != nil {
		return nil, err
	}
	// Only return a pending or ready order
	if order.Status != string(core.StatusPending) &&
		order.Status != string(core.StatusReady) {
		return nil, berrors.NotFoundError("no order matching request found")
	}
	return order, nil
}

func (ssa *SQLStorageAuthority) GetOrderForNames(ctx context.Context, req *sapb.GetOrderForNamesRequest) (*corepb.Order, error) {
	return ssa.SQLStorageAuthorityRO.GetOrderForNames(ctx, req)
}

// GetAuthorization2 returns the authz2 style authorization identified by the provided ID or an error.
// If no authorization is found matching the ID a berrors.NotFound type error is returned.
func (ssa *SQLStorageAuthorityRO) GetAuthorization2(ctx context.Context, req *sapb.AuthorizationID2) (*corepb.Authorization, error) {
	if req.Id == 0 {
		return nil, errIncompleteRequest
	}
	obj, err := ssa.dbReadOnlyMap.Get(authzModel{}, req.Id)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, berrors.NotFoundError("authorization %d not found", req.Id)
	}
	return modelToAuthzPB(*(obj.(*authzModel)))
}

func (ssa *SQLStorageAuthority) GetAuthorization2(ctx context.Context, req *sapb.AuthorizationID2) (*corepb.Authorization, error) {
	return ssa.SQLStorageAuthorityRO.GetAuthorization2(ctx, req)
}

// authzModelMapToPB converts a mapping of domain name to authzModels into a
// protobuf authorizations map
func authzModelMapToPB(m map[string]authzModel) (*sapb.Authorizations, error) {
	resp := &sapb.Authorizations{}
	for k, v := range m {
		authzPB, err := modelToAuthzPB(v)
		if err != nil {
			return nil, err
		}
		resp.Authz = append(resp.Authz, &sapb.Authorizations_MapElement{Domain: k, Authz: authzPB})
	}
	return resp, nil
}

// GetAuthorizations2 returns any valid or pending authorizations that exist for the list of domains
// provided. If both a valid and pending authorization exist only the valid one will be returned.
func (ssa *SQLStorageAuthorityRO) GetAuthorizations2(ctx context.Context, req *sapb.GetAuthorizationsRequest) (*sapb.Authorizations, error) {
	if len(req.Domains) == 0 || req.RegistrationID == 0 || req.Now == 0 {
		return nil, errIncompleteRequest
	}
	var authzModels []authzModel
	params := []interface{}{
		req.RegistrationID,
		statusUint(core.StatusValid),
		statusUint(core.StatusPending),
		time.Unix(0, req.Now),
		identifierTypeToUint[string(identifier.DNS)],
	}

	for _, name := range req.Domains {
		params = append(params, name)
	}

	query := fmt.Sprintf(
		`SELECT %s FROM authz2
			USE INDEX (regID_identifier_status_expires_idx)
			WHERE registrationID = ? AND
			status IN (?,?) AND
			expires > ? AND
			identifierType = ? AND
			identifierValue IN (%s)`,
		authzFields,
		db.QuestionMarks(len(req.Domains)),
	)

	_, err := ssa.dbReadOnlyMap.Select(
		&authzModels,
		query,
		params...,
	)
	if err != nil {
		return nil, err
	}

	if len(authzModels) == 0 {
		return &sapb.Authorizations{}, nil
	}

	authzModelMap := make(map[string]authzModel)
	for _, am := range authzModels {
		existing, present := authzModelMap[am.IdentifierValue]
		if !present || uintToStatus[existing.Status] == core.StatusPending && uintToStatus[am.Status] == core.StatusValid {
			authzModelMap[am.IdentifierValue] = am
		}
	}

	return authzModelMapToPB(authzModelMap)
}

func (ssa *SQLStorageAuthority) GetAuthorizations2(ctx context.Context, req *sapb.GetAuthorizationsRequest) (*sapb.Authorizations, error) {
	return ssa.SQLStorageAuthorityRO.GetAuthorizations2(ctx, req)
}

// GetPendingAuthorization2 returns the most recent Pending authorization with
// the given identifier, if available. This method only supports DNS identifier types.
// TODO(#5816): Consider removing this method, as it has no callers.
func (ssa *SQLStorageAuthorityRO) GetPendingAuthorization2(ctx context.Context, req *sapb.GetPendingAuthorizationRequest) (*corepb.Authorization, error) {
	if req.RegistrationID == 0 || req.IdentifierValue == "" || req.ValidUntil == 0 {
		return nil, errIncompleteRequest
	}
	var am authzModel
	err := ssa.dbReadOnlyMap.WithContext(ctx).SelectOne(
		&am,
		fmt.Sprintf(`SELECT %s FROM authz2 WHERE
			registrationID = :regID AND
			status = :status AND
			expires > :validUntil AND
			identifierType = :dnsType AND
			identifierValue = :ident
			ORDER BY expires ASC
			LIMIT 1 `, authzFields),
		map[string]interface{}{
			"regID":      req.RegistrationID,
			"status":     statusUint(core.StatusPending),
			"validUntil": time.Unix(0, req.ValidUntil),
			"dnsType":    identifierTypeToUint[string(identifier.DNS)],
			"ident":      req.IdentifierValue,
		},
	)
	if err != nil {
		if db.IsNoRows(err) {
			return nil, berrors.NotFoundError("pending authz not found")
		}
		return nil, err
	}
	return modelToAuthzPB(am)
}

func (ssa *SQLStorageAuthority) GetPendingAuthorization2(ctx context.Context, req *sapb.GetPendingAuthorizationRequest) (*corepb.Authorization, error) {
	return ssa.SQLStorageAuthorityRO.GetPendingAuthorization2(ctx, req)
}

// CountPendingAuthorizations2 returns the number of pending, unexpired authorizations
// for the given registration.
func (ssa *SQLStorageAuthorityRO) CountPendingAuthorizations2(ctx context.Context, req *sapb.RegistrationID) (*sapb.Count, error) {
	if req.Id == 0 {
		return nil, errIncompleteRequest
	}

	var count int64
	err := ssa.dbReadOnlyMap.WithContext(ctx).SelectOne(&count,
		`SELECT COUNT(*) FROM authz2 WHERE
		registrationID = :regID AND
		expires > :expires AND
		status = :status`,
		map[string]interface{}{
			"regID":   req.Id,
			"expires": ssa.clk.Now(),
			"status":  statusUint(core.StatusPending),
		},
	)
	if err != nil {
		return nil, err
	}
	return &sapb.Count{Count: count}, nil
}

func (ssa *SQLStorageAuthority) CountPendingAuthorizations2(ctx context.Context, req *sapb.RegistrationID) (*sapb.Count, error) {
	return ssa.SQLStorageAuthorityRO.CountPendingAuthorizations2(ctx, req)
}

// GetValidOrderAuthorizations2 is used to find the valid, unexpired authorizations
// associated with a specific order and account ID.
func (ssa *SQLStorageAuthorityRO) GetValidOrderAuthorizations2(ctx context.Context, req *sapb.GetValidOrderAuthorizationsRequest) (*sapb.Authorizations, error) {
	if req.AcctID == 0 || req.Id == 0 {
		return nil, errIncompleteRequest
	}

	var ams []authzModel
	_, err := ssa.dbReadOnlyMap.WithContext(ctx).Select(
		&ams,
		fmt.Sprintf(`SELECT %s FROM authz2
			LEFT JOIN orderToAuthz2 ON authz2.ID = orderToAuthz2.authzID
			WHERE authz2.registrationID = :regID AND
			authz2.expires > :expires AND
			authz2.status = :status AND
			orderToAuthz2.orderID = :orderID`,
			authzFields,
		),
		map[string]interface{}{
			"regID":   req.AcctID,
			"expires": ssa.clk.Now(),
			"status":  statusUint(core.StatusValid),
			"orderID": req.Id,
		},
	)
	if err != nil {
		return nil, err
	}

	byName := make(map[string]authzModel)
	for _, am := range ams {
		if uintToIdentifierType[am.IdentifierType] != string(identifier.DNS) {
			return nil, fmt.Errorf("unknown identifier type: %q on authz id %d", am.IdentifierType, am.ID)
		}
		existing, present := byName[am.IdentifierValue]
		if !present || am.Expires.After(existing.Expires) {
			byName[am.IdentifierValue] = am
		}
	}

	return authzModelMapToPB(byName)
}

func (ssa *SQLStorageAuthority) GetValidOrderAuthorizations2(ctx context.Context, req *sapb.GetValidOrderAuthorizationsRequest) (*sapb.Authorizations, error) {
	return ssa.SQLStorageAuthorityRO.GetValidOrderAuthorizations2(ctx, req)
}

// CountInvalidAuthorizations2 counts invalid authorizations for a user expiring
// in a given time range. This method only supports DNS identifier types.
func (ssa *SQLStorageAuthorityRO) CountInvalidAuthorizations2(ctx context.Context, req *sapb.CountInvalidAuthorizationsRequest) (*sapb.Count, error) {
	if req.RegistrationID == 0 || req.Hostname == "" || req.Range.Earliest == 0 || req.Range.Latest == 0 {
		return nil, errIncompleteRequest
	}

	var count int64
	err := ssa.dbReadOnlyMap.WithContext(ctx).SelectOne(
		&count,
		`SELECT COUNT(*) FROM authz2 WHERE
		registrationID = :regID AND
		status = :status AND
		expires > :expiresEarliest AND
		expires <= :expiresLatest AND
		identifierType = :dnsType AND
		identifierValue = :ident`,
		map[string]interface{}{
			"regID":           req.RegistrationID,
			"dnsType":         identifierTypeToUint[string(identifier.DNS)],
			"ident":           req.Hostname,
			"expiresEarliest": time.Unix(0, req.Range.Earliest),
			"expiresLatest":   time.Unix(0, req.Range.Latest),
			"status":          statusUint(core.StatusInvalid),
		},
	)
	if err != nil {
		return nil, err
	}
	return &sapb.Count{Count: count}, nil
}

func (ssa *SQLStorageAuthority) CountInvalidAuthorizations2(ctx context.Context, req *sapb.CountInvalidAuthorizationsRequest) (*sapb.Count, error) {
	return ssa.SQLStorageAuthorityRO.CountInvalidAuthorizations2(ctx, req)
}

// GetValidAuthorizations2 returns the latest authorization for all
// domain names that the account has authorizations for. This method
// only supports DNS identifier types.
func (ssa *SQLStorageAuthorityRO) GetValidAuthorizations2(ctx context.Context, req *sapb.GetValidAuthorizationsRequest) (*sapb.Authorizations, error) {
	if len(req.Domains) == 0 || req.RegistrationID == 0 || req.Now == 0 {
		return nil, errIncompleteRequest
	}

	query := fmt.Sprintf(
		`SELECT %s FROM authz2 WHERE
			registrationID = ? AND
			status = ? AND
			expires > ? AND
			identifierType = ? AND
			identifierValue IN (%s)`,
		authzFields,
		db.QuestionMarks(len(req.Domains)),
	)

	params := []interface{}{
		req.RegistrationID,
		statusUint(core.StatusValid),
		time.Unix(0, req.Now),
		identifierTypeToUint[string(identifier.DNS)],
	}
	for _, domain := range req.Domains {
		params = append(params, domain)
	}

	var authzModels []authzModel
	_, err := ssa.dbReadOnlyMap.Select(
		&authzModels,
		query,
		params...,
	)
	if err != nil {
		return nil, err
	}

	authzMap := make(map[string]authzModel, len(authzModels))
	for _, am := range authzModels {
		// Only allow DNS identifiers
		if uintToIdentifierType[am.IdentifierType] != string(identifier.DNS) {
			continue
		}
		// If there is an existing authorization in the map only replace it with one
		// which has a later expiry.
		if existing, present := authzMap[am.IdentifierValue]; present && am.Expires.Before(existing.Expires) {
			continue
		}
		authzMap[am.IdentifierValue] = am
	}
	return authzModelMapToPB(authzMap)
}

func (ssa *SQLStorageAuthority) GetValidAuthorizations2(ctx context.Context, req *sapb.GetValidAuthorizationsRequest) (*sapb.Authorizations, error) {
	return ssa.SQLStorageAuthorityRO.GetValidAuthorizations2(ctx, req)
}

// KeyBlocked checks if a key, indicated by a hash, is present in the blockedKeys table
func (ssa *SQLStorageAuthorityRO) KeyBlocked(ctx context.Context, req *sapb.KeyBlockedRequest) (*sapb.Exists, error) {
	if req == nil || req.KeyHash == nil {
		return nil, errIncompleteRequest
	}

	var id int64
	err := ssa.dbReadOnlyMap.SelectOne(&id, `SELECT ID FROM blockedKeys WHERE keyHash = ?`, req.KeyHash)
	if err != nil {
		if db.IsNoRows(err) {
			return &sapb.Exists{Exists: false}, nil
		}
		return nil, err
	}

	return &sapb.Exists{Exists: true}, nil
}

func (ssa *SQLStorageAuthority) KeyBlocked(ctx context.Context, req *sapb.KeyBlockedRequest) (*sapb.Exists, error) {
	return ssa.SQLStorageAuthorityRO.KeyBlocked(ctx, req)
}

// IncidentsForSerial queries each active incident table and returns every
// incident that currently impacts `req.Serial`.
func (ssa *SQLStorageAuthorityRO) IncidentsForSerial(ctx context.Context, req *sapb.Serial) (*sapb.Incidents, error) {
	if req == nil {
		return nil, errIncompleteRequest
	}

	var activeIncidents []incidentModel
	_, err := ssa.dbReadOnlyMap.Select(&activeIncidents, `SELECT * FROM incidents WHERE enabled = 1`)
	if err != nil {
		if db.IsNoRows(err) {
			return &sapb.Incidents{}, nil
		}
		return nil, err
	}

	var incidentsForSerial []*sapb.Incident
	for _, i := range activeIncidents {
		var count int
		err := ssa.dbIncidentsMap.SelectOne(&count, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE serial = ?",
			i.SerialTable), req.Serial)
		if err != nil {
			if db.IsNoRows(err) {
				continue
			}
			return nil, err
		}
		if count > 0 {
			incident := incidentModelToPB(i)
			incidentsForSerial = append(incidentsForSerial, &incident)
		}

	}
	if len(incidentsForSerial) == 0 {
		return &sapb.Incidents{}, nil
	}
	return &sapb.Incidents{Incidents: incidentsForSerial}, nil
}

func (ssa *SQLStorageAuthority) IncidentsForSerial(ctx context.Context, req *sapb.Serial) (*sapb.Incidents, error) {
	return ssa.SQLStorageAuthorityRO.IncidentsForSerial(ctx, req)
}

// SerialsForIncident queries the provided incident table and returns the
// resulting rows as a stream of `*sapb.IncidentSerial`s. An `io.EOF` error
// signals that there are no more serials to send. If the incident table in
// question contains zero rows, only an `io.EOF` error is returned.
func (ssa *SQLStorageAuthorityRO) SerialsForIncident(req *sapb.SerialsForIncidentRequest, stream sapb.StorageAuthorityReadOnly_SerialsForIncidentServer) error {
	if req.IncidentTable == "" {
		return errIncompleteRequest
	}

	// Check that `req.IncidentTable` is a valid incident table name.
	if !validIncidentTableRegexp.MatchString(req.IncidentTable) {
		return fmt.Errorf("malformed table name %q", req.IncidentTable)
	}

	selector, err := db.NewMappedSelector[incidentSerialModel](ssa.dbIncidentsMap)
	if err != nil {
		return fmt.Errorf("initializing db map: %w", err)
	}

	rows, err := selector.QueryFrom(stream.Context(), req.IncidentTable, "")
	if err != nil {
		return fmt.Errorf("starting db query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		// Scan the row into the model. Note: the fields must be passed in the
		// same order as the columns returned by the query above.
		ism, err := rows.Get()
		if err != nil {
			return err
		}

		err = stream.Send(
			&sapb.IncidentSerial{
				Serial:         ism.Serial,
				RegistrationID: ism.RegistrationID,
				OrderID:        ism.OrderID,
				LastNoticeSent: ism.LastNoticeSent.UnixNano(),
			})
		if err != nil {
			return err
		}
	}

	err = rows.Err()
	if err != nil {
		return err
	}
	return nil
}

func (ssa *SQLStorageAuthority) SerialsForIncident(req *sapb.SerialsForIncidentRequest, stream sapb.StorageAuthority_SerialsForIncidentServer) error {
	return ssa.SQLStorageAuthorityRO.SerialsForIncident(req, stream)
}

// GetRevokedCerts gets a request specifying an issuer and a period of time,
// and writes to the output stream the set of all certificates issued by that
// issuer which expire during that period of time and which have been revoked.
// The starting timestamp is treated as inclusive (certs with exactly that
// notAfter date are included), but the ending timestamp is exclusive (certs
// with exactly that notAfter date are *not* included).
func (ssa *SQLStorageAuthorityRO) GetRevokedCerts(req *sapb.GetRevokedCertsRequest, stream sapb.StorageAuthorityReadOnly_GetRevokedCertsServer) error {
	atTime := time.Unix(0, req.RevokedBefore)

	clauses := `
		WHERE notAfter >= ?
		AND notAfter < ?
		AND issuerID = ?
		AND status = ?`
	params := []interface{}{
		time.Unix(0, req.ExpiresAfter),
		time.Unix(0, req.ExpiresBefore),
		req.IssuerNameID,
		core.OCSPStatusRevoked,
	}

	selector, err := db.NewMappedSelector[crlEntryModel](ssa.dbReadOnlyMap)
	if err != nil {
		return fmt.Errorf("initializing db map: %w", err)
	}

	rows, err := selector.Query(stream.Context(), clauses, params...)
	if err != nil {
		return fmt.Errorf("reading db: %w", err)
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			ssa.log.AuditErrf("closing row reader: %s", err)
		}
	}()

	for rows.Next() {
		row, err := rows.Get()
		if err != nil {
			return fmt.Errorf("reading row: %w", err)
		}

		// Double-check that the cert wasn't revoked between the time at which we're
		// constructing this snapshot CRL and right now. If the cert was revoked
		// at-or-after the "atTime", we'll just include it in the next generation
		// of CRLs.
		if row.RevokedDate.After(atTime) || row.RevokedDate.Equal(atTime) {
			continue
		}

		err = stream.Send(&corepb.CRLEntry{
			Serial:    row.Serial,
			Reason:    int32(row.RevokedReason),
			RevokedAt: row.RevokedDate.UnixNano(),
		})
		if err != nil {
			return fmt.Errorf("sending crl entry: %w", err)
		}
	}

	err = rows.Err()
	if err != nil {
		return fmt.Errorf("iterating over row reader: %w", err)
	}

	return nil
}

func (ssa *SQLStorageAuthority) GetRevokedCerts(req *sapb.GetRevokedCertsRequest, stream sapb.StorageAuthority_GetRevokedCertsServer) error {
	return ssa.SQLStorageAuthorityRO.GetRevokedCerts(req, stream)
}

// GetMaxExpiration returns the timestamp of the farthest-future notAfter date
// found in the certificateStatus table. This provides an upper bound on how far
// forward operations that need to cover all currently-unexpired certificates
// have to look.
func (ssa *SQLStorageAuthorityRO) GetMaxExpiration(ctx context.Context, req *emptypb.Empty) (*timestamppb.Timestamp, error) {
	var model struct {
		MaxNotAfter time.Time `db:"maxNotAfter"`
	}
	err := ssa.dbReadOnlyMap.WithContext(ctx).SelectOne(
		&model,
		"SELECT MAX(notAfter) AS maxNotAfter FROM certificateStatus",
	)
	if err != nil {
		return nil, fmt.Errorf("selecting max notAfter: %w", err)
	}
	return timestamppb.New(model.MaxNotAfter), err
}

func (ssa *SQLStorageAuthority) GetMaxExpiration(ctx context.Context, req *emptypb.Empty) (*timestamppb.Timestamp, error) {
	return ssa.SQLStorageAuthorityRO.GetMaxExpiration(ctx, req)
}
