package notmain

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/linter"
	"github.com/letsencrypt/boulder/pkcs11helpers"
	"golang.org/x/crypto/ocsp"
	"gopkg.in/yaml.v3"
)

const configDateLayout = "2006-01-02 15:04:05"

type keyGenConfig struct {
	Type         string `yaml:"type"`
	RSAModLength uint   `yaml:"rsa-mod-length"`
	ECDSACurve   string `yaml:"ecdsa-curve"`
}

var allowedCurves = map[string]bool{
	"P-224": true,
	"P-256": true,
	"P-384": true,
	"P-521": true,
}

func (kgc keyGenConfig) validate() error {
	if kgc.Type == "" {
		return errors.New("key.type is required")
	}
	if kgc.Type != "rsa" && kgc.Type != "ecdsa" {
		return errors.New("key.type can only be 'rsa' or 'ecdsa'")
	}
	if kgc.Type == "rsa" && (kgc.RSAModLength != 2048 && kgc.RSAModLength != 4096) {
		return errors.New("key.rsa-mod-length can only be 2048 or 4096")
	}
	if kgc.Type == "rsa" && kgc.ECDSACurve != "" {
		return errors.New("if key.type = 'rsa' then key.ecdsa-curve is not used")
	}
	if kgc.Type == "ecdsa" && !allowedCurves[kgc.ECDSACurve] {
		return errors.New("key.ecdsa-curve can only be 'P-224', 'P-256', 'P-384', or 'P-521'")
	}
	if kgc.Type == "ecdsa" && kgc.RSAModLength != 0 {
		return errors.New("if key.type = 'ecdsa' then key.rsa-mod-length is not used")
	}

	return nil
}

type PKCS11KeyGenConfig struct {
	Module     string `yaml:"module"`
	PIN        string `yaml:"pin"`
	StoreSlot  uint   `yaml:"store-key-in-slot"`
	StoreLabel string `yaml:"store-key-with-label"`
}

func (pkgc PKCS11KeyGenConfig) validate() error {
	if pkgc.Module == "" {
		return errors.New("pkcs11.module is required")
	}
	if pkgc.StoreLabel == "" {
		return errors.New("pkcs11.store-key-with-label is required")
	}
	// key-slot is allowed to be 0 (which is a valid slot).
	// PIN is allowed to be "", which will commonly happen when
	// PIN entry is done via PED.
	return nil
}

// checkOutputFile returns an error if the filename is empty,
// or if a file already exists with that filename.
func checkOutputFile(filename, fieldname string) error {
	if filename == "" {
		return fmt.Errorf("outputs.%s is required", fieldname)
	}
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		return fmt.Errorf("outputs.%s is %q, which already exists",
			fieldname, filename)
	}
	return nil
}

type rootConfig struct {
	CeremonyType string             `yaml:"ceremony-type"`
	PKCS11       PKCS11KeyGenConfig `yaml:"pkcs11"`
	Key          keyGenConfig       `yaml:"key"`
	Outputs      struct {
		PublicKeyPath   string `yaml:"public-key-path"`
		CertificatePath string `yaml:"certificate-path"`
	} `yaml:"outputs"`
	CertProfile certProfile `yaml:"certificate-profile"`
	SkipLints   []string    `yaml:"skip-lints"`
}

func (rc rootConfig) validate() error {
	err := rc.PKCS11.validate()
	if err != nil {
		return err
	}

	// Key gen fields
	err = rc.Key.validate()
	if err != nil {
		return err
	}

	// Output fields
	err = checkOutputFile(rc.Outputs.PublicKeyPath, "public-key-path")
	if err != nil {
		return err
	}
	err = checkOutputFile(rc.Outputs.CertificatePath, "certificate-path")
	if err != nil {
		return err
	}

	// Certificate profile
	err = rc.CertProfile.verifyProfile(rootCert)
	if err != nil {
		return err
	}

	return nil
}

type PKCS11SigningConfig struct {
	Module       string `yaml:"module"`
	PIN          string `yaml:"pin"`
	SigningSlot  uint   `yaml:"signing-key-slot"`
	SigningLabel string `yaml:"signing-key-label"`
}

func (psc PKCS11SigningConfig) validate() error {
	if psc.Module == "" {
		return errors.New("pkcs11.module is required")
	}
	if psc.SigningLabel == "" {
		return errors.New("pkcs11.signing-key-label is required")
	}
	// key-slot is allowed to be 0 (which is a valid slot).
	return nil
}

type intermediateConfig struct {
	CeremonyType string              `yaml:"ceremony-type"`
	PKCS11       PKCS11SigningConfig `yaml:"pkcs11"`
	Inputs       struct {
		PublicKeyPath         string `yaml:"public-key-path"`
		IssuerCertificatePath string `yaml:"issuer-certificate-path"`
	} `yaml:"inputs"`
	Outputs struct {
		CertificatePath string `yaml:"certificate-path"`
	} `yaml:"outputs"`
	CertProfile certProfile `yaml:"certificate-profile"`
	SkipLints   []string    `yaml:"skip-lints"`
}

func (ic intermediateConfig) validate(ct certType) error {
	err := ic.PKCS11.validate()
	if err != nil {
		return err
	}

	// Input fields
	if ic.Inputs.PublicKeyPath == "" {
		return errors.New("inputs.public-key-path is required")
	}
	if ic.Inputs.IssuerCertificatePath == "" {
		return errors.New("inputs.issuer-certificate is required")
	}

	// Output fields
	err = checkOutputFile(ic.Outputs.CertificatePath, "certificate-path")
	if err != nil {
		return err
	}

	// Certificate profile
	err = ic.CertProfile.verifyProfile(ct)
	if err != nil {
		return err
	}

	return nil
}

type csrConfig struct {
	CeremonyType string              `yaml:"ceremony-type"`
	PKCS11       PKCS11SigningConfig `yaml:"pkcs11"`
	Inputs       struct {
		PublicKeyPath string `yaml:"public-key-path"`
	} `yaml:"inputs"`
	Outputs struct {
		CSRPath string `yaml:"csr-path"`
	} `yaml:"outputs"`
	CertProfile certProfile `yaml:"certificate-profile"`
}

func (cc csrConfig) validate() error {
	err := cc.PKCS11.validate()
	if err != nil {
		return err
	}

	// Input fields
	if cc.Inputs.PublicKeyPath == "" {
		return errors.New("inputs.public-key-path is required")
	}

	// Output fields
	err = checkOutputFile(cc.Outputs.CSRPath, "csr-path")
	if err != nil {
		return err
	}

	// Certificate profile
	err = cc.CertProfile.verifyProfile(requestCert)
	if err != nil {
		return err
	}

	return nil
}

type keyConfig struct {
	CeremonyType string             `yaml:"ceremony-type"`
	PKCS11       PKCS11KeyGenConfig `yaml:"pkcs11"`
	Key          keyGenConfig       `yaml:"key"`
	Outputs      struct {
		PublicKeyPath    string `yaml:"public-key-path"`
		PKCS11ConfigPath string `yaml:"pkcs11-config-path"`
	} `yaml:"outputs"`
}

func (kc keyConfig) validate() error {
	err := kc.PKCS11.validate()
	if err != nil {
		return err
	}

	// Key gen fields
	err = kc.Key.validate()
	if err != nil {
		return err
	}

	// Output fields
	err = checkOutputFile(kc.Outputs.PublicKeyPath, "public-key-path")
	if err != nil {
		return err
	}

	return nil
}

type ocspRespConfig struct {
	CeremonyType string              `yaml:"ceremony-type"`
	PKCS11       PKCS11SigningConfig `yaml:"pkcs11"`
	Inputs       struct {
		CertificatePath                string `yaml:"certificate-path"`
		IssuerCertificatePath          string `yaml:"issuer-certificate-path"`
		DelegatedIssuerCertificatePath string `yaml:"delegated-issuer-certificate-path"`
	} `yaml:"inputs"`
	Outputs struct {
		ResponsePath string `yaml:"response-path"`
	} `yaml:"outputs"`
	OCSPProfile struct {
		ThisUpdate string `yaml:"this-update"`
		NextUpdate string `yaml:"next-update"`
		Status     string `yaml:"status"`
	} `yaml:"ocsp-profile"`
}

func (orc ocspRespConfig) validate() error {
	err := orc.PKCS11.validate()
	if err != nil {
		return err
	}

	// Input fields
	if orc.Inputs.CertificatePath == "" {
		return errors.New("inputs.certificate-path is required")
	}
	if orc.Inputs.IssuerCertificatePath == "" {
		return errors.New("inputs.issuer-certificate-path is required")
	}
	// DelegatedIssuerCertificatePath may be omitted

	// Output fields
	err = checkOutputFile(orc.Outputs.ResponsePath, "response-path")
	if err != nil {
		return err
	}

	// OCSP fields
	if orc.OCSPProfile.ThisUpdate == "" {
		return errors.New("ocsp-profile.this-update is required")
	}
	if orc.OCSPProfile.NextUpdate == "" {
		return errors.New("ocsp-profile.next-update is required")
	}
	if orc.OCSPProfile.Status != "good" && orc.OCSPProfile.Status != "revoked" {
		return errors.New("ocsp-profile.status must be either \"good\" or \"revoked\"")
	}

	return nil
}

type crlConfig struct {
	CeremonyType string              `yaml:"ceremony-type"`
	PKCS11       PKCS11SigningConfig `yaml:"pkcs11"`
	Inputs       struct {
		IssuerCertificatePath string `yaml:"issuer-certificate-path"`
	} `yaml:"inputs"`
	Outputs struct {
		CRLPath string `yaml:"crl-path"`
	} `yaml:"outputs"`
	CRLProfile struct {
		ThisUpdate          string `yaml:"this-update"`
		NextUpdate          string `yaml:"next-update"`
		Number              int64  `yaml:"number"`
		RevokedCertificates []struct {
			CertificatePath  string `yaml:"certificate-path"`
			RevocationDate   string `yaml:"revocation-date"`
			RevocationReason int    `yaml:"revocation-reason"`
		} `yaml:"revoked-certificates"`
	} `yaml:"crl-profile"`
}

func (cc crlConfig) validate() error {
	err := cc.PKCS11.validate()
	if err != nil {
		return err
	}

	// Input fields
	if cc.Inputs.IssuerCertificatePath == "" {
		return errors.New("inputs.issuer-certificate-path is required")
	}

	// Output fields
	err = checkOutputFile(cc.Outputs.CRLPath, "crl-path")
	if err != nil {
		return err
	}

	// CRL profile fields
	if cc.CRLProfile.ThisUpdate == "" {
		return errors.New("crl-profile.this-update is required")
	}
	if cc.CRLProfile.NextUpdate == "" {
		return errors.New("crl-profile.next-update is required")
	}
	if cc.CRLProfile.Number == 0 {
		return errors.New("crl-profile.number must be non-zero")
	}
	for _, rc := range cc.CRLProfile.RevokedCertificates {
		if rc.CertificatePath == "" {
			return errors.New("crl-profile.revoked-certificates.certificate-path is required")
		}
		if rc.RevocationDate == "" {
			return errors.New("crl-profile.revoked-certificates.revocation-date is required")
		}
		if rc.RevocationReason == 0 {
			return errors.New("crl-profile.revoked-certificates.revocation-reason is required")
		}
	}

	return nil
}

// loadCert loads a PEM certificate specified by filename or returns an error
func loadCert(filename string) (cert *x509.Certificate, err error) {
	certPEM, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("No data in cert PEM file %s", filename)
	}
	cert, err = x509.ParseCertificate(block.Bytes)
	return
}

func equalPubKeys(a, b interface{}) bool {
	aBytes, err := x509.MarshalPKIXPublicKey(a)
	if err != nil {
		return false
	}
	bBytes, err := x509.MarshalPKIXPublicKey(b)
	if err != nil {
		return false
	}
	return bytes.Equal(aBytes, bBytes)
}

func openSigner(cfg PKCS11SigningConfig, pubKey crypto.PublicKey) (crypto.Signer, *hsmRandReader, error) {
	session, err := pkcs11helpers.Initialize(cfg.Module, cfg.SigningSlot, cfg.PIN)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup session and PKCS#11 context for slot %d: %s",
			cfg.SigningSlot, err)
	}
	log.Printf("Opened PKCS#11 session for slot %d\n", cfg.SigningSlot)
	signer, err := session.NewSigner(cfg.SigningLabel, pubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve private key handle: %s", err)
	}
	if !equalPubKeys(signer.Public(), pubKey) {
		return nil, nil, fmt.Errorf("signer pubkey did not match issuer pubkey")
	}
	log.Println("Retrieved private key handle")
	return signer, newRandReader(session), nil
}

func signAndWriteCert(tbs, issuer *x509.Certificate, subjectPubKey crypto.PublicKey, signer crypto.Signer, certPath string, skipLints []string) error {
	err := linter.Check(tbs, subjectPubKey, issuer, signer, skipLints)
	if err != nil {
		return fmt.Errorf("certificate failed pre-issuance lint: %w", err)
	}
	// x509.CreateCertificate uses a io.Reader here for signing methods that require
	// a source of randomness. Since PKCS#11 based signing generates needed randomness
	// at the HSM we don't need to pass a real reader. Instead of passing a nil reader
	// we use one that always returns errors in case the internal usage of this reader
	// changes.
	certBytes, err := x509.CreateCertificate(&failReader{}, tbs, issuer, subjectPubKey, signer)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %s", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	log.Printf("Signed certificate PEM:\n%s", pemBytes)
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return fmt.Errorf("failed to parse signed certificate: %s", err)
	}
	if tbs == issuer {
		// If cert is self-signed we need to populate the issuer subject key to
		// verify the signature
		issuer.PublicKey = cert.PublicKey
		issuer.PublicKeyAlgorithm = cert.PublicKeyAlgorithm
	}

	err = cert.CheckSignatureFrom(issuer)
	if err != nil {
		return fmt.Errorf("failed to verify certificate signature: %s", err)
	}
	err = writeFile(certPath, pemBytes)
	if err != nil {
		return fmt.Errorf("failed to write certificate to %q: %s", certPath, err)
	}
	log.Printf("Certificate written to %q\n", certPath)
	return nil
}

func rootCeremony(configBytes []byte) error {
	d := yaml.NewDecoder(bytes.NewReader(configBytes))
	d.KnownFields(true)

	var config rootConfig
	err := d.Decode(&config)
	if err != nil {
		return fmt.Errorf("failed to parse config: %s", err)
	}
	err = config.validate()
	if err != nil {
		return fmt.Errorf("failed to validate config: %s", err)
	}
	session, err := pkcs11helpers.Initialize(config.PKCS11.Module, config.PKCS11.StoreSlot, config.PKCS11.PIN)
	if err != nil {
		return fmt.Errorf("failed to setup session and PKCS#11 context for slot %d: %s", config.PKCS11.StoreSlot, err)
	}
	log.Printf("Opened PKCS#11 session for slot %d\n", config.PKCS11.StoreSlot)
	keyInfo, err := generateKey(session, config.PKCS11.StoreLabel, config.Outputs.PublicKeyPath, config.Key)
	if err != nil {
		return err
	}
	signer, err := session.NewSigner(config.PKCS11.StoreLabel, keyInfo.key)
	if err != nil {
		return fmt.Errorf("failed to retrieve signer: %s", err)
	}
	template, err := makeTemplate(newRandReader(session), &config.CertProfile, keyInfo.der, rootCert)
	if err != nil {
		return fmt.Errorf("failed to create certificate profile: %s", err)
	}

	err = signAndWriteCert(template, template, keyInfo.key, signer, config.Outputs.CertificatePath, config.SkipLints)
	if err != nil {
		return err
	}

	return nil
}

func intermediateCeremony(configBytes []byte, ct certType) error {
	d := yaml.NewDecoder(bytes.NewReader(configBytes))
	d.KnownFields(true)

	var config intermediateConfig
	err := d.Decode(&config)
	if err != nil {
		return fmt.Errorf("failed to parse config: %s", err)
	}
	err = config.validate(ct)
	if err != nil {
		return fmt.Errorf("failed to validate config: %s", err)
	}

	pubPEMBytes, err := os.ReadFile(config.Inputs.PublicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key %q: %s", config.Inputs.PublicKeyPath, err)
	}
	pubPEM, _ := pem.Decode(pubPEMBytes)
	if pubPEM == nil {
		return fmt.Errorf("failed to parse public key")
	}
	pub, err := x509.ParsePKIXPublicKey(pubPEM.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %s", err)
	}
	issuer, err := loadCert(config.Inputs.IssuerCertificatePath)
	if err != nil {
		return fmt.Errorf("failed to load issuer certificate %q: %s", config.Inputs.IssuerCertificatePath, err)
	}

	signer, randReader, err := openSigner(config.PKCS11, issuer.PublicKey)
	if err != nil {
		return err
	}

	template, err := makeTemplate(randReader, &config.CertProfile, pubPEM.Bytes, ct)
	if err != nil {
		return fmt.Errorf("failed to create certificate profile: %s", err)
	}
	template.AuthorityKeyId = issuer.SubjectKeyId

	err = signAndWriteCert(template, issuer, pub, signer, config.Outputs.CertificatePath, config.SkipLints)
	if err != nil {
		return err
	}

	return nil
}

func csrCeremony(configBytes []byte) error {
	d := yaml.NewDecoder(bytes.NewReader(configBytes))
	d.KnownFields(true)

	var config csrConfig
	err := d.Decode(&config)
	if err != nil {
		return fmt.Errorf("failed to parse config: %s", err)
	}
	err = config.validate()
	if err != nil {
		return fmt.Errorf("failed to validate config: %s", err)
	}

	pubPEMBytes, err := os.ReadFile(config.Inputs.PublicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key %q: %s", config.Inputs.PublicKeyPath, err)
	}
	pubPEM, _ := pem.Decode(pubPEMBytes)
	if pubPEM == nil {
		return fmt.Errorf("failed to parse public key")
	}
	pub, err := x509.ParsePKIXPublicKey(pubPEM.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %s", err)
	}

	signer, _, err := openSigner(config.PKCS11, pub)
	if err != nil {
		return err
	}

	csrDER, err := generateCSR(&config.CertProfile, signer)
	if err != nil {
		return fmt.Errorf("failed to generate CSR: %s", err)
	}
	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})
	err = writeFile(config.Outputs.CSRPath, csrPEM)
	if err != nil {
		return fmt.Errorf("failed to write CSR to %q: %s", config.Outputs.CSRPath, err)
	}
	log.Printf("CSR written to %q\n", config.Outputs.CSRPath)

	return nil
}

func keyCeremony(configBytes []byte) error {
	d := yaml.NewDecoder(bytes.NewReader(configBytes))
	d.KnownFields(true)

	var config keyConfig
	err := d.Decode(&config)
	if err != nil {
		return fmt.Errorf("failed to parse config: %s", err)
	}
	err = config.validate()
	if err != nil {
		return fmt.Errorf("failed to validate config: %s", err)
	}
	session, err := pkcs11helpers.Initialize(config.PKCS11.Module, config.PKCS11.StoreSlot, config.PKCS11.PIN)
	if err != nil {
		return fmt.Errorf("failed to setup session and PKCS#11 context for slot %d: %s", config.PKCS11.StoreSlot, err)
	}
	log.Printf("Opened PKCS#11 session for slot %d\n", config.PKCS11.StoreSlot)
	if _, err = generateKey(session, config.PKCS11.StoreLabel, config.Outputs.PublicKeyPath, config.Key); err != nil {
		return err
	}

	if config.Outputs.PKCS11ConfigPath != "" {
		contents := fmt.Sprintf(
			`{"module": %q, "tokenLabel": %q, "pin": %q}`,
			config.PKCS11.Module, config.PKCS11.StoreLabel, config.PKCS11.PIN,
		)
		err = writeFile(config.Outputs.PKCS11ConfigPath, []byte(contents))
		if err != nil {
			return err
		}
	}

	return nil
}

func ocspRespCeremony(configBytes []byte) error {
	d := yaml.NewDecoder(bytes.NewReader(configBytes))
	d.KnownFields(true)

	var config ocspRespConfig
	err := d.Decode(&config)
	if err != nil {
		return fmt.Errorf("failed to parse config: %s", err)
	}
	err = config.validate()
	if err != nil {
		return fmt.Errorf("failed to validate config: %s", err)
	}

	cert, err := loadCert(config.Inputs.CertificatePath)
	if err != nil {
		return fmt.Errorf("failed to load certificate %q: %s", config.Inputs.CertificatePath, err)
	}
	issuer, err := loadCert(config.Inputs.IssuerCertificatePath)
	if err != nil {
		return fmt.Errorf("failed to load issuer certificate %q: %s", config.Inputs.IssuerCertificatePath, err)
	}
	var signer crypto.Signer
	var delegatedIssuer *x509.Certificate
	if config.Inputs.DelegatedIssuerCertificatePath != "" {
		delegatedIssuer, err = loadCert(config.Inputs.DelegatedIssuerCertificatePath)
		if err != nil {
			return fmt.Errorf("failed to load delegated issuer certificate %q: %s", config.Inputs.DelegatedIssuerCertificatePath, err)
		}

		signer, _, err = openSigner(config.PKCS11, delegatedIssuer.PublicKey)
		if err != nil {
			return err
		}
	} else {
		signer, _, err = openSigner(config.PKCS11, issuer.PublicKey)
		if err != nil {
			return err
		}
	}

	thisUpdate, err := time.Parse(configDateLayout, config.OCSPProfile.ThisUpdate)
	if err != nil {
		return fmt.Errorf("unable to parse ocsp-profile.this-update: %s", err)
	}
	nextUpdate, err := time.Parse(configDateLayout, config.OCSPProfile.NextUpdate)
	if err != nil {
		return fmt.Errorf("unable to parse ocsp-profile.next-update: %s", err)
	}
	var status int
	switch config.OCSPProfile.Status {
	case "good":
		status = int(ocsp.Good)
	case "revoked":
		status = int(ocsp.Revoked)
	default:
		// this shouldn't happen if the config is validated
		return fmt.Errorf("unexpected ocsp-profile.stats: %s", config.OCSPProfile.Status)
	}

	resp, err := generateOCSPResponse(signer, issuer, delegatedIssuer, cert, thisUpdate, nextUpdate, status)
	if err != nil {
		return err
	}

	err = writeFile(config.Outputs.ResponsePath, resp)
	if err != nil {
		return fmt.Errorf("failed to write OCSP response to %q: %s", config.Outputs.ResponsePath, err)
	}

	return nil
}

func crlCeremony(configBytes []byte) error {
	d := yaml.NewDecoder(bytes.NewReader(configBytes))
	d.KnownFields(true)

	var config crlConfig
	err := d.Decode(&config)
	if err != nil {
		return fmt.Errorf("failed to parse config: %s", err)
	}
	err = config.validate()
	if err != nil {
		return fmt.Errorf("failed to validate config: %s", err)
	}

	issuer, err := loadCert(config.Inputs.IssuerCertificatePath)
	if err != nil {
		return fmt.Errorf("failed to load issuer certificate %q: %s", config.Inputs.IssuerCertificatePath, err)
	}
	signer, _, err := openSigner(config.PKCS11, issuer.PublicKey)
	if err != nil {
		return err
	}

	thisUpdate, err := time.Parse(configDateLayout, config.CRLProfile.ThisUpdate)
	if err != nil {
		return fmt.Errorf("unable to parse crl-profile.this-update: %s", err)
	}
	nextUpdate, err := time.Parse(configDateLayout, config.CRLProfile.NextUpdate)
	if err != nil {
		return fmt.Errorf("unable to parse crl-profile.next-update: %s", err)
	}

	var revokedCertificates []pkix.RevokedCertificate
	for _, rc := range config.CRLProfile.RevokedCertificates {
		cert, err := loadCert(rc.CertificatePath)
		if err != nil {
			return fmt.Errorf("failed to load revoked certificate %q: %s", rc.CertificatePath, err)
		}
		revokedAt, err := time.Parse(configDateLayout, rc.RevocationDate)
		if err != nil {
			return fmt.Errorf("unable to parse crl-profile.revoked-certificates.revocation-date")
		}
		revokedCert := pkix.RevokedCertificate{
			SerialNumber:   cert.SerialNumber,
			RevocationTime: revokedAt,
		}
		encReason, err := asn1.Marshal(rc.RevocationReason)
		if err != nil {
			return fmt.Errorf("failed to marshal revocation reason %q: %s", rc.RevocationReason, err)
		}
		revokedCert.Extensions = []pkix.Extension{{
			Id:    asn1.ObjectIdentifier{2, 5, 29, 21}, // id-ce-reasonCode
			Value: encReason,
		}}
		revokedCertificates = append(revokedCertificates, revokedCert)
	}

	crlBytes, err := generateCRL(signer, issuer, thisUpdate, nextUpdate, config.CRLProfile.Number, revokedCertificates)
	if err != nil {
		return err
	}

	log.Printf("Signed CRL PEM:\n%s", crlBytes)

	err = writeFile(config.Outputs.CRLPath, crlBytes)
	if err != nil {
		return fmt.Errorf("failed to write CRL to %q: %s", config.Outputs.CRLPath, err)
	}

	return nil
}

func main() {
	configPath := flag.String("config", "", "Path to ceremony configuration file")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("--config is required")
	}
	configBytes, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to read config file: %s", err)
	}
	var ct struct {
		CeremonyType string `yaml:"ceremony-type"`
	}
	err = yaml.Unmarshal(configBytes, &ct)
	if err != nil {
		log.Fatalf("Failed to parse config: %s", err)
	}

	switch ct.CeremonyType {
	case "root":
		err = rootCeremony(configBytes)
		if err != nil {
			log.Fatalf("root ceremony failed: %s", err)
		}
	case "cross-certificate":
		err = intermediateCeremony(configBytes, crossCert)
		if err != nil {
			log.Fatalf("cross-certificate ceremony failed: %s", err)
		}
	case "intermediate":
		err = intermediateCeremony(configBytes, intermediateCert)
		if err != nil {
			log.Fatalf("intermediate ceremony failed: %s", err)
		}
	case "cross-csr":
		err = csrCeremony(configBytes)
		if err != nil {
			log.Fatalf("cross-csr ceremony failed: %s", err)
		}
	case "ocsp-signer":
		err = intermediateCeremony(configBytes, ocspCert)
		if err != nil {
			log.Fatalf("ocsp signer ceremony failed: %s", err)
		}
	case "key":
		err = keyCeremony(configBytes)
		if err != nil {
			log.Fatalf("key ceremony failed: %s", err)
		}
	case "ocsp-response":
		err = ocspRespCeremony(configBytes)
		if err != nil {
			log.Fatalf("ocsp response ceremony failed: %s", err)
		}
	case "crl":
		err = crlCeremony(configBytes)
		if err != nil {
			log.Fatalf("crl ceremony failed: %s", err)
		}
	case "crl-signer":
		err = intermediateCeremony(configBytes, crlCert)
		if err != nil {
			log.Fatalf("crl signer ceremony failed: %s", err)
		}
	default:
		log.Fatalf("unknown ceremony-type, must be one of: root, intermediate, ocsp-signer, crl-signer, key, ocsp-response")
	}
}

func init() {
	cmd.RegisterCommand("ceremony", main)
}
