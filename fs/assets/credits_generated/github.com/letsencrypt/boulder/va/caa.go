package va

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	berrors "github.com/letsencrypt/boulder/errors"
	"github.com/letsencrypt/boulder/features"
	"github.com/letsencrypt/boulder/identifier"
	"github.com/letsencrypt/boulder/probs"
	vapb "github.com/letsencrypt/boulder/va/proto"
	"github.com/miekg/dns"
)

type caaParams struct {
	accountURIID     int64
	validationMethod core.AcmeChallenge
}

func (va *ValidationAuthorityImpl) IsCAAValid(ctx context.Context, req *vapb.IsCAAValidRequest) (*vapb.IsCAAValidResponse, error) {
	if req.Domain == "" || req.ValidationMethod == "" || req.AccountURIID == 0 {
		return nil, berrors.InternalServerError("incomplete IsCAAValid request")
	}

	validationMethod := core.AcmeChallenge(req.ValidationMethod)
	if !validationMethod.IsValid() {
		return nil, berrors.InternalServerError("unrecognized validation method %q", req.ValidationMethod)
	}

	acmeID := identifier.ACMEIdentifier{
		Type:  identifier.DNS,
		Value: req.Domain,
	}
	params := &caaParams{
		accountURIID:     req.AccountURIID,
		validationMethod: validationMethod,
	}
	if prob := va.checkCAA(ctx, acmeID, params); prob != nil {
		detail := fmt.Sprintf("While processing CAA for %s: %s", req.Domain, prob.Detail)
		return &vapb.IsCAAValidResponse{
			Problem: &corepb.ProblemDetails{
				ProblemType: string(prob.Type),
				Detail:      replaceInvalidUTF8([]byte(detail)),
			},
		}, nil
	}
	return &vapb.IsCAAValidResponse{}, nil
}

// checkCAA performs a CAA lookup & validation for the provided identifier. If
// the CAA lookup & validation fail a problem is returned.
func (va *ValidationAuthorityImpl) checkCAA(
	ctx context.Context,
	identifier identifier.ACMEIdentifier,
	params *caaParams) *probs.ProblemDetails {
	if params == nil || params.validationMethod == "" || params.accountURIID == 0 {
		return probs.ServerInternal("expected validationMethod or accountURIID not provided to checkCAA")
	}

	present, valid, response, err := va.checkCAARecords(ctx, identifier, params)
	if err != nil {
		return probs.DNS(err.Error())
	}

	va.log.AuditInfof("Checked CAA records for %s, [Present: %t, Account ID: %d, Challenge: %s, Valid for issuance: %t] Response=%q",
		identifier.Value, present, params.accountURIID, params.validationMethod, valid, response)
	if !valid {
		return probs.CAA(fmt.Sprintf("CAA record for %s prevents issuance", identifier.Value))
	}
	return nil
}

// CAASet consists of filtered CAA records
type CAASet struct {
	Issue     []*dns.CAA
	Issuewild []*dns.CAA
	Iodef     []*dns.CAA
	Unknown   []*dns.CAA
}

// returns true if any CAA records have unknown tag properties and are flagged critical.
func (caaSet CAASet) criticalUnknown() bool {
	if len(caaSet.Unknown) > 0 {
		for _, caaRecord := range caaSet.Unknown {
			// The critical flag is the bit with significance 128. However, many CAA
			// record users have misinterpreted the RFC and concluded that the bit
			// with significance 1 is the critical bit. This is sufficiently
			// widespread that that bit must reasonably be considered an alias for
			// the critical bit. The remaining bits are 0/ignore as proscribed by the
			// RFC.
			if (caaRecord.Flag & (128 | 1)) != 0 {
				return true
			}
		}
	}

	return false
}

// Filter CAA records by property
func newCAASet(CAAs []*dns.CAA) *CAASet {
	var filtered CAASet

	for _, caaRecord := range CAAs {
		switch strings.ToLower(caaRecord.Tag) {
		case "issue":
			filtered.Issue = append(filtered.Issue, caaRecord)
		case "issuewild":
			filtered.Issuewild = append(filtered.Issuewild, caaRecord)
		case "iodef":
			filtered.Iodef = append(filtered.Iodef, caaRecord)
		default:
			filtered.Unknown = append(filtered.Unknown, caaRecord)
		}
	}

	return &filtered
}

type caaResult struct {
	records  []*dns.CAA
	response string
	err      error
}

func parseResults(results []caaResult) (*CAASet, string, error) {
	// Return first result
	for _, res := range results {
		if res.err != nil {
			return nil, "", res.err
		}
		if len(res.records) > 0 {
			return newCAASet(res.records), res.response, nil
		}
	}
	return nil, "", nil
}

func (va *ValidationAuthorityImpl) parallelCAALookup(ctx context.Context, name string) []caaResult {
	labels := strings.Split(name, ".")
	results := make([]caaResult, len(labels))
	var wg sync.WaitGroup

	for i := 0; i < len(labels); i++ {
		// Start the concurrent DNS lookup.
		wg.Add(1)
		go func(name string, r *caaResult) {
			r.records, r.response, r.err = va.dnsClient.LookupCAA(ctx, name)
			wg.Done()
		}(strings.Join(labels[i:], "."), &results[i])
	}

	wg.Wait()
	return results
}

func (va *ValidationAuthorityImpl) getCAASet(ctx context.Context, hostname string) (*CAASet, string, error) {
	hostname = strings.TrimRight(hostname, ".")

	// See RFC 6844 "Certification Authority Processing" for pseudocode, as
	// amended by https://www.rfc-editor.org/errata/eid5065.
	// Essentially: check CAA records for the FDQN to be issued, and all
	// parent domains.
	//
	// The lookups are performed in parallel in order to avoid timing out
	// the RPC call.
	//
	// We depend on our resolver to snap CNAME and DNAME records.
	results := va.parallelCAALookup(ctx, hostname)
	return parseResults(results)
}

// checkCAARecords fetches the CAA records for the given identifier and then
// validates them. If the identifier argument's value has a wildcard prefix then
// the prefix is stripped and validation will be performed against the base
// domain, honouring any issueWild CAA records encountered as appropriate.
// checkCAARecords returns four values: the first is a bool indicating whether
// CAA records were present after filtering for known/supported CAA tags. The
// second is a bool indicating whether issuance for the identifier is valid. The
// unmodified *dns.CAA records that were processed/filtered are returned as the
// third argument. Any  errors encountered are returned as the fourth return
// value (or nil).
func (va *ValidationAuthorityImpl) checkCAARecords(
	ctx context.Context,
	identifier identifier.ACMEIdentifier,
	params *caaParams) (bool, bool, string, error) {
	hostname := strings.ToLower(identifier.Value)
	// If this is a wildcard name, remove the prefix
	var wildcard bool
	if strings.HasPrefix(hostname, `*.`) {
		hostname = strings.TrimPrefix(identifier.Value, `*.`)
		wildcard = true
	}
	caaSet, response, err := va.getCAASet(ctx, hostname)
	if err != nil {
		return false, false, "", err
	}
	present, valid := va.validateCAASet(caaSet, wildcard, params)
	return present, valid, response, nil
}

// validateCAASet checks a provided *CAASet. When the wildcard argument is true
// this means the CAASet's issueWild records must be validated as well. This
// function returns two booleans: the first indicates whether the CAASet was
// empty, the second indicates whether the CAASet is valid for issuance to
// proceed.
func (va *ValidationAuthorityImpl) validateCAASet(caaSet *CAASet, wildcard bool, params *caaParams) (present, valid bool) {
	if caaSet == nil {
		// No CAA records found, can issue
		va.metrics.caaCounter.WithLabelValues("no records").Inc()
		return false, true
	}

	if caaSet.criticalUnknown() {
		// Contains unknown critical directives
		va.metrics.caaCounter.WithLabelValues("record with unknown critical directive").Inc()
		return true, false
	}

	if len(caaSet.Issue) == 0 && !wildcard {
		// Although CAA records exist, none of them pertain to issuance in this case.
		// (e.g. there is only an issuewild directive, but we are checking for a
		// non-wildcard identifier, or there is only an iodef or non-critical unknown
		// directive.)
		va.metrics.caaCounter.WithLabelValues("no relevant records").Inc()
		return true, true
	}

	// Per RFC 8659 Section 5.3:
	//   - "Each issuewild Property MUST be ignored when processing a request for
	//     an FQDN that is not a Wildcard Domain Name."; and
	//   - "If at least one issuewild Property is specified in the Relevant RRset
	//     for a Wildcard Domain Name, each issue Property MUST be ignored when
	//     processing a request for that Wildcard Domain Name."
	// So we default to checking the `caaSet.Issue` records and only check
	// `caaSet.Issuewild` when `wildcard` is true and there are 1 or more
	// `Issuewild` records.
	records := caaSet.Issue
	if wildcard && len(caaSet.Issuewild) > 0 {
		records = caaSet.Issuewild
	}

	// There are CAA records pertaining to issuance in our case. Note that this
	// includes the case of the unsatisfiable CAA record value ";", used to
	// prevent issuance by any CA under any circumstance.
	//
	// Our CAA identity must be found in the chosen checkSet.
	for _, caa := range records {
		parsedDomain, parsedParams, err := parseCAARecord(caa)
		if err != nil {
			continue
		}

		if !caaDomainMatches(parsedDomain, va.issuerDomain) {
			continue
		}

		if features.Enabled(features.CAAAccountURI) {
			if !caaAccountURIMatches(parsedParams, va.accountURIPrefixes, params.accountURIID) {
				continue
			}
		}
		if features.Enabled(features.CAAValidationMethods) {
			if !caaValidationMethodMatches(parsedParams, params.validationMethod) {
				continue
			}
		}

		va.metrics.caaCounter.WithLabelValues("authorized").Inc()
		return true, true
	}

	// The list of authorized issuers is non-empty, but we are not in it. Fail.
	va.metrics.caaCounter.WithLabelValues("unauthorized").Inc()
	return true, false
}

// parseCAARecord extracts the domain and parameters (if any) from a
// issue/issuewild CAA record. This follows RFC 8659 Section 4.2 and Section 4.3
// (https://www.rfc-editor.org/rfc/rfc8659.html#section-4). It returns the
// domain name (which may be the empty string if the record forbids issuance)
// and a tag-value map of CAA parameters, or a descriptive error if the record
// is malformed.
func parseCAARecord(caa *dns.CAA) (string, map[string]string, error) {
	isWSP := func(r rune) bool {
		return r == '\t' || r == ' '
	}

	// Semi-colons (ASCII 0x3B) are prohibited from being specified in the
	// parameter tag or value, hence we can simply split on semi-colons.
	parts := strings.Split(caa.Value, ";")
	domain := strings.TrimFunc(parts[0], isWSP)
	paramList := parts[1:]
	parameters := make(map[string]string)

	// Handle the case where a semi-colon is specified following the domain
	// but no parameters are given.
	if len(paramList) == 1 && strings.TrimFunc(paramList[0], isWSP) == "" {
		return domain, parameters, nil
	}

	for _, parameter := range paramList {
		// A parameter tag cannot include equal signs (ASCII 0x3D),
		// however they are permitted in the value itself.
		tv := strings.SplitN(parameter, "=", 2)
		if len(tv) != 2 {
			return "", nil, fmt.Errorf("parameter not formatted as tag=value: %q", parameter)
		}

		tag := strings.TrimFunc(tv[0], isWSP)
		//lint:ignore S1029,SA6003 we iterate over runes because the RFC specifies ascii codepoints.
		for _, r := range []rune(tag) {
			// ASCII alpha/digits.
			// tag = (ALPHA / DIGIT) *( *("-") (ALPHA / DIGIT))
			if r < 0x30 || (r > 0x39 && r < 0x41) || (r > 0x5a && r < 0x61) || r > 0x7a {
				return "", nil, fmt.Errorf("tag contains disallowed character: %q", tag)
			}
		}

		value := strings.TrimFunc(tv[1], isWSP)
		//lint:ignore S1029,SA6003 we iterate over runes because the RFC specifies ascii codepoints.
		for _, r := range []rune(value) {
			// ASCII without whitespace/semi-colons.
			// value = *(%x21-3A / %x3C-7E)
			if r < 0x21 || (r > 0x3a && r < 0x3c) || r > 0x7e {
				return "", nil, fmt.Errorf("value contains disallowed character: %q", value)
			}
		}

		parameters[tag] = value
	}

	return domain, parameters, nil
}

// caaDomainMatches checks that the issuer domain name listed in the parsed
// CAA record matches the domain name we expect.
func caaDomainMatches(caaDomain string, issuerDomain string) bool {
	return caaDomain == issuerDomain
}

// caaAccountURIMatches checks that the accounturi CAA parameter, if present,
// matches one of the specific account URIs we expect. We support multiple
// account URI prefixes to handle accounts which were registered under ACMEv1.
// See RFC 8657 Section 3: https://www.rfc-editor.org/rfc/rfc8657.html#section-3
func caaAccountURIMatches(caaParams map[string]string, accountURIPrefixes []string, accountID int64) bool {
	accountURI, ok := caaParams["accounturi"]
	if !ok {
		return true
	}

	// If the accounturi is not formatted according to RFC 3986, reject it.
	_, err := url.Parse(accountURI)
	if err != nil {
		return false
	}

	for _, prefix := range accountURIPrefixes {
		if accountURI == fmt.Sprintf("%s%d", prefix, accountID) {
			return true
		}
	}
	return false
}

var validationMethodRegexp = regexp.MustCompile(`^[[:alnum:]-]+$`)

// caaValidationMethodMatches checks that the validationmethods CAA parameter,
// if present, contains the exact name of the ACME validation method used to
// validate this domain.
// See RFC 8657 Section 4: https://www.rfc-editor.org/rfc/rfc8657.html#section-4
func caaValidationMethodMatches(caaParams map[string]string, method core.AcmeChallenge) bool {
	commaSeparatedMethods, ok := caaParams["validationmethods"]
	if !ok {
		return true
	}

	for _, m := range strings.Split(commaSeparatedMethods, ",") {
		// If any listed method does not match the ABNF 1*(ALPHA / DIGIT / "-"),
		// immediately reject the whole record.
		if !validationMethodRegexp.MatchString(m) {
			return false
		}

		caaMethod := core.AcmeChallenge(m)
		if !caaMethod.IsValid() {
			continue
		}

		if caaMethod == method {
			return true
		}
	}
	return false
}
