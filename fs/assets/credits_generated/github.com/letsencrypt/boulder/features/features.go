//go:generate stringer -type=FeatureFlag

package features

import (
	"fmt"
	"strings"
	"sync"
)

type FeatureFlag int

const (
	unused FeatureFlag = iota // unused is used for testing
	//   Deprecated features, these can be removed once stripped from production configs
	StoreRevokerInfo

	//   Currently in-use features
	// Check CAA and respect validationmethods parameter.
	CAAValidationMethods
	// Check CAA and respect accounturi parameter.
	CAAAccountURI
	// EnforceMultiVA causes the VA to block on remote VA PerformValidation
	// requests in order to make a valid/invalid decision with the results.
	EnforceMultiVA
	// MultiVAFullResults will cause the main VA to wait for all of the remote VA
	// results, not just the threshold required to make a decision.
	MultiVAFullResults
	// MandatoryPOSTAsGET forbids legacy unauthenticated GET requests for ACME
	// resources.
	MandatoryPOSTAsGET
	// ECDSAForAll enables all accounts, regardless of their presence in the CA's
	// ecdsaAllowedAccounts config value, to get issuance from ECDSA issuers.
	ECDSAForAll
	// ServeRenewalInfo exposes the renewalInfo endpoint in the directory and for
	// GET requests. WARNING: This feature is a draft and highly unstable.
	ServeRenewalInfo
	// AllowUnrecognizedFeatures is internal to the features package: if true,
	// skip error when unrecognized feature flag names are passed.
	AllowUnrecognizedFeatures

	// ROCSPStage6 disables writing full OCSP Responses to MariaDB during
	// (pre)certificate issuance and during revocation. Because Stage 4 involved
	// disabling ocsp-updater, this means that no ocsp response bytes will be
	// written to the database anymore.
	ROCSPStage6
	// ROCSPStage7 disables generating OCSP responses during issuance and
	// revocation. This affects codepaths in both the RA (revocation) and the CA
	// (precert "birth certificates").
	ROCSPStage7

	// ExpirationMailerUsesJoin enables using a JOIN query in expiration-mailer
	// rather than a SELECT from certificateStatus followed by thousands of
	// one-row SELECTs from certificates.
	ExpirationMailerUsesJoin

	// CertCheckerChecksValidations enables an extra query for each certificate
	// checked, to find the relevant authzs. Since this query might be
	// expensive, we gate it behind a feature flag.
	CertCheckerChecksValidations

	// CertCheckerRequiresValidations causes cert-checker to fail if the
	// query enabled by CertCheckerChecksValidations didn't find corresponding
	// authorizations.
	CertCheckerRequiresValidations
)

// List of features and their default value, protected by fMu
var features = map[FeatureFlag]bool{
	unused:                         false,
	CAAValidationMethods:           false,
	CAAAccountURI:                  false,
	EnforceMultiVA:                 false,
	MultiVAFullResults:             false,
	MandatoryPOSTAsGET:             false,
	StoreRevokerInfo:               false,
	ECDSAForAll:                    false,
	ServeRenewalInfo:               false,
	AllowUnrecognizedFeatures:      false,
	ROCSPStage6:                    false,
	ROCSPStage7:                    false,
	ExpirationMailerUsesJoin:       false,
	CertCheckerChecksValidations:   false,
	CertCheckerRequiresValidations: false,
}

var fMu = new(sync.RWMutex)

var initial = map[FeatureFlag]bool{}

var nameToFeature = make(map[string]FeatureFlag, len(features))

func init() {
	for f, v := range features {
		nameToFeature[f.String()] = f
		initial[f] = v
	}
}

// Set accepts a list of features and whether they should
// be enabled or disabled. In the presence of unrecognized
// flags, it will return an error or not depending on the
// value of AllowUnrecognizedFeatures.
func Set(featureSet map[string]bool) error {
	fMu.Lock()
	defer fMu.Unlock()
	var unknown []string
	for n, v := range featureSet {
		f, present := nameToFeature[n]
		if present {
			features[f] = v
		} else {
			unknown = append(unknown, n)
		}
	}
	if len(unknown) > 0 && !features[AllowUnrecognizedFeatures] {
		return fmt.Errorf("unrecognized feature flag names: %s",
			strings.Join(unknown, ", "))
	}
	return nil
}

// Enabled returns true if the feature is enabled or false
// if it isn't, it will panic if passed a feature that it
// doesn't know.
func Enabled(n FeatureFlag) bool {
	fMu.RLock()
	defer fMu.RUnlock()
	v, present := features[n]
	if !present {
		panic(fmt.Sprintf("feature '%s' doesn't exist", n.String()))
	}
	return v
}

// Reset resets the features to their initial state
func Reset() {
	fMu.Lock()
	defer fMu.Unlock()
	for k, v := range initial {
		features[k] = v
	}
}
