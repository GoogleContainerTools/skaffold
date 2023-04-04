//go:build integration

package integration

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/eggsampler/acme/v3"

	"github.com/letsencrypt/boulder/test"
)

// TestTooBigOrderError tests that submitting an order with more than 100 names
// produces the expected problem result.
func TestTooBigOrderError(t *testing.T) {
	t.Parallel()
	os.Setenv("DIRECTORY", "http://boulder.service.consul:4001/directory")

	var domains []string
	for i := 0; i < 101; i++ {
		domains = append(domains, fmt.Sprintf("%d.example.com", i))
	}

	_, err := authAndIssue(nil, nil, domains)
	test.AssertError(t, err, "authAndIssue failed")

	var prob acme.Problem
	test.AssertErrorWraps(t, err, &prob)
	test.AssertEquals(t, prob.Type, "urn:ietf:params:acme:error:malformed")
	test.AssertEquals(t, prob.Detail, "Error creating new order :: Order cannot contain more than 100 DNS names")
}

// TestAccountEmailError tests that registering a new account, or updating an
// account, with invalid contact information produces the expected problem
// result to ACME clients.
func TestAccountEmailError(t *testing.T) {
	t.Parallel()
	os.Setenv("DIRECTORY", "http://boulder.service.consul:4001/directory")

	// The registrations.contact field is VARCHAR(191). 175 'a' characters plus
	// the prefix "mailto:" and the suffix "@a.com" makes exactly 191 bytes of
	// encoded JSON. The correct size to hit our maximum DB field length.
	var longStringBuf strings.Builder
	longStringBuf.WriteString("mailto:")
	for i := 0; i < 175; i++ {
		longStringBuf.WriteRune('a')
	}
	longStringBuf.WriteString("@a.com")

	createErrorPrefix := "Error creating new account :: "
	updateErrorPrefix := "Unable to update account :: "

	testCases := []struct {
		name               string
		contacts           []string
		expectedProbType   string
		expectedProbDetail string
	}{
		{
			name:               "empty contact",
			contacts:           []string{"mailto:valid@valid.com", ""},
			expectedProbType:   "urn:ietf:params:acme:error:invalidEmail",
			expectedProbDetail: `empty contact`,
		},
		{
			name:               "empty proto",
			contacts:           []string{"mailto:valid@valid.com", " "},
			expectedProbType:   "urn:ietf:params:acme:error:invalidEmail",
			expectedProbDetail: `contact method "" is not supported`,
		},
		{
			name:               "empty mailto",
			contacts:           []string{"mailto:valid@valid.com", "mailto:"},
			expectedProbType:   "urn:ietf:params:acme:error:invalidEmail",
			expectedProbDetail: `"" is not a valid e-mail address`,
		},
		{
			name:               "non-ascii mailto",
			contacts:           []string{"mailto:valid@valid.com", "mailto:cpu@l̴etsencrypt.org"},
			expectedProbType:   "urn:ietf:params:acme:error:invalidEmail",
			expectedProbDetail: `contact email ["mailto:cpu@l̴etsencrypt.org"] contains non-ASCII characters`,
		},
		{
			name:               "too many contacts",
			contacts:           []string{"a", "b", "c", "d"},
			expectedProbType:   "urn:ietf:params:acme:error:malformed",
			expectedProbDetail: `too many contacts provided: 4 > 3`,
		},
		{
			name:               "invalid contact",
			contacts:           []string{"mailto:valid@valid.com", "mailto:a@"},
			expectedProbType:   "urn:ietf:params:acme:error:invalidEmail",
			expectedProbDetail: `"a@" is not a valid e-mail address`,
		},
		{
			name:               "forbidden contact domain",
			contacts:           []string{"mailto:valid@valid.com", "mailto:a@example.com"},
			expectedProbType:   "urn:ietf:params:acme:error:invalidEmail",
			expectedProbDetail: "invalid contact domain. Contact emails @example.com are forbidden",
		},
		{
			name:               "contact domain invalid TLD",
			contacts:           []string{"mailto:valid@valid.com", "mailto:a@example.cpu"},
			expectedProbType:   "urn:ietf:params:acme:error:invalidEmail",
			expectedProbDetail: `contact email "a@example.cpu" has invalid domain : Domain name does not end with a valid public suffix (TLD)`,
		},
		{
			name:               "contact domain invalid",
			contacts:           []string{"mailto:valid@valid.com", "mailto:a@example./.com"},
			expectedProbType:   "urn:ietf:params:acme:error:invalidEmail",
			expectedProbDetail: "contact email \"a@example./.com\" has invalid domain : Domain name contains an invalid character",
		},
		{
			name: "too long contact",
			contacts: []string{
				longStringBuf.String(),
			},
			expectedProbType:   "urn:ietf:params:acme:error:invalidEmail",
			expectedProbDetail: `too many/too long contact(s). Please use shorter or fewer email addresses`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// First try registering a new account and ensuring the expected problem occurs
			var prob acme.Problem
			if _, err := makeClient(tc.contacts...); err != nil {
				test.AssertErrorWraps(t, err, &prob)
				test.AssertEquals(t, prob.Type, tc.expectedProbType)
				test.AssertEquals(t, prob.Detail, createErrorPrefix+tc.expectedProbDetail)
			} else if err == nil {
				t.Errorf("expected %s type problem for %q, got nil",
					tc.expectedProbType, strings.Join(tc.contacts, ","))
			}

			// Next try making a client with a good contact and updating with the test
			// case contact info. The same problem should occur.
			c, err := makeClient("mailto:valid@valid.com")
			test.AssertNotError(t, err, "failed to create account with valid contact")
			if _, err := c.UpdateAccount(c.Account, tc.contacts...); err != nil {
				test.AssertErrorWraps(t, err, &prob)
				test.AssertEquals(t, prob.Type, tc.expectedProbType)
				test.AssertEquals(t, prob.Detail, updateErrorPrefix+tc.expectedProbDetail)
			} else if err == nil {
				t.Errorf("expected %s type problem after updating account to %q, got nil",
					tc.expectedProbType, strings.Join(tc.contacts, ","))
			}
		})
	}
}
