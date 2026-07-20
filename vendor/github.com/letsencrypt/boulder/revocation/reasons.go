package revocation

import (
	"fmt"
)

// Reason is used to specify a certificate revocation reason
type Reason int64

// The enumerated reasons for revoking a certificate. See RFC 5280:
// https://datatracker.ietf.org/doc/html/rfc5280#section-5.3.1.
const (
	Unspecified          Reason = 0
	KeyCompromise        Reason = 1
	CACompromise         Reason = 2
	AffiliationChanged   Reason = 3
	Superseded           Reason = 4
	CessationOfOperation Reason = 5
	CertificateHold      Reason = 6
	// 7 is unused
	RemoveFromCRL      Reason = 8
	PrivilegeWithdrawn Reason = 9
	AACompromise       Reason = 10
)

// reasonToString provides a map from reason code to string. It is unexported
// to make it immutable.
var reasonToString = map[Reason]string{
	Unspecified:          "unspecified",
	KeyCompromise:        "keyCompromise",
	CACompromise:         "cACompromise",
	AffiliationChanged:   "affiliationChanged",
	Superseded:           "superseded",
	CessationOfOperation: "cessationOfOperation",
	CertificateHold:      "certificateHold",
	RemoveFromCRL:        "removeFromCRL",
	PrivilegeWithdrawn:   "privilegeWithdrawn",
	AACompromise:         "aAcompromise",
}

// String converts a revocation reason code (such as 0) into its corresponding
// reason string (e.g. "unspecified").
//
// The receiver *must* be one of the valid reason code constants defined in this
// package: this method will panic if called on an invalid Reason. It is
// expected that this method is only called on const Reasons, or after a call to
// UserAllowedReason or AdminAllowedReason.
func (r Reason) String() string {
	res, ok := reasonToString[r]
	if !ok {
		panic(fmt.Errorf("unrecognized revocation code %d", r))
	}
	return res
}

// StringToReason converts a revocation reason string (such as "keyCompromise")
// into the corresponding integer reason code (e.g. 1).
func StringToReason(s string) (Reason, error) {
	for code, str := range reasonToString {
		if s == str {
			return code, nil
		}
	}
	return 0, fmt.Errorf("unrecognized revocation reason %q", s)
}

// UserAllowedReason returns true if the given Reason is in the subset of
// Reasons which users are allowed to request.
func UserAllowedReason(r Reason) bool {
	switch r {
	case Unspecified,
		KeyCompromise,
		Superseded,
		CessationOfOperation:
		return true
	}
	return false
}

// AdminAllowedReason returns true if the given Reason is in the subset of
// Reasons which admins (i.e. people acting in CA Trusted Roles) are allowed
// to request. Reasons which do *not* appear here are those which are defined
// by RFC 5280 but are disallowed by the Baseline Requirements.
func AdminAllowedReason(r Reason) bool {
	switch r {
	case Unspecified,
		KeyCompromise,
		Superseded,
		CessationOfOperation,
		PrivilegeWithdrawn:
		return true
	}
	return false
}
