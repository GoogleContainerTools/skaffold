package ca

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	capb "github.com/letsencrypt/boulder/ca/proto"
	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	bcrl "github.com/letsencrypt/boulder/crl"
	"github.com/letsencrypt/boulder/crl/crl_x509"
	"github.com/letsencrypt/boulder/issuance"
	blog "github.com/letsencrypt/boulder/log"
)

type crlImpl struct {
	capb.UnimplementedCRLGeneratorServer
	issuers   map[issuance.IssuerNameID]*issuance.Issuer
	lifetime  time.Duration
	idpBase   string
	maxLogLen int
	log       blog.Logger
}

// NewCRLImpt returns a new object which fulfils the ca.proto CRLGenerator
// interface. It uses the list of issuers to determine what issuers it can
// issue CRLs from. lifetime sets the validity period (inclusive) of the
// resulting CRLs. idpBase is the base URL from which IssuingDistributionPoint
// URIs will constructed; it must use the http:// scheme.
func NewCRLImpl(issuers []*issuance.Issuer, lifetime time.Duration, idpBase string, maxLogLen int, logger blog.Logger) (*crlImpl, error) {
	issuersByNameID := make(map[issuance.IssuerNameID]*issuance.Issuer, len(issuers))
	for _, issuer := range issuers {
		issuersByNameID[issuer.Cert.NameID()] = issuer
	}

	if lifetime == 0 {
		logger.Warningf("got zero for crl lifetime; setting to default 9 days")
		lifetime = 9 * 24 * time.Hour
	} else if lifetime >= 10*24*time.Hour {
		return nil, fmt.Errorf("crl lifetime cannot be more than 10 days, got %q", lifetime)
	} else if lifetime <= 0*time.Hour {
		return nil, fmt.Errorf("crl lifetime must be positive, got %q", lifetime)
	}

	if !strings.HasPrefix(idpBase, "http://") {
		return nil, fmt.Errorf("issuingDistributionPoint base URI must use http:// scheme, got %q", idpBase)
	}
	if strings.HasSuffix(idpBase, "/") {
		return nil, fmt.Errorf("issuingDistributionPoint base URI must not end with a slash, got %q", idpBase)
	}

	return &crlImpl{
		issuers:   issuersByNameID,
		lifetime:  lifetime,
		idpBase:   idpBase,
		maxLogLen: maxLogLen,
		log:       logger,
	}, nil
}

func (ci *crlImpl) GenerateCRL(stream capb.CRLGenerator_GenerateCRLServer) error {
	var issuer *issuance.Issuer
	var template *crl_x509.RevocationList
	var shard int64
	rcs := make([]crl_x509.RevokedCertificate, 0)

	for {
		in, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch payload := in.Payload.(type) {
		case *capb.GenerateCRLRequest_Metadata:
			if template != nil {
				return errors.New("got more than one metadata message")
			}

			template, err = ci.metadataToTemplate(payload.Metadata)
			if err != nil {
				return err
			}

			var ok bool
			issuer, ok = ci.issuers[issuance.IssuerNameID(payload.Metadata.IssuerNameID)]
			if !ok {
				return fmt.Errorf("got unrecognized IssuerNameID: %d", payload.Metadata.IssuerNameID)
			}

			shard = payload.Metadata.ShardIdx

		case *capb.GenerateCRLRequest_Entry:
			rc, err := ci.entryToRevokedCertificate(payload.Entry)
			if err != nil {
				return err
			}

			rcs = append(rcs, *rc)

		default:
			return errors.New("got empty or malformed message in input stream")
		}
	}

	if template == nil {
		return errors.New("no crl metadata received")
	}

	// Add the Issuing Distribution Point extension.
	idp, err := makeIDPExt(ci.idpBase, issuer.Cert.NameID(), shard)
	if err != nil {
		return fmt.Errorf("creating IDP extension: %w", err)
	}
	template.ExtraExtensions = append(template.ExtraExtensions, *idp)

	// Compute a unique ID for this issuer-number-shard combo, to tie together all
	// the audit log lines related to its issuance.
	logID := blog.LogLineChecksum(fmt.Sprintf("%d", issuer.Cert.NameID()) + template.Number.String() + fmt.Sprintf("%d", shard))
	ci.log.AuditInfof(
		"Signing CRL: logID=[%s] issuer=[%s] number=[%s] shard=[%d] thisUpdate=[%s] nextUpdate=[%s] numEntries=[%d]",
		logID, issuer.Cert.Subject.CommonName, template.Number.String(), shard, template.ThisUpdate, template.NextUpdate, len(rcs),
	)

	if len(rcs) > 0 {
		builder := strings.Builder{}
		for i := 0; i < len(rcs); i += 1 {
			if builder.Len() == 0 {
				fmt.Fprintf(&builder, "Signing CRL: logID=[%s] entries=[", logID)
			}

			reason := 0
			if rcs[i].ReasonCode != nil {
				reason = *rcs[i].ReasonCode
			}
			fmt.Fprintf(&builder, "%x:%d,", rcs[i].SerialNumber.Bytes(), reason)

			if builder.Len() >= ci.maxLogLen {
				fmt.Fprint(&builder, "]")
				ci.log.AuditInfo(builder.String())
				builder = strings.Builder{}
			}
		}
		fmt.Fprint(&builder, "]")
		ci.log.AuditInfo(builder.String())
	}

	template.RevokedCertificates = rcs

	err = issuer.Linter.CheckCRL(template)
	if err != nil {
		return err
	}

	crlBytes, err := crl_x509.CreateRevocationList(
		rand.Reader,
		template,
		issuer.Cert.Certificate,
		issuer.Signer,
	)
	if err != nil {
		return fmt.Errorf("signing crl: %w", err)
	}

	hash := sha256.Sum256(crlBytes)
	ci.log.AuditInfof(
		"Signing CRL success: logID=[%s] size=[%d] hash=[%x]",
		logID, len(crlBytes), hash,
	)

	for i := 0; i < len(crlBytes); i += 1000 {
		j := i + 1000
		if j > len(crlBytes) {
			j = len(crlBytes)
		}
		err = stream.Send(&capb.GenerateCRLResponse{
			Chunk: crlBytes[i:j],
		})
		if err != nil {
			return err
		}
		if i%1000 == 0 {
			ci.log.Debugf("Wrote %d bytes to output stream", i*1000)
		}
	}

	return nil
}

func (ci *crlImpl) metadataToTemplate(meta *capb.CRLMetadata) (*crl_x509.RevocationList, error) {
	if meta.IssuerNameID == 0 || meta.ThisUpdate == 0 {
		return nil, errors.New("got incomplete metadata message")
	}
	thisUpdate := time.Unix(0, meta.ThisUpdate)
	number := bcrl.Number(thisUpdate)

	return &crl_x509.RevocationList{
		Number:     number,
		ThisUpdate: thisUpdate,
		NextUpdate: thisUpdate.Add(-time.Second).Add(ci.lifetime),
	}, nil
}

func (ci *crlImpl) entryToRevokedCertificate(entry *corepb.CRLEntry) (*crl_x509.RevokedCertificate, error) {
	serial, err := core.StringToSerial(entry.Serial)
	if err != nil {
		return nil, err
	}

	if entry.RevokedAt == 0 {
		return nil, errors.New("got empty or zero revocation timestamp")
	}
	revokedAt := time.Unix(0, entry.RevokedAt)

	var reason *int
	if entry.Reason != 0 {
		reason = new(int)
		*reason = int(entry.Reason)
	}

	return &crl_x509.RevokedCertificate{
		SerialNumber:   serial,
		RevocationTime: revokedAt,
		ReasonCode:     reason,
	}, nil
}

// distributionPointName represents the ASN.1 DistributionPointName CHOICE as
// defined in RFC 5280 Section 4.2.1.13. We only use one of the fields, so the
// others are omitted.
type distributionPointName struct {
	// Technically, FullName is of type GeneralNames, which is of type SEQUENCE OF
	// GeneralName. But GeneralName itself is of type CHOICE, and the ans1.Marhsal
	// function doesn't support marshalling structs to CHOICEs, so we have to use
	// asn1.RawValue and encode the GeneralName ourselves.
	FullName []asn1.RawValue `asn1:"optional,tag:0"`
}

// issuingDistributionPoint represents the ASN.1 IssuingDistributionPoint
// SEQUENCE as defined in RFC 5280 Section 5.2.5. We only use two of the fields,
// so the others are omitted.
type issuingDistributionPoint struct {
	DistributionPoint     distributionPointName `asn1:"optional,tag:0"`
	OnlyContainsUserCerts bool                  `asn1:"optional,tag:1"`
}

// makeIDPExt returns a critical IssuingDistributionPoint extension containing a
// URI built from the base url, the issuer's NameID, and the shard number. It
// also sets the OnlyContainsUserCerts boolean to true.
func makeIDPExt(base string, issuer issuance.IssuerNameID, shardIdx int64) (*pkix.Extension, error) {
	val := issuingDistributionPoint{
		DistributionPoint: distributionPointName{
			[]asn1.RawValue{ // GeneralNames
				{ // GeneralName
					Class: 2, // context-specific
					Tag:   6, // uniformResourceIdentifier, IA5String
					Bytes: []byte(fmt.Sprintf("%s/%d/%d.crl", base, issuer, shardIdx)),
				},
			},
		},
		OnlyContainsUserCerts: true,
	}

	valBytes, err := asn1.Marshal(val)
	if err != nil {
		return nil, err
	}

	return &pkix.Extension{
		Id:       asn1.ObjectIdentifier{2, 5, 29, 28}, // id-ce-issuingDistributionPoint
		Value:    valBytes,
		Critical: true,
	}, nil
}

// disabledCRLImpl implements the capb.CRLGeneratorServer interface, but returns
// an error for all gRPC methods. This is only used to replace a real impl when
// the CRLGenerator service is disabled.
// TODO(#6448): Remove this.
type disabledCRLImpl struct {
	capb.UnimplementedCRLGeneratorServer
}

// NewDiabledCRLImpl returns an object which implements the
// capb.CRLGeneratorServer interface but always returns errors.
func NewDisabledCRLImpl() *disabledCRLImpl {
	return &disabledCRLImpl{}
}

// GenerateCRL always returns an error because the service is disabled.
func (ci *disabledCRLImpl) GenerateCRL(stream capb.CRLGenerator_GenerateCRLServer) error {
	return errors.New("the CRLGenerator gRPC service is disabled")
}
