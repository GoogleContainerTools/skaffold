package cmd

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/letsencrypt/boulder/test"
)

func TestDBConfigURL(t *testing.T) {
	tests := []struct {
		conf     DBConfig
		expected string
	}{
		{
			// Test with one config file that has no trailing newline
			conf:     DBConfig{DBConnectFile: "testdata/test_dburl"},
			expected: "test@tcp(testhost:3306)/testDB?readTimeout=800ms&writeTimeout=800ms",
		},
		{
			// Test with a config file that *has* a trailing newline
			conf:     DBConfig{DBConnectFile: "testdata/test_dburl_newline"},
			expected: "test@tcp(testhost:3306)/testDB?readTimeout=800ms&writeTimeout=800ms",
		},
	}

	for _, tc := range tests {
		url, err := tc.conf.URL()
		test.AssertNotError(t, err, "Failed calling URL() on DBConfig")
		test.AssertEquals(t, url, tc.expected)
	}
}

func TestPasswordConfig(t *testing.T) {
	tests := []struct {
		pc       PasswordConfig
		expected string
	}{
		{pc: PasswordConfig{}, expected: ""},
		{pc: PasswordConfig{PasswordFile: "testdata/test_secret"}, expected: "secret"},
	}

	for _, tc := range tests {
		password, err := tc.pc.Pass()
		test.AssertNotError(t, err, "Failed to retrieve password")
		test.AssertEquals(t, password, tc.expected)
	}
}

func TestTLSConfigLoad(t *testing.T) {
	null := "/dev/null"
	nonExistent := "[nonexistent]"
	cert := "testdata/cert.pem"
	key := "testdata/key.pem"
	caCert := "testdata/minica.pem"
	testCases := []struct {
		TLSConfig
		want string
	}{
		{TLSConfig{nil, &null, &null}, "nil CertFile in TLSConfig"},
		{TLSConfig{&null, nil, &null}, "nil KeyFile in TLSConfig"},
		{TLSConfig{&null, &null, nil}, "nil CACertFile in TLSConfig"},
		{TLSConfig{&nonExistent, &key, &caCert}, "loading key pair.*no such file or directory"},
		{TLSConfig{&cert, &nonExistent, &caCert}, "loading key pair.*no such file or directory"},
		{TLSConfig{&cert, &key, &nonExistent}, "reading CA cert from.*no such file or directory"},
		{TLSConfig{&null, &key, &caCert}, "loading key pair.*failed to find any PEM data"},
		{TLSConfig{&cert, &null, &caCert}, "loading key pair.*failed to find any PEM data"},
		{TLSConfig{&cert, &key, &null}, "parsing CA certs"},
	}
	for _, tc := range testCases {
		var title [3]string
		if tc.CertFile == nil {
			title[0] = "nil"
		} else {
			title[0] = *tc.CertFile
		}
		if tc.KeyFile == nil {
			title[1] = "nil"
		} else {
			title[1] = *tc.KeyFile
		}
		if tc.CACertFile == nil {
			title[2] = "nil"
		} else {
			title[2] = *tc.CACertFile
		}
		t.Run(strings.Join(title[:], "_"), func(t *testing.T) {
			_, err := tc.TLSConfig.Load()
			if err == nil {
				t.Errorf("got no error")
			}
			if matched, _ := regexp.MatchString(tc.want, err.Error()); !matched {
				t.Errorf("got error %q, wanted %q", err, tc.want)
			}
		})
	}
}

func TestSampler(t *testing.T) {
	testCases := []struct {
		samplerate uint32
		span       map[string]interface{}
		sampled    bool
		rate       int
	}{
		// At sample rate 1, both of these should get sampled.
		{1, map[string]interface{}{"trace.trace_id": "foo"}, true, 1},
		{1, map[string]interface{}{"trace.trace_id": ""}, true, 1},
		// At sample rate 0, it should behave the same as sample rate 1.
		{0, map[string]interface{}{"trace.trace_id": "foo"}, true, 1},
		{0, map[string]interface{}{"trace.trace_id": ""}, true, 1},
		// At sample rate 2, only one of these should be sampled.
		{2, map[string]interface{}{"trace.trace_id": "foo"}, true, 2},
		{2, map[string]interface{}{"trace.trace_id": ""}, false, 2},
		// At sample rate 100, neither of these should be sampled.
		{100, map[string]interface{}{"trace.trace_id": "foo"}, false, 100},
		{100, map[string]interface{}{"trace.trace_id": ""}, false, 100},
		// A missing or non-string trace_id should result in sampling.
		{100, map[string]interface{}{}, true, 1},
		{100, map[string]interface{}{"trace.trace_id": 123}, true, 1},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Rate(%d) Span(%s)", tc.samplerate, tc.span), func(t *testing.T) {
			s := makeSampler(tc.samplerate)
			b, i := s(tc.span)
			test.AssertEquals(t, b, tc.sampled)
			test.AssertEquals(t, i, tc.rate)
		})
	}
}
