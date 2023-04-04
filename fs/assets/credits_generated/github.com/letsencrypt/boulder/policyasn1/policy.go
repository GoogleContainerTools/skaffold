// policyasn1 contains structures required to encode the RFC 5280
// PolicyInformation ASN.1 structures.
package policyasn1

import "encoding/asn1"

// CPSQualifierOID contains the id-qt-cps OID that is used to indicate the
// CPS policy qualifier type
var CPSQualifierOID = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 2, 1}

// PolicyQualifier represents the PolicyQualifierInfo ASN.1 structure
type PolicyQualifier struct {
	OID   asn1.ObjectIdentifier
	Value string `asn1:"optional,ia5"`
}

// PolicyInformation represents the PolicyInformation ASN.1 structure
type PolicyInformation struct {
	Policy     asn1.ObjectIdentifier
	Qualifiers []PolicyQualifier `asn1:"optional"`
}
