//go:build integration

package integration

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/test"
	ocsp_helper "github.com/letsencrypt/boulder/test/ocsp/helper"
	"golang.org/x/crypto/ocsp"
)

// certID matches the ASN.1 structure of the CertID sequence defined by RFC6960.
type certID struct {
	HashAlgorithm  pkix.AlgorithmIdentifier
	IssuerNameHash []byte
	IssuerKeyHash  []byte
	SerialNumber   *big.Int
}

func TestARI(t *testing.T) {
	t.Parallel()
	// This test is gated on the ServeRenewalInfo feature flag.
	if !strings.Contains(os.Getenv("BOULDER_CONFIG_DIR"), "test/config-next") {
		return
	}

	// Create an account.
	os.Setenv("DIRECTORY", "http://boulder.service.consul:4001/directory")
	client, err := makeClient("mailto:example@letsencrypt.org")
	test.AssertNotError(t, err, "creating acme client")

	// Create a private key.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	test.AssertNotError(t, err, "creating random cert key")

	// Issue a cert.
	name := random_domain()
	ir, err := authAndIssue(client, key, []string{name})
	test.AssertNotError(t, err, "failed to issue test cert")
	cert := ir.certs[0]

	// Leverage OCSP to get components of ARI request path.
	issuer, err := ocsp_helper.GetIssuer(cert)
	test.AssertNotError(t, err, "failed to get issuer cert")
	ocspReqBytes, err := ocsp.CreateRequest(cert, issuer, &ocsp.RequestOptions{Hash: crypto.SHA256})
	test.AssertNotError(t, err, "failed to build ocsp request")
	ocspReq, err := ocsp.ParseRequest(ocspReqBytes)
	test.AssertNotError(t, err, "failed to parse ocsp request")

	// Make ARI request.
	pathBytes, err := asn1.Marshal(certID{
		pkix.AlgorithmIdentifier{ // SHA256
			Algorithm:  asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1},
			Parameters: asn1.RawValue{Tag: 5 /* ASN.1 NULL */},
		},
		ocspReq.IssuerNameHash,
		ocspReq.IssuerKeyHash,
		cert.SerialNumber,
	})
	test.AssertNotError(t, err, "failed to marshal certID")
	url := fmt.Sprintf(
		"http://boulder.service.consul:4001/get/draft-ietf-acme-ari-00/renewalInfo/%s",
		base64.RawURLEncoding.EncodeToString(pathBytes),
	)
	resp, err := http.Get(url)
	test.AssertNotError(t, err, "ARI request should have succeeded")
	test.AssertEquals(t, resp.StatusCode, http.StatusOK)

	// Revoke the cert, then request ARI again, and the window should now be in
	// the past.
	err = client.RevokeCertificate(client.Account, cert, client.PrivateKey, 0)
	test.AssertNotError(t, err, "failed to revoke cert")
	resp, err = http.Get(url)
	test.AssertNotError(t, err, "ARI request should have succeeded")
	test.AssertEquals(t, resp.StatusCode, http.StatusOK)

	riBytes, err := io.ReadAll(resp.Body)
	test.AssertNotError(t, err, "failed to read ARI response")
	var ri core.RenewalInfo
	err = json.Unmarshal(riBytes, &ri)
	test.AssertNotError(t, err, "failed to parse ARI response")
	test.Assert(t, ri.SuggestedWindow.End.Before(time.Now()), "suggested window should end in the past")
	test.Assert(t, ri.SuggestedWindow.Start.Before(ri.SuggestedWindow.End), "suggested window should start before it ends")

	// Try to make a new cert for a new domain, but have it fail so only
	// a precert gets created.
	name = random_domain()
	err = ctAddRejectHost(name)
	test.AssertNotError(t, err, "failed to add ct-test-srv reject host")
	_, err = authAndIssue(client, key, []string{name})
	test.AssertError(t, err, "expected error from authAndIssue, was nil")
	cert, err = ctFindRejection([]string{name})
	test.AssertNotError(t, err, "failed to find rejected precert")

	// Get ARI path components.
	issuer, err = ocsp_helper.GetIssuer(cert)
	test.AssertNotError(t, err, "failed to get issuer cert")
	ocspReqBytes, err = ocsp.CreateRequest(cert, issuer, &ocsp.RequestOptions{Hash: crypto.SHA256})
	test.AssertNotError(t, err, "failed to build ocsp request")
	ocspReq, err = ocsp.ParseRequest(ocspReqBytes)
	test.AssertNotError(t, err, "failed to parse ocsp request")

	// Make ARI request.
	pathBytes, err = asn1.Marshal(certID{
		pkix.AlgorithmIdentifier{ // SHA256
			Algorithm:  asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1},
			Parameters: asn1.RawValue{Tag: 5 /* ASN.1 NULL */},
		},
		ocspReq.IssuerNameHash,
		ocspReq.IssuerKeyHash,
		cert.SerialNumber,
	})
	test.AssertNotError(t, err, "failed to marshal certID")
	url = fmt.Sprintf(
		"http://boulder.service.consul:4001/get/draft-ietf-acme-ari-00/renewalInfo/%s",
		base64.RawURLEncoding.EncodeToString(pathBytes),
	)
	resp, err = http.Get(url)
	test.AssertNotError(t, err, "ARI request should have succeeded")
	test.AssertEquals(t, resp.StatusCode, http.StatusNotFound)
}
