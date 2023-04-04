package updater

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"math/big"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-gorp/gorp/v3"
	"github.com/jmhodges/clock"
	capb "github.com/letsencrypt/boulder/ca/proto"
	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/db"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/sa"
	sapb "github.com/letsencrypt/boulder/sa/proto"
	"github.com/letsencrypt/boulder/sa/satest"
	"github.com/letsencrypt/boulder/test"
	isa "github.com/letsencrypt/boulder/test/inmem/sa"
	"github.com/letsencrypt/boulder/test/vars"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

var ctx = context.Background()

type mockOCSP struct {
	sleepTime time.Duration
}

func (ca *mockOCSP) GenerateOCSP(_ context.Context, req *capb.GenerateOCSPRequest, _ ...grpc.CallOption) (*capb.OCSPResponse, error) {
	time.Sleep(ca.sleepTime)
	return &capb.OCSPResponse{Response: []byte{1, 2, 3}}, nil
}

var log = blog.UseMock()

func setup(t *testing.T) (*OCSPUpdater, sapb.StorageAuthorityClient, *db.WrappedMap, clock.FakeClock, func()) {
	dbMap, err := sa.NewDbMap(vars.DBConnSA, sa.DbSettings{})
	test.AssertNotError(t, err, "Failed to create dbMap")
	readOnlyDb, err := sa.NewDbMap(vars.DBConnSAOcspUpdateRO, sa.DbSettings{})
	test.AssertNotError(t, err, "Failed to create dbMap")
	cleanUp := test.ResetBoulderTestDatabase(t)
	sa.SetSQLDebug(dbMap, log)

	fc := clock.NewFake()
	fc.Add(1 * time.Hour)

	ssa, err := sa.NewSQLStorageAuthority(dbMap, dbMap, nil, 1, fc, log, metrics.NoopRegisterer)
	test.AssertNotError(t, err, "Failed to create SA")

	updater, err := New(
		metrics.NoopRegisterer,
		fc,
		dbMap,
		readOnlyDb,
		strings.Fields("0 1 2 3 4 5 6 7 8 9 a b c d e f"),
		&mockOCSP{},
		1,
		time.Second,
		time.Minute,
		1.5,
		0,
		0,
		blog.NewMock(),
	)
	test.AssertNotError(t, err, "Failed to create newUpdater")

	return updater, isa.SA{Impl: ssa}, dbMap, fc, cleanUp
}

func nowNano(fc clock.Clock) int64 {
	return fc.Now().UnixNano()
}

func TestStalenessHistogram(t *testing.T) {
	updater, sac, _, fc, cleanUp := setup(t)
	defer cleanUp()

	reg := satest.CreateWorkingRegistration(t, sac)
	parsedCertA, err := core.LoadCert("testdata/test-cert.pem")
	test.AssertNotError(t, err, "Couldn't read test certificate")
	_, err = sac.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      parsedCertA.Raw,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   nowNano(fc),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add test-cert.pem")
	parsedCertB, err := core.LoadCert("testdata/test-cert-b.pem")
	test.AssertNotError(t, err, "Couldn't read test certificate")
	_, err = sac.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      parsedCertB.Raw,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   nowNano(fc),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add test-cert-b.pem")

	// Jump time forward by 2 hours so the ocspLastUpdate value will be older than
	// the earliest lastUpdate time we care about.
	fc.Set(fc.Now().Add(2 * time.Hour))
	earliest := fc.Now().Add(-time.Hour)

	// We should have 2 stale responses now.
	metas := updater.findStaleOCSPResponses(ctx, earliest, 10)
	var metaSlice []*sa.CertStatusMetadata
	for status := range metas {
		metaSlice = append(metaSlice, status)
	}
	test.AssertEquals(t, updater.readFailures.Value(), 0)
	test.AssertEquals(t, len(metaSlice), 2)

	test.AssertMetricWithLabelsEquals(t, updater.stalenessHistogram, prometheus.Labels{}, 2)
}

func TestGenerateAndStoreOCSPResponse(t *testing.T) {
	updater, sa, _, fc, cleanUp := setup(t)
	defer cleanUp()

	reg := satest.CreateWorkingRegistration(t, sa)
	parsedCert, err := core.LoadCert("testdata/test-cert.pem")
	test.AssertNotError(t, err, "Couldn't read test certificate")
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      parsedCert.Raw,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   nowNano(fc),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add test-cert.pem")

	fc.Set(fc.Now().Add(2 * time.Hour))
	earliest := fc.Now().Add(-time.Hour)
	metas := findStaleOCSPResponsesBuffered(ctx, updater, earliest)
	test.AssertEquals(t, updater.readFailures.Value(), 0)
	test.AssertEquals(t, len(metas), 1)
	meta := <-metas

	status, err := updater.generateResponse(ctx, meta)
	test.AssertNotError(t, err, "Couldn't generate OCSP response")
	err = updater.storeResponse(context.Background(), status)
	test.AssertNotError(t, err, "Couldn't store certificate status")
}

// findStaleOCSPResponsesBuffered runs findStaleOCSPResponses and returns
// it as a buffered channel. This is helpful for tests that want to test
// the length of the channel.
func findStaleOCSPResponsesBuffered(ctx context.Context, updater *OCSPUpdater, earliest time.Time) <-chan *sa.CertStatusMetadata {
	out := make(chan *sa.CertStatusMetadata, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(out)
		metas := updater.findStaleOCSPResponses(ctx, earliest, 10)
		for meta := range metas {
			out <- meta
		}
	}()
	wg.Wait()
	return out
}

func TestGenerateOCSPResponses(t *testing.T) {
	updater, sa, _, fc, cleanUp := setup(t)
	defer cleanUp()

	reg := satest.CreateWorkingRegistration(t, sa)
	parsedCertA, err := core.LoadCert("testdata/test-cert.pem")
	test.AssertNotError(t, err, "Couldn't read test certificate")
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      parsedCertA.Raw,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   nowNano(fc),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add test-cert.pem")
	parsedCertB, err := core.LoadCert("testdata/test-cert-b.pem")
	test.AssertNotError(t, err, "Couldn't read test certificate")
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      parsedCertB.Raw,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   nowNano(fc),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add test-cert-b.pem")

	// Jump time forward by 2 hours so the ocspLastUpdate value will be older than
	// the earliest lastUpdate time we care about.
	fc.Set(fc.Now().Add(2 * time.Hour))
	earliest := fc.Now().Add(-time.Hour)

	// We should have 2 stale responses now.
	statuses := findStaleOCSPResponsesBuffered(ctx, updater, earliest)
	test.AssertEquals(t, updater.readFailures.Value(), 0)
	test.AssertEquals(t, len(statuses), 2)

	// Hacky test of parallelism: Make each request to the CA take 1 second, and
	// produce 2 requests to the CA. If the pair of requests complete in about a
	// second, they were made in parallel.
	// Note that this test also tests the basic functionality of
	// generateOCSPResponses.
	start := time.Now()
	updater.ogc = &mockOCSP{time.Second}
	updater.parallelGenerateOCSPRequests = 10
	updater.generateOCSPResponses(ctx, statuses)
	elapsed := time.Since(start)
	if elapsed > 1500*time.Millisecond {
		t.Errorf("generateOCSPResponses took too long, expected it to make calls in parallel.")
	}

	// generateOCSPResponses should have updated the ocspLastUpdate for each
	// cert, so there shouldn't be any stale responses anymore.
	statuses = findStaleOCSPResponsesBuffered(ctx, updater, earliest)

	test.AssertEquals(t, updater.readFailures.Value(), 0)
	test.AssertEquals(t, len(statuses), 0)
}

func TestFindStaleOCSPResponses(t *testing.T) {
	updater, sa, _, fc, cleanUp := setup(t)
	defer cleanUp()

	// With no rows in the CertificateStatus table we shouldn't get an error.
	statuses := findStaleOCSPResponsesBuffered(ctx, updater, fc.Now())
	test.AssertEquals(t, updater.readFailures.Value(), 0)
	test.AssertEquals(t, len(statuses), 0)

	reg := satest.CreateWorkingRegistration(t, sa)
	parsedCert, err := core.LoadCert("testdata/test-cert.pem")
	test.AssertNotError(t, err, "Couldn't read test certificate")
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      parsedCert.Raw,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   nowNano(fc),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add test-cert.pem")

	// Jump time forward by 2 hours so the ocspLastUpdate value will be older than
	// the earliest lastUpdate time we care about.
	fc.Set(fc.Now().Add(2 * time.Hour))
	earliest := fc.Now().Add(-time.Hour)

	// We should have 1 stale response now.
	statuses = findStaleOCSPResponsesBuffered(ctx, updater, earliest)
	test.AssertEquals(t, updater.readFailures.Value(), 0)
	test.AssertEquals(t, len(statuses), 1)
	status := <-statuses

	// Generate and store an updated response, which will update the
	// ocspLastUpdate field for this cert.
	meta, err := updater.generateResponse(ctx, status)
	test.AssertNotError(t, err, "Couldn't generate OCSP response")
	err = updater.storeResponse(context.Background(), meta)
	test.AssertNotError(t, err, "Couldn't store OCSP response")

	// We should have 0 stale responses now.
	statuses = findStaleOCSPResponsesBuffered(ctx, updater, earliest)
	test.AssertEquals(t, updater.readFailures.Value(), 0)
	test.AssertEquals(t, len(statuses), 0)
}

func TestFindStaleOCSPResponsesRevokedReason(t *testing.T) {
	updater, sa, dbMap, fc, cleanUp := setup(t)
	defer cleanUp()

	reg := satest.CreateWorkingRegistration(t, sa)
	parsedCert, err := core.LoadCert("testdata/test-cert.pem")
	test.AssertNotError(t, err, "Couldn't read test certificate")
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      parsedCert.Raw,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   nowNano(fc),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add test-cert.pem")

	// Set a revokedReason to ensure it gets written into the OCSPResponse.
	_, err = dbMap.Exec(
		"UPDATE certificateStatus SET revokedReason = 1 WHERE serial = ?",
		core.SerialToString(parsedCert.SerialNumber))
	test.AssertNotError(t, err, "Couldn't update revokedReason")

	// Jump time forward by 2 hours so the ocspLastUpdate value will be older than
	// the earliest lastUpdate time we care about.
	fc.Set(fc.Now().Add(2 * time.Hour))
	earliest := fc.Now().Add(-time.Hour)

	statuses := findStaleOCSPResponsesBuffered(ctx, updater, earliest)
	test.AssertEquals(t, updater.readFailures.Value(), 0)
	test.AssertEquals(t, len(statuses), 1)
	status := <-statuses
	test.AssertEquals(t, int(status.RevokedReason), 1)
}

func TestPipelineTick(t *testing.T) {
	updater, sa, _, fc, cleanUp := setup(t)
	defer cleanUp()

	reg := satest.CreateWorkingRegistration(t, sa)
	parsedCert, err := core.LoadCert("testdata/test-cert.pem")
	test.AssertNotError(t, err, "Couldn't read test certificate")
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      parsedCert.Raw,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   nowNano(fc),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add test-cert.pem")

	updater.ocspMinTimeToExpiry = 1 * time.Hour
	earliest := fc.Now().Add(-time.Hour)
	updater.generateOCSPResponses(ctx, updater.processExpired(ctx, updater.findStaleOCSPResponses(ctx, earliest, 10)))
	test.AssertEquals(t, updater.readFailures.Value(), 0)

	certs := findStaleOCSPResponsesBuffered(ctx, updater, fc.Now().Add(-updater.ocspMinTimeToExpiry))
	test.AssertEquals(t, updater.readFailures.Value(), 0)
	test.AssertEquals(t, len(certs), 0)
}

// TestProcessExpired checks that the `processExpired` pipeline step
// updates the `IsExpired` field opportunistically as it encounters
// certificates that are expired but whose certificate status rows do not
// have `IsExpired` set, and that expired certs don't show up as having
// stale responses.
func TestProcessExpired(t *testing.T) {
	updater, sa, _, fc, cleanUp := setup(t)
	defer cleanUp()

	reg := satest.CreateWorkingRegistration(t, sa)
	parsedCert, err := core.LoadCert("testdata/test-cert.pem")
	test.AssertNotError(t, err, "Couldn't read test certificate")
	serial := core.SerialToString(parsedCert.SerialNumber)

	// Add a new test certificate
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      parsedCert.Raw,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   nowNano(fc),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add test-cert.pem")

	// Jump time forward by 2 hours so the ocspLastUpdate value will be older than
	// the earliest lastUpdate time we care about.
	fc.Set(fc.Now().Add(2 * time.Hour))
	earliest := fc.Now().Add(-time.Hour)

	// The certificate isn't expired, so the certificate status should have
	// a false `IsExpired` and it should show up as stale.
	statusPB, err := sa.GetCertificateStatus(ctx, &sapb.Serial{Serial: serial})
	test.AssertNotError(t, err, "Couldn't get the certificateStatus from the database")
	cs, err := bgrpc.PBToCertStatus(statusPB)
	test.AssertNotError(t, err, "Count't convert the certificateStatus from a PB")

	test.AssertEquals(t, cs.IsExpired, false)
	statuses := findStaleOCSPResponsesBuffered(ctx, updater, earliest)
	test.AssertEquals(t, updater.readFailures.Value(), 0)
	test.AssertEquals(t, len(statuses), 1)

	// Advance the clock to the point that the certificate we added is now expired
	fc.Set(parsedCert.NotAfter.Add(2 * time.Hour))
	earliest = fc.Now().Add(-time.Hour)
	updater.ocspMinTimeToExpiry = 1 * time.Hour

	// Run pipeline to find stale responses, mark expired, and generate new response.
	updater.generateOCSPResponses(ctx, updater.processExpired(ctx, updater.findStaleOCSPResponses(ctx, earliest, 10)))

	// Since we advanced the fakeclock beyond our test certificate's NotAfter we
	// expect the certificate status has been updated to have a true `IsExpired`
	statusPB, err = sa.GetCertificateStatus(ctx, &sapb.Serial{Serial: serial})
	test.AssertNotError(t, err, "Couldn't get the certificateStatus from the database")
	cs, err = bgrpc.PBToCertStatus(statusPB)
	test.AssertNotError(t, err, "Count't convert the certificateStatus from a PB")

	test.AssertEquals(t, cs.IsExpired, true)
	statuses = findStaleOCSPResponsesBuffered(ctx, updater, earliest)
	test.AssertEquals(t, updater.readFailures.Value(), 0)
	test.AssertEquals(t, len(statuses), 0)
}

func TestStoreResponseGuard(t *testing.T) {
	updater, sa, _, fc, cleanUp := setup(t)
	defer cleanUp()

	reg := satest.CreateWorkingRegistration(t, sa)
	parsedCert, err := core.LoadCert("testdata/test-cert.pem")
	test.AssertNotError(t, err, "Couldn't read test certificate")
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      parsedCert.Raw,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   nowNano(fc),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add test-cert.pem")

	fc.Set(fc.Now().Add(2 * time.Hour))
	earliest := fc.Now().Add(-time.Hour)
	metas := findStaleOCSPResponsesBuffered(ctx, updater, earliest)
	test.AssertEquals(t, updater.readFailures.Value(), 0)
	test.AssertEquals(t, len(metas), 1)
	meta := <-metas

	serialStr := core.SerialToString(parsedCert.SerialNumber)
	reason := int64(0)
	revokedDate := fc.Now().UnixNano()
	_, err = sa.RevokeCertificate(context.Background(), &sapb.RevokeCertificateRequest{
		Serial:   serialStr,
		Reason:   reason,
		Date:     revokedDate,
		Response: []byte("fakeocspbytes"),
	})
	test.AssertNotError(t, err, "Failed to revoked certificate")

	// Attempt to update OCSP response where status.Status is good but stored status
	// is revoked, this should fail silently
	status := statusFromMetaAndResp(meta, []byte("newfakeocspbytes"))
	err = updater.storeResponse(context.Background(), status)
	test.AssertNotError(t, err, "Failed to update certificate status")

	// Make sure the OCSP response hasn't actually changed
	unchangedStatus, err := sa.GetCertificateStatus(ctx, &sapb.Serial{Serial: core.SerialToString(parsedCert.SerialNumber)})
	test.AssertNotError(t, err, "Failed to get certificate status")
	test.AssertEquals(t, string(unchangedStatus.OcspResponse), "fakeocspbytes")

	// Changing the status to the stored status should allow the update to occur
	status.Status = core.OCSPStatusRevoked
	err = updater.storeResponse(context.Background(), status)
	test.AssertNotError(t, err, "Failed to updated certificate status")

	// Make sure the OCSP response has been updated
	changedStatus, err := sa.GetCertificateStatus(ctx, &sapb.Serial{Serial: core.SerialToString(parsedCert.SerialNumber)})
	test.AssertNotError(t, err, "Failed to get certificate status")
	test.AssertEquals(t, string(changedStatus.OcspResponse), "newfakeocspbytes")
}

func TestGenerateOCSPResponsePrecert(t *testing.T) {
	updater, sa, _, fc, cleanUp := setup(t)
	defer cleanUp()

	reg := satest.CreateWorkingRegistration(t, sa)

	// Create a throw-away self signed certificate with some names
	serial, testCert := test.ThrowAwayCert(t, 5)

	// Use AddPrecertificate to set up a precertificate, serials, and
	// certificateStatus row for the testcert.
	ocspResp := []byte{0, 0, 1}
	regID := reg.Id
	issuedTime := fc.Now().UnixNano()
	_, err := sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      testCert.Raw,
		RegID:    regID,
		Ocsp:     ocspResp,
		Issued:   issuedTime,
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add test-cert2.der")

	// Jump time forward by 2 hours so the ocspLastUpdate value will be older than
	// the earliest lastUpdate time we care about.
	fc.Set(fc.Now().Add(2 * time.Hour))
	earliest := fc.Now().Add(-time.Hour)

	// There should be one stale ocsp response found for the precert
	certs := findStaleOCSPResponsesBuffered(ctx, updater, earliest)
	test.AssertEquals(t, updater.readFailures.Value(), 0)
	test.AssertEquals(t, len(certs), 1)
	cert := <-certs
	test.AssertEquals(t, cert.Serial, serial)

	// Directly call generateResponse again with the same result. It should not
	// error and should instead update the precertificate's OCSP status even
	// though no certificate row exists.
	_, err = updater.generateResponse(ctx, cert)
	test.AssertNotError(t, err, "generateResponse for precert errored")
}

type mockOCSPRecordIssuer struct {
	gotIssuer bool
}

func (ca *mockOCSPRecordIssuer) GenerateOCSP(_ context.Context, req *capb.GenerateOCSPRequest, _ ...grpc.CallOption) (*capb.OCSPResponse, error) {
	ca.gotIssuer = req.IssuerID != 0 && req.Serial != ""
	return &capb.OCSPResponse{Response: []byte{1, 2, 3}}, nil
}

func TestIssuerInfo(t *testing.T) {
	updater, sa, _, fc, cleanUp := setup(t)
	defer cleanUp()
	m := mockOCSPRecordIssuer{}
	updater.ogc = &m
	reg := satest.CreateWorkingRegistration(t, sa)

	k, err := rsa.GenerateKey(rand.Reader, 512)
	test.AssertNotError(t, err, "rsa.GenerateKey failed")
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{"example.com"},
	}
	certA, err := x509.CreateCertificate(rand.Reader, template, template, &k.PublicKey, k)
	test.AssertNotError(t, err, "x509.CreateCertificate failed")

	now := fc.Now().UnixNano()
	id := int64(1234)
	_, err = sa.AddPrecertificate(context.Background(), &sapb.AddCertificateRequest{
		Der:      certA,
		RegID:    reg.Id,
		Ocsp:     []byte{1, 2, 3},
		Issued:   now,
		IssuerID: id,
	})
	test.AssertNotError(t, err, "sa.AddPrecertificate failed")

	fc.Add(time.Hour * 24 * 4)
	statuses := findStaleOCSPResponsesBuffered(ctx, updater, fc.Now().Add(-time.Hour))

	test.AssertEquals(t, updater.readFailures.Value(), 0)
	test.AssertEquals(t, len(statuses), 1)
	status := <-statuses
	test.AssertEquals(t, status.IssuerID, id)

	_, err = updater.generateResponse(context.Background(), status)
	test.AssertNotError(t, err, "generateResponse failed")
	test.Assert(t, m.gotIssuer, "generateResponse didn't send issuer information and serial")
}

type brokenDB struct{}

func (bdb *brokenDB) TableFor(_ reflect.Type, _ bool) (*gorp.TableMap, error) {
	return nil, errors.New("broken")
}

func (bdb *brokenDB) WithContext(_ context.Context) gorp.SqlExecutor {
	return nil
}

func TestTickSleep(t *testing.T) {
	updater, _, dbMap, fc, cleanUp := setup(t)
	defer cleanUp()

	// Test that when findStaleResponses fails the failure counter is
	// incremented and the clock moved forward by more than
	// updater.tickWindow
	updater.readOnlyDb = &brokenDB{}
	updater.readFailures.Add(2)
	before := fc.Now()
	updater.Tick()
	test.AssertEquals(t, updater.readFailures.Value(), 3)
	took := fc.Since(before)
	test.Assert(t, took > updater.tickWindow, "Clock didn't move forward enough")

	// Test when findStaleResponses works the failure counter is reset to
	// zero and the clock only moves by updater.tickWindow
	updater.readOnlyDb = dbMap
	before = fc.Now()
	updater.Tick()
	test.AssertEquals(t, updater.readFailures.Value(), 0)
	took = fc.Since(before)
	test.AssertEquals(t, took, updater.tickWindow)

}

func TestFindOCSPResponsesSleep(t *testing.T) {
	updater, _, dbMap, fc, cleanUp := setup(t)
	defer cleanUp()
	m := &brokenDB{}
	updater.readOnlyDb = m

	// Test when updateOCSPResponses fails the failure counter is incremented
	// and the clock moved forward by more than updater.tickWindow
	updater.readFailures.Add(2)
	before := fc.Now()
	updater.Tick()
	test.AssertEquals(t, updater.readFailures.Value(), 3)
	took := fc.Since(before)
	test.Assert(t, took > updater.tickWindow, "Clock didn't move forward enough")

	// Test when updateOCSPResponses works the failure counter is reset to zero
	// and the clock only moves by updater.tickWindow
	updater.readOnlyDb = dbMap
	before = fc.Now()
	updater.Tick()
	test.AssertEquals(t, updater.readFailures.Value(), 0)
	took = fc.Since(before)
	test.AssertEquals(t, took, updater.tickWindow)

}

func mkNewUpdaterWithStrings(t *testing.T, shards []string) (*OCSPUpdater, error) {
	dbMap, err := sa.NewDbMap(vars.DBConnSA, sa.DbSettings{})
	test.AssertNotError(t, err, "Failed to create dbMap")
	sa.SetSQLDebug(dbMap, log)

	fc := clock.NewFake()

	updater, err := New(
		metrics.NoopRegisterer,
		fc,
		dbMap,
		dbMap,
		shards,
		&mockOCSP{},
		1,
		time.Second,
		time.Minute,
		1.5,
		0,
		0,
		blog.NewMock(),
	)
	return updater, err
}

func TestUpdaterConfiguration(t *testing.T) {
	_, err := mkNewUpdaterWithStrings(t, strings.Fields("0 1 2 3 4 5 6 7 8 9 a B c d e f"))
	test.AssertError(t, err, "No uppercase allowed")

	_, err = mkNewUpdaterWithStrings(t, strings.Fields("0 1 g"))
	test.AssertError(t, err, "No letters > f allowed")

	_, err = mkNewUpdaterWithStrings(t, strings.Fields("0 *"))
	test.AssertError(t, err, "No special chars allowed")

	_, err = mkNewUpdaterWithStrings(t, strings.Fields("0 -1"))
	test.AssertError(t, err, "No negative numbers allowed")

	_, err = mkNewUpdaterWithStrings(t, strings.Fields("wazzup 0 a b c"))
	test.AssertError(t, err, "No multi-letter shards allowed")

	_, err = mkNewUpdaterWithStrings(t, []string{})
	test.AssertNotError(t, err, "Empty should be valid, meaning use old queries")
}

func TestGetQuestionsForShardList(t *testing.T) {
	test.AssertEquals(t, getQuestionsForShardList(2), "?,?")
	test.AssertEquals(t, getQuestionsForShardList(1), "?")
	test.AssertEquals(t, getQuestionsForShardList(16), "?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?")
}
