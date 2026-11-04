// Package http implements the HTTP transport protocol.
package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/groupcache/lru"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/capability"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/utils/ioutil"
)

type contextKey int

const initialRequestKey contextKey = iota

// RedirectPolicy controls how the HTTP transport follows redirects.
//
// The values mirror Git's http.followRedirects config:
// "true" follows redirects for all requests, "false" treats redirects as
// errors, and "initial" follows redirects only for the initial
// /info/refs discovery request. The zero value defaults to "initial".
type RedirectPolicy string

const (
	FollowInitialRedirects RedirectPolicy = "initial"
	FollowRedirects        RedirectPolicy = "true"
	NoFollowRedirects      RedirectPolicy = "false"
)

func withInitialRequest(ctx context.Context) context.Context {
	return context.WithValue(ctx, initialRequestKey, true)
}

func isInitialRequest(req *http.Request) bool {
	v, _ := req.Context().Value(initialRequestKey).(bool)
	return v
}

// it requires a bytes.Buffer, because we need to know the length
func applyHeadersToRequest(req *http.Request, content *bytes.Buffer, host string, requestType string) {
	req.Header.Add("User-Agent", capability.DefaultAgent())
	req.Header.Add("Host", host) // host:port

	if content == nil {
		req.Header.Add("Accept", "*/*")
		return
	}

	req.Header.Add("Accept", fmt.Sprintf("application/x-%s-result", requestType))
	req.Header.Add("Content-Type", fmt.Sprintf("application/x-%s-request", requestType))
	req.Header.Add("Content-Length", strconv.Itoa(content.Len()))
}

const infoRefsPath = "/info/refs"

func advertisedReferences(ctx context.Context, s *session, serviceName string) (ref *packp.AdvRefs, err error) {
	url := fmt.Sprintf(
		"%s%s?service=%s",
		s.endpoint.String(), infoRefsPath, serviceName,
	)

	req, err := newRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	s.ApplyAuthToRequest(req)
	applyHeadersToRequest(req, nil, s.endpoint.Host, serviceName)
	res, err := s.client.Do(req.WithContext(withInitialRequest(ctx)))
	if err != nil {
		return nil, err
	}

	if err := s.ModifyEndpointIfRedirect(res); err != nil {
		_ = res.Body.Close()
		return nil, err
	}
	defer ioutil.CheckClose(res.Body, &err)

	if err = NewErr(res); err != nil {
		return nil, err
	}

	ar := packp.NewAdvRefs()
	if err = ar.Decode(res.Body); err != nil {
		if err == packp.ErrEmptyAdvRefs {
			err = transport.ErrEmptyRemoteRepository
		}

		return nil, err
	}

	// Git 2.41+ returns a zero-id plus capabilities when an empty
	// repository is being cloned. This skips the existing logic within
	// advrefs_decode.decodeFirstHash, which expects a flush-pkt instead.
	//
	// This logic aligns with plumbing/transport/internal/common/common.go.
	if ar.IsEmpty() &&
		// Empty repositories are valid for git-receive-pack.
		transport.ReceivePackServiceName != serviceName {
		return nil, transport.ErrEmptyRemoteRepository
	}

	transport.FilterUnsupportedCapabilities(ar.Capabilities)
	s.advRefs = ar

	return ar, nil
}

type client struct {
	client     *http.Client
	transports *lru.Cache
	mutex      sync.RWMutex
	follow     RedirectPolicy
}

// ClientOptions holds user configurable options for the client.
type ClientOptions struct {
	// CacheMaxEntries is the max no. of entries that the transport objects
	// cache will hold at any given point of time. It must be a positive integer.
	// Calling `client.addTransport()` after the cache has reached the specified
	// size, will result in the least recently used transport getting deleted
	// before the provided transport is added to the cache.
	CacheMaxEntries int

	// RedirectPolicy controls redirect handling. Supported values are
	// "true", "false", and "initial". The zero value defaults to
	// "initial", matching Git's http.followRedirects default.
	RedirectPolicy RedirectPolicy
}

var (
	// defaultTransportCacheSize is the default capacity of the transport objects cache.
	// Its value is 0 because transport caching is turned off by default and is an
	// opt-in feature.
	defaultTransportCacheSize = 0

	// DefaultClient is the default HTTP client, which uses a net/http client configured
	// with http.DefaultTransport.
	DefaultClient = NewClient(nil)
)

// NewClient creates a new client with a custom net/http client.
// See `InstallProtocol` to install and override default http client.
// If the net/http client is nil or empty, it will use a net/http client configured
// with http.DefaultTransport.
//
// Note that for HTTP client cannot distinguish between private repositories and
// unexistent repositories on GitHub. So it returns `ErrAuthorizationRequired`
// for both.
func NewClient(c *http.Client) transport.Transport {
	if c == nil {
		c = &http.Client{
			Transport: http.DefaultTransport,
		}
	}
	return NewClientWithOptions(c, &ClientOptions{
		CacheMaxEntries: defaultTransportCacheSize,
	})
}

// NewClientWithOptions returns a new client configured with the provided net/http client
// and other custom options specific to the client.
// If the net/http client is nil or empty, it will use a net/http client configured
// with http.DefaultTransport.
//
// Credentials this client adds where the transport cannot see them are not
// subject to the redirect stripping described on AuthMethod: a RoundTripper
// injects after the hop is decided, and Client.Jar is consulted after
// CheckRedirect, so a domain cookie still follows a redirect to a subdomain
// the transport counts as another origin. Apply them in an AuthMethod instead
// if that is not wanted. A CheckRedirect hook set on this client runs
// alongside the transport's own, but any header it adds when a redirect
// leaves the repository's origin is discarded the same way.
//
// A RoundTripper is therefore also how to authenticate to a new origin a
// redirect has moved the repository to: match on the request URL and inject
// the credential only for that origin, so it is not sent anywhere else. The
// transport keeps its own CheckRedirect on the copy it makes of this client,
// so the policy and the origin checks still apply.
//
// None of this applies to a RoundTripper that follows redirects itself:
// CheckRedirect is not consulted then, so no stripping happens at all.
func NewClientWithOptions(c *http.Client, opts *ClientOptions) transport.Transport {
	if c == nil {
		c = &http.Client{
			Transport: http.DefaultTransport,
		}
	}
	cl := &client{
		client: c,
		follow: FollowInitialRedirects,
	}

	if opts != nil {
		if opts.CacheMaxEntries > 0 {
			cl.transports = lru.New(opts.CacheMaxEntries)
		}
		if opts.RedirectPolicy != "" {
			cl.follow = opts.RedirectPolicy
		}
	}
	return cl
}

func (c *client) NewUploadPackSession(ep *transport.Endpoint, auth transport.AuthMethod) (
	transport.UploadPackSession, error) {

	return newUploadPackSession(c, ep, auth)
}

func (c *client) NewReceivePackSession(ep *transport.Endpoint, auth transport.AuthMethod) (
	transport.ReceivePackSession, error) {

	return newReceivePackSession(c, ep, auth)
}

type session struct {
	auth     AuthMethod
	client   *http.Client
	endpoint *transport.Endpoint
	advRefs  *packp.AdvRefs
}

func transportWithInsecureTLS(transport *http.Transport) {
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	transport.TLSClientConfig.InsecureSkipVerify = true
}

func transportWithClientCert(transport *http.Transport, cert, key []byte) error {
	keyPair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return err
	}
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	transport.TLSClientConfig.Certificates = []tls.Certificate{keyPair}
	return nil
}

func transportWithCABundle(transport *http.Transport, caBundle []byte) error {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return err
	}
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	rootCAs.AppendCertsFromPEM(caBundle)
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	transport.TLSClientConfig.RootCAs = rootCAs
	return nil
}

func transportWithProxy(transport *http.Transport, proxyURL *url.URL) {
	transport.Proxy = http.ProxyURL(proxyURL)
}

func configureTransport(transport *http.Transport, ep *transport.Endpoint) error {
	if len(ep.ClientCert) > 0 && len(ep.ClientKey) > 0 {
		if err := transportWithClientCert(transport, ep.ClientCert, ep.ClientKey); err != nil {
			return err
		}
	}
	if len(ep.CaBundle) > 0 {
		if err := transportWithCABundle(transport, ep.CaBundle); err != nil {
			return err
		}
	}
	if ep.InsecureSkipTLS {
		transportWithInsecureTLS(transport)
	}

	if ep.Proxy.URL != "" {
		proxyURL, err := ep.Proxy.FullURL()
		if err != nil {
			return err
		}
		transportWithProxy(transport, proxyURL)
	}
	return nil
}

func newSession(c *client, ep *transport.Endpoint, auth transport.AuthMethod) (*session, error) {
	var httpClient *http.Client

	// We need to configure the http transport if there are transport specific
	// options present in the endpoint.
	if len(ep.ClientKey) > 0 || len(ep.ClientCert) > 0 || len(ep.CaBundle) > 0 || ep.InsecureSkipTLS || ep.Proxy.URL != "" {
		var transport *http.Transport
		// if the client wasn't configured to have a cache for transports then just configure
		// the transport and use it directly, otherwise try to use the cache.
		if c.transports == nil {
			tr, ok := c.client.Transport.(*http.Transport)
			if !ok {
				return nil, fmt.Errorf("expected underlying client transport to be of type: %s; got: %s",
					reflect.TypeOf(transport), reflect.TypeOf(c.client.Transport))
			}

			transport = tr.Clone()
			if err := configureTransport(transport, ep); err != nil {
				return nil, err
			}
		} else {
			transportOpts := transportOptions{
				clientCert:      string(ep.ClientCert),
				clientKey:       string(ep.ClientKey),
				caBundle:        string(ep.CaBundle),
				insecureSkipTLS: ep.InsecureSkipTLS,
			}
			if ep.Proxy.URL != "" {
				proxyURL, err := ep.Proxy.FullURL()
				if err != nil {
					return nil, err
				}
				transportOpts.proxyURL = *proxyURL
			}
			var found bool
			transport, found = c.fetchTransport(transportOpts)

			if !found {
				transport = c.client.Transport.(*http.Transport).Clone()
				if err := configureTransport(transport, ep); err != nil {
					return nil, err
				}
				c.addTransport(transportOpts, transport)
			}
		}

		httpClient = c.cloneHTTPClient(transport)
	} else {
		httpClient = c.cloneHTTPClient(c.client.Transport)
	}

	s := &session{
		auth:     basicAuthFromEndpoint(ep),
		client:   httpClient,
		endpoint: ep,
	}
	if auth != nil {
		a, ok := auth.(AuthMethod)
		if !ok {
			return nil, transport.ErrInvalidAuthMethod
		}

		s.auth = a
	}

	return s, nil
}

func (s *session) ApplyAuthToRequest(req *http.Request) {
	if s.auth == nil {
		return
	}

	s.auth.SetAuth(req)
}

func (s *session) ModifyEndpointIfRedirect(res *http.Response) error {
	if res.Request == nil {
		return nil
	}
	if s.endpoint == nil {
		return fmt.Errorf("http redirect: nil endpoint")
	}

	r := res.Request
	if !strings.HasSuffix(r.URL.Path, infoRefsPath) {
		return fmt.Errorf("http redirect: target %q does not end with %s", r.URL.Path, infoRefsPath)
	}
	// A scheme is case-insensitive per RFC 3986, and checkRedirect folds case
	// when it reads the same hop, so fold here too rather than reject a
	// spelling that check let through. url.Parse and transport.NewEndpoint
	// both lowercase what they parse, so only a hand-built URL or Endpoint
	// arrives uppercased. The folded form is what gets stored below, so every
	// later request built from the endpoint carries the canonical spelling.
	scheme := strings.ToLower(r.URL.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("http redirect: unsupported scheme %q", r.URL.Scheme)
	}
	// schemeUpgrade rather than an inline comparison, so the one cross-scheme
	// change go-git permits has a single definition shared with
	// credentialsMayFollow.
	if !strings.EqualFold(scheme, s.endpoint.Protocol) && !schemeUpgrade(s.endpoint.Protocol, scheme) {
		return fmt.Errorf("http redirect: changes scheme from %q to %q", s.endpoint.Protocol, r.URL.Scheme)
	}

	host := endpointHost(r.URL.Hostname())
	port, err := endpointPort(r.URL.Port())
	if err != nil {
		return err
	}

	// The session stores the endpoint and re-applies its credentials on every
	// later request, so clear them once the redirect has left the origin they
	// were issued for. This uses the same predicate as stripCredentials, so
	// both halves share one definition of an origin.
	//
	// The two are deliberately asymmetric in one respect: stripCredentials is
	// sticky over the whole chain, so an origin -> evil -> origin redirect
	// leaves the discovery GET's later hops unauthenticated even though the
	// chain returned home. This compares the endpoint only against the final
	// URL, so the same round trip leaves the session authenticated. That is
	// not a leak - the final URL's origin is the original one - but it means
	// such a chain can make the discovery GET anonymous while the session's
	// POSTs are authenticated, which can surface as a confusing 401.
	if !credentialsMayFollow(endpointURL(s.endpoint), r.URL) {
		s.endpoint.User = ""
		s.endpoint.Password = ""
		s.auth = nil
	}

	s.endpoint.Host = host
	s.endpoint.Port = port

	s.endpoint.Protocol = scheme
	s.endpoint.Path = r.URL.Path[:len(r.URL.Path)-len(infoRefsPath)]
	return nil
}

func endpointHost(host string) string {
	if strings.Contains(host, ":") {
		return "[" + host + "]"
	}

	return host
}

func endpointPort(port string) (int, error) {
	if port == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(port)
	if err != nil {
		return 0, fmt.Errorf("http redirect: invalid port %q", port)
	}

	return parsed, nil
}

// schemeUpgrade reports whether the scheme transition from one URL to another
// is the one cross-scheme change go-git permits: a plain-http origin upgrading
// to https. It strictly improves confidentiality and is how servers steer
// clients off cleartext.
//
// Permitting it at all is a deliberate deviation: curl, git and the Fetch
// standard all count scheme as part of host identity and drop credentials on
// the upgrade. Auth is sent pre-emptively here, so an http origin has already
// spent its credential in cleartext on the first request and refusing the
// upgrade would break the clone without unspending it. The host is unchanged,
// where an on-path attacker needs a valid certificate to receive anything.
func schemeUpgrade(from, to string) bool {
	return strings.EqualFold(from, "http") && strings.EqualFold(to, "https")
}

// canonicalHost returns u's hostname in the form origins are compared in.
//
// An address literal is normalised by netip, so the many spellings of one
// address are one origin. Two literals are the same origin exactly when netip
// parses them to the same Addr, which is also how the WHATWG URL Standard
// compares hosts. An IPv4-mapped literal is deliberately not unmapped onto
// the IPv4 it dials: reaching the same endpoint is not the same authority,
// since net/http sends the literal as written in Host and a server may route
// the two spellings to different virtual hosts.
//
// netip also keeps a scope zone verbatim, which is what origin comparison
// needs: net resolves a zone to an interface by exact name, so folding %eth0
// onto %ETH0 would call two hosts the same origin that net dials down
// different interfaces.
//
// A registered name is ASCII-lowercased. That fold is the only liberty taken;
// every other difference in spelling is a different origin.
//
// A trailing root dot is one such difference and is kept, for the same reason
// as the IPv4-mapped literal: curl and the WHATWG URL Standard both hold
// "example.com." and "example.com" to be distinct hosts, and although
// crypto/tls and crypto/x509 fold the dot when they authenticate the peer,
// net/http sends the name as written in Host.
//
// The fold is deliberately ASCII-only. strings.ToLower and strings.EqualFold
// apply Unicode case mapping, which folds U+03C2 onto U+03C3 and so would
// call two hosts the same origin when they resolve to different servers. An
// ASCII-only fold cannot merge two names DNS keeps apart.
//
// No IDNA mapping is applied either, so a unicode hostname is a different
// origin from the punycode encoding of it, and from another Unicode case of
// itself, even though all three reach the same server. Mapping through
// golang.org/x/net/idna would join them, but it can only widen this equality,
// never narrow it, so leaving it out can cost a credential across such a
// redirect and cannot forward one. Against that cost, go-git pins x/net while
// net/http uses the copy vendored into the toolchain: the two are versioned
// separately, so a release that moves the Unicode tables under one and not
// the other would have this merge origins net/http still dials apart. That is
// the failure this comparison exists to prevent, and comparing bytes has no
// such mode.
func canonicalHost(u *url.URL) string {
	host := u.Hostname()
	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.String()
	}
	b := []byte(host)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

// effectivePort returns u's port as the connection will use it: the scheme's
// well-known port when the URL does not spell one out, and without leading
// zeroes, so "https://x", "https://x:443" and "https://x:0443" all agree.
func effectivePort(u *url.URL) string {
	port := u.Port()
	if port == "" {
		switch strings.ToLower(u.Scheme) {
		case "http":
			return "80"
		case "https":
			return "443"
		default:
			return ""
		}
	}
	if trimmed := strings.TrimLeft(port, "0"); trimmed != "" {
		return trimmed
	}
	return "0"
}

// credentialsMayFollow reports whether credentials issued for one URL may be
// sent to another.
//
// The relation is deliberately asymmetric: scheme, host and effective port
// must all match, except that a plain http origin may upgrade to https on the
// same host (see schemeUpgrade). That exception is confined to the two
// default ports: 80 to 443 is the upgrade servers actually steer clients
// through, whereas a non-default port carries no such convention, so
// http://host:8080 to https://host:8443 is a move to another origin like any
// other port change.
//
// Host matching is exact. Unlike Go's http.Client, which forwards credentials
// from a host to any subdomain of it, a subdomain is a different origin here —
// matching canonical git and libcurl.
func credentialsMayFollow(from, to *url.URL) bool {
	if canonicalHost(from) != canonicalHost(to) {
		return false
	}
	if strings.EqualFold(from.Scheme, to.Scheme) {
		return effectivePort(from) == effectivePort(to)
	}
	return schemeUpgrade(from.Scheme, to.Scheme) &&
		effectivePort(from) == "80" && effectivePort(to) == "443"
}

// endpointURL renders an Endpoint's origin as a URL, so that the session's
// credential clearing and the per-hop stripping share one definition of an
// origin and cannot drift apart.
func endpointURL(ep *transport.Endpoint) *url.URL {
	host := strings.Trim(ep.Host, "[]")
	if ep.Port != 0 {
		host = net.JoinHostPort(host, strconv.Itoa(ep.Port))
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return &url.URL{Scheme: ep.Protocol, Host: host}
}

func (c *client) cloneHTTPClient(transport http.RoundTripper) *http.Client {
	return &http.Client{
		Transport:     transport,
		CheckRedirect: wrapCheckRedirect(c.follow, c.client.CheckRedirect),
		Jar:           c.client.Jar,
		Timeout:       c.client.Timeout,
	}
}

func wrapCheckRedirect(policy RedirectPolicy, next func(*http.Request, []*http.Request) error) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if err := checkRedirect(req, via, policy); err != nil {
			return err
		}
		// Strip before the caller's hook so it observes what will actually
		// be sent, and again afterwards so a hook of the common "preserve
		// my headers across redirects" shape - which copies from via[0],
		// the original unsanitized request - cannot reinstate them.
		// Carrying credentials across an origin boundary is deliberately
		// unsupported.
		stripCredentials(req, via)
		if next != nil {
			if err := next(req, via); err != nil {
				return err
			}
		}
		stripCredentials(req, via)
		return nil
	}
}

// safeHeaders lists the headers go-git sets itself, none of which can carry a
// caller credential. stripCredentials keeps only these when a redirect leaves
// the credential's origin. Adding a name here makes it forwardable across an
// origin boundary - do not add anything a caller can put a secret in.
//
// This narrows rather than eliminates the exposure: an AuthMethod that writes
// a credential into one of these names directly - for example
// Header.Set("User-Agent", "token "+secret) - still survives a cross-origin
// redirect. Such a value is also sent to the origin and to any proxy in path,
// so it should not be placed there whether or not a redirect follows.
var safeHeaders = map[string]struct{}{
	"User-Agent":     {},
	"Host":           {},
	"Accept":         {},
	"Content-Type":   {},
	"Content-Length": {},
}

func filterHeaders(h http.Header) http.Header {
	filtered := make(http.Header)
	for key, values := range h {
		if _, ok := safeHeaders[http.CanonicalHeaderKey(key)]; ok {
			filtered[key] = values
		}
	}
	return filtered
}

// stripCredentials removes credentials from req once the redirect chain has
// left the origin of the original, credential-bearing request.
//
// CheckRedirect is the only hook that runs while a redirected request's
// headers are still mutable: http.Client.Do performs the entire chain
// internally, so anything the transport does after Do returns - including
// ModifyEndpointIfRedirect - is too late for the hops themselves.
//
// Two subtleties:
//
//   - net/http rebuilds every redirect request from the original request's
//     headers before calling this, so a header removed at one hop reappears
//     at the next. The decision is therefore recomputed per hop.
//   - The decision is sticky: once the chain has left the origin, credentials
//     stay gone even if a later hop returns to it. Stickiness is derived from
//     via rather than stored, because this closure is shared across a
//     session's requests.
//
// Stripping keeps only the headers go-git sets itself (safeHeaders). An
// allowlist is used rather than a list of credential header names because
// caller credentials arrive under names that cannot be enumerated -
// PRIVATE-TOKEN, X-Api-Key, gateway headers - which is exactly what
// net/http's fixed list of sensitive header names gets wrong. It is also
// immune to header-name canonicalisation: an AuthMethod that writes a raw map
// key is still removed.
func stripCredentials(req *http.Request, via []*http.Request) {
	if len(via) == 0 {
		return
	}
	// net/http sets a URL on every request it builds, and req.URL is non-nil
	// by construction: checkRedirect dereferences req.URL.Scheme on each path
	// that returns nil, so it runs first or not at all. This nil check and
	// the two in crossedOrigin are defensive, against a synthetic caller.
	// Each treats a URL it cannot read as an origin crossing; removing one
	// panics in canonicalHost rather than leaking.
	if origin := via[0].URL; origin != nil && !crossedOrigin(origin, req, via) {
		return
	}
	req.Header = filterHeaders(req.Header)
	if req.URL != nil {
		req.URL.User = nil
	}
}

// crossedOrigin reports whether any hop so far, including the pending one, has
// left origin.
func crossedOrigin(origin *url.URL, req *http.Request, via []*http.Request) bool {
	if req.URL == nil || !credentialsMayFollow(origin, req.URL) {
		return true
	}
	for _, prev := range via[1:] {
		if prev.URL == nil || !credentialsMayFollow(origin, prev.URL) {
			return true
		}
	}
	return false
}

// redactedURL returns the string form of u with the userinfo password
// replaced, for use in error messages. (*url.URL).String() renders the
// password verbatim, and request URLs are built from the endpoint, which
// carries whatever credentials the caller put in the clone URL.
func redactedURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	if u.User == nil {
		return u.String()
	}
	if _, hasPassword := u.User.Password(); !hasPassword {
		return u.String()
	}
	redacted := *u
	redacted.User = url.UserPassword(u.User.Username(), "REDACTED")
	return redacted.String()
}

// redactedRawURL is redactedURL for a string that may not parse. Request URLs
// are assembled from Endpoint.String(), which re-emits Endpoint.Path raw, so a
// path holding a stray percent produces a string url.Parse rejects. url.Parse
// reports the input verbatim and applies no redaction of its own; only
// http.Client strips a password, and only from errors it raises itself.
func redactedRawURL(raw string) string {
	i := strings.Index(raw, "://")
	if i < 0 {
		return raw
	}
	authority := raw[i+3:]
	if end := strings.IndexByte(authority, '/'); end >= 0 {
		authority = authority[:end]
	}
	at := strings.LastIndexByte(authority, '@')
	if at < 0 {
		return raw
	}
	colon := strings.IndexByte(authority[:at], ':')
	if colon < 0 {
		// Username only, left alone, as redactedURL leaves it.
		return raw
	}
	return raw[:i+3+colon+1] + "REDACTED" + raw[i+3+at:]
}

// newRequest wraps http.NewRequest so that a URL it cannot parse does not
// reach the caller with the endpoint's credentials still in it.
func newRequest(method, rawURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		var uerr *url.Error
		if errors.As(err, &uerr) {
			uerr.URL = redactedRawURL(uerr.URL)
		}
		return nil, err
	}
	return req, nil
}

func checkRedirect(req *http.Request, via []*http.Request, policy RedirectPolicy) error {
	// CheckRedirect is the only hook that runs before the next hop leaves
	// the client. ModifyEndpointIfRedirect inspects the chain after
	// client.Do has followed all of it, so a hop rejected there has already
	// carried the request headers to its server.
	//
	// The wording matches the message ModifyEndpointIfRedirect produces for
	// the same hop, which this check reaches first.
	if len(via) != 0 {
		// A hop whose scheme cannot be read cannot be shown not to have been
		// https, so it is assumed to have been, and a cleartext target is
		// rejected. Skipping the comparison instead would let an
		// undeterminable hop turn the check off, which is the wrong default
		// for a credential control; crossedOrigin fails closed the same way.
		// An empty scheme is as unreadable as a nil URL, so both take the
		// assumed-https default.
		//
		// The comparisons fold case because a scheme is case-insensitive per
		// RFC 3986. net/url lowercases what it parses, so only a hand-built
		// URL reaches here uppercased - the same synthetic caller the nil
		// checks guard against - and for that caller "HTTPS" to "http" is
		// still a downgrade.
		prevScheme := "https"
		if prev := via[len(via)-1]; prev.URL != nil && prev.URL.Scheme != "" {
			prevScheme = prev.URL.Scheme
		}
		if strings.EqualFold(prevScheme, "https") && strings.EqualFold(req.URL.Scheme, "http") {
			return fmt.Errorf("http redirect: changes scheme from %q to %q: %s",
				prevScheme, req.URL.Scheme, redactedURL(req.URL))
		}
	}

	switch policy {
	case FollowRedirects:
	case NoFollowRedirects:
		return fmt.Errorf("http redirect: redirects disabled to %s", redactedURL(req.URL))
	case "", FollowInitialRedirects:
		if !isInitialRequest(req) {
			return fmt.Errorf("http redirect: redirect on non-initial request to %s", redactedURL(req.URL))
		}
	default:
		return fmt.Errorf("http redirect: invalid redirect policy %q", policy)
	}
	// Folded for the same reason as the guard above: a scheme is
	// case-insensitive per RFC 3986, so a spelling the downgrade check read
	// as https must not be rejected here as a scheme go-git cannot speak.
	if !strings.EqualFold(req.URL.Scheme, "http") && !strings.EqualFold(req.URL.Scheme, "https") {
		return fmt.Errorf("http redirect: unsupported scheme %q", req.URL.Scheme)
	}
	if len(via) >= 10 {
		return fmt.Errorf("http redirect: too many redirects")
	}
	return nil
}

func (*session) Close() error {
	return nil
}

// AuthMethod is concrete implementation of common.AuthMethod for HTTP services
//
// Headers SetAuth adds are dropped when a redirect leaves the repository's
// origin: only the headers the transport sets itself survive that boundary.
// This applies to non-credential headers too, so an implementation that adds a
// trace or tenant header loses it on such a hop.
//
// That filter matches header names, not values. An implementation that writes
// a credential into a name the transport also uses - User-Agent, Host, Accept,
// Content-Type, Content-Length - has that value carried across the boundary
// with the name. Such a credential is also sent to the origin and to any proxy
// in path, so it should not be placed there whether or not a redirect follows.
type AuthMethod interface {
	transport.AuthMethod
	SetAuth(r *http.Request)
}

func basicAuthFromEndpoint(ep *transport.Endpoint) *BasicAuth {
	u := ep.User
	if u == "" {
		return nil
	}

	return &BasicAuth{u, ep.Password}
}

// BasicAuth represent a HTTP basic auth
type BasicAuth struct {
	Username, Password string
}

func (a *BasicAuth) SetAuth(r *http.Request) {
	if a == nil {
		return
	}

	r.SetBasicAuth(a.Username, a.Password)
}

// Name is name of the auth
func (a *BasicAuth) Name() string {
	return "http-basic-auth"
}

func (a *BasicAuth) String() string {
	masked := "*******"
	if a.Password == "" {
		masked = "<empty>"
	}

	return fmt.Sprintf("%s - %s:%s", a.Name(), a.Username, masked)
}

// TokenAuth implements an http.AuthMethod that can be used with http transport
// to authenticate with HTTP token authentication (also known as bearer
// authentication).
//
// IMPORTANT: If you are looking to use OAuth tokens with popular servers (e.g.
// GitHub, Bitbucket, GitLab) you should use BasicAuth instead. These servers
// use basic HTTP authentication, with the OAuth token as user or password.
// Check the documentation of your git server for details.
type TokenAuth struct {
	Token string
}

func (a *TokenAuth) SetAuth(r *http.Request) {
	if a == nil {
		return
	}
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.Token))
}

// Name is name of the auth
func (a *TokenAuth) Name() string {
	return "http-token-auth"
}

func (a *TokenAuth) String() string {
	masked := "*******"
	if a.Token == "" {
		masked = "<empty>"
	}
	return fmt.Sprintf("%s - %s", a.Name(), masked)
}

// Err is a dedicated error to return errors based on status code
type Err struct {
	Response *http.Response
	Reason   string
}

// NewErr returns a new Err based on a http response and closes response body
// if needed
func NewErr(r *http.Response) error {
	if r.StatusCode >= http.StatusOK && r.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	var reason string

	// If a response message is present, add it to error
	var messageBuffer bytes.Buffer
	if r.Body != nil {
		messageLength, _ := messageBuffer.ReadFrom(r.Body)
		if messageLength > 0 {
			reason = messageBuffer.String()
		}
		_ = r.Body.Close()
	}

	switch r.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", transport.ErrAuthenticationRequired, reason)
	case http.StatusForbidden:
		return fmt.Errorf("%w: %s", transport.ErrAuthorizationFailed, reason)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", transport.ErrRepositoryNotFound, reason)
	}

	return plumbing.NewUnexpectedError(&Err{r, reason})
}

// StatusCode returns the status code of the response
func (e *Err) StatusCode() int {
	return e.Response.StatusCode
}

func (e *Err) Error() string {
	return fmt.Sprintf("unexpected requesting %q status code: %d",
		redactedURL(e.Response.Request.URL), e.Response.StatusCode,
	)
}
