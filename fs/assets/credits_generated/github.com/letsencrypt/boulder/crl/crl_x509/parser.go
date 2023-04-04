// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package crl_x509

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/crypto/cryptobyte"
	cryptobyte_asn1 "golang.org/x/crypto/cryptobyte/asn1"
)

// isPrintable reports whether the given b is in the ASN.1 PrintableString set.
// This is a simplified version of encoding/asn1.isPrintable.
func isPrintable(b byte) bool {
	return 'a' <= b && b <= 'z' ||
		'A' <= b && b <= 'Z' ||
		'0' <= b && b <= '9' ||
		'\'' <= b && b <= ')' ||
		'+' <= b && b <= '/' ||
		b == ' ' ||
		b == ':' ||
		b == '=' ||
		b == '?' ||
		// This is technically not allowed in a PrintableString.
		// However, x509 certificates with wildcard strings don't
		// always use the correct string type so we permit it.
		b == '*' ||
		// This is not technically allowed either. However, not
		// only is it relatively common, but there are also a
		// handful of CA certificates that contain it. At least
		// one of which will not expire until 2027.
		b == '&'
}

// parseASN1String parses the ASN.1 string types T61String, PrintableString,
// UTF8String, BMPString, IA5String, and NumericString. This is mostly copied
// from the respective encoding/asn1.parse... methods, rather than just
// increasing the API surface of that package.
func parseASN1String(tag cryptobyte_asn1.Tag, value []byte) (string, error) {
	switch tag {
	case cryptobyte_asn1.T61String:
		return string(value), nil
	case cryptobyte_asn1.PrintableString:
		for _, b := range value {
			if !isPrintable(b) {
				return "", errors.New("invalid PrintableString")
			}
		}
		return string(value), nil
	case cryptobyte_asn1.UTF8String:
		if !utf8.Valid(value) {
			return "", errors.New("invalid UTF-8 string")
		}
		return string(value), nil
	case cryptobyte_asn1.Tag(asn1.TagBMPString):
		if len(value)%2 != 0 {
			return "", errors.New("invalid BMPString")
		}

		// Strip terminator if present.
		if l := len(value); l >= 2 && value[l-1] == 0 && value[l-2] == 0 {
			value = value[:l-2]
		}

		s := make([]uint16, 0, len(value)/2)
		for len(value) > 0 {
			s = append(s, uint16(value[0])<<8+uint16(value[1]))
			value = value[2:]
		}

		return string(utf16.Decode(s)), nil
	case cryptobyte_asn1.IA5String:
		s := string(value)
		if isIA5String(s) != nil {
			return "", errors.New("invalid IA5String")
		}
		return s, nil
	case cryptobyte_asn1.Tag(asn1.TagNumericString):
		for _, b := range value {
			if !('0' <= b && b <= '9' || b == ' ') {
				return "", errors.New("invalid NumericString")
			}
		}
		return string(value), nil
	}
	return "", fmt.Errorf("unsupported string type: %v", tag)
}

// parseName parses a DER encoded Name as defined in RFC 5280. We may
// want to export this function in the future for use in crypto/tls.
func parseName(raw cryptobyte.String) (*pkix.RDNSequence, error) {
	if !raw.ReadASN1(&raw, cryptobyte_asn1.SEQUENCE) {
		return nil, errors.New("x509: invalid RDNSequence")
	}

	var rdnSeq pkix.RDNSequence
	for !raw.Empty() {
		var rdnSet pkix.RelativeDistinguishedNameSET
		var set cryptobyte.String
		if !raw.ReadASN1(&set, cryptobyte_asn1.SET) {
			return nil, errors.New("x509: invalid RDNSequence")
		}
		for !set.Empty() {
			var atav cryptobyte.String
			if !set.ReadASN1(&atav, cryptobyte_asn1.SEQUENCE) {
				return nil, errors.New("x509: invalid RDNSequence: invalid attribute")
			}
			var attr pkix.AttributeTypeAndValue
			if !atav.ReadASN1ObjectIdentifier(&attr.Type) {
				return nil, errors.New("x509: invalid RDNSequence: invalid attribute type")
			}
			var rawValue cryptobyte.String
			var valueTag cryptobyte_asn1.Tag
			if !atav.ReadAnyASN1(&rawValue, &valueTag) {
				return nil, errors.New("x509: invalid RDNSequence: invalid attribute value")
			}
			var err error
			attr.Value, err = parseASN1String(valueTag, rawValue)
			if err != nil {
				return nil, fmt.Errorf("x509: invalid RDNSequence: invalid attribute value: %s", err)
			}
			rdnSet = append(rdnSet, attr)
		}

		rdnSeq = append(rdnSeq, rdnSet)
	}

	return &rdnSeq, nil
}

func parseAI(der cryptobyte.String) (pkix.AlgorithmIdentifier, error) {
	ai := pkix.AlgorithmIdentifier{}
	if !der.ReadASN1ObjectIdentifier(&ai.Algorithm) {
		return ai, errors.New("x509: malformed OID")
	}
	if der.Empty() {
		return ai, nil
	}
	var params cryptobyte.String
	var tag cryptobyte_asn1.Tag
	if !der.ReadAnyASN1Element(&params, &tag) {
		return ai, errors.New("x509: malformed parameters")
	}
	ai.Parameters.Tag = int(tag)
	ai.Parameters.FullBytes = params
	return ai, nil
}

func parseTime(der *cryptobyte.String) (time.Time, error) {
	var t time.Time
	switch {
	case der.PeekASN1Tag(cryptobyte_asn1.UTCTime):
		if !der.ReadASN1UTCTime(&t) {
			return t, errors.New("x509: malformed UTCTime")
		}
	case der.PeekASN1Tag(cryptobyte_asn1.GeneralizedTime):
		if !der.ReadASN1GeneralizedTime(&t) {
			return t, errors.New("x509: malformed GeneralizedTime")
		}
	default:
		return t, errors.New("x509: unsupported time format")
	}
	return t, nil
}

func parseExtension(der cryptobyte.String) (pkix.Extension, error) {
	var ext pkix.Extension
	if !der.ReadASN1ObjectIdentifier(&ext.Id) {
		return ext, errors.New("x509: malformed extension OID field")
	}
	if der.PeekASN1Tag(cryptobyte_asn1.BOOLEAN) {
		if !der.ReadASN1Boolean(&ext.Critical) {
			return ext, errors.New("x509: malformed extension critical field")
		}
	}
	var val cryptobyte.String
	if !der.ReadASN1(&val, cryptobyte_asn1.OCTET_STRING) {
		return ext, errors.New("x509: malformed extension value field")
	}
	ext.Value = val
	return ext, nil
}

// The X.509 standards confusingly 1-indexed the version names, but 0-indexed
// the actual encoded version, so the version for X.509v2 is 1.
const x509v2Version = 1
