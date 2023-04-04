package issuance

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	ct "github.com/google/certificate-transparency-go"
	cttls "github.com/google/certificate-transparency-go/tls"
	ctx509 "github.com/google/certificate-transparency-go/x509"
	"github.com/jmhodges/clock"
	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/linter"
	"github.com/letsencrypt/boulder/policyasn1"
	"github.com/letsencrypt/boulder/privatekey"
	"github.com/letsencrypt/pkcs11key/v4"
	"golang.org/x/crypto/ocsp"
)

// ProfileConfig describes the certificate issuance constraints for all issuers.
type ProfileConfig struct {
	AllowMustStaple bool
	AllowCTPoison   bool
	AllowSCTList    bool
	AllowCommonName bool

	Policies            []PolicyInformation
	MaxValidityPeriod   cmd.ConfigDuration
	MaxValidityBackdate cmd.ConfigDuration
}

// PolicyInformation describes a policy
type PolicyInformation struct {
	OID        string
	Qualifiers []PolicyQualifier
}

// PolicyQualifier describes a policy qualifier
type PolicyQualifier struct {
	Type  string
	Value string
}

// IssuerConfig describes the constraints on and URLs used by a single issuer.
type IssuerConfig struct {
	UseForRSALeaves   bool
	UseForECDSALeaves bool

	IssuerURL string
	OCSPURL   string
	CRLURL    string

	Location IssuerLoc
}

// IssuerLoc describes the on-disk location and parameters that an issuer
// should use to retrieve its certificate and private key.
// Only one of File, ConfigFile, or PKCS11 should be set.
type IssuerLoc struct {
	// A file from which a private key will be read and parsed.
	File string
	// A file from which a pkcs11key.Config will be read and parsed, if File is not set.
	ConfigFile string
	// An in-memory pkcs11key.Config, which will be used if ConfigFile is not set.
	PKCS11 *pkcs11key.Config
	// A file from which a certificate will be read and parsed.
	CertFile string
	// Number of sessions to open with the HSM. For maximum performance,
	// this should be equal to the number of cores in the HSM. Defaults to 1.
	NumSessions int
}

// LoadIssuer loads a signer (private key) and certificate from the locations specified.
func LoadIssuer(location IssuerLoc) (*Certificate, crypto.Signer, error) {
	issuerCert, err := LoadCertificate(location.CertFile)
	if err != nil {
		return nil, nil, err
	}

	signer, err := loadSigner(location, issuerCert)
	if err != nil {
		return nil, nil, err
	}

	if !core.KeyDigestEquals(signer.Public(), issuerCert.PublicKey) {
		return nil, nil, fmt.Errorf("Issuer key did not match issuer cert %s", location.CertFile)
	}
	return issuerCert, signer, err
}

func LoadCertificate(path string) (*Certificate, error) {
	cert, err := core.LoadCert(path)
	if err != nil {
		return nil, err
	}
	return NewCertificate(cert)
}

func loadSigner(location IssuerLoc, cert *Certificate) (crypto.Signer, error) {
	if location.File != "" {
		signer, _, err := privatekey.Load(location.File)
		if err != nil {
			return nil, err
		}
		return signer, nil
	}

	var pkcs11Config *pkcs11key.Config
	if location.ConfigFile != "" {
		contents, err := os.ReadFile(location.ConfigFile)
		if err != nil {
			return nil, err
		}
		pkcs11Config = new(pkcs11key.Config)
		err = json.Unmarshal(contents, pkcs11Config)
		if err != nil {
			return nil, err
		}
	} else {
		pkcs11Config = location.PKCS11
	}

	if pkcs11Config.Module == "" ||
		pkcs11Config.TokenLabel == "" ||
		pkcs11Config.PIN == "" {
		return nil, fmt.Errorf("Missing a field in pkcs11Config %#v", pkcs11Config)
	}

	numSessions := location.NumSessions
	if numSessions <= 0 {
		numSessions = 1
	}

	return pkcs11key.NewPool(numSessions, pkcs11Config.Module,
		pkcs11Config.TokenLabel, pkcs11Config.PIN, cert.PublicKey)
}

// Profile is the validated structure created by reading in ProfileConfigs and IssuerConfigs
type Profile struct {
	useForRSALeaves   bool
	useForECDSALeaves bool

	allowMustStaple bool
	allowCTPoison   bool
	allowSCTList    bool
	allowCommonName bool

	sigAlg    x509.SignatureAlgorithm
	ocspURL   string
	crlURL    string
	issuerURL string
	policies  *pkix.Extension

	maxBackdate time.Duration
	maxValidity time.Duration
}

func parseOID(oidStr string) (asn1.ObjectIdentifier, error) {
	var oid asn1.ObjectIdentifier
	for _, a := range strings.Split(oidStr, ".") {
		i, err := strconv.Atoi(a)
		if err != nil {
			return nil, err
		}
		if i <= 0 {
			return nil, errors.New("OID components must be >= 1")
		}
		oid = append(oid, i)
	}
	return oid, nil
}

var stringToQualifierType = map[string]asn1.ObjectIdentifier{
	"id-qt-cps": policyasn1.CPSQualifierOID,
}

// NewProfile synthesizes the profile config and issuer config into a single
// object, and checks various aspects for correctness.
func NewProfile(profileConfig ProfileConfig, issuerConfig IssuerConfig) (*Profile, error) {
	if issuerConfig.IssuerURL == "" {
		return nil, errors.New("Issuer URL is required")
	}
	if issuerConfig.OCSPURL == "" {
		return nil, errors.New("OCSP URL is required")
	}
	sp := &Profile{
		useForRSALeaves:   issuerConfig.UseForRSALeaves,
		useForECDSALeaves: issuerConfig.UseForECDSALeaves,
		allowMustStaple:   profileConfig.AllowMustStaple,
		allowCTPoison:     profileConfig.AllowCTPoison,
		allowSCTList:      profileConfig.AllowSCTList,
		allowCommonName:   profileConfig.AllowCommonName,
		issuerURL:         issuerConfig.IssuerURL,
		crlURL:            issuerConfig.CRLURL,
		ocspURL:           issuerConfig.OCSPURL,
		maxBackdate:       profileConfig.MaxValidityBackdate.Duration,
		maxValidity:       profileConfig.MaxValidityPeriod.Duration,
	}
	if len(profileConfig.Policies) > 0 {
		var policies []policyasn1.PolicyInformation
		for _, policyConfig := range profileConfig.Policies {
			id, err := parseOID(policyConfig.OID)
			if err != nil {
				return nil, fmt.Errorf("failed parsing policy OID %q: %s", policyConfig.OID, err)
			}
			pi := policyasn1.PolicyInformation{Policy: id}
			for _, qualifierConfig := range policyConfig.Qualifiers {
				qt, ok := stringToQualifierType[qualifierConfig.Type]
				if !ok {
					return nil, fmt.Errorf("unknown qualifier type: %s", qualifierConfig.Type)
				}
				pq := policyasn1.PolicyQualifier{
					OID:   qt,
					Value: qualifierConfig.Value,
				}
				pi.Qualifiers = append(pi.Qualifiers, pq)
			}
			policies = append(policies, pi)
		}
		policyExtBytes, err := asn1.Marshal(policies)
		if err != nil {
			return nil, err
		}
		sp.policies = &pkix.Extension{
			Id:    asn1.ObjectIdentifier{2, 5, 29, 32},
			Value: policyExtBytes,
		}
	}
	return sp, nil
}

// requestValid verifies the passed IssuanceRequest against the profile. If the
// request doesn't match the signing profile an error is returned.
func (p *Profile) requestValid(clk clock.Clock, req *IssuanceRequest) error {
	switch req.PublicKey.(type) {
	case *rsa.PublicKey:
		if !p.useForRSALeaves {
			return errors.New("cannot sign RSA public keys")
		}
	case *ecdsa.PublicKey:
		if !p.useForECDSALeaves {
			return errors.New("cannot sign ECDSA public keys")
		}
	default:
		return errors.New("unsupported public key type")
	}

	if !p.allowMustStaple && req.IncludeMustStaple {
		return errors.New("must-staple extension cannot be included")
	}

	if !p.allowCTPoison && req.IncludeCTPoison {
		return errors.New("ct poison extension cannot be included")
	}

	if !p.allowSCTList && req.SCTList != nil {
		return errors.New("sct list extension cannot be included")
	}

	if req.IncludeCTPoison && req.SCTList != nil {
		return errors.New("cannot include both ct poison and sct list extensions")
	}

	if !p.allowCommonName && req.CommonName != "" {
		return errors.New("common name cannot be included")
	}

	// The validity period is calculated inclusive of the whole second represented
	// by the notAfter timestamp.
	validity := req.NotAfter.Add(time.Second).Sub(req.NotBefore)
	if validity <= 0 {
		return errors.New("NotAfter must be after NotBefore")
	}
	if validity > p.maxValidity {
		return fmt.Errorf("validity period is more than the maximum allowed period (%s>%s)", validity, p.maxValidity)
	}
	backdatedBy := clk.Now().Sub(req.NotBefore)
	if backdatedBy > p.maxBackdate {
		return fmt.Errorf("NotBefore is backdated more than the maximum allowed period (%s>%s)", backdatedBy, p.maxBackdate)
	}
	if backdatedBy < 0 {
		return errors.New("NotBefore is in the future")
	}

	// We use 19 here because a 20-byte serial could produce >20 octets when
	// encoded in ASN.1. That happens when the first byte is >0x80. See
	// https://letsencrypt.org/docs/a-warm-welcome-to-asn1-and-der/#integer-encoding
	if len(req.Serial) > 19 || len(req.Serial) < 9 {
		return errors.New("serial must be between 9 and 19 bytes")
	}

	return nil
}

var defaultEKU = []x509.ExtKeyUsage{
	x509.ExtKeyUsageServerAuth,
	x509.ExtKeyUsageClientAuth,
}

func (p *Profile) generateTemplate() *x509.Certificate {
	template := &x509.Certificate{
		SignatureAlgorithm:    p.sigAlg,
		ExtKeyUsage:           defaultEKU,
		OCSPServer:            []string{p.ocspURL},
		IssuingCertificateURL: []string{p.issuerURL},
		BasicConstraintsValid: true,
	}

	if p.crlURL != "" {
		template.CRLDistributionPoints = []string{p.crlURL}
	}

	if p.policies != nil {
		template.ExtraExtensions = []pkix.Extension{*p.policies}
	}

	return template
}

// IssuerID is a statistically-unique small ID computed from a hash over the
// entirety of the issuer certificate.
// DEPRECATED: This identifier is being phased out in favor of IssuerNameID.
// It exists in the database in certificateStatus rows for certs issued prior
// to approximately November 2021, but is not being written for new rows.
type IssuerID int64

// IssuerNameID is a statistically-unique small ID which can be computed from
// both CA and end-entity certs to link them together into a validation chain.
// It is computed as a truncated hash over the issuer Subject Name bytes, or
// over the end-entity's Issuer Name bytes, which are required to be equal.
type IssuerNameID int64

// Certificate embeds an *x509.Certificate and represents the added semantics
// that this certificate can be used for issuance.
type Certificate struct {
	*x509.Certificate
	id       IssuerID
	nameID   IssuerNameID
	nameHash [20]byte
	keyHash  [20]byte
}

// NewCertificate wraps an in-memory cert in an issuance.Certificate, marking it
// as an issuer cert. It may fail if the certificate does not contain the
// attributes expected of an issuer certificate.
func NewCertificate(ic *x509.Certificate) (*Certificate, error) {
	res := Certificate{Certificate: ic}

	// Compute ic.ID()
	h := sha256.Sum256(ic.Raw)
	res.id = IssuerID(big.NewInt(0).SetBytes(h[:4]).Int64())

	// Compute ic.NameID()
	res.nameID = truncatedHash(ic.RawSubject)

	// Compute ic.NameHash()
	res.nameHash = sha1.Sum(ic.RawSubject)

	// Compute ic.KeyHash()
	// The issuerKeyHash in OCSP requests is constructed over the DER encoding of
	// the public key per RFC6960 (defined in RFC4055 for RSA and RFC5480 for
	// ECDSA). We can't use MarshalPKIXPublicKey for this since it encodes keys
	// using the SPKI structure itself, and we just want the contents of the
	// subjectPublicKey for the hash, so we need to extract it ourselves.
	var spki struct {
		Algorithm pkix.AlgorithmIdentifier
		PublicKey asn1.BitString
	}
	_, err := asn1.Unmarshal(ic.RawSubjectPublicKeyInfo, &spki)
	if err != nil {
		return nil, err
	}
	res.keyHash = sha1.Sum(spki.PublicKey.RightAlign())

	return &res, nil
}

// ID returns the IssuerID (a truncated hash over the raw bytes of the whole
// cert) of this issuer certificate.
// DEPRECATED: Use .NameID() instead.
func (ic *Certificate) ID() IssuerID {
	return ic.id
}

// NameID returns the IssuerNameID (a truncated hash over the raw bytes of the
// Subject Distinguished Name) of this issuer certificate. Useful for storing as
// a lookup key in contexts that don't expect hash collisions.
func (ic *Certificate) NameID() IssuerNameID {
	return ic.nameID
}

// NameHash returns the SHA1 hash over the issuer certificate's Subject
// Distinguished Name. This is one of the values used to uniquely identify the
// issuer cert in an RFC6960 + RFC5019 OCSP request.
func (ic *Certificate) NameHash() [20]byte {
	return ic.nameHash
}

// KeyHash returns the SHA1 hash over the issuer certificate's Subject Public
// Key Info. This is one of the values used to uniquely identify the issuer cert
// in an RFC6960 + RFC5019 OCSP request.
func (ic *Certificate) KeyHash() [20]byte {
	return ic.keyHash
}

// GetIssuerNameID returns the IssuerNameID (a truncated hash over the raw bytes
// of the Issuer Distinguished Name) of the given end-entity certificate.
// Useful for performing lookups in contexts that don't expect hash collisions.
func GetIssuerNameID(ee *x509.Certificate) IssuerNameID {
	return truncatedHash(ee.RawIssuer)
}

// GetOCSPIssuerNameID returns the IssuerNameID (a truncated hash over the raw
// bytes of the Responder Distinguished Name) of the given OCSP Response.
// As per the OCSP spec, it is technically possible for this field to not be
// populated: the OCSP Response can instead contain a SHA-1 hash of the Issuer
// Public Key as the Responder ID. The Go stdlib always uses the DN, though.
func GetOCSPIssuerNameID(resp *ocsp.Response) IssuerNameID {
	return truncatedHash(resp.RawResponderName)
}

// truncatedHash computes a truncated SHA1 hash across arbitrary bytes. Uses
// SHA1 because that is the algorithm most commonly used in OCSP requests.
// PURPOSEFULLY NOT EXPORTED. Exists only to ensure that the implementations of
// Certificate.NameID() and GetIssuerNameID() never diverge. Use those instead.
func truncatedHash(name []byte) IssuerNameID {
	h := crypto.SHA1.New()
	h.Write(name)
	s := h.Sum(nil)
	return IssuerNameID(big.NewInt(0).SetBytes(s[:7]).Int64())
}

// Issuer is capable of issuing new certificates
// TODO(#5086): make Cert and Signer private when they're no longer needed by ca.internalIssuer
type Issuer struct {
	Cert    *Certificate
	Signer  crypto.Signer
	Profile *Profile
	Linter  *linter.Linter
	Clk     clock.Clock
}

// NewIssuer constructs an Issuer on the heap, verifying that the profile
// is well-formed.
func NewIssuer(cert *Certificate, signer crypto.Signer, profile *Profile, linter *linter.Linter, clk clock.Clock) (*Issuer, error) {
	switch k := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		profile.sigAlg = x509.SHA256WithRSA
	case *ecdsa.PublicKey:
		switch k.Curve {
		case elliptic.P256():
			profile.sigAlg = x509.ECDSAWithSHA256
		case elliptic.P384():
			profile.sigAlg = x509.ECDSAWithSHA384
		default:
			return nil, fmt.Errorf("unsupported ECDSA curve: %s", k.Curve.Params().Name)
		}
	default:
		return nil, errors.New("unsupported issuer key type")
	}

	if profile.useForRSALeaves || profile.useForECDSALeaves {
		if cert.KeyUsage&x509.KeyUsageCertSign == 0 {
			return nil, errors.New("end-entity signing cert does not have keyUsage certSign")
		}
	}
	// TODO(#5086): Only do this check for ocsp-issuing issuers.
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return nil, errors.New("end-entity ocsp signing cert does not have keyUsage digitalSignature")
	}

	i := &Issuer{
		Cert:    cert,
		Signer:  signer,
		Profile: profile,
		Linter:  linter,
		Clk:     clk,
	}
	return i, nil
}

// Algs provides the list of leaf certificate public key algorithms for which
// this issuer is willing to issue. This is not necessarily the same as the
// public key algorithm or signature algorithm in this issuer's own cert.
func (i *Issuer) Algs() []x509.PublicKeyAlgorithm {
	var algs []x509.PublicKeyAlgorithm
	if i.Profile.useForRSALeaves {
		algs = append(algs, x509.RSA)
	}
	if i.Profile.useForECDSALeaves {
		algs = append(algs, x509.ECDSA)
	}
	return algs
}

// Name provides the Common Name specified in the issuer's certificate.
func (i *Issuer) Name() string {
	return i.Cert.Subject.CommonName
}

// ID provides a stable ID for an issuer's certificate. This is used for
// identifying which issuer issued a certificate in the certificateStatus table.
func (i *Issuer) ID() IssuerID {
	return i.Cert.ID()
}

var ctPoisonExt = pkix.Extension{
	// OID for CT poison, RFC 6962 (was never assigned a proper id-pe- name)
	Id:       asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 4, 3},
	Value:    asn1.NullBytes,
	Critical: true,
}

// OID for SCT list, RFC 6962 (was never assigned a proper id-pe- name)
var sctListOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 4, 2}

func generateSCTListExt(scts []ct.SignedCertificateTimestamp) (pkix.Extension, error) {
	list := ctx509.SignedCertificateTimestampList{}
	for _, sct := range scts {
		sctBytes, err := cttls.Marshal(sct)
		if err != nil {
			return pkix.Extension{}, err
		}
		list.SCTList = append(list.SCTList, ctx509.SerializedSCT{Val: sctBytes})
	}
	listBytes, err := cttls.Marshal(list)
	if err != nil {
		return pkix.Extension{}, err
	}
	extBytes, err := asn1.Marshal(listBytes)
	if err != nil {
		return pkix.Extension{}, err
	}
	return pkix.Extension{
		Id:    sctListOID,
		Value: extBytes,
	}, nil
}

var mustStapleExt = pkix.Extension{
	// RFC 7633: id-pe-tlsfeature OBJECT IDENTIFIER ::=  { id-pe 24 }
	Id: asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 1, 24},
	// ASN.1 encoding of:
	// SEQUENCE
	//   INTEGER 5
	// where "5" is the status_request feature (RFC 6066)
	Value: []byte{0x30, 0x03, 0x02, 0x01, 0x05},
}

func generateSKID(pk crypto.PublicKey) ([]byte, error) {
	pkBytes, err := x509.MarshalPKIXPublicKey(pk)
	if err != nil {
		return nil, err
	}
	var pkixPublicKey struct {
		Algo      pkix.AlgorithmIdentifier
		BitString asn1.BitString
	}
	if _, err := asn1.Unmarshal(pkBytes, &pkixPublicKey); err != nil {
		return nil, err
	}
	skid := sha1.Sum(pkixPublicKey.BitString.Bytes)
	return skid[:], nil
}

// IssuanceRequest describes a certificate issuance request
type IssuanceRequest struct {
	PublicKey crypto.PublicKey

	Serial []byte

	NotBefore time.Time
	NotAfter  time.Time

	CommonName string
	DNSNames   []string

	IncludeMustStaple bool
	IncludeCTPoison   bool
	SCTList           []ct.SignedCertificateTimestamp
}

// Issue generates a certificate from the provided issuance request and
// signs it. Before signing the certificate with the issuer's private
// key, it is signed using a throwaway key so that it can be linted using
// zlint. If the linting fails, an error is returned and the certificate
// is not signed using the issuer's key.
func (i *Issuer) Issue(req *IssuanceRequest) ([]byte, error) {
	// check request is valid according to the issuance profile
	err := i.Profile.requestValid(i.Clk, req)
	if err != nil {
		return nil, err
	}

	// generate template from the issuance profile
	template := i.Profile.generateTemplate()

	// populate template from the issuance request
	template.NotBefore, template.NotAfter = req.NotBefore, req.NotAfter
	template.SerialNumber = big.NewInt(0).SetBytes(req.Serial)
	if req.CommonName != "" {
		template.Subject.CommonName = req.CommonName
	}
	template.DNSNames = req.DNSNames
	template.AuthorityKeyId = i.Cert.SubjectKeyId
	skid, err := generateSKID(req.PublicKey)
	if err != nil {
		return nil, err
	}
	template.SubjectKeyId = skid
	switch req.PublicKey.(type) {
	case *rsa.PublicKey:
		template.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	case *ecdsa.PublicKey:
		template.KeyUsage = x509.KeyUsageDigitalSignature
	}

	if req.IncludeCTPoison {
		template.ExtraExtensions = append(template.ExtraExtensions, ctPoisonExt)
	} else if req.SCTList != nil {
		sctListExt, err := generateSCTListExt(req.SCTList)
		if err != nil {
			return nil, err
		}
		template.ExtraExtensions = append(template.ExtraExtensions, sctListExt)
	}

	if req.IncludeMustStaple {
		template.ExtraExtensions = append(template.ExtraExtensions, mustStapleExt)
	}

	// check that the tbsCertificate is properly formed by signing it
	// with a throwaway key and then linting it using zlint
	err = i.Linter.Check(template, req.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("tbsCertificate linting failed: %w", err)
	}

	return x509.CreateCertificate(rand.Reader, template, i.Cert.Certificate, req.PublicKey, i.Signer)
}

func ContainsMustStaple(extensions []pkix.Extension) bool {
	for _, ext := range extensions {
		if ext.Id.Equal(mustStapleExt.Id) && bytes.Equal(ext.Value, mustStapleExt.Value) {
			return true
		}
	}
	return false
}

func containsCTPoison(extensions []pkix.Extension) bool {
	for _, ext := range extensions {
		if ext.Id.Equal(ctPoisonExt.Id) && bytes.Equal(ext.Value, asn1.NullBytes) {
			return true
		}
	}
	return false
}

// RequestFromPrecert constructs a final certificate IssuanceRequest matching
// the provided precertificate. It returns an error if the precertificate doesn't
// contain the CT poison extension.
func RequestFromPrecert(precert *x509.Certificate, scts []ct.SignedCertificateTimestamp) (*IssuanceRequest, error) {
	if !containsCTPoison(precert.Extensions) {
		return nil, errors.New("provided certificate doesn't contain the CT poison extension")
	}
	return &IssuanceRequest{
		PublicKey:         precert.PublicKey,
		Serial:            precert.SerialNumber.Bytes(),
		NotBefore:         precert.NotBefore,
		NotAfter:          precert.NotAfter,
		CommonName:        precert.Subject.CommonName,
		DNSNames:          precert.DNSNames,
		IncludeMustStaple: ContainsMustStaple(precert.Extensions),
		SCTList:           scts,
	}, nil
}

// LoadChain takes a list of filenames containing pem-formatted certificates,
// and returns a chain representing all of those certificates in order. It
// ensures that the resulting chain is valid. The final file is expected to be
// a root certificate, which the chain will be verified against, but which will
// not be included in the resulting chain.
func LoadChain(certFiles []string) ([]*Certificate, error) {
	if len(certFiles) < 2 {
		return nil, errors.New(
			"each chain must have at least two certificates: an intermediate and a root")
	}

	// Pre-load all the certificates to make validation easier.
	certs := make([]*Certificate, len(certFiles))
	var err error
	for i := 0; i < len(certFiles); i++ {
		certs[i], err = LoadCertificate(certFiles[i])
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate %q: %w", certFiles[i], err)
		}
	}

	// Iterate over all certs except for the last, checking that their signature
	// comes from the next cert in the list.
	chain := make([]*Certificate, len(certFiles)-1)
	for i := 0; i < len(certs)-1; i++ {
		err = certs[i].CheckSignatureFrom(certs[i+1].Certificate)
		if err != nil {
			return nil, fmt.Errorf("failed to verify signature from %q to %q (%q to %q): %w",
				certs[i+1].Subject, certs[i].Subject, certFiles[i+1], certFiles[i], err)
		}
		chain[i] = certs[i]
	}

	// Verify that the last cert is self-signed.
	lastCert := certs[len(certs)-1]
	err = lastCert.CheckSignatureFrom(lastCert.Certificate)
	if err != nil {
		return nil, fmt.Errorf(
			"final cert in chain (%q; %q) must be self-signed (used only for validation): %w",
			lastCert.Subject, certFiles[len(certFiles)-1], err)
	}

	return chain, nil
}
