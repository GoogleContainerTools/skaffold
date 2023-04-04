package notmain

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/jmhodges/clock"
	capb "github.com/letsencrypt/boulder/ca/proto"
	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	berrors "github.com/letsencrypt/boulder/errors"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	"github.com/letsencrypt/boulder/issuance"
	blog "github.com/letsencrypt/boulder/log"
	sapb "github.com/letsencrypt/boulder/sa/proto"
	"github.com/letsencrypt/boulder/test"
)

type mockSA struct {
	certificates    []*corepb.Certificate
	precertificates []core.Certificate
	clk             clock.FakeClock
}

func (m *mockSA) AddCertificate(ctx context.Context, req *sapb.AddCertificateRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	parsed, err := x509.ParseCertificate(req.Der)
	if err != nil {
		return nil, err
	}
	cert := &corepb.Certificate{
		Der:            req.Der,
		RegistrationID: req.RegID,
		Serial:         core.SerialToString(parsed.SerialNumber),
		Issued:         req.Issued,
	}
	m.certificates = append(m.certificates, cert)
	return nil, nil
}

func (m *mockSA) GetCertificate(ctx context.Context, req *sapb.Serial, _ ...grpc.CallOption) (*corepb.Certificate, error) {
	if len(m.certificates) == 0 {
		return nil, berrors.NotFoundError("no certs stored")
	}
	for _, cert := range m.certificates {
		if cert.Serial == req.Serial {
			return cert, nil
		}
	}
	return nil, berrors.NotFoundError("no cert stored for requested serial")
}

func (m *mockSA) AddPrecertificate(ctx context.Context, req *sapb.AddCertificateRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	if core.IsAnyNilOrZero(req.Der, req.Issued, req.RegID, req.IssuerID) {
		return nil, berrors.InternalServerError("Incomplete request")
	}
	parsed, err := x509.ParseCertificate(req.Der)
	if err != nil {
		return nil, err
	}
	precert := core.Certificate{
		DER:            req.Der,
		RegistrationID: req.RegID,
		Serial:         core.SerialToString(parsed.SerialNumber),
	}
	if req.Issued == 0 {
		precert.Issued = m.clk.Now()
	} else {
		precert.Issued = time.Unix(0, req.Issued)
	}
	m.precertificates = append(m.precertificates, precert)
	return &emptypb.Empty{}, nil
}

func (m *mockSA) GetPrecertificate(ctx context.Context, req *sapb.Serial, _ ...grpc.CallOption) (*corepb.Certificate, error) {
	if len(m.precertificates) == 0 {
		return nil, berrors.NotFoundError("no precerts stored")
	}
	for _, precert := range m.precertificates {
		if precert.Serial == req.Serial {
			return bgrpc.CertToPB(precert), nil
		}
	}
	return nil, berrors.NotFoundError("no precert stored for requested serial")
}

func (m *mockSA) AddSerial(ctx context.Context, req *sapb.AddSerialRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

type mockOCSPA struct{}

func (ca *mockOCSPA) GenerateOCSP(context.Context, *capb.GenerateOCSPRequest, ...grpc.CallOption) (*capb.OCSPResponse, error) {
	return &capb.OCSPResponse{
		Response: []byte("HI"),
	}, nil
}

func TestParseLine(t *testing.T) {
	issuer, err := issuance.LoadCertificate("../../test/hierarchy/int-e1.cert.pem")
	test.AssertNotError(t, err, "failed to load test issuer")
	signer, err := test.LoadSigner("../../test/hierarchy/int-e1.key.pem")
	test.AssertNotError(t, err, "failed to load test signer")
	cert, err := core.LoadCert("../../test/hierarchy/ee-e1.cert.pem")
	test.AssertNotError(t, err, "failed to load test cert")
	certStr := hex.EncodeToString(cert.Raw)
	precertTmpl := x509.Certificate{
		SerialNumber: big.NewInt(0),
		NotBefore:    time.Now(),
		ExtraExtensions: []pkix.Extension{
			{Id: asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 4, 3}, Critical: true, Value: []byte{0x05, 0x00}},
		},
	}
	precertDER, err := x509.CreateCertificate(rand.Reader, &precertTmpl, issuer.Certificate, signer.Public(), signer)
	test.AssertNotError(t, err, "failed to generate test precert")
	precertStr := hex.EncodeToString(precertDER)

	opf := &orphanFinder{
		sa:       &mockSA{},
		ca:       &mockOCSPA{},
		logger:   blog.UseMock(),
		issuers:  map[issuance.IssuerNameID]*issuance.Certificate{issuer.NameID(): issuer},
		backdate: time.Hour,
	}

	logLine := func(typ orphanType, der, issuerID, regID, orderID string) string {
		return fmt.Sprintf(
			"0000-00-00T00:00:00+00:00 hostname boulder-ca[pid]: "+
				"[AUDIT] Failed RPC to store at SA, orphaning %s: "+
				"serial=[unused], cert=[%s], issuerID=[%s], regID=[%s], orderID=[%s], err=[context deadline exceeded]",
			typ, der, issuerID, regID, orderID)
	}

	testCases := []struct {
		Name           string
		LogLine        string
		ExpectFound    bool
		ExpectAdded    bool
		ExpectNoErrors bool
		ExpectAddedDER string
		ExpectRegID    int
	}{
		{
			Name:           "Empty line",
			LogLine:        "",
			ExpectFound:    false,
			ExpectAdded:    false,
			ExpectNoErrors: false,
		},
		{
			Name:           "Empty cert in line",
			LogLine:        logLine(certOrphan, "", "1", "1337", "0"),
			ExpectFound:    true,
			ExpectAdded:    false,
			ExpectNoErrors: false,
		},
		{
			Name:           "Invalid cert in line",
			LogLine:        logLine(certOrphan, "deadbeef", "", "", ""),
			ExpectFound:    true,
			ExpectAdded:    false,
			ExpectNoErrors: false,
		},
		{
			Name:           "Valid cert in line",
			LogLine:        logLine(certOrphan, certStr, "1", "1001", "0"),
			ExpectFound:    true,
			ExpectAdded:    true,
			ExpectAddedDER: certStr,
			ExpectRegID:    1001,
			ExpectNoErrors: true,
		},
		{
			Name:        "Already inserted cert in line",
			LogLine:     logLine(certOrphan, certStr, "1", "1001", "0"),
			ExpectFound: true,
			// ExpectAdded is false because we have already added this cert in the
			// previous "Valid cert in line" test case.
			ExpectAdded:    false,
			ExpectNoErrors: true,
		},
		{
			Name:           "Empty precert in line",
			LogLine:        logLine(precertOrphan, "", "1", "1337", "0"),
			ExpectFound:    true,
			ExpectAdded:    false,
			ExpectNoErrors: false,
		},
		{
			Name:           "Invalid precert in line",
			LogLine:        logLine(precertOrphan, "deadbeef", "", "", ""),
			ExpectFound:    true,
			ExpectAdded:    false,
			ExpectNoErrors: false,
		},
		{
			Name:           "Valid precert in line",
			LogLine:        logLine(precertOrphan, precertStr, "1", "9999", "0"),
			ExpectFound:    true,
			ExpectAdded:    true,
			ExpectAddedDER: precertStr,
			ExpectRegID:    9999,
			ExpectNoErrors: true,
		},
		{
			Name:        "Already inserted precert in line",
			LogLine:     logLine(precertOrphan, precertStr, "1", "1001", "0"),
			ExpectFound: true,
			// ExpectAdded is false because we have already added this cert in the
			// previous "Valid cert in line" test case.
			ExpectAdded:    false,
			ExpectNoErrors: true,
		},
		{
			Name:           "Unknown orphan type",
			LogLine:        logLine(unknownOrphan, precertStr, "1", "1001", "0"),
			ExpectFound:    false,
			ExpectAdded:    false,
			ExpectNoErrors: false,
		},
		{
			Name:           "Empty issuerID in line",
			LogLine:        logLine(precertOrphan, precertStr, "", "1001", "0"),
			ExpectFound:    true,
			ExpectAdded:    false,
			ExpectNoErrors: false,
		},
		{
			Name:           "Zero issuerID in line",
			LogLine:        logLine(precertOrphan, precertStr, "0", "1001", "0"),
			ExpectFound:    true,
			ExpectAdded:    false,
			ExpectNoErrors: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()
			opf.logger.(*blog.Mock).Clear()
			found, added, typ := opf.storeLogLine(ctx, tc.LogLine)
			test.AssertEquals(t, found, tc.ExpectFound)
			test.AssertEquals(t, added, tc.ExpectAdded)
			logs := opf.logger.(*blog.Mock).GetAllMatching("ERR:")
			if tc.ExpectNoErrors {
				test.AssertEquals(t, len(logs), 0)
			}

			if tc.ExpectAdded {
				// Decode the precert/cert DER we expect the testcase added to get the
				// certificate serial
				der, _ := hex.DecodeString(tc.ExpectAddedDER)
				testCert, _ := x509.ParseCertificate(der)
				testCertSerial := core.SerialToString(testCert.SerialNumber)

				// Fetch the precert/cert using the correct mock SA function
				var storedCert *corepb.Certificate
				switch typ {
				case precertOrphan:
					storedCert, err = opf.sa.GetPrecertificate(ctx, &sapb.Serial{Serial: testCertSerial})
					test.AssertNotError(t, err, "Error getting test precert serial from SA")
				case certOrphan:
					storedCert, err = opf.sa.GetCertificate(ctx, &sapb.Serial{Serial: testCertSerial})
					test.AssertNotError(t, err, "Error getting test cert serial from SA")
				default:
					t.Fatalf("unknown orphan type returned: %s", typ)
				}
				// The orphan should have been added with the correct registration ID from the log line
				test.AssertEquals(t, storedCert.RegistrationID, int64(tc.ExpectRegID))
				// The Issued timestamp should be the certificate's NotBefore timestamp offset by the backdate
				expectedIssued := testCert.NotBefore.Add(opf.backdate).UnixNano()
				test.AssertEquals(t, storedCert.Issued, expectedIssued)
			}
		})
	}
}

func TestNotOrphan(t *testing.T) {
	ctx := context.Background()
	opf := &orphanFinder{
		sa:       &mockSA{},
		ca:       &mockOCSPA{},
		logger:   blog.UseMock(),
		backdate: time.Hour,
	}

	found, added, typ := opf.storeLogLine(ctx, "cert=fakeout")
	test.AssertEquals(t, found, false)
	test.AssertEquals(t, added, false)
	test.AssertEquals(t, typ, unknownOrphan)
	logs := opf.logger.(*blog.Mock).GetAllMatching("ERR:")
	if len(logs) != 0 {
		t.Error("Found error logs:")
		for _, ll := range logs {
			t.Error(ll)
		}
	}
}
