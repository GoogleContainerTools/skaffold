package ca

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/beeker1121/goque"
	ct "github.com/google/certificate-transparency-go"
	cttls "github.com/google/certificate-transparency-go/tls"
	"github.com/jmhodges/clock"
	"github.com/miekg/pkcs11"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/ocsp"

	capb "github.com/letsencrypt/boulder/ca/proto"
	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	csrlib "github.com/letsencrypt/boulder/csr"
	berrors "github.com/letsencrypt/boulder/errors"
	"github.com/letsencrypt/boulder/features"
	"github.com/letsencrypt/boulder/goodkey"
	"github.com/letsencrypt/boulder/issuance"
	blog "github.com/letsencrypt/boulder/log"
	sapb "github.com/letsencrypt/boulder/sa/proto"
)

type certificateType string

const (
	precertType = certificateType("precertificate")
	certType    = certificateType("certificate")
)

// Two maps of keys to Issuers. Lookup by PublicKeyAlgorithm is useful for
// determining which issuer to use to sign a given (pre)cert, based on its
// PublicKeyAlgorithm. Lookup by NameID is useful for looking up the appropriate
// issuer based on the issuer of a given (pre)certificate.
type issuerMaps struct {
	byAlg    map[x509.PublicKeyAlgorithm]*issuance.Issuer
	byNameID map[issuance.IssuerNameID]*issuance.Issuer
}

// certificateAuthorityImpl represents a CA that signs certificates.
// It can sign OCSP responses as well, but only via delegation to an ocspImpl.
type certificateAuthorityImpl struct {
	capb.UnimplementedCertificateAuthorityServer
	capb.UnimplementedOCSPGeneratorServer
	sa      sapb.StorageAuthorityCertificateClient
	pa      core.PolicyAuthority
	issuers issuerMaps
	// TODO(#6448): Remove these.
	ocsp capb.OCSPGeneratorServer
	crl  capb.CRLGeneratorServer

	// This is temporary, and will be used for testing and slow roll-out
	// of ECDSA issuance, but will then be removed.
	ecdsaAllowList     *ECDSAAllowList
	prefix             int // Prepended to the serial number
	validityPeriod     time.Duration
	backdate           time.Duration
	maxNames           int
	keyPolicy          goodkey.KeyPolicy
	orphanQueue        *goque.Queue
	clk                clock.Clock
	log                blog.Logger
	signatureCount     *prometheus.CounterVec
	orphanCount        *prometheus.CounterVec
	adoptedOrphanCount *prometheus.CounterVec
	signErrorCount     *prometheus.CounterVec
}

// makeIssuerMaps processes a list of issuers into a set of maps, mapping
// nearly-unique identifiers of those issuers to the issuers themselves. Note
// that, if two issuers have the same nearly-unique ID, the *latter* one in
// the input list "wins".
func makeIssuerMaps(issuers []*issuance.Issuer) issuerMaps {
	issuersByAlg := make(map[x509.PublicKeyAlgorithm]*issuance.Issuer, 2)
	issuersByNameID := make(map[issuance.IssuerNameID]*issuance.Issuer, len(issuers))
	for _, issuer := range issuers {
		for _, alg := range issuer.Algs() {
			// TODO(#5259): Enforce that there is only one issuer for each algorithm,
			// instead of taking the first issuer for each algorithm type.
			if issuersByAlg[alg] == nil {
				issuersByAlg[alg] = issuer
			}
		}
		issuersByNameID[issuer.Cert.NameID()] = issuer
	}
	return issuerMaps{issuersByAlg, issuersByNameID}
}

// NewCertificateAuthorityImpl creates a CA instance that can sign certificates
// from any number of issuance.Issuers according to their profiles, and can sign
// OCSP (via delegation to an ocspImpl and its issuers).
func NewCertificateAuthorityImpl(
	sa sapb.StorageAuthorityCertificateClient,
	pa core.PolicyAuthority,
	ocsp capb.OCSPGeneratorServer,
	crl capb.CRLGeneratorServer,
	boulderIssuers []*issuance.Issuer,
	ecdsaAllowList *ECDSAAllowList,
	certExpiry time.Duration,
	certBackdate time.Duration,
	serialPrefix int,
	maxNames int,
	keyPolicy goodkey.KeyPolicy,
	orphanQueue *goque.Queue,
	logger blog.Logger,
	stats prometheus.Registerer,
	signatureCount *prometheus.CounterVec,
	signErrorCount *prometheus.CounterVec,
	clk clock.Clock,
) (*certificateAuthorityImpl, error) {
	var ca *certificateAuthorityImpl
	var err error

	// TODO(briansmith): Make the backdate setting mandatory after the
	// production ca.json has been updated to include it. Until then, manually
	// default to 1h, which is the backdating duration we currently use.
	if certBackdate == 0 {
		certBackdate = time.Hour
	}

	if serialPrefix <= 0 || serialPrefix >= 256 {
		err = errors.New("Must have a positive non-zero serial prefix less than 256 for CA.")
		return nil, err
	}

	issuers := makeIssuerMaps(boulderIssuers)

	orphanCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orphans",
			Help: "Number of orphaned certificates labelled by type (precert, cert)",
		},
		[]string{"type"})
	stats.MustRegister(orphanCount)

	adoptedOrphanCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "adopted_orphans",
			Help: "Number of orphaned certificates adopted from the orphan queue by type (precert, cert)",
		},
		[]string{"type"})
	stats.MustRegister(adoptedOrphanCount)

	ca = &certificateAuthorityImpl{
		sa:                 sa,
		pa:                 pa,
		ocsp:               ocsp,
		crl:                crl,
		issuers:            issuers,
		validityPeriod:     certExpiry,
		backdate:           certBackdate,
		prefix:             serialPrefix,
		maxNames:           maxNames,
		keyPolicy:          keyPolicy,
		orphanQueue:        orphanQueue,
		log:                logger,
		signatureCount:     signatureCount,
		orphanCount:        orphanCount,
		adoptedOrphanCount: adoptedOrphanCount,
		signErrorCount:     signErrorCount,
		clk:                clk,
		ecdsaAllowList:     ecdsaAllowList,
	}

	return ca, nil
}

// noteSignError is called after operations that may cause a PKCS11 signing error.
func (ca *certificateAuthorityImpl) noteSignError(err error) {
	var pkcs11Error *pkcs11.Error
	if errors.As(err, &pkcs11Error) {
		ca.signErrorCount.WithLabelValues("HSM").Inc()
	}
}

var ocspStatusToCode = map[string]int{
	"good":    ocsp.Good,
	"revoked": ocsp.Revoked,
	"unknown": ocsp.Unknown,
}

func (ca *certificateAuthorityImpl) IssuePrecertificate(ctx context.Context, issueReq *capb.IssueCertificateRequest) (*capb.IssuePrecertificateResponse, error) {
	// issueReq.orderID may be zero, for ACMEv1 requests.
	if core.IsAnyNilOrZero(issueReq, issueReq.Csr, issueReq.RegistrationID) {
		return nil, berrors.InternalServerError("Incomplete issue certificate request")
	}

	serialBigInt, validity, err := ca.generateSerialNumberAndValidity()
	if err != nil {
		return nil, err
	}

	serialHex := core.SerialToString(serialBigInt)
	regID := issueReq.RegistrationID
	nowNanos := ca.clk.Now().UnixNano()
	expiresNanos := validity.NotAfter.UnixNano()
	_, err = ca.sa.AddSerial(ctx, &sapb.AddSerialRequest{
		Serial:  serialHex,
		RegID:   regID,
		Created: nowNanos,
		Expires: expiresNanos,
	})
	if err != nil {
		return nil, err
	}

	precertDER, ocspResp, issuer, err := ca.issuePrecertificateInner(ctx, issueReq, serialBigInt, validity)
	if err != nil {
		return nil, err
	}
	issuerID := issuer.Cert.NameID()

	req := &sapb.AddCertificateRequest{
		Der:      precertDER,
		RegID:    regID,
		Ocsp:     ocspResp,
		Issued:   nowNanos,
		IssuerID: int64(issuerID),
	}

	_, err = ca.sa.AddPrecertificate(ctx, req)
	if err != nil {
		ca.orphanCount.With(prometheus.Labels{"type": "precert"}).Inc()
		err = berrors.InternalServerError(err.Error())
		// Note: This log line is parsed by cmd/orphan-finder. If you make any
		// changes here, you should make sure they are reflected in orphan-finder.
		ca.log.AuditErrf("Failed RPC to store at SA, orphaning precertificate: serial=[%s], cert=[%s], issuerID=[%d], regID=[%d], orderID=[%d], err=[%v]",
			serialHex, hex.EncodeToString(precertDER), issuerID, issueReq.RegistrationID, issueReq.OrderID, err)
		if ca.orphanQueue != nil {
			ca.queueOrphan(&orphanedCert{
				DER:      precertDER,
				RegID:    regID,
				OCSPResp: ocspResp,
				Precert:  true,
				IssuerID: int64(issuerID),
			})
		}
		return nil, err
	}

	return &capb.IssuePrecertificateResponse{
		DER: precertDER,
	}, nil
}

// IssueCertificateForPrecertificate takes a precertificate and a set
// of SCTs for that precertificate and uses the signer to create and
// sign a certificate from them. The poison extension is removed and a
// SCT list extension is inserted in its place. Except for this and the
// signature the certificate exactly matches the precertificate. After
// the certificate is signed a OCSP response is generated and the
// response and certificate are stored in the database.
//
// It's critical not to sign two different final certificates for the same
// precertificate. This can happen, for instance, if the caller provides a
// different set of SCTs on subsequent calls to  IssueCertificateForPrecertificate.
// We rely on the RA not to call IssueCertificateForPrecertificate twice for the
// same serial. This is accomplished by the fact that
// IssueCertificateForPrecertificate is only ever called in a straight-through
// RPC path without retries. If there is any error, including a networking
// error, the whole certificate issuance attempt fails and any subsequent
// issuance will use a different serial number.
//
// We also check that the provided serial number does not already exist as a
// final certificate, but this is just a belt-and-suspenders measure, since
// there could be race conditions where two goroutines are issuing for the same
// serial number at the same time.
func (ca *certificateAuthorityImpl) IssueCertificateForPrecertificate(ctx context.Context, req *capb.IssueCertificateForPrecertificateRequest) (*corepb.Certificate, error) {
	// issueReq.orderID may be zero, for ACMEv1 requests.
	if core.IsAnyNilOrZero(req, req.DER, req.SCTs, req.RegistrationID) {
		return nil, berrors.InternalServerError("Incomplete cert for precertificate request")
	}

	precert, err := x509.ParseCertificate(req.DER)
	if err != nil {
		return nil, err
	}

	serialHex := core.SerialToString(precert.SerialNumber)
	if _, err = ca.sa.GetCertificate(ctx, &sapb.Serial{Serial: serialHex}); err == nil {
		err = berrors.InternalServerError("issuance of duplicate final certificate requested: %s", serialHex)
		ca.log.AuditErr(err.Error())
		return nil, err
	} else if !errors.Is(err, berrors.NotFound) {
		return nil, fmt.Errorf("error checking for duplicate issuance of %s: %s", serialHex, err)
	}
	var scts []ct.SignedCertificateTimestamp
	for _, sctBytes := range req.SCTs {
		var sct ct.SignedCertificateTimestamp
		_, err = cttls.Unmarshal(sctBytes, &sct)
		if err != nil {
			return nil, err
		}
		scts = append(scts, sct)
	}

	issuer, ok := ca.issuers.byNameID[issuance.GetIssuerNameID(precert)]
	if !ok {
		return nil, berrors.InternalServerError("no issuer found for Issuer Name %s", precert.Issuer)
	}

	issuanceReq, err := issuance.RequestFromPrecert(precert, scts)
	if err != nil {
		return nil, err
	}

	names := strings.Join(issuanceReq.DNSNames, ", ")

	ca.log.AuditInfof("Signing cert: serial=[%s] regID=[%d] names=[%s] precert=[%s]",
		serialHex, req.RegistrationID, names, hex.EncodeToString(precert.Raw))

	certDER, err := issuer.Issue(issuanceReq)
	if err != nil {
		ca.noteSignError(err)
		ca.log.AuditErrf("Signing cert failed: serial=[%s] regID=[%d] names=[%s] err=[%v]",
			serialHex, req.RegistrationID, names, err)
		return nil, berrors.InternalServerError("failed to sign precertificate: %s", err)
	}

	ca.signatureCount.With(prometheus.Labels{"purpose": string(certType), "issuer": issuer.Name()}).Inc()
	ca.log.AuditInfof("Signing cert success: serial=[%s] regID=[%d] names=[%s] certificate=[%s]",
		serialHex, req.RegistrationID, names, hex.EncodeToString(certDER))

	err = ca.storeCertificate(ctx, req.RegistrationID, req.OrderID, precert.SerialNumber, certDER, int64(issuer.Cert.NameID()))
	if err != nil {
		return nil, err
	}

	return &corepb.Certificate{
		RegistrationID: req.RegistrationID,
		Serial:         core.SerialToString(precert.SerialNumber),
		Der:            certDER,
		Digest:         core.Fingerprint256(certDER),
		Issued:         precert.NotBefore.UnixNano(),
		Expires:        precert.NotAfter.UnixNano(),
	}, nil
}

type validity struct {
	NotBefore time.Time
	NotAfter  time.Time
}

func (ca *certificateAuthorityImpl) generateSerialNumberAndValidity() (*big.Int, validity, error) {
	// We want 136 bits of random number, plus an 8-bit instance id prefix.
	const randBits = 136
	serialBytes := make([]byte, randBits/8+1)
	serialBytes[0] = byte(ca.prefix)
	_, err := rand.Read(serialBytes[1:])
	if err != nil {
		err = berrors.InternalServerError("failed to generate serial: %s", err)
		ca.log.AuditErrf("Serial randomness failed, err=[%v]", err)
		return nil, validity{}, err
	}
	serialBigInt := big.NewInt(0)
	serialBigInt = serialBigInt.SetBytes(serialBytes)

	notBefore := ca.clk.Now().Add(-ca.backdate)
	validity := validity{
		NotBefore: notBefore,
		NotAfter:  notBefore.Add(ca.validityPeriod - time.Second),
	}

	return serialBigInt, validity, nil
}

func (ca *certificateAuthorityImpl) issuePrecertificateInner(ctx context.Context, issueReq *capb.IssueCertificateRequest, serialBigInt *big.Int, validity validity) ([]byte, []byte, *issuance.Issuer, error) {
	csr, err := x509.ParseCertificateRequest(issueReq.Csr)
	if err != nil {
		return nil, nil, nil, err
	}

	err = csrlib.VerifyCSR(ctx, csr, ca.maxNames, &ca.keyPolicy, ca.pa)
	if err != nil {
		ca.log.AuditErr(err.Error())
		// VerifyCSR returns berror instances that can be passed through as-is
		// without wrapping.
		return nil, nil, nil, err
	}

	var issuer *issuance.Issuer
	var ok bool
	if issueReq.IssuerNameID == 0 {
		// Use the issuer which corresponds to the algorithm of the public key
		// contained in the CSR, unless we have an allowlist of registration IDs
		// for ECDSA, in which case switch all not-allowed accounts to RSA issuance.
		alg := csr.PublicKeyAlgorithm
		if alg == x509.ECDSA && !features.Enabled(features.ECDSAForAll) && ca.ecdsaAllowList != nil && !ca.ecdsaAllowList.permitted(issueReq.RegistrationID) {
			alg = x509.RSA
		}
		issuer, ok = ca.issuers.byAlg[alg]
		if !ok {
			return nil, nil, nil, berrors.InternalServerError("no issuer found for public key algorithm %s", csr.PublicKeyAlgorithm)
		}
	} else {
		issuer, ok = ca.issuers.byNameID[issuance.IssuerNameID(issueReq.IssuerNameID)]
		if !ok {
			return nil, nil, nil, berrors.InternalServerError("no issuer found for IssuerNameID %d", issueReq.IssuerNameID)
		}
	}

	if issuer.Cert.NotAfter.Before(validity.NotAfter) {
		err = berrors.InternalServerError("cannot issue a certificate that expires after the issuer certificate")
		ca.log.AuditErr(err.Error())
		return nil, nil, nil, err
	}

	serialHex := core.SerialToString(serialBigInt)

	var ocspResp []byte
	if !features.Enabled(features.ROCSPStage7) {
		// Generate ocsp response before issuing precertificate
		ocspRespPB, err := ca.ocsp.GenerateOCSP(ctx, &capb.GenerateOCSPRequest{
			Serial:   serialHex,
			IssuerID: int64(issuer.Cert.NameID()),
			Status:   string(core.OCSPStatusGood),
		})
		if err != nil {
			err = berrors.InternalServerError(err.Error())
			ca.log.AuditInfof("OCSP Signing for precertificate failure: serial=[%s] err=[%s]", serialHex, err)
			return nil, nil, nil, err
		}
		ocspResp = ocspRespPB.Response
	}

	ca.log.AuditInfof("Signing precert: serial=[%s] regID=[%d] names=[%s] csr=[%s]",
		serialHex, issueReq.RegistrationID, strings.Join(csr.DNSNames, ", "), hex.EncodeToString(csr.Raw))

	certDER, err := issuer.Issue(&issuance.IssuanceRequest{
		PublicKey:         csr.PublicKey,
		Serial:            serialBigInt.Bytes(),
		CommonName:        csr.Subject.CommonName,
		DNSNames:          csr.DNSNames,
		IncludeCTPoison:   true,
		IncludeMustStaple: issuance.ContainsMustStaple(csr.Extensions),
		NotBefore:         validity.NotBefore,
		NotAfter:          validity.NotAfter,
	})
	if err != nil {
		ca.noteSignError(err)
		ca.log.AuditErrf("Signing precert failed: serial=[%s] regID=[%d] names=[%s] err=[%v]",
			serialHex, issueReq.RegistrationID, strings.Join(csr.DNSNames, ", "), err)
		return nil, nil, nil, berrors.InternalServerError("failed to sign precertificate: %s", err)
	}

	ca.signatureCount.With(prometheus.Labels{"purpose": string(precertType), "issuer": issuer.Name()}).Inc()
	ca.log.AuditInfof("Signing precert success: serial=[%s] regID=[%d] names=[%s] precertificate=[%s]",
		serialHex, issueReq.RegistrationID, strings.Join(csr.DNSNames, ", "), hex.EncodeToString(certDER))

	return certDER, ocspResp, issuer, nil
}

func (ca *certificateAuthorityImpl) storeCertificate(
	ctx context.Context,
	regID int64,
	orderID int64,
	serialBigInt *big.Int,
	certDER []byte,
	issuerID int64) error {
	var err error
	_, err = ca.sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    certDER,
		RegID:  regID,
		Issued: ca.clk.Now().UnixNano(),
	})
	if err != nil {
		ca.orphanCount.With(prometheus.Labels{"type": "cert"}).Inc()
		err = berrors.InternalServerError(err.Error())
		// Note: This log line is parsed by cmd/orphan-finder. If you make any
		// changes here, you should make sure they are reflected in orphan-finder.
		ca.log.AuditErrf("Failed RPC to store at SA, orphaning certificate: serial=[%s], cert=[%s], issuerID=[%d], regID=[%d], orderID=[%d], err=[%v]",
			core.SerialToString(serialBigInt), hex.EncodeToString(certDER), issuerID, regID, orderID, err)
		if ca.orphanQueue != nil {
			ca.queueOrphan(&orphanedCert{
				DER:      certDER,
				RegID:    regID,
				IssuerID: issuerID,
			})
		}
		return err
	}
	return nil
}

type orphanedCert struct {
	DER      []byte
	OCSPResp []byte
	RegID    int64
	Precert  bool
	IssuerID int64
}

func (ca *certificateAuthorityImpl) queueOrphan(o *orphanedCert) {
	if _, err := ca.orphanQueue.EnqueueObject(o); err != nil {
		ca.log.AuditErrf("failed to queue orphan for integration: %s", err)
	}
}

// OrphanIntegrationLoop runs a loop executing integrateOrphans and then waiting a minute.
// It is split out into a separate function called directly by boulder-ca in order to make
// testing the orphan queue functionality somewhat more simple.
func (ca *certificateAuthorityImpl) OrphanIntegrationLoop() {
	for {
		err := ca.integrateOrphan()
		if err != nil {
			if err == goque.ErrEmpty {
				time.Sleep(time.Minute)
				continue
			}
			ca.log.AuditErrf("failed to integrate orphaned certs: %s", err)
			time.Sleep(time.Second)
		}
	}
}

// integrateOrpan removes an orphan from the queue and adds it to the database. The
// item isn't dequeued until it is actually added to the database to prevent items from
// being lost if the CA is restarted between the item being dequeued and being added to
// the database. It calculates the issuance time by subtracting the backdate period from
// the notBefore time.
func (ca *certificateAuthorityImpl) integrateOrphan() error {
	item, err := ca.orphanQueue.Peek()
	if err != nil {
		if err == goque.ErrEmpty {
			return goque.ErrEmpty
		}
		return fmt.Errorf("failed to peek into orphan queue: %s", err)
	}
	var orphan orphanedCert
	if err = item.ToObject(&orphan); err != nil {
		return fmt.Errorf("failed to marshal orphan: %s", err)
	}
	cert, err := x509.ParseCertificate(orphan.DER)
	if err != nil {
		return fmt.Errorf("failed to parse orphan: %s", err)
	}
	// When calculating the `NotBefore` at issuance time, we subtracted
	// ca.backdate. Now, to calculate the actual issuance time from the NotBefore,
	// we reverse the process and add ca.backdate.
	issued := cert.NotBefore.Add(ca.backdate)
	if orphan.Precert {
		_, err = ca.sa.AddPrecertificate(context.Background(), &sapb.AddCertificateRequest{
			Der:      orphan.DER,
			RegID:    orphan.RegID,
			Ocsp:     orphan.OCSPResp,
			Issued:   issued.UnixNano(),
			IssuerID: orphan.IssuerID,
		})
		if err != nil && !errors.Is(err, berrors.Duplicate) {
			return fmt.Errorf("failed to store orphaned precertificate: %s", err)
		}
	} else {
		_, err = ca.sa.AddCertificate(context.Background(), &sapb.AddCertificateRequest{
			Der:    orphan.DER,
			RegID:  orphan.RegID,
			Issued: issued.UnixNano(),
		})
		if err != nil && !errors.Is(err, berrors.Duplicate) {
			return fmt.Errorf("failed to store orphaned certificate: %s", err)
		}
	}
	if _, err = ca.orphanQueue.Dequeue(); err != nil {
		return fmt.Errorf("failed to dequeue integrated orphaned certificate: %s", err)
	}
	ca.log.AuditInfof("Incorporated orphaned certificate: serial=[%s] cert=[%s] regID=[%d]",
		core.SerialToString(cert.SerialNumber), hex.EncodeToString(orphan.DER), orphan.RegID)
	typ := "cert"
	if orphan.Precert {
		typ = "precert"
	}
	ca.adoptedOrphanCount.With(prometheus.Labels{"type": typ}).Inc()
	return nil
}

// GenerateOCSP is simply a passthrough to ocspImpl.GenerateOCSP so that other
// services which need to talk to the CA anyway can do so without configuring
// two separate gRPC service backends.
// TODO(#6448): Remove this passthrough to fully separate the services.
func (ca *certificateAuthorityImpl) GenerateOCSP(ctx context.Context, req *capb.GenerateOCSPRequest) (*capb.OCSPResponse, error) {
	return ca.ocsp.GenerateOCSP(ctx, req)
}

// GenerateCRL is simply a passthrough to crlImpl.GenerateCRL so that other
// services which need to talk to the CA anyway can do so without configuring
// two separate gRPC service backends.
// TODO(#6448): Remove this passthrough to fully separate the services.
func (ca *certificateAuthorityImpl) GenerateCRL(stream capb.CertificateAuthority_GenerateCRLServer) error {
	return ca.crl.GenerateCRL(stream)
}
