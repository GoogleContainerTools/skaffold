package policy

import (
	"os"
	"testing"

	"github.com/letsencrypt/boulder/core"
	berrors "github.com/letsencrypt/boulder/errors"
	"github.com/letsencrypt/boulder/features"
	"github.com/letsencrypt/boulder/identifier"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/test"
	"gopkg.in/yaml.v3"
)

var enabledChallenges = map[core.AcmeChallenge]bool{
	core.ChallengeTypeHTTP01: true,
	core.ChallengeTypeDNS01:  true,
}

func paImpl(t *testing.T) *AuthorityImpl {
	pa, err := New(enabledChallenges, blog.NewMock())
	if err != nil {
		t.Fatalf("Couldn't create policy implementation: %s", err)
	}
	return pa
}

func TestWillingToIssue(t *testing.T) {
	testCases := []struct {
		domain string
		err    error
	}{
		{``, errEmptyName},                    // Empty name
		{`zomb!.com`, errInvalidDNSCharacter}, // ASCII character out of range
		{`emailaddress@myseriously.present.com`, errInvalidDNSCharacter},
		{`user:pass@myseriously.present.com`, errInvalidDNSCharacter},
		{`zömbo.com`, errInvalidDNSCharacter},                              // non-ASCII character
		{`127.0.0.1`, errIPAddress},                                        // IPv4 address
		{`fe80::1:1`, errInvalidDNSCharacter},                              // IPv6 addresses
		{`[2001:db8:85a3:8d3:1319:8a2e:370:7348]`, errInvalidDNSCharacter}, // unexpected IPv6 variants
		{`[2001:db8:85a3:8d3:1319:8a2e:370:7348]:443`, errInvalidDNSCharacter},
		{`2001:db8::/32`, errInvalidDNSCharacter},
		{`a.b.c.d.e.f.g.h.i.j.k`, errTooManyLabels}, // Too many labels (>10)

		{`www.0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef012345.com`, errNameTooLong}, // Too long (254 characters)

		{`www.ef0123456789abcdef013456789abcdef012345.789abcdef012345679abcdef0123456789abcdef01234.6789abcdef0123456789abcdef0.23456789abcdef0123456789a.cdef0123456789abcdef0123456789ab.def0123456789abcdef0123456789.bcdef0123456789abcdef012345.com`, nil}, // OK, not too long (240 characters)

		{`www.abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz.com`, errLabelTooLong}, // Label too long (>63 characters)

		{`www.-ombo.com`, errInvalidDNSCharacter}, // Label starts with '-'
		{`www.zomb-.com`, errInvalidDNSCharacter}, // Label ends with '-'
		{`xn--.net`, errInvalidDNSCharacter},      // Label ends with '-'
		{`-0b.net`, errInvalidDNSCharacter},       // First label begins with '-'
		{`-0.net`, errInvalidDNSCharacter},        // First label begins with '-'
		{`-.net`, errInvalidDNSCharacter},         // First label is only '-'
		{`---.net`, errInvalidDNSCharacter},       // First label is only hyphens
		{`0`, errTooFewLabels},
		{`1`, errTooFewLabels},
		{`*`, errInvalidDNSCharacter},
		{`**`, errInvalidDNSCharacter},
		{`*.*`, errWildcardNotSupported},
		{`zombo*com`, errInvalidDNSCharacter},
		{`*.com`, errWildcardNotSupported},
		{`*.zombo.com`, errWildcardNotSupported},
		{`..a`, errLabelTooShort},
		{`a..a`, errLabelTooShort},
		{`.a..a`, errLabelTooShort},
		{`..foo.com`, errLabelTooShort},
		{`.`, errNameEndsInDot},
		{`..`, errNameEndsInDot},
		{`a..`, errNameEndsInDot},
		{`.....`, errNameEndsInDot},
		{`.a.`, errNameEndsInDot},
		{`www.zombo.com.`, errNameEndsInDot},
		{`www.zombo_com.com`, errInvalidDNSCharacter},
		{`\uFEFF`, errInvalidDNSCharacter}, // Byte order mark
		{`\uFEFFwww.zombo.com`, errInvalidDNSCharacter},
		{`www.zom\u202Ebo.com`, errInvalidDNSCharacter}, // Right-to-Left Override
		{`\u202Ewww.zombo.com`, errInvalidDNSCharacter},
		{`www.zom\u200Fbo.com`, errInvalidDNSCharacter}, // Right-to-Left Mark
		{`\u200Fwww.zombo.com`, errInvalidDNSCharacter},
		// Underscores are technically disallowed in DNS. Some DNS
		// implementations accept them but we will be conservative.
		{`www.zom_bo.com`, errInvalidDNSCharacter},
		{`zombocom`, errTooFewLabels},
		{`localhost`, errTooFewLabels},
		{`mail`, errTooFewLabels},

		// disallow capitalized letters for #927
		{`CapitalizedLetters.com`, errInvalidDNSCharacter},

		{`example.acting`, errNonPublic},
		{`example.internal`, errNonPublic},
		// All-numeric final label not okay.
		{`www.zombo.163`, errNonPublic},
		{`xn--109-3veba6djs1bfxlfmx6c9g.xn--f1awi.xn--p1ai`, errMalformedIDN}, // Not in Unicode NFC
		{`bq--abwhky3f6fxq.jakacomo.com`, errInvalidRLDH},
		// Three hyphens starting at third second char of first label.
		{`bq---abwhky3f6fxq.jakacomo.com`, errInvalidRLDH},
		// Three hyphens starting at second char of first label.
		{`h---test.hk2yz.org`, errInvalidRLDH},
	}

	shouldBeTLDError := []string{
		`co.uk`,
		`foo.bd`,
	}

	shouldBeBlocked := []string{
		`highvalue.website1.org`,
		`website2.co.uk`,
		`www.website3.com`,
		`lots.of.labels.website4.com`,
		`banned.in.dc.com`,
		`bad.brains.banned.in.dc.com`,
	}
	blocklistContents := []string{
		`website2.com`,
		`website2.org`,
		`website2.co.uk`,
		`website3.com`,
		`website4.com`,
	}
	exactBlocklistContents := []string{
		`www.website1.org`,
		`highvalue.website1.org`,
		`dl.website1.org`,
	}
	adminBlockedContents := []string{
		`banned.in.dc.com`,
	}

	shouldBeAccepted := []string{
		`lowvalue.website1.org`,
		`website4.sucks`,
		"www.unrelated.com",
		"unrelated.com",
		"www.8675309.com",
		"8675309.com",
		"web5ite2.com",
		"www.web-site2.com",
	}

	policy := blockedNamesPolicy{
		HighRiskBlockedNames: blocklistContents,
		ExactBlockedNames:    exactBlocklistContents,
		AdminBlockedNames:    adminBlockedContents,
	}

	yamlPolicyBytes, err := yaml.Marshal(policy)
	test.AssertNotError(t, err, "Couldn't YAML serialize blocklist")
	yamlPolicyFile, _ := os.CreateTemp("", "test-blocklist.*.yaml")
	defer os.Remove(yamlPolicyFile.Name())
	err = os.WriteFile(yamlPolicyFile.Name(), yamlPolicyBytes, 0640)
	test.AssertNotError(t, err, "Couldn't write YAML blocklist")

	pa := paImpl(t)

	err = pa.SetHostnamePolicyFile(yamlPolicyFile.Name())
	test.AssertNotError(t, err, "Couldn't load rules")

	// Test for invalid identifier type
	ident := identifier.ACMEIdentifier{Type: "ip", Value: "example.com"}
	err = pa.willingToIssue(ident)
	if err != errInvalidIdentifier {
		t.Error("Identifier was not correctly forbidden: ", ident)
	}

	// Test syntax errors
	for _, tc := range testCases {
		ident := identifier.DNSIdentifier(tc.domain)
		err := pa.willingToIssue(ident)
		if err != tc.err {
			t.Errorf("WillingToIssue(%q) = %q, expected %q", tc.domain, err, tc.err)
		}
	}

	// Invalid encoding
	err = pa.willingToIssue(identifier.DNSIdentifier("www.xn--m.com"))
	test.AssertError(t, err, "WillingToIssue didn't fail on a malformed IDN")
	// Valid encoding
	err = pa.willingToIssue(identifier.DNSIdentifier("www.xn--mnich-kva.com"))
	test.AssertNotError(t, err, "WillingToIssue failed on a properly formed IDN")
	// IDN TLD
	err = pa.willingToIssue(identifier.DNSIdentifier("xn--example--3bhk5a.xn--p1ai"))
	test.AssertNotError(t, err, "WillingToIssue failed on a properly formed domain with IDN TLD")
	features.Reset()

	// Test domains that are equal to public suffixes
	for _, domain := range shouldBeTLDError {
		ident := identifier.DNSIdentifier(domain)
		err := pa.willingToIssue(ident)
		if err != errICANNTLD {
			t.Error("Identifier was not correctly forbidden: ", ident, err)
		}
	}

	// Test expected blocked domains
	for _, domain := range shouldBeBlocked {
		ident := identifier.DNSIdentifier(domain)
		err := pa.willingToIssue(ident)
		if err != errPolicyForbidden {
			t.Error("Identifier was not correctly forbidden: ", ident, err)
		}
	}

	// Test acceptance of good names
	for _, domain := range shouldBeAccepted {
		ident := identifier.DNSIdentifier(domain)
		err := pa.willingToIssue(ident)
		test.AssertNotError(t, err, "identiier was incorrectly forbidden")
	}
}

func TestWillingToIssueWildcard(t *testing.T) {
	bannedDomains := []string{
		"zombo.gov.us",
	}
	exactBannedDomains := []string{
		"highvalue.letsdecrypt.org",
	}
	pa := paImpl(t)

	bannedBytes, err := yaml.Marshal(blockedNamesPolicy{
		HighRiskBlockedNames: bannedDomains,
		ExactBlockedNames:    exactBannedDomains,
	})
	test.AssertNotError(t, err, "Couldn't serialize banned list")
	f, _ := os.CreateTemp("", "test-wildcard-banlist.*.yaml")
	defer os.Remove(f.Name())
	err = os.WriteFile(f.Name(), bannedBytes, 0640)
	test.AssertNotError(t, err, "Couldn't write serialized banned list to file")
	err = pa.SetHostnamePolicyFile(f.Name())
	test.AssertNotError(t, err, "Couldn't load policy contents from file")

	testCases := []struct {
		Name        string
		Ident       identifier.ACMEIdentifier
		ExpectedErr error
	}{
		{
			Name:        "Non-DNS identifier",
			Ident:       identifier.ACMEIdentifier{Type: "nickname", Value: "cpu"},
			ExpectedErr: errInvalidIdentifier,
		},
		{
			Name:        "Too many wildcards",
			Ident:       identifier.DNSIdentifier("ok.*.whatever.*.example.com"),
			ExpectedErr: errTooManyWildcards,
		},
		{
			Name:        "Misplaced wildcard",
			Ident:       identifier.DNSIdentifier("ok.*.whatever.example.com"),
			ExpectedErr: errMalformedWildcard,
		},
		{
			Name:        "Missing ICANN TLD",
			Ident:       identifier.DNSIdentifier("*.ok.madeup"),
			ExpectedErr: errNonPublic,
		},
		{
			Name:        "Wildcard for ICANN TLD",
			Ident:       identifier.DNSIdentifier("*.com"),
			ExpectedErr: errICANNTLDWildcard,
		},
		{
			Name:        "Forbidden base domain",
			Ident:       identifier.DNSIdentifier("*.zombo.gov.us"),
			ExpectedErr: errPolicyForbidden,
		},
		// We should not allow getting a wildcard for that would cover an exact
		// blocklist domain
		{
			Name:        "Wildcard for ExactBlocklist base domain",
			Ident:       identifier.DNSIdentifier("*.letsdecrypt.org"),
			ExpectedErr: errPolicyForbidden,
		},
		// We should allow a wildcard for a domain that doesn't match the exact
		// blocklist domain
		{
			Name:        "Wildcard for non-matching subdomain of ExactBlocklist domain",
			Ident:       identifier.DNSIdentifier("*.lowvalue.letsdecrypt.org"),
			ExpectedErr: nil,
		},
		// We should allow getting a wildcard for an exact blocklist domain since it
		// only covers subdomains, not the exact name.
		{
			Name:        "Wildcard for ExactBlocklist domain",
			Ident:       identifier.DNSIdentifier("*.highvalue.letsdecrypt.org"),
			ExpectedErr: nil,
		},
		{
			Name:        "Valid wildcard domain",
			Ident:       identifier.DNSIdentifier("*.everything.is.possible.at.zombo.com"),
			ExpectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := pa.willingToIssueWildcard(tc.Ident)
			test.AssertEquals(t, result, tc.ExpectedErr)
		})
	}
}

// TestWillingToIssueWildcards tests that more than one rejected identifier
// results in an error with suberrors.
func TestWillingToIssueWildcards(t *testing.T) {
	banned := []string{
		"letsdecrypt.org",
	}
	pa := paImpl(t)

	bannedBytes, err := yaml.Marshal(blockedNamesPolicy{
		HighRiskBlockedNames: banned,
		ExactBlockedNames:    banned,
	})
	test.AssertNotError(t, err, "Couldn't serialize banned list")
	f, _ := os.CreateTemp("", "test-wildcard-banlist.*.yaml")
	defer os.Remove(f.Name())
	err = os.WriteFile(f.Name(), bannedBytes, 0640)
	test.AssertNotError(t, err, "Couldn't write serialized banned list to file")
	err = pa.SetHostnamePolicyFile(f.Name())
	test.AssertNotError(t, err, "Couldn't load policy contents from file")

	idents := []identifier.ACMEIdentifier{
		identifier.DNSIdentifier("perfectly-fine.com"),
		identifier.DNSIdentifier("letsdecrypt.org"),
		identifier.DNSIdentifier("ok.*.this.is.a.*.weird.one.com"),
		identifier.DNSIdentifier("also-perfectly-fine.com"),
	}

	err = pa.WillingToIssueWildcards(idents)
	test.AssertError(t, err, "Expected err from WillingToIssueWildcards")

	var berr *berrors.BoulderError
	test.AssertErrorWraps(t, err, &berr)
	test.AssertEquals(t, len(berr.SubErrors), 2)
	test.AssertEquals(t, berr.Error(), "Cannot issue for \"letsdecrypt.org\": The ACME server refuses to issue a certificate for this domain name, because it is forbidden by policy (and 1 more problems. Refer to sub-problems for more information.)")

	subErrMap := make(map[string]berrors.SubBoulderError, len(berr.SubErrors))

	for _, subErr := range berr.SubErrors {
		subErrMap[subErr.Identifier.Value] = subErr
	}

	subErrA, foundA := subErrMap["letsdecrypt.org"]
	subErrB, foundB := subErrMap["ok.*.this.is.a.*.weird.one.com"]
	test.AssertEquals(t, foundA, true)
	test.AssertEquals(t, foundB, true)

	test.AssertEquals(t, subErrA.Type, berrors.RejectedIdentifier)
	test.AssertEquals(t, subErrB.Type, berrors.Malformed)

	// Test willing to issue with only *one* bad identifier.
	err = pa.WillingToIssueWildcards([]identifier.ACMEIdentifier{
		identifier.DNSIdentifier("letsdecrypt.org"),
	})
	// It should error
	test.AssertError(t, err, "Expected err from WillingToIssueWildcards")

	test.AssertErrorWraps(t, err, &berr)
	// There should be *no* suberrors because there was only one error overall.
	test.AssertEquals(t, len(berr.SubErrors), 0)
	test.AssertEquals(t, berr.Error(), "Cannot issue for \"letsdecrypt.org\": The ACME server refuses to issue a certificate for this domain name, because it is forbidden by policy")
}

func TestChallengesFor(t *testing.T) {
	pa := paImpl(t)

	challenges, err := pa.ChallengesFor(identifier.ACMEIdentifier{})
	test.AssertNotError(t, err, "ChallengesFor failed")

	test.Assert(t, len(challenges) == len(enabledChallenges), "Wrong number of challenges returned")

	seenChalls := make(map[core.AcmeChallenge]bool)
	for _, challenge := range challenges {
		test.Assert(t, !seenChalls[challenge.Type], "should not already have seen this type")
		seenChalls[challenge.Type] = true

		test.Assert(t, enabledChallenges[challenge.Type], "Unsupported challenge returned")
	}
	test.AssertEquals(t, len(seenChalls), len(enabledChallenges))

}

func TestChallengesForWildcard(t *testing.T) {
	// wildcardIdent is an identifier for a wildcard domain name
	wildcardIdent := identifier.ACMEIdentifier{
		Type:  identifier.DNS,
		Value: "*.zombo.com",
	}

	mustConstructPA := func(t *testing.T, enabledChallenges map[core.AcmeChallenge]bool) *AuthorityImpl {
		pa, err := New(enabledChallenges, blog.NewMock())
		test.AssertNotError(t, err, "Couldn't create policy implementation")
		return pa
	}

	// First try to get a challenge for the wildcard ident without the
	// DNS-01 challenge type enabled. This should produce an error
	var enabledChallenges = map[core.AcmeChallenge]bool{
		core.ChallengeTypeHTTP01: true,
		core.ChallengeTypeDNS01:  false,
	}
	pa := mustConstructPA(t, enabledChallenges)
	_, err := pa.ChallengesFor(wildcardIdent)
	test.AssertError(t, err, "ChallengesFor did not error for a wildcard ident "+
		"when DNS-01 was disabled")
	test.AssertEquals(t, err.Error(), "Challenges requested for wildcard "+
		"identifier but DNS-01 challenge type is not enabled")

	// Try again with DNS-01 enabled. It should not error and
	// should return only one DNS-01 type challenge
	enabledChallenges[core.ChallengeTypeDNS01] = true
	pa = mustConstructPA(t, enabledChallenges)
	challenges, err := pa.ChallengesFor(wildcardIdent)
	test.AssertNotError(t, err, "ChallengesFor errored for a wildcard ident "+
		"unexpectedly")
	test.AssertEquals(t, len(challenges), 1)
	test.AssertEquals(t, challenges[0].Type, core.ChallengeTypeDNS01)
}

// TestMalformedExactBlocklist tests that loading a YAML policy file with an
// invalid exact blocklist entry will fail as expected.
func TestMalformedExactBlocklist(t *testing.T) {
	pa := paImpl(t)

	exactBannedDomains := []string{
		// Only one label - not valid
		"com",
	}
	bannedDomains := []string{
		"placeholder.domain.not.important.for.this.test.com",
	}

	// Create YAML for the exactBannedDomains
	bannedBytes, err := yaml.Marshal(blockedNamesPolicy{
		HighRiskBlockedNames: bannedDomains,
		ExactBlockedNames:    exactBannedDomains,
	})
	test.AssertNotError(t, err, "Couldn't serialize banned list")

	// Create a temp file for the YAML contents
	f, _ := os.CreateTemp("", "test-invalid-exactblocklist.*.yaml")
	defer os.Remove(f.Name())
	// Write the YAML to the temp file
	err = os.WriteFile(f.Name(), bannedBytes, 0640)
	test.AssertNotError(t, err, "Couldn't write serialized banned list to file")

	// Try to use the YAML tempfile as the hostname policy. It should produce an
	// error since the exact blocklist contents are malformed.
	err = pa.SetHostnamePolicyFile(f.Name())
	test.AssertError(t, err, "Loaded invalid exact blocklist content without error")
	test.AssertEquals(t, err.Error(), "Malformed ExactBlockedNames entry, only one label: \"com\"")
}

func TestValidEmailError(t *testing.T) {
	err := ValidEmail("(๑•́ ω •̀๑)")
	test.AssertEquals(t, err.Error(), "\"(๑•́ ω •̀๑)\" is not a valid e-mail address")

	err = ValidEmail("john.smith@gmail.com #replace with real email")
	test.AssertEquals(t, err.Error(), "\"john.smith@gmail.com #replace with real email\" is not a valid e-mail address")

	err = ValidEmail("example@example.com")
	test.AssertEquals(t, err.Error(), "invalid contact domain. Contact emails @example.com are forbidden")

	err = ValidEmail("example@-foobar.com")
	test.AssertEquals(t, err.Error(), "contact email \"example@-foobar.com\" has invalid domain : Domain name contains an invalid character")
}
