package subscriber

import (
	"time"

	"github.com/zmap/zcrypto/x509"
	"github.com/zmap/zlint/v3/lint"
	"github.com/zmap/zlint/v3/util"

	"github.com/letsencrypt/boulder/linter/lints"
)

type certValidityTooLong struct{}

func init() {
	lint.RegisterLint(&lint.Lint{
		Name:          "e_validity_period_greater_than_25_years",
		Description:   "Let's Encrypt Root CA Certificates have Validity Periods of up to 25 years",
		Citation:      "CPS: 7.1",
		Source:        lints.LetsEncryptCPSRoot,
		EffectiveDate: lints.CPSV33Date,
		Lint:          NewCertValidityTooLong,
	})
}

func NewCertValidityTooLong() lint.LintInterface {
	return &certValidityTooLong{}
}

func (l *certValidityTooLong) CheckApplies(c *x509.Certificate) bool {
	return util.IsRootCA(c)
}

func (l *certValidityTooLong) Execute(c *x509.Certificate) *lint.LintResult {
	// CPS 7.1: "Root CA Certificate Validity Period: Up to 25 years."
	maxValidity := 25 * 365 * lints.BRDay

	// RFC 5280 4.1.2.5: "The validity period for a certificate is the period
	// of time from notBefore through notAfter, inclusive."
	certValidity := c.NotAfter.Add(time.Second).Sub(c.NotBefore)

	if certValidity > maxValidity {
		return &lint.LintResult{Status: lint.Error}
	}

	return &lint.LintResult{Status: lint.Pass}
}
