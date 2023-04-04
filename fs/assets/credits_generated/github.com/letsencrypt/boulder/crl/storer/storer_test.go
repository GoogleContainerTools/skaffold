package storer

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jmhodges/clock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/letsencrypt/boulder/crl/crl_x509"
	cspb "github.com/letsencrypt/boulder/crl/storer/proto"
	"github.com/letsencrypt/boulder/issuance"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/test"
)

type fakeUploadCRLServerStream struct {
	grpc.ServerStream
	input <-chan *cspb.UploadCRLRequest
}

func (s *fakeUploadCRLServerStream) Recv() (*cspb.UploadCRLRequest, error) {
	next, ok := <-s.input
	if !ok {
		return nil, io.EOF
	}
	return next, nil
}

func (s *fakeUploadCRLServerStream) SendAndClose(*emptypb.Empty) error {
	return nil
}

func (s *fakeUploadCRLServerStream) Context() context.Context {
	return context.Background()
}

func setupTestUploadCRL(t *testing.T) (*crlStorer, *issuance.Issuer) {
	t.Helper()

	r3, err := issuance.LoadCertificate("../../test/hierarchy/int-r3.cert.pem")
	test.AssertNotError(t, err, "loading fake RSA issuer cert")
	e1, e1Signer, err := issuance.LoadIssuer(issuance.IssuerLoc{
		File:     "../../test/hierarchy/int-e1.key.pem",
		CertFile: "../../test/hierarchy/int-e1.cert.pem",
	})
	test.AssertNotError(t, err, "loading fake ECDSA issuer cert")

	storer, err := New(
		[]*issuance.Certificate{r3, e1},
		nil, "le-crl.s3.us-west.amazonaws.com",
		metrics.NoopRegisterer, blog.NewMock(), clock.NewFake(),
	)
	test.AssertNotError(t, err, "creating test crl-storer")

	return storer, &issuance.Issuer{Cert: e1, Signer: e1Signer}
}

// Test that we get an error when no metadata is sent.
func TestUploadCRLNoMetadata(t *testing.T) {
	storer, _ := setupTestUploadCRL(t)
	errs := make(chan error, 1)

	ins := make(chan *cspb.UploadCRLRequest)
	go func() {
		errs <- storer.UploadCRL(&fakeUploadCRLServerStream{input: ins})
	}()
	close(ins)
	err := <-errs
	test.AssertError(t, err, "can't upload CRL with no metadata")
	test.AssertContains(t, err.Error(), "no metadata")
}

// Test that we get an error when incomplete metadata is sent.
func TestUploadCRLIncompleteMetadata(t *testing.T) {
	storer, _ := setupTestUploadCRL(t)
	errs := make(chan error, 1)

	ins := make(chan *cspb.UploadCRLRequest)
	go func() {
		errs <- storer.UploadCRL(&fakeUploadCRLServerStream{input: ins})
	}()
	ins <- &cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_Metadata{
			Metadata: &cspb.CRLMetadata{},
		},
	}
	close(ins)
	err := <-errs
	test.AssertError(t, err, "can't upload CRL with incomplete metadata")
	test.AssertContains(t, err.Error(), "incomplete metadata")
}

// Test that we get an error when a bad issuer is sent.
func TestUploadCRLUnrecognizedIssuer(t *testing.T) {
	storer, _ := setupTestUploadCRL(t)
	errs := make(chan error, 1)

	ins := make(chan *cspb.UploadCRLRequest)
	go func() {
		errs <- storer.UploadCRL(&fakeUploadCRLServerStream{input: ins})
	}()
	ins <- &cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_Metadata{
			Metadata: &cspb.CRLMetadata{
				IssuerNameID: 1,
				Number:       1,
			},
		},
	}
	close(ins)
	err := <-errs
	test.AssertError(t, err, "can't upload CRL with unrecognized issuer")
	test.AssertContains(t, err.Error(), "unrecognized")
}

// Test that we get an error when two metadata are sent.
func TestUploadCRLMultipleMetadata(t *testing.T) {
	storer, iss := setupTestUploadCRL(t)
	errs := make(chan error, 1)

	ins := make(chan *cspb.UploadCRLRequest)
	go func() {
		errs <- storer.UploadCRL(&fakeUploadCRLServerStream{input: ins})
	}()
	ins <- &cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_Metadata{
			Metadata: &cspb.CRLMetadata{
				IssuerNameID: int64(iss.Cert.NameID()),
				Number:       1,
			},
		},
	}
	ins <- &cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_Metadata{
			Metadata: &cspb.CRLMetadata{
				IssuerNameID: int64(iss.Cert.NameID()),
				Number:       1,
			},
		},
	}
	close(ins)
	err := <-errs
	test.AssertError(t, err, "can't upload CRL with multiple metadata")
	test.AssertContains(t, err.Error(), "more than one")
}

// Test that we get an error when a malformed CRL is sent.
func TestUploadCRLMalformedBytes(t *testing.T) {
	storer, iss := setupTestUploadCRL(t)
	errs := make(chan error, 1)

	ins := make(chan *cspb.UploadCRLRequest)
	go func() {
		errs <- storer.UploadCRL(&fakeUploadCRLServerStream{input: ins})
	}()
	ins <- &cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_Metadata{
			Metadata: &cspb.CRLMetadata{
				IssuerNameID: int64(iss.Cert.NameID()),
				Number:       1,
			},
		},
	}
	ins <- &cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_CrlChunk{
			CrlChunk: []byte("this is not a valid crl"),
		},
	}
	close(ins)
	err := <-errs
	test.AssertError(t, err, "can't upload unparsable CRL")
	test.AssertContains(t, err.Error(), "parsing CRL")
}

// Test that we get an error when an invalid CRL (signed by a throwaway
// private key but tagged as being from a "real" issuer) is sent.
func TestUploadCRLInvalidSignature(t *testing.T) {
	storer, iss := setupTestUploadCRL(t)
	errs := make(chan error, 1)

	ins := make(chan *cspb.UploadCRLRequest)
	go func() {
		errs <- storer.UploadCRL(&fakeUploadCRLServerStream{input: ins})
	}()
	ins <- &cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_Metadata{
			Metadata: &cspb.CRLMetadata{
				IssuerNameID: int64(iss.Cert.NameID()),
				Number:       1,
			},
		},
	}
	fakeSigner, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	test.AssertNotError(t, err, "creating throwaway signer")
	crlBytes, err := crl_x509.CreateRevocationList(
		rand.Reader,
		&crl_x509.RevocationList{
			ThisUpdate: time.Now(),
			NextUpdate: time.Now().Add(time.Hour),
			Number:     big.NewInt(1),
		},
		iss.Cert.Certificate,
		fakeSigner,
	)
	test.AssertNotError(t, err, "creating test CRL")
	ins <- &cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_CrlChunk{
			CrlChunk: crlBytes,
		},
	}
	close(ins)
	err = <-errs
	test.AssertError(t, err, "can't upload unverifiable CRL")
	test.AssertContains(t, err.Error(), "validating signature")
}

// Test that we get an error if the CRL Numbers mismatch.
func TestUploadCRLMismatchedNumbers(t *testing.T) {
	storer, iss := setupTestUploadCRL(t)
	errs := make(chan error, 1)

	ins := make(chan *cspb.UploadCRLRequest)
	go func() {
		errs <- storer.UploadCRL(&fakeUploadCRLServerStream{input: ins})
	}()
	ins <- &cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_Metadata{
			Metadata: &cspb.CRLMetadata{
				IssuerNameID: int64(iss.Cert.NameID()),
				Number:       1,
			},
		},
	}
	crlBytes, err := crl_x509.CreateRevocationList(
		rand.Reader,
		&crl_x509.RevocationList{
			ThisUpdate: time.Now(),
			NextUpdate: time.Now().Add(time.Hour),
			Number:     big.NewInt(2),
		},
		iss.Cert.Certificate,
		iss.Signer,
	)
	test.AssertNotError(t, err, "creating test CRL")
	ins <- &cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_CrlChunk{
			CrlChunk: crlBytes,
		},
	}
	close(ins)
	err = <-errs
	test.AssertError(t, err, "can't upload CRL with mismatched number")
	test.AssertContains(t, err.Error(), "mismatched")
}

type fakeS3Putter struct {
	expectBytes []byte
}

func (p *fakeS3Putter) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	recvBytes, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(p.expectBytes, recvBytes) {
		return nil, errors.New("received bytes did not match expectation")
	}
	return &s3.PutObjectOutput{}, nil
}

// Test that the correct bytes get propagated to S3.
func TestUploadCRLSuccess(t *testing.T) {
	storer, iss := setupTestUploadCRL(t)
	errs := make(chan error, 1)

	ins := make(chan *cspb.UploadCRLRequest)
	go func() {
		errs <- storer.UploadCRL(&fakeUploadCRLServerStream{input: ins})
	}()
	ins <- &cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_Metadata{
			Metadata: &cspb.CRLMetadata{
				IssuerNameID: int64(iss.Cert.NameID()),
				Number:       1,
			},
		},
	}
	crlBytes, err := crl_x509.CreateRevocationList(
		rand.Reader,
		&crl_x509.RevocationList{
			ThisUpdate: time.Now(),
			NextUpdate: time.Now().Add(time.Hour),
			Number:     big.NewInt(1),
			RevokedCertificates: []crl_x509.RevokedCertificate{
				{SerialNumber: big.NewInt(123), RevocationTime: time.Now().Add(-time.Hour)},
			},
		},
		iss.Cert.Certificate,
		iss.Signer,
	)
	test.AssertNotError(t, err, "creating test CRL")
	storer.s3Client = &fakeS3Putter{expectBytes: crlBytes}
	ins <- &cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_CrlChunk{
			CrlChunk: crlBytes,
		},
	}
	close(ins)
	err = <-errs
	test.AssertNotError(t, err, "uploading valid CRL should work")
}

type brokenS3Putter struct{}

func (p *brokenS3Putter) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return nil, errors.New("sorry")
}

// Test that we get an error when S3 falls over.
func TestUploadCRLBrokenS3(t *testing.T) {
	storer, iss := setupTestUploadCRL(t)
	errs := make(chan error, 1)

	ins := make(chan *cspb.UploadCRLRequest)
	go func() {
		errs <- storer.UploadCRL(&fakeUploadCRLServerStream{input: ins})
	}()
	ins <- &cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_Metadata{
			Metadata: &cspb.CRLMetadata{
				IssuerNameID: int64(iss.Cert.NameID()),
				Number:       1,
			},
		},
	}
	crlBytes, err := crl_x509.CreateRevocationList(
		rand.Reader,
		&crl_x509.RevocationList{
			ThisUpdate: time.Now(),
			NextUpdate: time.Now().Add(time.Hour),
			Number:     big.NewInt(1),
			RevokedCertificates: []crl_x509.RevokedCertificate{
				{SerialNumber: big.NewInt(123), RevocationTime: time.Now().Add(-time.Hour)},
			},
		},
		iss.Cert.Certificate,
		iss.Signer,
	)
	test.AssertNotError(t, err, "creating test CRL")
	storer.s3Client = &brokenS3Putter{}
	ins <- &cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_CrlChunk{
			CrlChunk: crlBytes,
		},
	}
	close(ins)
	err = <-errs
	test.AssertError(t, err, "uploading to broken S3 should fail")
	test.AssertContains(t, err.Error(), "uploading to S3")
}
