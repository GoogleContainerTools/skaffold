// Package http implements the HTTP transport protocol.
package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
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

	req, err := http.NewRequest(http.MethodGet, url, nil)
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
	if r.URL.Scheme != "http" && r.URL.Scheme != "https" {
		return fmt.Errorf("http redirect: unsupported scheme %q", r.URL.Scheme)
	}
	if r.URL.Scheme != s.endpoint.Protocol &&
		!(s.endpoint.Protocol == "http" && r.URL.Scheme == "https") {
		return fmt.Errorf("http redirect: changes scheme from %q to %q", s.endpoint.Protocol, r.URL.Scheme)
	}

	host := endpointHost(r.URL.Hostname())
	port, err := endpointPort(r.URL.Port())
	if err != nil {
		return err
	}

	if host != s.endpoint.Host || effectivePort(r.URL.Scheme, port) != effectivePort(s.endpoint.Protocol, s.endpoint.Port) {
		s.endpoint.User = ""
		s.endpoint.Password = ""
		s.auth = nil
	}

	s.endpoint.Host = host
	s.endpoint.Port = port

	s.endpoint.Protocol = r.URL.Scheme
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

func effectivePort(scheme string, port int) int {
	if port != 0 {
		return port
	}

	switch strings.ToLower(scheme) {
	case "http":
		return 80
	case "https":
		return 443
	default:
		return 0
	}
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
		if next != nil {
			return next(req, via)
		}
		return nil
	}
}

func checkRedirect(req *http.Request, via []*http.Request, policy RedirectPolicy) error {
	switch policy {
	case FollowRedirects:
	case NoFollowRedirects:
		return fmt.Errorf("http redirect: redirects disabled to %s", req.URL)
	case "", FollowInitialRedirects:
		if !isInitialRequest(req) {
			return fmt.Errorf("http redirect: redirect on non-initial request to %s", req.URL)
		}
	default:
		return fmt.Errorf("http redirect: invalid redirect policy %q", policy)
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
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
		e.Response.Request.URL, e.Response.StatusCode,
	)
}
