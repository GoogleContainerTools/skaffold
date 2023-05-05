package probs

import (
	"testing"

	"net/http"

	"github.com/letsencrypt/boulder/identifier"
	"github.com/letsencrypt/boulder/test"
)

func TestProblemDetails(t *testing.T) {
	pd := &ProblemDetails{
		Type:       MalformedProblem,
		Detail:     "Wat? o.O",
		HTTPStatus: 403,
	}
	test.AssertEquals(t, pd.Error(), "malformed :: Wat? o.O")
}

func TestProblemDetailsToStatusCode(t *testing.T) {
	testCases := []struct {
		pb         *ProblemDetails
		statusCode int
	}{
		{&ProblemDetails{Type: ConnectionProblem}, http.StatusBadRequest},
		{&ProblemDetails{Type: MalformedProblem}, http.StatusBadRequest},
		{&ProblemDetails{Type: ServerInternalProblem}, http.StatusInternalServerError},
		{&ProblemDetails{Type: TLSProblem}, http.StatusBadRequest},
		{&ProblemDetails{Type: UnauthorizedProblem}, http.StatusForbidden},
		{&ProblemDetails{Type: RateLimitedProblem}, statusTooManyRequests},
		{&ProblemDetails{Type: BadNonceProblem}, http.StatusBadRequest},
		{&ProblemDetails{Type: InvalidEmailProblem}, http.StatusBadRequest},
		{&ProblemDetails{Type: "foo"}, http.StatusInternalServerError},
		{&ProblemDetails{Type: "foo", HTTPStatus: 200}, 200},
		{&ProblemDetails{Type: ConnectionProblem, HTTPStatus: 200}, 200},
		{&ProblemDetails{Type: AccountDoesNotExistProblem}, http.StatusBadRequest},
		{&ProblemDetails{Type: BadRevocationReasonProblem}, http.StatusBadRequest},
	}

	for _, c := range testCases {
		p := ProblemDetailsToStatusCode(c.pb)
		if c.statusCode != p {
			t.Errorf("Incorrect status code for %s. Expected %d, got %d", c.pb.Type, c.statusCode, p)
		}
	}
}

func TestProblemDetailsConvenience(t *testing.T) {
	testCases := []struct {
		pb           *ProblemDetails
		expectedType ProblemType
		statusCode   int
		detail       string
	}{
		{InvalidEmail("invalid email detail"), InvalidEmailProblem, http.StatusBadRequest, "invalid email detail"},
		{ConnectionFailure("connection failure detail"), ConnectionProblem, http.StatusBadRequest, "connection failure detail"},
		{Malformed("malformed detail"), MalformedProblem, http.StatusBadRequest, "malformed detail"},
		{ServerInternal("internal error detail"), ServerInternalProblem, http.StatusInternalServerError, "internal error detail"},
		{Unauthorized("unauthorized detail"), UnauthorizedProblem, http.StatusForbidden, "unauthorized detail"},
		{RateLimited("rate limited detail"), RateLimitedProblem, statusTooManyRequests, "rate limited detail"},
		{BadNonce("bad nonce detail"), BadNonceProblem, http.StatusBadRequest, "bad nonce detail"},
		{TLSError("TLS error detail"), TLSProblem, http.StatusBadRequest, "TLS error detail"},
		{RejectedIdentifier("rejected identifier detail"), RejectedIdentifierProblem, http.StatusBadRequest, "rejected identifier detail"},
		{AccountDoesNotExist("no account detail"), AccountDoesNotExistProblem, http.StatusBadRequest, "no account detail"},
		{BadRevocationReason("only reason xxx is supported"), BadRevocationReasonProblem, http.StatusBadRequest, "only reason xxx is supported"},
	}

	for _, c := range testCases {
		if c.pb.Type != c.expectedType {
			t.Errorf("Incorrect problem type. Expected %s got %s", c.expectedType, c.pb.Type)
		}

		if c.pb.HTTPStatus != c.statusCode {
			t.Errorf("Incorrect HTTP Status. Expected %d got %d", c.statusCode, c.pb.HTTPStatus)
		}

		if c.pb.Detail != c.detail {
			t.Errorf("Incorrect detail message. Expected %s got %s", c.detail, c.pb.Detail)
		}

		if subProbLen := len(c.pb.SubProblems); subProbLen != 0 {
			t.Errorf("Incorrect SubProblems. Expected 0, found %d", subProbLen)
		}
	}
}

// TestWithSubProblems tests that a new problem can be constructed by adding
// subproblems.
func TestWithSubProblems(t *testing.T) {
	topProb := &ProblemDetails{
		Type:       RateLimitedProblem,
		Detail:     "don't you think you have enough certificates already?",
		HTTPStatus: statusTooManyRequests,
	}
	subProbs := []SubProblemDetails{
		{
			Identifier: identifier.DNSIdentifier("example.com"),
			ProblemDetails: ProblemDetails{
				Type:       RateLimitedProblem,
				Detail:     "don't you think you have enough certificates already?",
				HTTPStatus: statusTooManyRequests,
			},
		},
		{
			Identifier: identifier.DNSIdentifier("what about example.com"),
			ProblemDetails: ProblemDetails{
				Type:       MalformedProblem,
				Detail:     "try a real identifier value next time",
				HTTPStatus: http.StatusConflict,
			},
		},
	}

	outResult := topProb.WithSubProblems(subProbs)

	// The outResult should be a new, distinct problem details instance
	test.AssertNotEquals(t, topProb, outResult)
	// The outResult problem details should have the correct sub problems
	test.AssertDeepEquals(t, outResult.SubProblems, subProbs)
	// Adding another sub problem shouldn't squash the original sub problems
	anotherSubProb := SubProblemDetails{
		Identifier: identifier.DNSIdentifier("another ident"),
		ProblemDetails: ProblemDetails{
			Type:       RateLimitedProblem,
			Detail:     "yet another rate limit err",
			HTTPStatus: statusTooManyRequests,
		},
	}
	outResult = outResult.WithSubProblems([]SubProblemDetails{anotherSubProb})
	test.AssertDeepEquals(t, outResult.SubProblems, append(subProbs, anotherSubProb))
}
