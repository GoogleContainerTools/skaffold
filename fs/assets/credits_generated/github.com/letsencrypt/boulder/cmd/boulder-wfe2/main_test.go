package notmain

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/issuance"
	"github.com/letsencrypt/boulder/test"
)

func TestLoadChain_Valid(t *testing.T) {
	issuer, chainPEM, err := loadChain([]string{
		"../../test/test-ca-cross.pem",
		"../../test/test-root2.pem",
	})
	test.AssertNotError(t, err, "Should load valid chain")

	expectedIssuer, err := core.LoadCert("../../test/test-ca-cross.pem")
	test.AssertNotError(t, err, "Failed to load test issuer")

	chainIssuerPEM, rest := pem.Decode(chainPEM)
	test.AssertNotNil(t, chainIssuerPEM, "Failed to decode chain PEM")
	parsedIssuer, err := x509.ParseCertificate(chainIssuerPEM.Bytes)
	test.AssertNotError(t, err, "Failed to parse chain PEM")

	// The three versions of the intermediate (the one loaded by us, the one
	// returned by loadChain, and the one parsed from the chain) should be equal.
	test.AssertByteEquals(t, issuer.Raw, expectedIssuer.Raw)
	test.AssertByteEquals(t, parsedIssuer.Raw, expectedIssuer.Raw)

	// The chain should contain nothing else.
	rootIssuerPEM, _ := pem.Decode(rest)
	if rootIssuerPEM != nil {
		t.Error("Expected chain PEM to contain one cert and nothing else")
	}
}

func TestLoadChain_TooShort(t *testing.T) {
	_, _, err := loadChain([]string{"/path/to/one/cert.pem"})
	test.AssertError(t, err, "Should reject too-short chain")
}

func TestLoadChain_Unloadable(t *testing.T) {
	_, _, err := loadChain([]string{
		"does-not-exist.pem",
		"../../test/test-root2.pem",
	})
	test.AssertError(t, err, "Should reject unloadable chain")

	_, _, err = loadChain([]string{
		"../../test/test-ca-cross.pem",
		"does-not-exist.pem",
	})
	test.AssertError(t, err, "Should reject unloadable chain")

	invalidPEMFile, _ := os.CreateTemp("", "invalid.pem")
	err = os.WriteFile(invalidPEMFile.Name(), []byte(""), 0640)
	test.AssertNotError(t, err, "Error writing invalid PEM tmp file")
	_, _, err = loadChain([]string{
		invalidPEMFile.Name(),
		"../../test/test-root2.pem",
	})
	test.AssertError(t, err, "Should reject unloadable chain")
}

func TestLoadChain_InvalidSig(t *testing.T) {
	_, _, err := loadChain([]string{
		"../../test/test-root2.pem",
		"../../test/test-ca-cross.pem",
	})
	test.AssertError(t, err, "Should reject invalid signature")
}

func TestLoadChain_NoRoot(t *testing.T) {
	// TODO(#5251): Implement this when we have a hierarchy which includes two
	// CA certs, neither of which is a root.
}

func TestLoadCertificateChains(t *testing.T) {
	// Read some cert bytes to use for expected chain content
	certBytesA, err := os.ReadFile("../../test/test-ca.pem")
	test.AssertNotError(t, err, "Error reading../../test/test-ca.pem")
	certBytesB, err := os.ReadFile("../../test/test-ca2.pem")
	test.AssertNotError(t, err, "Error reading../../test/test-ca2.pem")

	// Make a .pem file with invalid contents
	invalidPEMFile, _ := os.CreateTemp("", "invalid.pem")
	err = os.WriteFile(invalidPEMFile.Name(), []byte(""), 0640)
	test.AssertNotError(t, err, "Error writing invalid PEM tmp file")

	// Make a .pem file with a valid cert but also some leftover bytes
	leftoverPEMFile, _ := os.CreateTemp("", "leftovers.pem")
	leftovers := "vegan curry, cold rice, soy milk"
	leftoverBytes := append(certBytesA, []byte(leftovers)...)
	err = os.WriteFile(leftoverPEMFile.Name(), leftoverBytes, 0640)
	test.AssertNotError(t, err, "Error writing leftover PEM tmp file")

	// Make a .pem file that is test-ca2.pem but with Windows/DOS CRLF line
	// endings
	crlfPEM, _ := os.CreateTemp("", "crlf.pem")
	crlfPEMBytes := []byte(strings.Replace(string(certBytesB), "\n", "\r\n", -1))
	err = os.WriteFile(crlfPEM.Name(), crlfPEMBytes, 0640)
	test.AssertNotError(t, err, "os.WriteFile failed")

	// Make a .pem file that is test-ca.pem but with no trailing newline
	abruptPEM, _ := os.CreateTemp("", "abrupt.pem")
	abruptPEMBytes := certBytesA[:len(certBytesA)-1]
	err = os.WriteFile(abruptPEM.Name(), abruptPEMBytes, 0640)
	test.AssertNotError(t, err, "os.WriteFile failed")

	testCases := []struct {
		Name            string
		Input           map[string][]string
		ExpectedMap     map[issuance.IssuerNameID][]byte
		ExpectedError   error
		AllowEmptyChain bool
	}{
		{
			Name:  "No input",
			Input: nil,
		},
		{
			Name: "AIA Issuer without chain files",
			Input: map[string][]string{
				"http://break.the.chain.com": {},
			},
			ExpectedError: fmt.Errorf(
				"CertificateChain entry for AIA issuer url \"http://break.the.chain.com\" " +
					"has no chain file names configured"),
		},
		{
			Name: "Missing chain file",
			Input: map[string][]string{
				"http://where.is.my.mind": {"/tmp/does.not.exist.pem"},
			},
			ExpectedError: fmt.Errorf("CertificateChain entry for AIA issuer url \"http://where.is.my.mind\" " +
				"has an invalid chain file: \"/tmp/does.not.exist.pem\" - error reading " +
				"contents: open /tmp/does.not.exist.pem: no such file or directory"),
		},
		{
			Name: "PEM chain file with Windows CRLF line endings",
			Input: map[string][]string{
				"http://windows.sad.zone": {crlfPEM.Name()},
			},
			ExpectedError: fmt.Errorf("CertificateChain entry for AIA issuer url \"http://windows.sad.zone\" "+
				"has an invalid chain file: %q - contents had CRLF line endings", crlfPEM.Name()),
		},
		{
			Name: "Invalid PEM chain file",
			Input: map[string][]string{
				"http://ok.go": {invalidPEMFile.Name()},
			},
			ExpectedError: fmt.Errorf(
				"CertificateChain entry for AIA issuer url \"http://ok.go\" has an "+
					"invalid chain file: %q - contents did not decode as PEM",
				invalidPEMFile.Name()),
		},
		{
			Name: "PEM chain file that isn't a cert",
			Input: map[string][]string{
				"http://not-a-cert.com": {"../../test/test-root.key"},
			},
			ExpectedError: fmt.Errorf(
				"CertificateChain entry for AIA issuer url \"http://not-a-cert.com\" has " +
					"an invalid chain file: \"../../test/test-root.key\" - PEM block type " +
					"incorrect, found \"PRIVATE KEY\", expected \"CERTIFICATE\""),
		},
		{
			Name: "PEM chain file with leftover bytes",
			Input: map[string][]string{
				"http://tasty.leftovers.com": {leftoverPEMFile.Name()},
			},
			ExpectedError: fmt.Errorf(
				"CertificateChain entry for AIA issuer url \"http://tasty.leftovers.com\" "+
					"has an invalid chain file: %q - PEM contents had unused remainder input "+
					"(%d bytes)",
				leftoverPEMFile.Name(),
				len([]byte(leftovers)),
			),
		},
		{
			Name: "One PEM file chain",
			Input: map[string][]string{
				"http://single-cert-chain.com": {"../../test/test-ca.pem"},
			},
			ExpectedMap: map[issuance.IssuerNameID][]byte{
				issuance.IssuerNameID(37287262753088952): []byte(fmt.Sprintf("\n%s", string(certBytesA))),
			},
		},
		{
			Name: "Two PEM file chain",
			Input: map[string][]string{
				"http://two-cert-chain.com": {"../../test/test-ca.pem", "../../test/test-ca2.pem"},
			},
			ExpectedMap: map[issuance.IssuerNameID][]byte{
				issuance.IssuerNameID(37287262753088952): []byte(fmt.Sprintf("\n%s\n%s", string(certBytesA), string(certBytesB))),
			},
		},
		{
			Name: "One PEM file chain, no trailing newline",
			Input: map[string][]string{
				"http://single-cert-chain.nonewline.com": {abruptPEM.Name()},
			},
			ExpectedMap: map[issuance.IssuerNameID][]byte{
				// NOTE(@cpu): There should be a trailing \n added by the WFE that we
				// expect in the format specifier below.
				issuance.IssuerNameID(37287262753088952): []byte(fmt.Sprintf("\n%s\n", string(abruptPEMBytes))),
			},
		},
		{
			Name:            "Two PEM file chain, don't require at least one chain",
			AllowEmptyChain: true,
			Input: map[string][]string{
				"http://two-cert-chain.com": {"../../test/test-ca.pem", "../../test/test-ca2.pem"},
			},
			ExpectedMap: map[issuance.IssuerNameID][]byte{
				issuance.IssuerNameID(37287262753088952): []byte(fmt.Sprintf("\n%s\n%s", string(certBytesA), string(certBytesB))),
			},
		},
		{
			Name:            "Empty chain, don't require at least one chain",
			AllowEmptyChain: true,
			Input: map[string][]string{
				"http://two-cert-chain.com": {},
			},
			ExpectedMap: map[issuance.IssuerNameID][]byte{},
		},
		{
			Name: "Empty chain",
			Input: map[string][]string{
				"http://two-cert-chain.com": {},
			},
			ExpectedError: fmt.Errorf(
				"CertificateChain entry for AIA issuer url %q has no chain "+
					"file names configured",
				"http://two-cert-chain.com"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			resultMap, issuers, err := loadCertificateChains(tc.Input, !tc.AllowEmptyChain)
			if tc.ExpectedError == nil && err != nil {
				t.Errorf("Expected nil error, got %#v\n", err)
			} else if tc.ExpectedError != nil && err == nil {
				t.Errorf("Expected non-nil error, got nil err")
			} else if tc.ExpectedError != nil {
				test.AssertEquals(t, err.Error(), tc.ExpectedError.Error())
			}
			test.AssertEquals(t, len(resultMap), len(tc.ExpectedMap))
			test.AssertEquals(t, len(issuers), len(tc.ExpectedMap))
			for nameid, chain := range resultMap {
				test.Assert(t, bytes.Equal(chain, tc.ExpectedMap[nameid]), "Chain bytes did not match expected")
			}
		})
	}
}
