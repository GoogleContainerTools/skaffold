package test

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"testing"
)

// LoadSigner loads a PEM private key specified by filename or returns an error.
// Can be paired with issuance.LoadCertificate to get both a CA cert and its
// associated private key for use in signing throwaway test certs.
func LoadSigner(filename string) (crypto.Signer, error) {
	keyBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// pem.Decode does not return an error as its 2nd arg, but instead the "rest"
	// that was leftover from parsing the PEM block. We only care if the decoded
	// PEM block was empty for this test function.
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, errors.New("Unable to decode private key PEM bytes")
	}

	// Try decoding as an RSA private key
	if rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return rsaKey, nil
	}

	// Try decoding as a PKCS8 private key
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		// Determine the key's true type and return it as a crypto.Signer
		switch k := key.(type) {
		case *rsa.PrivateKey:
			return k, nil
		case *ecdsa.PrivateKey:
			return k, nil
		}
	}

	// Try as an ECDSA private key
	if ecdsaKey, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return ecdsaKey, nil
	}

	// Nothing worked! Fail hard.
	return nil, errors.New("Unable to decode private key PEM bytes")
}

// ThrowAwayCert is a small test helper function that creates a self-signed
// certificate for nameCount random example.com subdomains and returns the
// parsed certificate  and the random serial in string form or aborts the test.
// The certificate returned from this function is the bare minimum needed for
// most tests and isn't a robust example of a complete end entity certificate.
func ThrowAwayCert(t *testing.T, nameCount int) (string, *x509.Certificate) {
	var serialBytes [16]byte
	_, _ = rand.Read(serialBytes[:])
	sn := big.NewInt(0).SetBytes(serialBytes[:])

	return ThrowAwayCertWithSerial(t, nameCount, sn, nil)
}

// ThrowAwayCertWithSerial is a small test helper function that creates a
// certificate for nameCount random example.com subdomains and returns the
// parsed certificate and the serial in string form or aborts the test.
// The new throwaway certificate is always self-signed (with a random key),
// but will appear to be issued from issuer if provided.
// The certificate returned from this function is the bare minimum needed for
// most tests and isn't a robust example of a complete end entity certificate.
func ThrowAwayCertWithSerial(t *testing.T, nameCount int, sn *big.Int, issuer *x509.Certificate) (string, *x509.Certificate) {
	k, err := rsa.GenerateKey(rand.Reader, 512)
	AssertNotError(t, err, "rsa.GenerateKey failed")

	var names []string
	for i := 0; i < nameCount; i++ {
		var nameBytes [3]byte
		_, _ = rand.Read(nameBytes[:])
		names = append(names, fmt.Sprintf("%s.example.com", hex.EncodeToString(nameBytes[:])))
	}

	template := &x509.Certificate{
		SerialNumber:          sn,
		DNSNames:              names,
		IssuingCertificateURL: []string{"http://localhost:4001/acme/issuer-cert/1234"},
	}

	if issuer == nil {
		issuer = template
	}

	testCertDER, err := x509.CreateCertificate(rand.Reader, template, issuer, &k.PublicKey, k)
	AssertNotError(t, err, "x509.CreateCertificate failed")
	testCert, err := x509.ParseCertificate(testCertDER)
	AssertNotError(t, err, "failed to parse self-signed cert DER")
	return fmt.Sprintf("%036x", sn), testCert
}
