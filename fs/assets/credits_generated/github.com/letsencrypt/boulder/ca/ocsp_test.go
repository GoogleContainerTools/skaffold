package ca

import (
	"context"
	"crypto/x509"
	"encoding/hex"
	"testing"
	"time"

	capb "github.com/letsencrypt/boulder/ca/proto"
	"github.com/letsencrypt/boulder/core"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/test"
	"golang.org/x/crypto/ocsp"
)

func serial(t *testing.T) []byte {
	serial, err := hex.DecodeString("aabbccddeeffaabbccddeeff000102030405")
	if err != nil {
		t.Fatal(err)
	}
	return serial

}

func TestOCSP(t *testing.T) {
	testCtx := setup(t)
	ca, err := NewCertificateAuthorityImpl(
		&mockSA{},
		testCtx.pa,
		testCtx.ocsp,
		testCtx.crl,
		testCtx.boulderIssuers,
		nil,
		testCtx.certExpiry,
		testCtx.certBackdate,
		testCtx.serialPrefix,
		testCtx.maxNames,
		testCtx.keyPolicy,
		nil,
		testCtx.logger,
		testCtx.stats,
		testCtx.signatureCount,
		testCtx.signErrorCount,
		testCtx.fc)
	test.AssertNotError(t, err, "Failed to create CA")
	ocspi := testCtx.ocsp

	// Issue a certificate from the RSA issuer caCert, then check OCSP comes from the same issuer.
	rsaIssuerID := ca.issuers.byAlg[x509.RSA].ID()
	rsaCertPB, err := ca.IssuePrecertificate(ctx, &capb.IssueCertificateRequest{Csr: CNandSANCSR, RegistrationID: arbitraryRegID})
	test.AssertNotError(t, err, "Failed to issue certificate")
	rsaCert, err := x509.ParseCertificate(rsaCertPB.DER)
	test.AssertNotError(t, err, "Failed to parse rsaCert")
	rsaOCSPPB, err := ocspi.GenerateOCSP(ctx, &capb.GenerateOCSPRequest{
		Serial:   core.SerialToString(rsaCert.SerialNumber),
		IssuerID: int64(rsaIssuerID),
		Status:   string(core.OCSPStatusGood),
	})
	test.AssertNotError(t, err, "Failed to generate OCSP")
	rsaOCSP, err := ocsp.ParseResponse(rsaOCSPPB.Response, caCert.Certificate)
	test.AssertNotError(t, err, "Failed to parse / validate OCSP for rsaCert")
	test.AssertEquals(t, rsaOCSP.Status, 0)
	test.AssertEquals(t, rsaOCSP.RevocationReason, 0)
	test.AssertEquals(t, rsaOCSP.SerialNumber.Cmp(rsaCert.SerialNumber), 0)

	// Issue a certificate from the ECDSA issuer caCert2, then check OCSP comes from the same issuer.
	ecdsaIssuerID := ca.issuers.byAlg[x509.ECDSA].ID()
	ecdsaCertPB, err := ca.IssuePrecertificate(ctx, &capb.IssueCertificateRequest{Csr: ECDSACSR, RegistrationID: arbitraryRegID})
	test.AssertNotError(t, err, "Failed to issue certificate")
	ecdsaCert, err := x509.ParseCertificate(ecdsaCertPB.DER)
	test.AssertNotError(t, err, "Failed to parse ecdsaCert")
	ecdsaOCSPPB, err := ocspi.GenerateOCSP(ctx, &capb.GenerateOCSPRequest{
		Serial:   core.SerialToString(ecdsaCert.SerialNumber),
		IssuerID: int64(ecdsaIssuerID),
		Status:   string(core.OCSPStatusGood),
	})
	test.AssertNotError(t, err, "Failed to generate OCSP")
	ecdsaOCSP, err := ocsp.ParseResponse(ecdsaOCSPPB.Response, caCert2.Certificate)
	test.AssertNotError(t, err, "Failed to parse / validate OCSP for ecdsaCert")
	test.AssertEquals(t, ecdsaOCSP.Status, 0)
	test.AssertEquals(t, ecdsaOCSP.RevocationReason, 0)
	test.AssertEquals(t, ecdsaOCSP.SerialNumber.Cmp(ecdsaCert.SerialNumber), 0)

	// GenerateOCSP with a bad IssuerID should fail.
	_, err = ocspi.GenerateOCSP(context.Background(), &capb.GenerateOCSPRequest{
		Serial:   core.SerialToString(rsaCert.SerialNumber),
		IssuerID: int64(666),
		Status:   string(core.OCSPStatusGood),
	})
	test.AssertError(t, err, "GenerateOCSP didn't fail with invalid IssuerID")

	// GenerateOCSP with a bad Serial should fail.
	_, err = ocspi.GenerateOCSP(context.Background(), &capb.GenerateOCSPRequest{
		Serial:   "BADDECAF",
		IssuerID: int64(rsaIssuerID),
		Status:   string(core.OCSPStatusGood),
	})
	test.AssertError(t, err, "GenerateOCSP didn't fail with invalid Serial")

	// GenerateOCSP with a valid-but-nonexistent Serial should *not* fail.
	_, err = ocspi.GenerateOCSP(context.Background(), &capb.GenerateOCSPRequest{
		Serial:   "03DEADBEEFBADDECAFFADEFACECAFE30",
		IssuerID: int64(rsaIssuerID),
		Status:   string(core.OCSPStatusGood),
	})
	test.AssertNotError(t, err, "GenerateOCSP failed with fake-but-valid Serial")
}

// Set up an ocspLogQueue with a very long period and a large maxLen,
// to ensure any buffered entries get flushed on `.stop()`.
func TestOcspLogFlushOnExit(t *testing.T) {
	t.Parallel()
	log := blog.NewMock()
	stats := metrics.NoopRegisterer
	queue := newOCSPLogQueue(4000, 10000*time.Millisecond, stats, log)
	go queue.loop()
	queue.enqueue(serial(t), time.Now(), ocsp.Good, ocsp.Unspecified)
	queue.stop()

	expected := []string{
		"INFO: [AUDIT] OCSP signed: aabbccddeeffaabbccddeeff000102030405:_,",
	}
	test.AssertDeepEquals(t, log.GetAll(), expected)
}

// Ensure log lines are sent when they exceed maxLen.
func TestOcspFlushOnLength(t *testing.T) {
	t.Parallel()
	log := blog.NewMock()
	stats := metrics.NoopRegisterer
	queue := newOCSPLogQueue(100, 100*time.Millisecond, stats, log)
	go queue.loop()
	for i := 0; i < 5; i++ {
		queue.enqueue(serial(t), time.Now(), ocsp.Good, ocsp.Unspecified)
	}
	queue.stop()

	expected := []string{
		"INFO: [AUDIT] OCSP signed: aabbccddeeffaabbccddeeff000102030405:_,aabbccddeeffaabbccddeeff000102030405:_,",
		"INFO: [AUDIT] OCSP signed: aabbccddeeffaabbccddeeff000102030405:_,aabbccddeeffaabbccddeeff000102030405:_,",
		"INFO: [AUDIT] OCSP signed: aabbccddeeffaabbccddeeff000102030405:_,",
	}
	test.AssertDeepEquals(t, log.GetAll(), expected)
}

// Ensure log lines are sent after a timeout.
func TestOcspFlushOnTimeout(t *testing.T) {
	t.Parallel()
	log := blog.NewWaitingMock()
	stats := metrics.NoopRegisterer
	queue := newOCSPLogQueue(90000, 10*time.Millisecond, stats, log)

	go queue.loop()
	queue.enqueue(serial(t), time.Now(), ocsp.Good, ocsp.Unspecified)

	expected := "INFO: [AUDIT] OCSP signed: aabbccddeeffaabbccddeeff000102030405:_,"
	logLines, err := log.WaitForMatch("OCSP signed", 50*time.Millisecond)
	test.AssertNotError(t, err, "error in mock log")
	test.AssertDeepEquals(t, logLines, expected)
	queue.stop()
}

// If the deadline passes and nothing has been logged, we should not log a blank line.
func TestOcspNoEmptyLines(t *testing.T) {
	t.Parallel()
	log := blog.NewMock()
	stats := metrics.NoopRegisterer
	queue := newOCSPLogQueue(90000, 10*time.Millisecond, stats, log)

	go queue.loop()
	time.Sleep(50 * time.Millisecond)
	queue.stop()

	test.AssertDeepEquals(t, log.GetAll(), []string{})
}

// If the maxLogLen is shorter than one entry, log everything immediately.
func TestOcspLogWhenMaxLogLenIsShort(t *testing.T) {
	t.Parallel()
	log := blog.NewMock()
	stats := metrics.NoopRegisterer
	queue := newOCSPLogQueue(3, 10000*time.Millisecond, stats, log)
	go queue.loop()
	queue.enqueue(serial(t), time.Now(), ocsp.Good, ocsp.Unspecified)
	queue.stop()

	expected := []string{
		"INFO: [AUDIT] OCSP signed: aabbccddeeffaabbccddeeff000102030405:_,",
	}
	test.AssertDeepEquals(t, log.GetAll(), expected)
}

// Enqueueing entries after stop causes panic.
func TestOcspLogPanicsOnEnqueueAfterStop(t *testing.T) {
	t.Parallel()

	log := blog.NewMock()
	stats := metrics.NoopRegisterer
	queue := newOCSPLogQueue(4000, 10000*time.Millisecond, stats, log)
	go queue.loop()
	queue.stop()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	queue.enqueue(serial(t), time.Now(), ocsp.Good, ocsp.Unspecified)
}

// Ensure revoke reason gets set.
func TestOcspRevokeReasonIsSet(t *testing.T) {
	t.Parallel()
	log := blog.NewMock()
	stats := metrics.NoopRegisterer
	queue := newOCSPLogQueue(100, 100*time.Millisecond, stats, log)
	go queue.loop()

	queue.enqueue(serial(t), time.Now(), ocsp.Revoked, ocsp.KeyCompromise)
	queue.enqueue(serial(t), time.Now(), ocsp.Revoked, ocsp.CACompromise)
	queue.stop()

	expected := []string{
		"INFO: [AUDIT] OCSP signed: aabbccddeeffaabbccddeeff000102030405:1,aabbccddeeffaabbccddeeff000102030405:2,",
	}
	test.AssertDeepEquals(t, log.GetAll(), expected)
}
