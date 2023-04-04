package bdns

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jmhodges/clock"
	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"

	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
)

func parseCidr(network string, comment string) net.IPNet {
	_, net, err := net.ParseCIDR(network)
	if err != nil {
		panic(fmt.Sprintf("error parsing %s (%s): %s", network, comment, err))
	}
	return *net
}

var (
	// Private CIDRs to ignore
	privateNetworks = []net.IPNet{
		// RFC1918
		// 10.0.0.0/8
		{
			IP:   []byte{10, 0, 0, 0},
			Mask: []byte{255, 0, 0, 0},
		},
		// 172.16.0.0/12
		{
			IP:   []byte{172, 16, 0, 0},
			Mask: []byte{255, 240, 0, 0},
		},
		// 192.168.0.0/16
		{
			IP:   []byte{192, 168, 0, 0},
			Mask: []byte{255, 255, 0, 0},
		},
		// RFC5735
		// 127.0.0.0/8
		{
			IP:   []byte{127, 0, 0, 0},
			Mask: []byte{255, 0, 0, 0},
		},
		// RFC1122 Section 3.2.1.3
		// 0.0.0.0/8
		{
			IP:   []byte{0, 0, 0, 0},
			Mask: []byte{255, 0, 0, 0},
		},
		// RFC3927
		// 169.254.0.0/16
		{
			IP:   []byte{169, 254, 0, 0},
			Mask: []byte{255, 255, 0, 0},
		},
		// RFC 5736
		// 192.0.0.0/24
		{
			IP:   []byte{192, 0, 0, 0},
			Mask: []byte{255, 255, 255, 0},
		},
		// RFC 5737
		// 192.0.2.0/24
		{
			IP:   []byte{192, 0, 2, 0},
			Mask: []byte{255, 255, 255, 0},
		},
		// 198.51.100.0/24
		{
			IP:   []byte{198, 51, 100, 0},
			Mask: []byte{255, 255, 255, 0},
		},
		// 203.0.113.0/24
		{
			IP:   []byte{203, 0, 113, 0},
			Mask: []byte{255, 255, 255, 0},
		},
		// RFC 3068
		// 192.88.99.0/24
		{
			IP:   []byte{192, 88, 99, 0},
			Mask: []byte{255, 255, 255, 0},
		},
		// RFC 2544, Errata 423
		// 198.18.0.0/15
		{
			IP:   []byte{198, 18, 0, 0},
			Mask: []byte{255, 254, 0, 0},
		},
		// RFC 3171
		// 224.0.0.0/4
		{
			IP:   []byte{224, 0, 0, 0},
			Mask: []byte{240, 0, 0, 0},
		},
		// RFC 1112
		// 240.0.0.0/4
		{
			IP:   []byte{240, 0, 0, 0},
			Mask: []byte{240, 0, 0, 0},
		},
		// RFC 919 Section 7
		// 255.255.255.255/32
		{
			IP:   []byte{255, 255, 255, 255},
			Mask: []byte{255, 255, 255, 255},
		},
		// RFC 6598
		// 100.64.0.0/10
		{
			IP:   []byte{100, 64, 0, 0},
			Mask: []byte{255, 192, 0, 0},
		},
	}
	// Sourced from https://www.iana.org/assignments/iana-ipv6-special-registry/iana-ipv6-special-registry.xhtml
	// where Global, Source, or Destination is False
	privateV6Networks = []net.IPNet{
		parseCidr("::/128", "RFC 4291: Unspecified Address"),
		parseCidr("::1/128", "RFC 4291: Loopback Address"),
		parseCidr("::ffff:0:0/96", "RFC 4291: IPv4-mapped Address"),
		parseCidr("100::/64", "RFC 6666: Discard Address Block"),
		parseCidr("2001::/23", "RFC 2928: IETF Protocol Assignments"),
		parseCidr("2001:2::/48", "RFC 5180: Benchmarking"),
		parseCidr("2001:db8::/32", "RFC 3849: Documentation"),
		parseCidr("2001::/32", "RFC 4380: TEREDO"),
		parseCidr("fc00::/7", "RFC 4193: Unique-Local"),
		parseCidr("fe80::/10", "RFC 4291: Section 2.5.6 Link-Scoped Unicast"),
		parseCidr("ff00::/8", "RFC 4291: Section 2.7"),
		// We disable validations to IPs under the 6to4 anycase prefix because
		// there's too much risk of a malicious actor advertising the prefix and
		// answering validations for a 6to4 host they do not control.
		// https://community.letsencrypt.org/t/problems-validating-ipv6-against-host-running-6to4/18312/9
		parseCidr("2002::/16", "RFC 7526: 6to4 anycast prefix deprecated"),
	}
)

// Client queries for DNS records
type Client interface {
	LookupTXT(context.Context, string) (txts []string, err error)
	LookupHost(context.Context, string) ([]net.IP, error)
	LookupCAA(context.Context, string) ([]*dns.CAA, string, error)
}

// impl represents a client that talks to an external resolver
type impl struct {
	dnsClient                exchanger
	servers                  ServerProvider
	allowRestrictedAddresses bool
	maxTries                 int
	clk                      clock.Clock
	log                      blog.Logger

	queryTime         *prometheus.HistogramVec
	totalLookupTime   *prometheus.HistogramVec
	timeoutCounter    *prometheus.CounterVec
	idMismatchCounter *prometheus.CounterVec
}

var _ Client = &impl{}

type exchanger interface {
	Exchange(m *dns.Msg, a string) (*dns.Msg, time.Duration, error)
}

// New constructs a new DNS resolver object that utilizes the
// provided list of DNS servers for resolution.
func New(
	readTimeout time.Duration,
	servers ServerProvider,
	stats prometheus.Registerer,
	clk clock.Clock,
	maxTries int,
	log blog.Logger,
) Client {
	dnsClient := new(dns.Client)

	// Set timeout for underlying net.Conn
	dnsClient.ReadTimeout = readTimeout
	dnsClient.Net = "udp"

	queryTime := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dns_query_time",
			Help:    "Time taken to perform a DNS query",
			Buckets: metrics.InternetFacingBuckets,
		},
		[]string{"qtype", "result", "authenticated_data", "resolver"},
	)
	totalLookupTime := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dns_total_lookup_time",
			Help:    "Time taken to perform a DNS lookup, including all retried queries",
			Buckets: metrics.InternetFacingBuckets,
		},
		[]string{"qtype", "result", "authenticated_data", "retries", "resolver"},
	)
	timeoutCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dns_timeout",
			Help: "Counter of various types of DNS query timeouts",
		},
		[]string{"qtype", "type", "resolver", "isTLD"},
	)
	idMismatchCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dns_id_mismatch",
			Help: "Counter of DNS ErrId errors sliced by query type and resolver",
		},
		[]string{"qtype", "resolver"},
	)
	stats.MustRegister(queryTime, totalLookupTime, timeoutCounter, idMismatchCounter)

	return &impl{
		dnsClient:                dnsClient,
		servers:                  servers,
		allowRestrictedAddresses: false,
		maxTries:                 maxTries,
		clk:                      clk,
		queryTime:                queryTime,
		totalLookupTime:          totalLookupTime,
		timeoutCounter:           timeoutCounter,
		idMismatchCounter:        idMismatchCounter,
		log:                      log,
	}
}

// NewTest constructs a new DNS resolver object that utilizes the
// provided list of DNS servers for resolution and will allow loopback addresses.
// This constructor should *only* be called from tests (unit or integration).
func NewTest(
	readTimeout time.Duration,
	servers ServerProvider,
	stats prometheus.Registerer,
	clk clock.Clock,
	maxTries int,
	log blog.Logger) Client {
	resolver := New(readTimeout, servers, stats, clk, maxTries, log)
	resolver.(*impl).allowRestrictedAddresses = true
	return resolver
}

// exchangeOne performs a single DNS exchange with a randomly chosen server
// out of the server list, returning the response, time, and error (if any).
// We assume that the upstream resolver requests and validates DNSSEC records
// itself.
func (dnsClient *impl) exchangeOne(ctx context.Context, hostname string, qtype uint16) (resp *dns.Msg, err error) {
	m := new(dns.Msg)
	// Set question type
	m.SetQuestion(dns.Fqdn(hostname), qtype)
	// Set the AD bit in the query header so that the resolver knows that
	// we are interested in this bit in the response header. If this isn't
	// set the AD bit in the response is useless (RFC 6840 Section 5.7).
	// This has no security implications, it simply allows us to gather
	// metrics about the percentage of responses that are secured with
	// DNSSEC.
	m.AuthenticatedData = true
	// Tell the resolver that we're willing to receive responses up to 4096 bytes.
	// This happens sometimes when there are a very large number of CAA records
	// present.
	m.SetEdns0(4096, false)

	servers, err := dnsClient.servers.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to list DNS servers: %w", err)
	}
	chosenServerIndex := 0
	chosenServer := servers[chosenServerIndex]

	start := dnsClient.clk.Now()
	client := dnsClient.dnsClient
	qtypeStr := dns.TypeToString[qtype]
	tries := 1
	defer func() {
		result, authenticated := "failed", ""
		if resp != nil {
			result = dns.RcodeToString[resp.Rcode]
			authenticated = fmt.Sprintf("%t", resp.AuthenticatedData)
		}
		dnsClient.totalLookupTime.With(prometheus.Labels{
			"qtype":              qtypeStr,
			"result":             result,
			"authenticated_data": authenticated,
			"retries":            strconv.Itoa(tries),
			"resolver":           chosenServer,
		}).Observe(dnsClient.clk.Since(start).Seconds())
	}()
	for {
		ch := make(chan dnsResp, 1)

		go func() {
			rsp, rtt, err := client.Exchange(m, chosenServer)
			result, authenticated := "failed", ""
			if rsp != nil {
				result = dns.RcodeToString[rsp.Rcode]
				authenticated = fmt.Sprintf("%t", rsp.AuthenticatedData)
			}
			if err != nil {
				logDNSError(dnsClient.log, chosenServer, hostname, m, rsp, err)
				if err == dns.ErrId {
					dnsClient.idMismatchCounter.With(prometheus.Labels{
						"qtype":    qtypeStr,
						"resolver": chosenServer,
					}).Inc()
				}
			}
			dnsClient.queryTime.With(prometheus.Labels{
				"qtype":              qtypeStr,
				"result":             result,
				"authenticated_data": authenticated,
				"resolver":           chosenServer,
			}).Observe(rtt.Seconds())
			ch <- dnsResp{m: rsp, err: err}
		}()
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				dnsClient.timeoutCounter.With(prometheus.Labels{
					"qtype":    qtypeStr,
					"type":     "deadline exceeded",
					"resolver": chosenServer,
					"isTLD":    isTLD(hostname),
				}).Inc()
			} else if ctx.Err() == context.Canceled {
				dnsClient.timeoutCounter.With(prometheus.Labels{
					"qtype":    qtypeStr,
					"type":     "canceled",
					"resolver": chosenServer,
					"isTLD":    isTLD(hostname),
				}).Inc()
			} else {
				dnsClient.timeoutCounter.With(prometheus.Labels{
					"qtype":    qtypeStr,
					"type":     "unknown",
					"resolver": chosenServer,
				}).Inc()
			}
			err = ctx.Err()
			return
		case r := <-ch:
			if r.err != nil {
				var operr *net.OpError
				ok := errors.As(r.err, &operr)
				isRetryable := ok && operr.Temporary()
				hasRetriesLeft := tries < dnsClient.maxTries
				if isRetryable && hasRetriesLeft {
					tries++
					// Chose a new server to retry the query with by incrementing the
					// chosen server index modulo the number of servers. This ensures that
					// if one dns server isn't available we retry with the next in the
					// list.
					chosenServerIndex = (chosenServerIndex + 1) % len(servers)
					chosenServer = servers[chosenServerIndex]
					continue
				} else if isRetryable && !hasRetriesLeft {
					dnsClient.timeoutCounter.With(prometheus.Labels{
						"qtype":    qtypeStr,
						"type":     "out of retries",
						"resolver": chosenServer,
						"isTLD":    isTLD(hostname),
					}).Inc()
				}
			}
			resp, err = r.m, r.err
			return
		}
	}

}

// isTLD returns a simplified view of whether something is a TLD: does it have
// any dots in it? This returns true or false as a string, and is meant solely
// for Prometheus metrics.
func isTLD(hostname string) string {
	if strings.Contains(hostname, ".") {
		return "false"
	} else {
		return "true"
	}
}

type dnsResp struct {
	m   *dns.Msg
	err error
}

// LookupTXT sends a DNS query to find all TXT records associated with
// the provided hostname which it returns along with the returned
// DNS authority section.
func (dnsClient *impl) LookupTXT(ctx context.Context, hostname string) ([]string, error) {
	var txt []string
	dnsType := dns.TypeTXT
	r, err := dnsClient.exchangeOne(ctx, hostname, dnsType)
	if err != nil {
		return nil, &Error{dnsType, hostname, err, -1}
	}
	if r.Rcode != dns.RcodeSuccess {
		return nil, &Error{dnsType, hostname, nil, r.Rcode}
	}

	for _, answer := range r.Answer {
		if answer.Header().Rrtype == dnsType {
			if txtRec, ok := answer.(*dns.TXT); ok {
				txt = append(txt, strings.Join(txtRec.Txt, ""))
			}
		}
	}

	return txt, err
}

func isPrivateV4(ip net.IP) bool {
	for _, net := range privateNetworks {
		if net.Contains(ip) {
			return true
		}
	}
	return false
}

func isPrivateV6(ip net.IP) bool {
	for _, net := range privateV6Networks {
		if net.Contains(ip) {
			return true
		}
	}
	return false
}

func (dnsClient *impl) lookupIP(ctx context.Context, hostname string, ipType uint16) ([]dns.RR, error) {
	resp, err := dnsClient.exchangeOne(ctx, hostname, ipType)
	if err != nil {
		return nil, &Error{ipType, hostname, err, -1}
	}
	if resp.Rcode != dns.RcodeSuccess {
		return nil, &Error{ipType, hostname, nil, resp.Rcode}
	}
	return resp.Answer, nil
}

// LookupHost sends a DNS query to find all A and AAAA records associated with
// the provided hostname. This method assumes that the external resolver will
// chase CNAME/DNAME aliases and return relevant records. It will retry
// requests in the case of temporary network errors. It returns an error if
// both the A and AAAA lookups fail or are empty, but succeeds otherwise.
func (dnsClient *impl) LookupHost(ctx context.Context, hostname string) ([]net.IP, error) {
	var recordsA, recordsAAAA []dns.RR
	var errA, errAAAA error
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		recordsA, errA = dnsClient.lookupIP(ctx, hostname, dns.TypeA)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		recordsAAAA, errAAAA = dnsClient.lookupIP(ctx, hostname, dns.TypeAAAA)
	}()
	wg.Wait()

	var addrsA []net.IP
	if errA == nil {
		for _, answer := range recordsA {
			if answer.Header().Rrtype == dns.TypeA {
				a, ok := answer.(*dns.A)
				if ok && a.A.To4() != nil && (!isPrivateV4(a.A) || dnsClient.allowRestrictedAddresses) {
					addrsA = append(addrsA, a.A)
				}
			}
		}
		if len(addrsA) == 0 {
			errA = fmt.Errorf("no valid A records found for %s", hostname)
		}
	}

	var addrsAAAA []net.IP
	if errAAAA == nil {
		for _, answer := range recordsAAAA {
			if answer.Header().Rrtype == dns.TypeAAAA {
				aaaa, ok := answer.(*dns.AAAA)
				if ok && aaaa.AAAA.To16() != nil && (!isPrivateV6(aaaa.AAAA) || dnsClient.allowRestrictedAddresses) {
					addrsAAAA = append(addrsAAAA, aaaa.AAAA)
				}
			}
		}
		if len(addrsAAAA) == 0 {
			errAAAA = fmt.Errorf("no valid AAAA records found for %s", hostname)
		}
	}

	if errA != nil && errAAAA != nil {
		// Construct a new error from both underlying errors. We can only use %w for
		// one of them, because the go error unwrapping protocol doesn't support
		// branching. We don't use ProblemDetails and SubProblemDetails here, because
		// this error will get wrapped in a DNSError and further munged by higher
		// layers in the stack.
		return nil, fmt.Errorf("%w; %s", errA, errAAAA)
	}

	return append(addrsA, addrsAAAA...), nil
}

// LookupCAA sends a DNS query to find all CAA records associated with
// the provided hostname and the complete dig-style RR `response`. This
// response is quite verbose, however it's only populated when the CAA
// response is non-empty.
func (dnsClient *impl) LookupCAA(ctx context.Context, hostname string) ([]*dns.CAA, string, error) {
	dnsType := dns.TypeCAA
	r, err := dnsClient.exchangeOne(ctx, hostname, dnsType)
	if err != nil {
		return nil, "", &Error{dnsType, hostname, err, -1}
	}

	if r.Rcode == dns.RcodeServerFailure {
		return nil, "", &Error{dnsType, hostname, nil, r.Rcode}
	}

	var CAAs []*dns.CAA
	for _, answer := range r.Answer {
		if caaR, ok := answer.(*dns.CAA); ok {
			CAAs = append(CAAs, caaR)
		}
	}
	var response string
	if len(CAAs) > 0 {
		response = r.String()
	}
	return CAAs, response, nil
}

// logDNSError logs the provided err result from making a query for hostname to
// the chosenServer. If the err is a `dns.ErrId` instance then the Base64
// encoded bytes of the query (and if not-nil, the response) in wire format
// is logged as well. This function is called from exchangeOne only for the case
// where an error occurs querying a hostname that indicates a problem between
// the VA and the chosenServer.
func logDNSError(
	logger blog.Logger,
	chosenServer string,
	hostname string,
	msg, resp *dns.Msg,
	underlying error) {
	// We don't expect logDNSError to be called with a nil msg or err but
	// if it happens return early. We allow resp to be nil.
	if msg == nil || len(msg.Question) == 0 || underlying == nil {
		return
	}
	queryType := dns.TypeToString[msg.Question[0].Qtype]

	// If the error indicates there was a query/response ID mismatch then we want
	// to log more detail.
	if underlying == dns.ErrId {
		packedMsgBytes, err := msg.Pack()
		if err != nil {
			logger.Errf("logDNSError failed to pack msg: %v", err)
			return
		}
		encodedMsg := base64.StdEncoding.EncodeToString(packedMsgBytes)

		var encodedResp string
		var respQname string
		if resp != nil {
			packedRespBytes, err := resp.Pack()
			if err != nil {
				logger.Errf("logDNSError failed to pack resp: %v", err)
				return
			}
			encodedResp = base64.StdEncoding.EncodeToString(packedRespBytes)
			if len(resp.Answer) > 0 && resp.Answer[0].Header() != nil {
				respQname = resp.Answer[0].Header().Name
			}
		}

		logger.Infof(
			"logDNSError ID mismatch chosenServer=[%s] hostname=[%s] respHostname=[%s] queryType=[%s] msg=[%s] resp=[%s] err=[%s]",
			chosenServer,
			hostname,
			respQname,
			queryType,
			encodedMsg,
			encodedResp,
			underlying)
	} else {
		// Otherwise log a general DNS error
		logger.Infof("logDNSError chosenServer=[%s] hostname=[%s] queryType=[%s] err=[%s]",
			chosenServer,
			hostname,
			queryType,
			underlying)
	}
}
