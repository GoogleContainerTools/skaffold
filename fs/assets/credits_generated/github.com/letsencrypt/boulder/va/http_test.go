package va

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	mrand "math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/letsencrypt/boulder/bdns"
	"github.com/letsencrypt/boulder/core"
	berrors "github.com/letsencrypt/boulder/errors"
	"github.com/letsencrypt/boulder/identifier"
	"github.com/letsencrypt/boulder/probs"
	"github.com/letsencrypt/boulder/test"
	"github.com/miekg/dns"

	"testing"
)

func httpChallenge() core.Challenge {
	return createChallenge(core.ChallengeTypeHTTP01)
}

// TestDialerMismatchError tests that using a preresolvedDialer for one host for
// a dial to another host produces the expected dialerMismatchError.
func TestDialerMismatchError(t *testing.T) {
	d := preresolvedDialer{
		ip:       net.ParseIP("127.0.0.1"),
		port:     1337,
		hostname: "letsencrypt.org",
	}

	expectedErr := dialerMismatchError{
		dialerHost: d.hostname,
		dialerIP:   d.ip.String(),
		dialerPort: d.port,
		host:       "lettuceencrypt.org",
	}

	_, err := d.DialContext(
		context.Background(),
		"tincan-and-string",
		"lettuceencrypt.org:80")
	test.AssertEquals(t, err.Error(), expectedErr.Error())
}

// TestPreresolvedDialerTimeout tests that the preresolvedDialer's DialContext
// will timeout after the expected singleDialTimeout. This ensures timeouts at
// the TCP level are handled correctly.
func TestPreresolvedDialerTimeout(t *testing.T) {
	va, _ := setup(nil, 0, "", nil)
	// Timeouts below 50ms tend to be flaky.
	va.singleDialTimeout = 50 * time.Millisecond

	// The context timeout needs to be larger than the singleDialTimeout
	ctxTimeout := 500 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	started := time.Now()

	va.dnsClient = dnsMockReturnsUnroutable{&bdns.MockClient{}}
	// NOTE(@jsha): The only method I've found so far to trigger a connect timeout
	// is to connect to an unrouteable IP address. This usually generates
	// a connection timeout, but will rarely return "Network unreachable" instead.
	// If we get that, just retry until we get something other than "Network unreachable".
	var prob *probs.ProblemDetails
	for i := 0; i < 20; i++ {
		_, _, prob = va.fetchHTTP(ctx, "unroutable.invalid", "/.well-known/acme-challenge/whatever")
		if prob != nil && strings.Contains(prob.Detail, "Network unreachable") {
			continue
		} else {
			break
		}
	}
	if prob == nil {
		t.Fatalf("Connection should've timed out")
	}
	took := time.Since(started)

	// Check that the HTTP connection doesn't return too fast, and times
	// out after the expected time
	if took < va.singleDialTimeout {
		t.Fatalf("fetch returned before %s (%s) with %#v", va.singleDialTimeout, took, prob)
	}
	if took > 2*va.singleDialTimeout {
		t.Fatalf("fetch didn't timeout after %s", va.singleDialTimeout)
	}
	test.AssertEquals(t, prob.Type, probs.ConnectionProblem)
	expectMatch := regexp.MustCompile(
		"Fetching http://unroutable.invalid/.well-known/acme-challenge/.*: Timeout during connect")
	if !expectMatch.MatchString(prob.Detail) {
		t.Errorf("Problem details incorrect. Got %q, expected to match %q",
			prob.Detail, expectMatch)
	}
}

func TestHTTPTransport(t *testing.T) {
	dummyDialerFunc := func(_ context.Context, _, _ string) (net.Conn, error) {
		return nil, nil
	}
	transport := httpTransport(dummyDialerFunc)
	// The HTTP Transport should have a TLS config that skips verifying
	// certificates.
	test.AssertEquals(t, transport.TLSClientConfig.InsecureSkipVerify, true)
	// Keep alives should be disabled
	test.AssertEquals(t, transport.DisableKeepAlives, true)
	test.AssertEquals(t, transport.MaxIdleConns, 1)
	test.AssertEquals(t, transport.IdleConnTimeout.String(), "1s")
	test.AssertEquals(t, transport.TLSHandshakeTimeout.String(), "10s")
}

func TestHTTPValidationTarget(t *testing.T) {
	// NOTE(@cpu): See `bdns/mocks.go` and the mock `LookupHost` function for the
	// hostnames used in this test.
	testCases := []struct {
		Name          string
		Host          string
		ExpectedError error
		ExpectedIPs   []string
	}{
		{
			Name:          "No IPs for host",
			Host:          "always.invalid",
			ExpectedError: berrors.DNSError("No valid IP addresses found for always.invalid"),
		},
		{
			Name:        "Only IPv4 addrs for host",
			Host:        "some.example.com",
			ExpectedIPs: []string{"127.0.0.1"},
		},
		{
			Name:        "Only IPv6 addrs for host",
			Host:        "ipv6.localhost",
			ExpectedIPs: []string{"::1"},
		},
		{
			Name: "Both IPv6 and IPv4 addrs for host",
			Host: "ipv4.and.ipv6.localhost",
			// In this case we expect 1 IPv6 address first, and then 1 IPv4 address
			ExpectedIPs: []string{"::1", "127.0.0.1"},
		},
	}

	const (
		examplePort  = 1234
		examplePath  = "/.well-known/path/i/took"
		exampleQuery = "my-path=was&my=own"
	)

	va, _ := setup(nil, 0, "", nil)
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			target, err := va.newHTTPValidationTarget(
				context.Background(),
				tc.Host,
				examplePort,
				examplePath,
				exampleQuery)
			if err != nil && tc.ExpectedError == nil {
				t.Fatalf("Unexpected error from NewHTTPValidationTarget: %v", err)
			} else if err != nil && tc.ExpectedError != nil {
				test.AssertMarshaledEquals(t, err, tc.ExpectedError)
			} else if err == nil {
				// The target should be populated.
				test.AssertNotEquals(t, target.host, "")
				test.AssertNotEquals(t, target.port, 0)
				test.AssertNotEquals(t, target.path, "")
				// Calling ip() on the target should give the expected IPs in the right
				// order.
				for i, expectedIP := range tc.ExpectedIPs {
					gotIP := target.ip()
					if gotIP == nil {
						t.Errorf("Expected IP %d to be %s got nil", i, expectedIP)
					} else {
						test.AssertEquals(t, gotIP.String(), expectedIP)
					}
					// Advance to the next IP
					_ = target.nextIP()
				}
			}
		})
	}
}

func TestExtractRequestTarget(t *testing.T) {
	mustURL := func(t *testing.T, rawURL string) *url.URL {
		urlOb, err := url.Parse(rawURL)
		if err != nil {
			t.Fatalf("Unable to parse raw URL %q: %v", rawURL, err)
			return nil
		}
		return urlOb
	}

	testCases := []struct {
		Name          string
		Req           *http.Request
		ExpectedError error
		ExpectedHost  string
		ExpectedPort  int
	}{
		{
			Name:          "nil input req",
			ExpectedError: fmt.Errorf("redirect HTTP request was nil"),
		},
		{
			Name: "invalid protocol scheme",
			Req: &http.Request{
				URL: mustURL(t, "gopher://letsencrypt.org"),
			},
			ExpectedError: fmt.Errorf("Invalid protocol scheme in redirect target. " +
				`Only "http" and "https" protocol schemes are supported, ` +
				`not "gopher"`),
		},
		{
			Name: "invalid explicit port",
			Req: &http.Request{
				URL: mustURL(t, "https://weird.port.letsencrypt.org:9999"),
			},
			ExpectedError: fmt.Errorf("Invalid port in redirect target. Only ports 80 " +
				"and 443 are supported, not 9999"),
		},
		{
			Name: "invalid empty hostname",
			Req: &http.Request{
				URL: mustURL(t, "https:///who/needs/a/hostname?not=me"),
			},
			ExpectedError: errors.New("Invalid empty hostname in redirect target"),
		},
		{
			Name: "invalid .well-known hostname",
			Req: &http.Request{
				URL: mustURL(t, "https://my.webserver.is.misconfigured.well-known/acme-challenge/xxx"),
			},
			ExpectedError: errors.New(`Invalid host in redirect target "my.webserver.is.misconfigured.well-known". Check webserver config for missing '/' in redirect target.`),
		},
		{
			Name: "invalid non-iana hostname",
			Req: &http.Request{
				URL: mustURL(t, "https://my.tld.is.cpu/pretty/cool/right?yeah=Ithoughtsotoo"),
			},
			ExpectedError: errors.New("Invalid hostname in redirect target, must end in IANA registered TLD"),
		},
		{
			Name: "bare IP",
			Req: &http.Request{
				URL: mustURL(t, "https://10.10.10.10"),
			},
			ExpectedError: fmt.Errorf(`Invalid host in redirect target "10.10.10.10". ` +
				"Only domain names are supported, not IP addresses"),
		},
		{
			Name: "valid HTTP redirect, explicit port",
			Req: &http.Request{
				URL: mustURL(t, "http://cpu.letsencrypt.org:80"),
			},
			ExpectedHost: "cpu.letsencrypt.org",
			ExpectedPort: 80,
		},
		{
			Name: "valid HTTP redirect, implicit port",
			Req: &http.Request{
				URL: mustURL(t, "http://cpu.letsencrypt.org"),
			},
			ExpectedHost: "cpu.letsencrypt.org",
			ExpectedPort: 80,
		},
		{
			Name: "valid HTTPS redirect, explicit port",
			Req: &http.Request{
				URL: mustURL(t, "https://cpu.letsencrypt.org:443/hello.world"),
			},
			ExpectedHost: "cpu.letsencrypt.org",
			ExpectedPort: 443,
		},
		{
			Name: "valid HTTPS redirect, implicit port",
			Req: &http.Request{
				URL: mustURL(t, "https://cpu.letsencrypt.org/hello.world"),
			},
			ExpectedHost: "cpu.letsencrypt.org",
			ExpectedPort: 443,
		},
	}

	va, _ := setup(nil, 0, "", nil)
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			host, port, err := va.extractRequestTarget(tc.Req)
			if err != nil && tc.ExpectedError == nil {
				t.Errorf("Expected nil err got %v", err)
			} else if err != nil && tc.ExpectedError != nil {
				test.AssertEquals(t, err.Error(), tc.ExpectedError.Error())
			} else if err == nil && tc.ExpectedError != nil {
				t.Errorf("Expected err %v, got nil", tc.ExpectedError)
			} else {
				test.AssertEquals(t, host, tc.ExpectedHost)
				test.AssertEquals(t, port, tc.ExpectedPort)
			}
		})
	}
}

// TestHTTPValidationDNSError attempts validation for a domain name that always
// generates a DNS error, and checks that a log line with the detailed error is
// generated.
func TestHTTPValidationDNSError(t *testing.T) {
	va, mockLog := setup(nil, 0, "", nil)

	_, _, prob := va.fetchHTTP(ctx, "always.error", "/.well-known/acme-challenge/whatever")
	test.AssertError(t, prob, "Expected validation fetch to fail")
	matchingLines := mockLog.GetAllMatching(`read udp: some net error`)
	if len(matchingLines) != 1 {
		t.Errorf("Didn't see expected DNS error logged. Instead, got:\n%s",
			strings.Join(mockLog.GetAllMatching(`.*`), "\n"))
	}
}

// TestHTTPValidationDNSIdMismatchError tests that performing an HTTP-01
// challenge with a domain name that always returns a DNS ID mismatch error from
// the mock resolver results in valid query/response data being logged in
// a format we can decode successfully.
func TestHTTPValidationDNSIdMismatchError(t *testing.T) {
	va, mockLog := setup(nil, 0, "", nil)

	_, _, prob := va.fetchHTTP(ctx, "id.mismatch", "/.well-known/acme-challenge/whatever")
	test.AssertError(t, prob, "Expected validation fetch to fail")
	matchingLines := mockLog.GetAllMatching(`logDNSError ID mismatch`)
	if len(matchingLines) != 1 {
		t.Errorf("Didn't see expected DNS error logged. Instead, got:\n%s",
			strings.Join(mockLog.GetAllMatching(`.*`), "\n"))
	}
	expectedRegex := regexp.MustCompile(
		`INFO: logDNSError ID mismatch ` +
			`chosenServer=\[mock.server\] ` +
			`hostname=\[id\.mismatch\] ` +
			`respHostname=\[id\.mismatch\.\] ` +
			`queryType=\[A\] ` +
			`msg=\[([A-Za-z0-9+=/\=]+)\] ` +
			`resp=\[([A-Za-z0-9+=/\=]+)\] ` +
			`err\=\[dns: id mismatch\]`,
	)

	matches := expectedRegex.FindAllStringSubmatch(matchingLines[0], -1)
	test.AssertEquals(t, len(matches), 1)
	submatches := matches[0]
	test.AssertEquals(t, len(submatches), 3)

	msgBytes, err := base64.StdEncoding.DecodeString(submatches[1])
	test.AssertNotError(t, err, "bad base64 encoded query msg")
	msg := new(dns.Msg)
	err = msg.Unpack(msgBytes)
	test.AssertNotError(t, err, "bad packed query msg")

	respBytes, err := base64.StdEncoding.DecodeString(submatches[2])
	test.AssertNotError(t, err, "bad base64 encoded resp msg")
	resp := new(dns.Msg)
	err = resp.Unpack(respBytes)
	test.AssertNotError(t, err, "bad packed response msg")
}

func TestSetupHTTPValidation(t *testing.T) {
	va, _ := setup(nil, 0, "", nil)

	mustTarget := func(t *testing.T, host string, port int, path string) *httpValidationTarget {
		target, err := va.newHTTPValidationTarget(
			context.Background(),
			host,
			port,
			path,
			"")
		if err != nil {
			t.Fatalf("Failed to construct httpValidationTarget for %q", host)
			return nil
		}
		return target
	}

	httpInputURL := "http://ipv4.and.ipv6.localhost/yellow/brick/road"
	httpsInputURL := "https://ipv4.and.ipv6.localhost/yellow/brick/road"

	testCases := []struct {
		Name           string
		InputURL       string
		InputTarget    *httpValidationTarget
		ExpectedRecord core.ValidationRecord
		ExpectedDialer *preresolvedDialer
		ExpectedError  error
	}{
		{
			Name:          "nil target",
			InputURL:      httpInputURL,
			ExpectedError: fmt.Errorf("httpValidationTarget can not be nil"),
		},
		{
			Name:          "empty input URL",
			InputTarget:   &httpValidationTarget{},
			ExpectedError: fmt.Errorf("reqURL can not be nil"),
		},
		{
			Name:     "target with no IPs",
			InputURL: httpInputURL,
			InputTarget: &httpValidationTarget{
				host: "foobar",
				port: va.httpPort,
				path: "idk",
			},
			ExpectedRecord: core.ValidationRecord{
				URL:      "http://ipv4.and.ipv6.localhost/yellow/brick/road",
				Hostname: "foobar",
				Port:     strconv.Itoa(va.httpPort),
			},
			ExpectedError: fmt.Errorf(`host "foobar" has no IP addresses remaining to use`),
		},
		{
			Name:        "HTTP input req",
			InputTarget: mustTarget(t, "ipv4.and.ipv6.localhost", va.httpPort, "/yellow/brick/road"),
			InputURL:    httpInputURL,
			ExpectedRecord: core.ValidationRecord{
				Hostname:          "ipv4.and.ipv6.localhost",
				Port:              strconv.Itoa(va.httpPort),
				URL:               "http://ipv4.and.ipv6.localhost/yellow/brick/road",
				AddressesResolved: []net.IP{net.ParseIP("::1"), net.ParseIP("127.0.0.1")},
				AddressUsed:       net.ParseIP("::1"),
			},
			ExpectedDialer: &preresolvedDialer{
				ip:      net.ParseIP("::1"),
				port:    va.httpPort,
				timeout: va.singleDialTimeout,
			},
		},
		{
			Name:        "HTTPS input req",
			InputTarget: mustTarget(t, "ipv4.and.ipv6.localhost", va.httpsPort, "/yellow/brick/road"),
			InputURL:    httpsInputURL,
			ExpectedRecord: core.ValidationRecord{
				Hostname:          "ipv4.and.ipv6.localhost",
				Port:              strconv.Itoa(va.httpsPort),
				URL:               "https://ipv4.and.ipv6.localhost/yellow/brick/road",
				AddressesResolved: []net.IP{net.ParseIP("::1"), net.ParseIP("127.0.0.1")},
				AddressUsed:       net.ParseIP("::1"),
			},
			ExpectedDialer: &preresolvedDialer{
				ip:      net.ParseIP("::1"),
				port:    va.httpsPort,
				timeout: va.singleDialTimeout,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			outDialer, outRecord, err := va.setupHTTPValidation(tc.InputURL, tc.InputTarget)
			if err != nil && tc.ExpectedError == nil {
				t.Errorf("Expected nil error, got %v", err)
			} else if err == nil && tc.ExpectedError != nil {
				t.Errorf("Expected %v error, got nil", tc.ExpectedError)
			} else if err != nil && tc.ExpectedError != nil {
				test.AssertEquals(t, err.Error(), tc.ExpectedError.Error())
			}
			if tc.ExpectedDialer == nil && outDialer != nil {
				t.Errorf("Expected nil dialer, got %v", outDialer)
			} else if tc.ExpectedDialer != nil {
				test.AssertMarshaledEquals(t, outDialer, tc.ExpectedDialer)
			}
			// In all cases we expect there to have been a validation record
			test.AssertMarshaledEquals(t, outRecord, tc.ExpectedRecord)
		})
	}
}

// A more concise version of httpSrv() that supports http.go tests
func httpTestSrv(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewUnstartedServer(mux)

	server.Start()
	httpPort := getPort(server)

	// A path that always returns an OK response
	mux.HandleFunc("/ok", func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
		fmt.Fprint(resp, "ok")
	})

	// A path that always times out by sleeping longer than the validation context
	// allows
	mux.HandleFunc("/timeout", func(resp http.ResponseWriter, req *http.Request) {
		time.Sleep(time.Second)
		resp.WriteHeader(http.StatusOK)
		fmt.Fprint(resp, "sorry, I'm a slow server")
	})

	// A path that always redirects to itself, creating a loop that will terminate
	// when detected.
	mux.HandleFunc("/loop", func(resp http.ResponseWriter, req *http.Request) {
		http.Redirect(
			resp,
			req,
			fmt.Sprintf("http://example.com:%d/loop", httpPort),
			http.StatusMovedPermanently)
	})

	// A path that sequentially redirects, creating an incrementing redirect
	// that will terminate when the redirect limit is reached and ensures each
	// URL is different than the last.
	for i := 0; i <= maxRedirect+1; i++ {
		// Need to re-scope i so it iterates properly in the function
		i := i
		mux.HandleFunc(fmt.Sprintf("/max-redirect/%d", i),
			func(resp http.ResponseWriter, req *http.Request) {
				http.Redirect(
					resp,
					req,
					fmt.Sprintf("http://example.com:%d/max-redirect/%d", httpPort, i+1),
					http.StatusMovedPermanently,
				)
			})
	}

	// A path that always redirects to a URL with a non-HTTP/HTTPs protocol scheme
	mux.HandleFunc("/redir-bad-proto", func(resp http.ResponseWriter, req *http.Request) {
		http.Redirect(
			resp,
			req,
			"gopher://example.com",
			http.StatusMovedPermanently,
		)
	})

	// A path that always redirects to a URL with a port other than the configured
	// HTTP/HTTPS port
	mux.HandleFunc("/redir-bad-port", func(resp http.ResponseWriter, req *http.Request) {
		http.Redirect(
			resp,
			req,
			"https://example.com:1987",
			http.StatusMovedPermanently,
		)
	})

	// A path that always redirects to a URL with a bare IP address
	mux.HandleFunc("/redir-bad-host", func(resp http.ResponseWriter, req *http.Request) {
		http.Redirect(
			resp,
			req,
			"https://127.0.0.1",
			http.StatusMovedPermanently,
		)
	})

	mux.HandleFunc("/bad-status-code", func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusGone)
		fmt.Fprint(resp, "sorry, I'm gone")
	})

	// A path that always responds with a 303 redirect
	mux.HandleFunc("/303-see-other", func(resp http.ResponseWriter, req *http.Request) {
		http.Redirect(
			resp,
			req,
			"http://example.org/303-see-other",
			http.StatusSeeOther,
		)
	})

	tooLargeBuf := bytes.NewBuffer([]byte{})
	for i := 0; i < maxResponseSize+10; i++ {
		tooLargeBuf.WriteByte(byte(97))
	}
	mux.HandleFunc("/resp-too-big", func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
		fmt.Fprint(resp, tooLargeBuf)
	})

	// Create a buffer that starts with invalid UTF8 and is bigger than
	// maxResponseSize
	tooLargeInvalidUTF8 := bytes.NewBuffer([]byte{})
	tooLargeInvalidUTF8.WriteString("f\xffoo")
	tooLargeInvalidUTF8.Write(tooLargeBuf.Bytes())
	// invalid-utf8-body Responds with body that is larger than
	// maxResponseSize and starts with an invalid UTF8 string. This is to
	// test the codepath where invalid UTF8 is converted to valid UTF8
	// that can be passed as an error message via grpc.
	mux.HandleFunc("/invalid-utf8-body", func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
		fmt.Fprint(resp, tooLargeInvalidUTF8)
	})

	mux.HandleFunc("/redir-path-too-long", func(resp http.ResponseWriter, req *http.Request) {
		http.Redirect(
			resp,
			req,
			"https://example.com/this-is-too-long-01234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789",
			http.StatusMovedPermanently)
	})

	// A path that redirects to an uppercase public suffix (#4215)
	mux.HandleFunc("/redir-uppercase-publicsuffix", func(resp http.ResponseWriter, req *http.Request) {
		http.Redirect(
			resp,
			req,
			"http://example.COM/ok",
			http.StatusMovedPermanently)
	})

	// A path that returns a body containing printf formatting verbs
	mux.HandleFunc("/printf-verbs", func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
		fmt.Fprint(resp, "%"+"2F.well-known%"+"2F"+tooLargeBuf.String())
	})

	return server
}

type testNetErr struct{}

func (e *testNetErr) Error() string {
	return "testNetErr"
}

func (e *testNetErr) Temporary() bool {
	return false
}

func (e *testNetErr) Timeout() bool {
	return false
}

func TestFallbackErr(t *testing.T) {
	untypedErr := errors.New("the least interesting kind of error")
	berr := berrors.InternalServerError("code violet: class neptune")
	netOpErr := &net.OpError{
		Op:  "siphon",
		Err: fmt.Errorf("port was clogged. please empty packets"),
	}
	netDialOpErr := &net.OpError{
		Op:  "dial",
		Err: fmt.Errorf("your call is important to us - please stay on the line"),
	}
	netErr := &testNetErr{}

	testCases := []struct {
		Name           string
		Err            error
		ExpectFallback bool
	}{
		{
			Name: "Nil error",
			Err:  nil,
		},
		{
			Name: "Standard untyped error",
			Err:  untypedErr,
		},
		{
			Name: "A Boulder error instance",
			Err:  berr,
		},
		{
			Name: "A non-dial net.OpError instance",
			Err:  netOpErr,
		},
		{
			Name:           "A dial net.OpError instance",
			Err:            netDialOpErr,
			ExpectFallback: true,
		},
		{
			Name: "A generic net.Error instance",
			Err:  netErr,
		},
		{
			Name: "A URL error wrapping a standard error",
			Err: &url.Error{
				Op:  "ivy",
				URL: "https://en.wikipedia.org/wiki/Operation_Ivy_(band)",
				Err: errors.New("take warning"),
			},
		},
		{
			Name: "A URL error wrapping a nil error",
			Err: &url.Error{
				Err: nil,
			},
		},
		{
			Name: "A URL error wrapping a Boulder error instance",
			Err: &url.Error{
				Err: berr,
			},
		},
		{
			Name: "A URL error wrapping a non-dial net OpError",
			Err: &url.Error{
				Err: netOpErr,
			},
		},
		{
			Name: "A URL error wrapping a dial net.OpError",
			Err: &url.Error{
				Err: netDialOpErr,
			},
			ExpectFallback: true,
		},
		{
			Name: "A URL error wrapping a generic net Error",
			Err: &url.Error{
				Err: netErr,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if isFallback := fallbackErr(tc.Err); isFallback != tc.ExpectFallback {
				t.Errorf(
					"Expected fallbackErr for %t to be %v was %v\n",
					tc.Err, tc.ExpectFallback, isFallback)
			}
		})
	}
}

func TestFetchHTTP(t *testing.T) {
	// Create a test server
	testSrv := httpTestSrv(t)
	defer testSrv.Close()

	// Setup a VA. By providing the testSrv to setup the VA will use the testSrv's
	// randomly assigned port as its HTTP port.
	va, _ := setup(testSrv, 0, "", nil)

	// We need to know the randomly assigned HTTP port for testcases as well
	httpPort := getPort(testSrv)

	// For the looped test case we expect one validation record per redirect
	// until boulder detects that a url has been used twice indicating a
	// redirect loop. Because it is hitting the /loop endpoint it will encounter
	// this scenario after the base url and fail on the second time hitting the
	// redirect with a port definition. On i=0 it will encounter the first
	// redirect to the url with a port definition and on i=1 it will encounter
	// the second redirect to the url with the port and get an expected error.
	expectedLoopRecords := []core.ValidationRecord{}
	for i := 0; i < 2; i++ {
		// The first request will not have a port # in the URL.
		url := "http://example.com/loop"
		if i != 0 {
			url = fmt.Sprintf("http://example.com:%d/loop", httpPort)
		}
		expectedLoopRecords = append(expectedLoopRecords,
			core.ValidationRecord{
				Hostname:          "example.com",
				Port:              strconv.Itoa(httpPort),
				URL:               url,
				AddressesResolved: []net.IP{net.ParseIP("127.0.0.1")},
				AddressUsed:       net.ParseIP("127.0.0.1"),
			})
	}

	// For the too many redirect test case we expect one validation record per
	// redirect up to maxRedirect (inclusive). There is also +1 record for the
	// base lookup, giving a termination criteria of > maxRedirect+1
	expectedTooManyRedirRecords := []core.ValidationRecord{}
	for i := 0; i <= maxRedirect+1; i++ {
		// The first request will not have a port # in the URL.
		url := "http://example.com/max-redirect/0"
		if i != 0 {
			url = fmt.Sprintf("http://example.com:%d/max-redirect/%d", httpPort, i)
		}
		expectedTooManyRedirRecords = append(expectedTooManyRedirRecords,
			core.ValidationRecord{
				Hostname:          "example.com",
				Port:              strconv.Itoa(httpPort),
				URL:               url,
				AddressesResolved: []net.IP{net.ParseIP("127.0.0.1")},
				AddressUsed:       net.ParseIP("127.0.0.1"),
			})
	}

	expectedTruncatedResp := bytes.NewBuffer([]byte{})
	for i := 0; i < maxResponseSize; i++ {
		expectedTruncatedResp.WriteByte(byte(97))
	}

	testCases := []struct {
		Name            string
		Host            string
		Path            string
		ExpectedBody    string
		ExpectedRecords []core.ValidationRecord
		ExpectedProblem *probs.ProblemDetails
	}{
		{
			Name: "No IPs for host",
			Host: "always.invalid",
			Path: "/.well-known/whatever",
			ExpectedProblem: probs.DNS(
				"No valid IP addresses found for always.invalid"),
			// There are no validation records in this case because the base record
			// is only constructed once a URL is made.
			ExpectedRecords: nil,
		},
		{
			Name: "Timeout for host",
			Host: "example.com",
			Path: "/timeout",
			ExpectedProblem: probs.ConnectionFailure(
				"127.0.0.1: Fetching http://example.com/timeout: " +
					"Timeout after connect (your server may be slow or overloaded)"),
			ExpectedRecords: []core.ValidationRecord{
				{
					Hostname:          "example.com",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://example.com/timeout",
					AddressesResolved: []net.IP{net.ParseIP("127.0.0.1")},
					AddressUsed:       net.ParseIP("127.0.0.1"),
				},
			},
		},
		{
			Name: "Redirect loop",
			Host: "example.com",
			Path: "/loop",
			ExpectedProblem: probs.ConnectionFailure(fmt.Sprintf(
				"127.0.0.1: Fetching http://example.com:%d/loop: Redirect loop detected", httpPort)),
			ExpectedRecords: expectedLoopRecords,
		},
		{
			Name: "Too many redirects",
			Host: "example.com",
			Path: "/max-redirect/0",
			ExpectedProblem: probs.ConnectionFailure(fmt.Sprintf(
				"127.0.0.1: Fetching http://example.com:%d/max-redirect/12: Too many redirects", httpPort)),
			ExpectedRecords: expectedTooManyRedirRecords,
		},
		{
			Name: "Redirect to bad protocol",
			Host: "example.com",
			Path: "/redir-bad-proto",
			ExpectedProblem: probs.ConnectionFailure(
				"127.0.0.1: Fetching gopher://example.com: Invalid protocol scheme in " +
					`redirect target. Only "http" and "https" protocol schemes ` +
					`are supported, not "gopher"`),
			ExpectedRecords: []core.ValidationRecord{
				{
					Hostname:          "example.com",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://example.com/redir-bad-proto",
					AddressesResolved: []net.IP{net.ParseIP("127.0.0.1")},
					AddressUsed:       net.ParseIP("127.0.0.1"),
				},
			},
		},
		{
			Name: "Redirect to bad port",
			Host: "example.com",
			Path: "/redir-bad-port",
			ExpectedProblem: probs.ConnectionFailure(fmt.Sprintf(
				"127.0.0.1: Fetching https://example.com:1987: Invalid port in redirect target. "+
					"Only ports %d and 443 are supported, not 1987", httpPort)),
			ExpectedRecords: []core.ValidationRecord{
				{
					Hostname:          "example.com",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://example.com/redir-bad-port",
					AddressesResolved: []net.IP{net.ParseIP("127.0.0.1")},
					AddressUsed:       net.ParseIP("127.0.0.1"),
				},
			},
		},
		{
			Name: "Redirect to bad host (bare IP address)",
			Host: "example.com",
			Path: "/redir-bad-host",
			ExpectedProblem: probs.ConnectionFailure(
				"127.0.0.1: Fetching https://127.0.0.1: Invalid host in redirect target " +
					`"127.0.0.1". Only domain names are supported, not IP addresses`),
			ExpectedRecords: []core.ValidationRecord{
				{
					Hostname:          "example.com",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://example.com/redir-bad-host",
					AddressesResolved: []net.IP{net.ParseIP("127.0.0.1")},
					AddressUsed:       net.ParseIP("127.0.0.1"),
				},
			},
		},
		{
			Name: "Redirect to long path",
			Host: "example.com",
			Path: "/redir-path-too-long",
			ExpectedProblem: probs.ConnectionFailure(
				"127.0.0.1: Fetching https://example.com/this-is-too-long-01234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789: Redirect target too long"),
			ExpectedRecords: []core.ValidationRecord{
				{
					Hostname:          "example.com",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://example.com/redir-path-too-long",
					AddressesResolved: []net.IP{net.ParseIP("127.0.0.1")},
					AddressUsed:       net.ParseIP("127.0.0.1"),
				},
			},
		},
		{
			Name: "Wrong HTTP status code",
			Host: "example.com",
			Path: "/bad-status-code",
			ExpectedProblem: probs.Unauthorized(
				"127.0.0.1: Invalid response from http://example.com/bad-status-code: 410"),
			ExpectedRecords: []core.ValidationRecord{
				{
					Hostname:          "example.com",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://example.com/bad-status-code",
					AddressesResolved: []net.IP{net.ParseIP("127.0.0.1")},
					AddressUsed:       net.ParseIP("127.0.0.1"),
				},
			},
		},
		{
			Name: "HTTP status code 303 redirect",
			Host: "example.com",
			Path: "/303-see-other",
			ExpectedProblem: probs.ConnectionFailure(
				"127.0.0.1: Fetching http://example.org/303-see-other: received disallowed redirect status code"),
			ExpectedRecords: []core.ValidationRecord{
				{
					Hostname:          "example.com",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://example.com/303-see-other",
					AddressesResolved: []net.IP{net.ParseIP("127.0.0.1")},
					AddressUsed:       net.ParseIP("127.0.0.1"),
				},
			},
		},
		{
			Name: "Response too large",
			Host: "example.com",
			Path: "/resp-too-big",
			ExpectedProblem: probs.Unauthorized(fmt.Sprintf(
				"127.0.0.1: Invalid response from http://example.com/resp-too-big: %q", expectedTruncatedResp.String(),
			)),
			ExpectedRecords: []core.ValidationRecord{
				{
					Hostname:          "example.com",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://example.com/resp-too-big",
					AddressesResolved: []net.IP{net.ParseIP("127.0.0.1")},
					AddressUsed:       net.ParseIP("127.0.0.1"),
				},
			},
		},
		{
			Name: "Broken IPv6 only",
			Host: "ipv6.localhost",
			Path: "/ok",
			ExpectedProblem: probs.ConnectionFailure(
				"::1: Fetching http://ipv6.localhost/ok: Error getting validation data"),
			ExpectedRecords: []core.ValidationRecord{
				{
					Hostname:          "ipv6.localhost",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://ipv6.localhost/ok",
					AddressesResolved: []net.IP{net.ParseIP("::1")},
					AddressUsed:       net.ParseIP("::1"),
				},
			},
		},
		{
			Name:         "Dual homed w/ broken IPv6, working IPv4",
			Host:         "ipv4.and.ipv6.localhost",
			Path:         "/ok",
			ExpectedBody: "ok",
			ExpectedRecords: []core.ValidationRecord{
				{
					Hostname:          "ipv4.and.ipv6.localhost",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://ipv4.and.ipv6.localhost/ok",
					AddressesResolved: []net.IP{net.ParseIP("::1"), net.ParseIP("127.0.0.1")},
					// The first validation record should have used the IPv6 addr
					AddressUsed: net.ParseIP("::1"),
				},
				{
					Hostname:          "ipv4.and.ipv6.localhost",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://ipv4.and.ipv6.localhost/ok",
					AddressesResolved: []net.IP{net.ParseIP("::1"), net.ParseIP("127.0.0.1")},
					// The second validation record should have used the IPv4 addr as a fallback
					AddressUsed: net.ParseIP("127.0.0.1"),
				},
			},
		},
		{
			Name:         "Working IPv4 only",
			Host:         "example.com",
			Path:         "/ok",
			ExpectedBody: "ok",
			ExpectedRecords: []core.ValidationRecord{
				{
					Hostname:          "example.com",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://example.com/ok",
					AddressesResolved: []net.IP{net.ParseIP("127.0.0.1")},
					AddressUsed:       net.ParseIP("127.0.0.1"),
				},
			},
		},
		{
			Name:         "Redirect to uppercase Public Suffix",
			Host:         "example.com",
			Path:         "/redir-uppercase-publicsuffix",
			ExpectedBody: "ok",
			ExpectedRecords: []core.ValidationRecord{
				{
					Hostname:          "example.com",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://example.com/redir-uppercase-publicsuffix",
					AddressesResolved: []net.IP{net.ParseIP("127.0.0.1")},
					AddressUsed:       net.ParseIP("127.0.0.1"),
				},
				{
					Hostname:          "example.com",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://example.com/ok",
					AddressesResolved: []net.IP{net.ParseIP("127.0.0.1")},
					AddressUsed:       net.ParseIP("127.0.0.1"),
				},
			},
		},
		{
			Name: "Reflected response body containing printf verbs",
			Host: "example.com",
			Path: "/printf-verbs",
			ExpectedProblem: &probs.ProblemDetails{
				Type: probs.UnauthorizedProblem,
				Detail: fmt.Sprintf("127.0.0.1: Invalid response from http://example.com/printf-verbs: %q",
					("%2F.well-known%2F" + expectedTruncatedResp.String())[:maxResponseSize]),
				HTTPStatus: http.StatusForbidden,
			},
			ExpectedRecords: []core.ValidationRecord{
				{
					Hostname:          "example.com",
					Port:              strconv.Itoa(httpPort),
					URL:               "http://example.com/printf-verbs",
					AddressesResolved: []net.IP{net.ParseIP("127.0.0.1")},
					AddressUsed:       net.ParseIP("127.0.0.1"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
			defer cancel()
			body, records, prob := va.fetchHTTP(ctx, tc.Host, tc.Path)
			if prob != nil && tc.ExpectedProblem == nil {
				t.Errorf("expected nil prob, got %#v\n", prob)
			} else if prob == nil && tc.ExpectedProblem != nil {
				t.Errorf("expected %#v prob, got nil", tc.ExpectedProblem)
			} else if prob != nil && tc.ExpectedProblem != nil {
				test.AssertMarshaledEquals(t, prob, tc.ExpectedProblem)
			} else {
				test.AssertEquals(t, string(body), tc.ExpectedBody)
			}
			// in all cases we expect validation records to be present and matching expected
			test.AssertMarshaledEquals(t, records, tc.ExpectedRecords)
		})
	}
}

// All paths that get assigned to tokens MUST be valid tokens
const pathWrongToken = "i6lNAC4lOOLYCl-A08VJt9z_tKYvVk63Dumo8icsBjQ"
const path404 = "404"
const path500 = "500"
const pathFound = "GBq8SwWq3JsbREFdCamk5IX3KLsxW5ULeGs98Ajl_UM"
const pathMoved = "5J4FIMrWNfmvHZo-QpKZngmuhqZGwRm21-oEgUDstJM"
const pathRedirectInvalidPort = "port-redirect"
const pathWait = "wait"
const pathWaitLong = "wait-long"
const pathReLookup = "7e-P57coLM7D3woNTp_xbJrtlkDYy6PWf3mSSbLwCr4"
const pathReLookupInvalid = "re-lookup-invalid"
const pathRedirectToFailingURL = "re-to-failing-url"
const pathLooper = "looper"
const pathValid = "valid"
const rejectUserAgent = "rejectMe"

func httpSrv(t *testing.T, token string) *httptest.Server {
	m := http.NewServeMux()

	server := httptest.NewUnstartedServer(m)

	defaultToken := token
	currentToken := defaultToken

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, path404) {
			t.Logf("HTTPSRV: Got a 404 req\n")
			http.NotFound(w, r)
		} else if strings.HasSuffix(r.URL.Path, path500) {
			t.Logf("HTTPSRV: Got a 500 req\n")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		} else if strings.HasSuffix(r.URL.Path, pathMoved) {
			t.Logf("HTTPSRV: Got a http.StatusMovedPermanently redirect req\n")
			if currentToken == defaultToken {
				currentToken = pathMoved
			}
			http.Redirect(w, r, pathValid, http.StatusMovedPermanently)
		} else if strings.HasSuffix(r.URL.Path, pathFound) {
			t.Logf("HTTPSRV: Got a http.StatusFound redirect req\n")
			if currentToken == defaultToken {
				currentToken = pathFound
			}
			http.Redirect(w, r, pathMoved, http.StatusFound)
		} else if strings.HasSuffix(r.URL.Path, pathWait) {
			t.Logf("HTTPSRV: Got a wait req\n")
			time.Sleep(time.Second * 3)
		} else if strings.HasSuffix(r.URL.Path, pathWaitLong) {
			t.Logf("HTTPSRV: Got a wait-long req\n")
			time.Sleep(time.Second * 10)
		} else if strings.HasSuffix(r.URL.Path, pathReLookup) {
			t.Logf("HTTPSRV: Got a redirect req to a valid hostname\n")
			if currentToken == defaultToken {
				currentToken = pathReLookup
			}
			port := getPort(server)
			http.Redirect(w, r, fmt.Sprintf("http://other.valid.com:%d/path", port), http.StatusFound)
		} else if strings.HasSuffix(r.URL.Path, pathReLookupInvalid) {
			t.Logf("HTTPSRV: Got a redirect req to an invalid hostname\n")
			http.Redirect(w, r, "http://invalid.invalid/path", http.StatusFound)
		} else if strings.HasSuffix(r.URL.Path, pathRedirectToFailingURL) {
			t.Logf("HTTPSRV: Redirecting to a URL that will fail\n")
			port := getPort(server)
			http.Redirect(w, r, fmt.Sprintf("http://other.valid.com:%d/%s", port, path500), http.StatusMovedPermanently)
		} else if strings.HasSuffix(r.URL.Path, pathLooper) {
			t.Logf("HTTPSRV: Got a loop req\n")
			http.Redirect(w, r, r.URL.String(), http.StatusMovedPermanently)
		} else if strings.HasSuffix(r.URL.Path, pathRedirectInvalidPort) {
			t.Logf("HTTPSRV: Got a port redirect req\n")
			// Port 8080 is not the VA's httpPort or httpsPort and should be rejected
			http.Redirect(w, r, "http://other.valid.com:8080/path", http.StatusFound)
		} else if r.Header.Get("User-Agent") == rejectUserAgent {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("found trap User-Agent"))
		} else {
			t.Logf("HTTPSRV: Got a valid req\n")
			t.Logf("HTTPSRV: Path = %s\n", r.URL.Path)

			ch := core.Challenge{Token: currentToken}
			keyAuthz, _ := ch.ExpectedKeyAuthorization(accountKey)
			t.Logf("HTTPSRV: Key Authz = '%s%s'\n", keyAuthz, "\\n\\r \\t")

			fmt.Fprint(w, keyAuthz, "\n\r \t")
			currentToken = defaultToken
		}
	})

	server.Start()
	return server
}

func TestHTTPBadPort(t *testing.T) {
	hs := httpSrv(t, expectedToken)
	defer hs.Close()

	va, _ := setup(hs, 0, "", nil)

	// Pick a random port between 40000 and 65000 - with great certainty we won't
	// have an HTTP server listening on this port and the test will fail as
	// intended
	badPort := 40000 + mrand.Intn(25000)
	va.httpPort = badPort

	_, prob := va.validateHTTP01(ctx, dnsi("localhost"), httpChallenge())
	if prob == nil {
		t.Fatalf("Server's down; expected refusal. Where did we connect?")
	}
	test.AssertEquals(t, prob.Type, probs.ConnectionProblem)
	if !strings.Contains(prob.Detail, "Connection refused") {
		t.Errorf("Expected a connection refused error, got %q", prob.Detail)
	}
}

func TestHTTPKeyAuthorizationFileMismatch(t *testing.T) {
	m := http.NewServeMux()
	hs := httptest.NewUnstartedServer(m)
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("\xef\xffAABBCC"))
	})
	hs.Start()

	va, _ := setup(hs, 0, "", nil)
	_, prob := va.validateHTTP01(ctx, dnsi("localhost.com"), httpChallenge())

	if prob == nil {
		t.Fatalf("Expected validation to fail when file mismatched.")
	}
	expected := `The key authorization file from the server did not match this challenge "LoqXcYV8q5ONbJQxbmR7SCTNo3tiAXDfowyjxAjEuX0.9jg46WB3rR_AHD-EBXdN7cBkH1WOu0tA3M9fm21mqTI" != "\xef\xffAABBCC"`
	if prob.Detail != expected {
		t.Errorf("validation failed with %s, expected %s", prob.Detail, expected)
	}
}

func TestHTTP(t *testing.T) {
	// NOTE: We do not attempt to shut down the server. The problem is that the
	// "wait-long" handler sleeps for ten seconds, but this test finishes in less
	// than that. So if we try to call hs.Close() at the end of the test, we'll be
	// closing the test server while a request is still pending. Unfortunately,
	// there appears to be an issue in httptest that trips Go's race detector when
	// that happens, failing the test. So instead, we live with leaving the server
	// around till the process exits.
	// TODO(#1989): close hs
	hs := httpSrv(t, expectedToken)

	va, log := setup(hs, 0, "", nil)

	chall := httpChallenge()
	t.Logf("Trying to validate: %+v\n", chall)
	_, prob := va.validateHTTP01(ctx, dnsi("localhost.com"), chall)
	if prob != nil {
		t.Errorf("Unexpected failure in HTTP validation: %s", prob)
	}
	test.AssertEquals(t, len(log.GetAllMatching(`\[AUDIT\] `)), 1)

	log.Clear()
	setChallengeToken(&chall, path404)
	_, prob = va.validateHTTP01(ctx, dnsi("localhost.com"), chall)
	if prob == nil {
		t.Fatalf("Should have found a 404 for the challenge.")
	}
	test.AssertEquals(t, prob.Type, probs.UnauthorizedProblem)
	test.AssertEquals(t, len(log.GetAllMatching(`\[AUDIT\] `)), 1)

	log.Clear()
	setChallengeToken(&chall, pathWrongToken)
	// The "wrong token" will actually be the expectedToken.  It's wrong
	// because it doesn't match pathWrongToken.
	_, prob = va.validateHTTP01(ctx, dnsi("localhost.com"), chall)
	if prob == nil {
		t.Fatalf("Should have found the wrong token value.")
	}
	test.AssertEquals(t, prob.Type, probs.UnauthorizedProblem)
	test.AssertEquals(t, len(log.GetAllMatching(`\[AUDIT\] `)), 1)

	log.Clear()
	setChallengeToken(&chall, pathMoved)
	_, prob = va.validateHTTP01(ctx, dnsi("localhost.com"), chall)
	if prob != nil {
		t.Fatalf("Failed to follow http.StatusMovedPermanently redirect")
	}
	redirectValid := `following redirect to host "" url "http://localhost.com/.well-known/acme-challenge/` + pathValid + `"`
	matchedValidRedirect := log.GetAllMatching(redirectValid)
	test.AssertEquals(t, len(matchedValidRedirect), 1)

	log.Clear()
	setChallengeToken(&chall, pathFound)
	_, prob = va.validateHTTP01(ctx, dnsi("localhost.com"), chall)
	if prob != nil {
		t.Fatalf("Failed to follow http.StatusFound redirect")
	}
	redirectMoved := `following redirect to host "" url "http://localhost.com/.well-known/acme-challenge/` + pathMoved + `"`
	matchedMovedRedirect := log.GetAllMatching(redirectMoved)
	test.AssertEquals(t, len(matchedValidRedirect), 1)
	test.AssertEquals(t, len(matchedMovedRedirect), 1)

	ipIdentifier := identifier.ACMEIdentifier{Type: identifier.IdentifierType("ip"), Value: "127.0.0.1"}
	_, prob = va.validateHTTP01(ctx, ipIdentifier, chall)
	if prob == nil {
		t.Fatalf("IdentifierType IP shouldn't have worked.")
	}
	test.AssertEquals(t, prob.Type, probs.MalformedProblem)

	_, prob = va.validateHTTP01(ctx, identifier.ACMEIdentifier{Type: identifier.DNS, Value: "always.invalid"}, chall)
	if prob == nil {
		t.Fatalf("Domain name is invalid.")
	}
	test.AssertEquals(t, prob.Type, probs.DNSProblem)
}

func TestHTTPTimeout(t *testing.T) {
	hs := httpSrv(t, expectedToken)
	// TODO(#1989): close hs

	va, _ := setup(hs, 0, "", nil)

	chall := httpChallenge()
	setChallengeToken(&chall, pathWaitLong)

	started := time.Now()
	timeout := 250 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, prob := va.validateHTTP01(ctx, dnsi("localhost"), chall)
	if prob == nil {
		t.Fatalf("Connection should've timed out")
	}

	took := time.Since(started)
	// Check that the HTTP connection doesn't return before a timeout, and times
	// out after the expected time
	if took < timeout-200*time.Millisecond {
		t.Fatalf("HTTP timed out before %s: %s with %s", timeout, took, prob)
	}
	if took > 2*timeout {
		t.Fatalf("HTTP connection didn't timeout after %s", timeout)
	}
	test.AssertEquals(t, prob.Type, probs.ConnectionProblem)
	test.AssertEquals(t, prob.Detail, "127.0.0.1: Fetching http://localhost/.well-known/acme-challenge/wait-long: Timeout after connect (your server may be slow or overloaded)")
}

// dnsMockReturnsUnroutable is a DNSClient mock that always returns an
// unroutable address for LookupHost. This is useful in testing connect
// timeouts.
type dnsMockReturnsUnroutable struct {
	*bdns.MockClient
}

func (mock dnsMockReturnsUnroutable) LookupHost(_ context.Context, hostname string) ([]net.IP, error) {
	return []net.IP{net.ParseIP("198.51.100.1")}, nil
}

// TestHTTPDialTimeout tests that we give the proper "Timeout during connect"
// error when dial fails. We do this by using a mock DNS client that resolves
// everything to an unroutable IP address.
func TestHTTPDialTimeout(t *testing.T) {
	va, _ := setup(nil, 0, "", nil)

	started := time.Now()
	timeout := 250 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	va.dnsClient = dnsMockReturnsUnroutable{&bdns.MockClient{}}
	// The only method I've found so far to trigger a connect timeout is to
	// connect to an unrouteable IP address. This usually generates a connection
	// timeout, but will rarely return "Network unreachable" instead. If we get
	// that, just retry until we get something other than "Network unreachable".
	var prob *probs.ProblemDetails
	for i := 0; i < 20; i++ {
		_, prob = va.validateHTTP01(ctx, dnsi("unroutable.invalid"), httpChallenge())
		if prob != nil && strings.Contains(prob.Detail, "Network unreachable") {
			continue
		} else {
			break
		}
	}
	if prob == nil {
		t.Fatalf("Connection should've timed out")
	}
	took := time.Since(started)
	// Check that the HTTP connection doesn't return too fast, and times
	// out after the expected time
	if took < (timeout-200*time.Millisecond)/2 {
		t.Fatalf("HTTP returned before %s (%s) with %#v", timeout, took, prob)
	}
	if took > 2*timeout {
		t.Fatalf("HTTP connection didn't timeout after %s seconds", timeout)
	}
	test.AssertEquals(t, prob.Type, probs.ConnectionProblem)
	expectMatch := regexp.MustCompile(
		"Fetching http://unroutable.invalid/.well-known/acme-challenge/.*: Timeout during connect")
	if !expectMatch.MatchString(prob.Detail) {
		t.Errorf("Problem details incorrect. Got %q, expected to match %q",
			prob.Detail, expectMatch)
	}
}

func TestHTTPRedirectLookup(t *testing.T) {
	hs := httpSrv(t, expectedToken)
	defer hs.Close()
	va, log := setup(hs, 0, "", nil)

	chall := httpChallenge()
	setChallengeToken(&chall, pathMoved)
	_, prob := va.validateHTTP01(ctx, dnsi("localhost.com"), chall)
	if prob != nil {
		t.Fatalf("Unexpected failure in redirect (%s): %s", pathMoved, prob)
	}
	redirectValid := `following redirect to host "" url "http://localhost.com/.well-known/acme-challenge/` + pathValid + `"`
	matchedValidRedirect := log.GetAllMatching(redirectValid)
	test.AssertEquals(t, len(matchedValidRedirect), 1)
	test.AssertEquals(t, len(log.GetAllMatching(`Resolved addresses for localhost.com: \[127.0.0.1\]`)), 2)

	log.Clear()
	setChallengeToken(&chall, pathFound)
	_, prob = va.validateHTTP01(ctx, dnsi("localhost.com"), chall)
	if prob != nil {
		t.Fatalf("Unexpected failure in redirect (%s): %s", pathFound, prob)
	}
	redirectMoved := `following redirect to host "" url "http://localhost.com/.well-known/acme-challenge/` + pathMoved + `"`
	matchedMovedRedirect := log.GetAllMatching(redirectMoved)
	test.AssertEquals(t, len(matchedMovedRedirect), 1)
	test.AssertEquals(t, len(log.GetAllMatching(`Resolved addresses for localhost.com: \[127.0.0.1\]`)), 3)

	log.Clear()
	setChallengeToken(&chall, pathReLookupInvalid)
	_, err := va.validateHTTP01(ctx, dnsi("localhost.com"), chall)
	test.AssertError(t, err, chall.Token)
	test.AssertEquals(t, len(log.GetAllMatching(`Resolved addresses for localhost.com: \[127.0.0.1\]`)), 1)
	test.AssertDeepEquals(t, err, probs.ConnectionFailure(`127.0.0.1: Fetching http://invalid.invalid/path: Invalid hostname in redirect target, must end in IANA registered TLD`))

	log.Clear()
	setChallengeToken(&chall, pathReLookup)
	_, prob = va.validateHTTP01(ctx, dnsi("localhost.com"), chall)
	if prob != nil {
		t.Fatalf("Unexpected error in redirect (%s): %s", pathReLookup, prob)
	}
	redirectPattern := `following redirect to host "" url "http://other.valid.com:\d+/path"`
	test.AssertEquals(t, len(log.GetAllMatching(redirectPattern)), 1)
	test.AssertEquals(t, len(log.GetAllMatching(`Resolved addresses for localhost.com: \[127.0.0.1\]`)), 1)
	test.AssertEquals(t, len(log.GetAllMatching(`Resolved addresses for other.valid.com: \[127.0.0.1\]`)), 1)

	log.Clear()
	setChallengeToken(&chall, pathRedirectInvalidPort)
	_, prob = va.validateHTTP01(ctx, dnsi("localhost.com"), chall)
	test.AssertNotNil(t, prob, "Problem details for pathRedirectInvalidPort should not be nil")
	test.AssertEquals(t, prob.Detail, fmt.Sprintf(
		"127.0.0.1: Fetching http://other.valid.com:8080/path: Invalid port in redirect target. "+
			"Only ports %d and %d are supported, not 8080", va.httpPort, va.httpsPort))

	// This case will redirect from a valid host to a host that is throwing
	// HTTP 500 errors. The test case is ensuring that the connection error
	// is referencing the redirected to host, instead of the original host.
	log.Clear()
	setChallengeToken(&chall, pathRedirectToFailingURL)
	_, prob = va.validateHTTP01(ctx, dnsi("localhost.com"), chall)
	test.AssertNotNil(t, prob, "Problem Details should not be nil")
	test.AssertDeepEquals(t, prob,
		probs.Unauthorized(
			fmt.Sprintf("127.0.0.1: Invalid response from http://other.valid.com:%d/500: 500",
				va.httpPort)))
}

func TestHTTPRedirectLoop(t *testing.T) {
	hs := httpSrv(t, expectedToken)
	defer hs.Close()
	va, _ := setup(hs, 0, "", nil)

	chall := httpChallenge()
	setChallengeToken(&chall, "looper")
	_, prob := va.validateHTTP01(ctx, dnsi("localhost"), chall)
	if prob == nil {
		t.Fatalf("Challenge should have failed for %s", chall.Token)
	}
}

func TestHTTPRedirectUserAgent(t *testing.T) {
	hs := httpSrv(t, expectedToken)
	defer hs.Close()
	va, _ := setup(hs, 0, "", nil)
	va.userAgent = rejectUserAgent

	chall := httpChallenge()
	setChallengeToken(&chall, pathMoved)
	_, prob := va.validateHTTP01(ctx, dnsi("localhost"), chall)
	if prob == nil {
		t.Fatalf("Challenge with rejectUserAgent should have failed (%s).", pathMoved)
	}

	setChallengeToken(&chall, pathFound)
	_, prob = va.validateHTTP01(ctx, dnsi("localhost"), chall)
	if prob == nil {
		t.Fatalf("Challenge with rejectUserAgent should have failed (%s).", pathFound)
	}
}

func getPort(hs *httptest.Server) int {
	url, err := url.Parse(hs.URL)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse hs URL: %q - %s", hs.URL, err.Error()))
	}
	_, portString, err := net.SplitHostPort(url.Host)
	if err != nil {
		panic(fmt.Sprintf("Failed to split hs URL host: %q - %s", url.Host, err.Error()))
	}
	port, err := strconv.ParseInt(portString, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse hs URL port: %q - %s", portString, err.Error()))
	}
	return int(port)
}

func TestValidateHTTP(t *testing.T) {
	chall := core.HTTPChallenge01("")
	setChallengeToken(&chall, core.NewToken())

	hs := httpSrv(t, chall.Token)
	defer hs.Close()

	va, _ := setup(hs, 0, "", nil)

	_, prob := va.validateChallenge(ctx, dnsi("localhost"), chall)
	test.Assert(t, prob == nil, "validation failed")
}

func TestLimitedReader(t *testing.T) {
	chall := core.HTTPChallenge01("")
	setChallengeToken(&chall, core.NewToken())

	hs := httpSrv(t, "012345\xff67890123456789012345678901234567890123456789012345678901234567890123456789")
	va, _ := setup(hs, 0, "", nil)
	defer hs.Close()

	_, prob := va.validateChallenge(ctx, dnsi("localhost"), chall)

	test.AssertEquals(t, prob.Type, probs.UnauthorizedProblem)
	test.Assert(t, strings.HasPrefix(prob.Detail, "127.0.0.1: Invalid response from "),
		"Expected failure due to truncation")

	if !utf8.ValidString(prob.Detail) {
		t.Errorf("Problem Detail contained an invalid UTF-8 string")
	}
}
