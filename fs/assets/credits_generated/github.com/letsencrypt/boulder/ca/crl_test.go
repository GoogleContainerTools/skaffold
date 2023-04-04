package ca

import (
	"crypto/x509"
	"io"
	"testing"
	"time"

	"google.golang.org/grpc"

	capb "github.com/letsencrypt/boulder/ca/proto"
	corepb "github.com/letsencrypt/boulder/core/proto"
	"github.com/letsencrypt/boulder/test"
)

type mockGenerateCRLBidiStream struct {
	grpc.ServerStream
	input  <-chan *capb.GenerateCRLRequest
	output chan<- *capb.GenerateCRLResponse
}

func (s mockGenerateCRLBidiStream) Recv() (*capb.GenerateCRLRequest, error) {
	next, ok := <-s.input
	if !ok {
		return nil, io.EOF
	}
	return next, nil
}

func (s mockGenerateCRLBidiStream) Send(entry *capb.GenerateCRLResponse) error {
	s.output <- entry
	return nil
}

func TestGenerateCRL(t *testing.T) {
	testCtx := setup(t)
	crli := testCtx.crl
	errs := make(chan error, 1)

	// Test that we get an error when no metadata is sent.
	ins := make(chan *capb.GenerateCRLRequest)
	go func() {
		errs <- crli.GenerateCRL(mockGenerateCRLBidiStream{input: ins, output: nil})
	}()
	close(ins)
	err := <-errs
	test.AssertError(t, err, "can't generate CRL with no metadata")
	test.AssertContains(t, err.Error(), "no crl metadata received")

	// Test that we get an error when incomplete metadata is sent.
	ins = make(chan *capb.GenerateCRLRequest)
	go func() {
		errs <- crli.GenerateCRL(mockGenerateCRLBidiStream{input: ins, output: nil})
	}()
	ins <- &capb.GenerateCRLRequest{
		Payload: &capb.GenerateCRLRequest_Metadata{
			Metadata: &capb.CRLMetadata{},
		},
	}
	close(ins)
	err = <-errs
	test.AssertError(t, err, "can't generate CRL with incomplete metadata")
	test.AssertContains(t, err.Error(), "got incomplete metadata message")

	// Test that we get an error when unrecognized metadata is sent.
	ins = make(chan *capb.GenerateCRLRequest)
	go func() {
		errs <- crli.GenerateCRL(mockGenerateCRLBidiStream{input: ins, output: nil})
	}()
	ins <- &capb.GenerateCRLRequest{
		Payload: &capb.GenerateCRLRequest_Metadata{
			Metadata: &capb.CRLMetadata{
				IssuerNameID: 1,
				ThisUpdate:   time.Now().UnixNano(),
			},
		},
	}
	close(ins)
	err = <-errs
	test.AssertError(t, err, "can't generate CRL with bad metadata")
	test.AssertContains(t, err.Error(), "got unrecognized IssuerNameID")

	// Test that we get an error when two metadata are sent.
	ins = make(chan *capb.GenerateCRLRequest)
	go func() {
		errs <- crli.GenerateCRL(mockGenerateCRLBidiStream{input: ins, output: nil})
	}()
	ins <- &capb.GenerateCRLRequest{
		Payload: &capb.GenerateCRLRequest_Metadata{
			Metadata: &capb.CRLMetadata{
				IssuerNameID: int64(testCtx.boulderIssuers[0].Cert.NameID()),
				ThisUpdate:   time.Now().UnixNano(),
			},
		},
	}
	ins <- &capb.GenerateCRLRequest{
		Payload: &capb.GenerateCRLRequest_Metadata{
			Metadata: &capb.CRLMetadata{
				IssuerNameID: int64(testCtx.boulderIssuers[0].Cert.NameID()),
				ThisUpdate:   time.Now().UnixNano(),
			},
		},
	}
	close(ins)
	err = <-errs
	test.AssertError(t, err, "can't generate CRL with duplicate metadata")
	test.AssertContains(t, err.Error(), "got more than one metadata message")

	// Test that we get an error when an entry has a bad serial.
	ins = make(chan *capb.GenerateCRLRequest)
	go func() {
		errs <- crli.GenerateCRL(mockGenerateCRLBidiStream{input: ins, output: nil})
	}()
	ins <- &capb.GenerateCRLRequest{
		Payload: &capb.GenerateCRLRequest_Entry{
			Entry: &corepb.CRLEntry{
				Serial:    "123",
				Reason:    1,
				RevokedAt: time.Now().UnixNano(),
			},
		},
	}
	close(ins)
	err = <-errs
	test.AssertError(t, err, "can't generate CRL with bad serials")
	test.AssertContains(t, err.Error(), "Invalid serial number")

	// Test that we get an error when an entry has a bad revocation time.
	ins = make(chan *capb.GenerateCRLRequest)
	go func() {
		errs <- crli.GenerateCRL(mockGenerateCRLBidiStream{input: ins, output: nil})
	}()
	ins <- &capb.GenerateCRLRequest{
		Payload: &capb.GenerateCRLRequest_Entry{
			Entry: &corepb.CRLEntry{
				Serial:    "deadbeefdeadbeefdeadbeefdeadbeefdead",
				Reason:    1,
				RevokedAt: 0,
			},
		},
	}
	close(ins)
	err = <-errs
	test.AssertError(t, err, "can't generate CRL with bad serials")
	test.AssertContains(t, err.Error(), "got empty or zero revocation timestamp")

	// Test that generating an empty CRL works.
	ins = make(chan *capb.GenerateCRLRequest)
	outs := make(chan *capb.GenerateCRLResponse)
	go func() {
		errs <- crli.GenerateCRL(mockGenerateCRLBidiStream{input: ins, output: outs})
		close(outs)
	}()
	crlBytes := make([]byte, 0)
	done := make(chan struct{})
	go func() {
		for resp := range outs {
			crlBytes = append(crlBytes, resp.Chunk...)
		}
		close(done)
	}()
	ins <- &capb.GenerateCRLRequest{
		Payload: &capb.GenerateCRLRequest_Metadata{
			Metadata: &capb.CRLMetadata{
				IssuerNameID: int64(testCtx.boulderIssuers[0].Cert.NameID()),
				ThisUpdate:   time.Now().UnixNano(),
			},
		},
	}
	close(ins)
	err = <-errs
	<-done
	test.AssertNotError(t, err, "generating empty CRL should work")
	test.Assert(t, len(crlBytes) > 0, "should have gotten some CRL bytes")
	crl, err := x509.ParseCRL(crlBytes)
	test.AssertNotError(t, err, "should be able to parse empty CRL")
	test.AssertEquals(t, len(crl.TBSCertList.RevokedCertificates), 0)
	err = testCtx.boulderIssuers[0].Cert.CheckCRLSignature(crl)
	test.AssertNotError(t, err, "CRL signature should validate")

	// Test that generating a CRL with some entries works.
	ins = make(chan *capb.GenerateCRLRequest)
	outs = make(chan *capb.GenerateCRLResponse)
	go func() {
		errs <- crli.GenerateCRL(mockGenerateCRLBidiStream{input: ins, output: outs})
		close(outs)
	}()
	crlBytes = make([]byte, 0)
	done = make(chan struct{})
	go func() {
		for resp := range outs {
			crlBytes = append(crlBytes, resp.Chunk...)
		}
		close(done)
	}()
	ins <- &capb.GenerateCRLRequest{
		Payload: &capb.GenerateCRLRequest_Metadata{
			Metadata: &capb.CRLMetadata{
				IssuerNameID: int64(testCtx.boulderIssuers[0].Cert.NameID()),
				ThisUpdate:   time.Now().UnixNano(),
			},
		},
	}
	ins <- &capb.GenerateCRLRequest{
		Payload: &capb.GenerateCRLRequest_Entry{
			Entry: &corepb.CRLEntry{
				Serial:    "000000000000000000000000000000000000",
				RevokedAt: time.Now().UnixNano(),
				// Reason 0, Unspecified, is omitted.
			},
		},
	}
	ins <- &capb.GenerateCRLRequest{
		Payload: &capb.GenerateCRLRequest_Entry{
			Entry: &corepb.CRLEntry{
				Serial:    "111111111111111111111111111111111111",
				Reason:    1, // keyCompromise
				RevokedAt: time.Now().UnixNano(),
			},
		},
	}
	ins <- &capb.GenerateCRLRequest{
		Payload: &capb.GenerateCRLRequest_Entry{
			Entry: &corepb.CRLEntry{
				Serial:    "444444444444444444444444444444444444",
				Reason:    4, // superseded
				RevokedAt: time.Now().UnixNano(),
			},
		},
	}
	ins <- &capb.GenerateCRLRequest{
		Payload: &capb.GenerateCRLRequest_Entry{
			Entry: &corepb.CRLEntry{
				Serial:    "555555555555555555555555555555555555",
				Reason:    5, // cessationOfOperation
				RevokedAt: time.Now().UnixNano(),
			},
		},
	}
	ins <- &capb.GenerateCRLRequest{
		Payload: &capb.GenerateCRLRequest_Entry{
			Entry: &corepb.CRLEntry{
				Serial:    "999999999999999999999999999999999999",
				Reason:    9, // privilegeWithdrawn
				RevokedAt: time.Now().UnixNano(),
			},
		},
	}
	close(ins)
	err = <-errs
	<-done
	test.AssertNotError(t, err, "generating empty CRL should work")
	test.Assert(t, len(crlBytes) > 0, "should have gotten some CRL bytes")
	crl, err = x509.ParseCRL(crlBytes)
	test.AssertNotError(t, err, "should be able to parse empty CRL")
	test.AssertEquals(t, len(crl.TBSCertList.RevokedCertificates), 5)
	err = testCtx.boulderIssuers[0].Cert.CheckCRLSignature(crl)
	test.AssertNotError(t, err, "CRL signature should validate")
}
