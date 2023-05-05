package grpc

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"gopkg.in/go-jose/go-jose.v2"

	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	"github.com/letsencrypt/boulder/identifier"
	"github.com/letsencrypt/boulder/probs"
	"github.com/letsencrypt/boulder/test"
)

const JWK1JSON = `{"kty":"RSA","n":"vuc785P8lBj3fUxyZchF_uZw6WtbxcorqgTyq-qapF5lrO1U82Tp93rpXlmctj6fyFHBVVB5aXnUHJ7LZeVPod7Wnfl8p5OyhlHQHC8BnzdzCqCMKmWZNX5DtETDId0qzU7dPzh0LP0idt5buU7L9QNaabChw3nnaL47iu_1Di5Wp264p2TwACeedv2hfRDjDlJmaQXuS8Rtv9GnRWyC9JBu7XmGvGDziumnJH7Hyzh3VNu-kSPQD3vuAFgMZS6uUzOztCkT0fpOalZI6hqxtWLvXUMj-crXrn-Maavz8qRhpAyp5kcYk3jiHGgQIi7QSK2JIdRJ8APyX9HlmTN5AQ","e":"AQAB"}`

func TestProblemDetails(t *testing.T) {
	pb, err := ProblemDetailsToPB(nil)
	test.AssertNotEquals(t, err, "problemDetailToPB failed")
	test.Assert(t, pb == nil, "Returned corepb.ProblemDetails is not nil")

	prob := &probs.ProblemDetails{Type: probs.TLSProblem, Detail: "asd", HTTPStatus: 200}
	pb, err = ProblemDetailsToPB(prob)
	test.AssertNotError(t, err, "problemDetailToPB failed")
	test.Assert(t, pb != nil, "return corepb.ProblemDetails is nill")
	test.AssertDeepEquals(t, pb.ProblemType, string(prob.Type))
	test.AssertEquals(t, pb.Detail, prob.Detail)
	test.AssertEquals(t, int(pb.HttpStatus), prob.HTTPStatus)

	recon, err := PBToProblemDetails(pb)
	test.AssertNotError(t, err, "PBToProblemDetails failed")
	test.AssertDeepEquals(t, recon, prob)

	recon, err = PBToProblemDetails(nil)
	test.AssertNotError(t, err, "PBToProblemDetails failed")
	test.Assert(t, recon == nil, "Returned core.PRoblemDetails is not nil")
	_, err = PBToProblemDetails(&corepb.ProblemDetails{})
	test.AssertError(t, err, "PBToProblemDetails did not fail")
	test.AssertEquals(t, err, ErrMissingParameters)
	_, err = PBToProblemDetails(&corepb.ProblemDetails{ProblemType: ""})
	test.AssertError(t, err, "PBToProblemDetails did not fail")
	test.AssertEquals(t, err, ErrMissingParameters)
	_, err = PBToProblemDetails(&corepb.ProblemDetails{Detail: ""})
	test.AssertError(t, err, "PBToProblemDetails did not fail")
	test.AssertEquals(t, err, ErrMissingParameters)
}

func TestChallenge(t *testing.T) {
	var jwk jose.JSONWebKey
	err := json.Unmarshal([]byte(JWK1JSON), &jwk)
	test.AssertNotError(t, err, "Failed to unmarshal test key")
	validated := time.Now().Round(0).UTC()
	chall := core.Challenge{
		Type:                     core.ChallengeTypeDNS01,
		Status:                   core.StatusValid,
		Token:                    "asd",
		ProvidedKeyAuthorization: "keyauth",
		Validated:                &validated,
	}

	pb, err := ChallengeToPB(chall)
	test.AssertNotError(t, err, "ChallengeToPB failed")
	test.Assert(t, pb != nil, "Returned corepb.Challenge is nil")

	recon, err := PBToChallenge(pb)
	test.AssertNotError(t, err, "PBToChallenge failed")
	test.AssertDeepEquals(t, recon, chall)

	ip := net.ParseIP("1.1.1.1")
	chall.ValidationRecord = []core.ValidationRecord{
		{
			Hostname:          "host",
			Port:              "2020",
			AddressesResolved: []net.IP{ip},
			AddressUsed:       ip,
			URL:               "url",
			AddressesTried:    []net.IP{ip},
		},
	}
	chall.Error = &probs.ProblemDetails{Type: probs.TLSProblem, Detail: "asd", HTTPStatus: 200}
	pb, err = ChallengeToPB(chall)
	test.AssertNotError(t, err, "ChallengeToPB failed")
	test.Assert(t, pb != nil, "Returned corepb.Challenge is nil")

	recon, err = PBToChallenge(pb)
	test.AssertNotError(t, err, "PBToChallenge failed")
	test.AssertDeepEquals(t, recon, chall)

	_, err = PBToChallenge(nil)
	test.AssertError(t, err, "PBToChallenge did not fail")
	test.AssertEquals(t, err, ErrMissingParameters)
	_, err = PBToChallenge(&corepb.Challenge{})
	test.AssertError(t, err, "PBToChallenge did not fail")
	test.AssertEquals(t, err, ErrMissingParameters)
}

func TestValidationRecord(t *testing.T) {
	ip := net.ParseIP("1.1.1.1")
	vr := core.ValidationRecord{
		Hostname:          "host",
		Port:              "2020",
		AddressesResolved: []net.IP{ip},
		AddressUsed:       ip,
		URL:               "url",
		AddressesTried:    []net.IP{ip},
	}

	pb, err := ValidationRecordToPB(vr)
	test.AssertNotError(t, err, "ValidationRecordToPB failed")
	test.Assert(t, pb != nil, "Return core.ValidationRecord is nil")

	recon, err := PBToValidationRecord(pb)
	test.AssertNotError(t, err, "PBToValidationRecord failed")
	test.AssertDeepEquals(t, recon, vr)
}

func TestValidationResult(t *testing.T) {
	ip := net.ParseIP("1.1.1.1")
	vrA := core.ValidationRecord{
		Hostname:          "hostA",
		Port:              "2020",
		AddressesResolved: []net.IP{ip},
		AddressUsed:       ip,
		URL:               "urlA",
		AddressesTried:    []net.IP{ip},
	}
	vrB := core.ValidationRecord{
		Hostname:          "hostB",
		Port:              "2020",
		AddressesResolved: []net.IP{ip},
		AddressUsed:       ip,
		URL:               "urlB",
		AddressesTried:    []net.IP{ip},
	}
	result := []core.ValidationRecord{vrA, vrB}
	prob := &probs.ProblemDetails{Type: probs.TLSProblem, Detail: "asd", HTTPStatus: 200}

	pb, err := ValidationResultToPB(result, prob)
	test.AssertNotError(t, err, "ValidationResultToPB failed")
	test.Assert(t, pb != nil, "Returned vapb.ValidationResult is nil")

	reconResult, reconProb, err := pbToValidationResult(pb)
	test.AssertNotError(t, err, "pbToValidationResult failed")
	test.AssertDeepEquals(t, reconResult, result)
	test.AssertDeepEquals(t, reconProb, prob)
}

func TestRegistration(t *testing.T) {
	contacts := []string{"email"}
	var key jose.JSONWebKey
	err := json.Unmarshal([]byte(`
		{
			"e": "AQAB",
			"kty": "RSA",
			"n": "tSwgy3ORGvc7YJI9B2qqkelZRUC6F1S5NwXFvM4w5-M0TsxbFsH5UH6adigV0jzsDJ5imAechcSoOhAh9POceCbPN1sTNwLpNbOLiQQ7RD5mY_pSUHWXNmS9R4NZ3t2fQAzPeW7jOfF0LKuJRGkekx6tXP1uSnNibgpJULNc4208dgBaCHo3mvaE2HV2GmVl1yxwWX5QZZkGQGjNDZYnjFfa2DKVvFs0QbAk21ROm594kAxlRlMMrvqlf24Eq4ERO0ptzpZgm_3j_e4hGRD39gJS7kAzK-j2cacFQ5Qi2Y6wZI2p-FCq_wiYsfEAIkATPBiLKl_6d_Jfcvs_impcXQ"
		}
	`), &key)
	test.AssertNotError(t, err, "Could not unmarshal testing key")
	createdAt := time.Now().Round(0).UTC()
	inReg := core.Registration{
		ID:        1,
		Key:       &key,
		Contact:   &contacts,
		Agreement: "yup",
		InitialIP: net.ParseIP("1.1.1.1"),
		CreatedAt: &createdAt,
		Status:    core.StatusValid,
	}
	pbReg, err := RegistrationToPB(inReg)
	test.AssertNotError(t, err, "registrationToPB failed")
	outReg, err := PbToRegistration(pbReg)
	test.AssertNotError(t, err, "PbToRegistration failed")
	test.AssertDeepEquals(t, inReg, outReg)

	inReg.Contact = nil
	pbReg, err = RegistrationToPB(inReg)
	test.AssertNotError(t, err, "registrationToPB failed")
	pbReg.Contact = []string{}
	outReg, err = PbToRegistration(pbReg)
	test.AssertNotError(t, err, "PbToRegistration failed")
	test.AssertDeepEquals(t, inReg, outReg)

	var empty []string
	inReg.Contact = &empty
	pbReg, err = RegistrationToPB(inReg)
	test.AssertNotError(t, err, "registrationToPB failed")
	outReg, err = PbToRegistration(pbReg)
	test.AssertNotError(t, err, "PbToRegistration failed")
	test.Assert(t, *outReg.Contact != nil, "Empty slice was converted to a nil slice")
}

func TestAuthz(t *testing.T) {
	exp := time.Now().AddDate(0, 0, 1).UTC()
	identifier := identifier.ACMEIdentifier{Type: identifier.DNS, Value: "example.com"}
	challA := core.Challenge{
		Type:                     core.ChallengeTypeDNS01,
		Status:                   core.StatusPending,
		Token:                    "asd",
		ProvidedKeyAuthorization: "keyauth",
	}
	challB := core.Challenge{
		Type:                     core.ChallengeTypeDNS01,
		Status:                   core.StatusPending,
		Token:                    "asd2",
		ProvidedKeyAuthorization: "keyauth4",
	}
	inAuthz := core.Authorization{
		ID:             "1",
		Identifier:     identifier,
		RegistrationID: 5,
		Status:         core.StatusPending,
		Expires:        &exp,
		Challenges:     []core.Challenge{challA, challB},
	}

	pbAuthz, err := AuthzToPB(inAuthz)
	test.AssertNotError(t, err, "AuthzToPB failed")
	outAuthz, err := PBToAuthz(pbAuthz)
	test.AssertNotError(t, err, "pbToAuthz failed")
	test.AssertDeepEquals(t, inAuthz, outAuthz)
}

func TestCert(t *testing.T) {
	now := time.Now().Round(0)
	cert := core.Certificate{
		RegistrationID: 1,
		Serial:         "serial",
		Digest:         "digest",
		DER:            []byte{255},
		Issued:         now,
		Expires:        now.Add(time.Hour),
	}

	certPB := CertToPB(cert)
	outCert, _ := PBToCert(certPB)

	test.AssertDeepEquals(t, cert, outCert)
}

func TestOrderValid(t *testing.T) {
	testCases := []struct {
		Name          string
		Order         *corepb.Order
		ExpectedValid bool
	}{
		{
			Name: "All valid",
			Order: &corepb.Order{
				Id:                1,
				RegistrationID:    1,
				Expires:           1,
				CertificateSerial: "",
				V2Authorizations:  []int64{},
				Names:             []string{"example.com"},
				BeganProcessing:   false,
				Created:           1,
			},
			ExpectedValid: true,
		},
		{
			Name: "Serial empty",
			Order: &corepb.Order{
				Id:               1,
				RegistrationID:   1,
				Expires:          1,
				V2Authorizations: []int64{},
				Names:            []string{"example.com"},
				BeganProcessing:  false,
				Created:          1,
			},
			ExpectedValid: true,
		},
		{
			Name:  "All zero",
			Order: &corepb.Order{},
		},
		{
			Name: "ID 0",
			Order: &corepb.Order{
				Id:                0,
				RegistrationID:    1,
				Expires:           1,
				CertificateSerial: "",
				V2Authorizations:  []int64{},
				Names:             []string{"example.com"},
				BeganProcessing:   false,
			},
		},
		{
			Name: "Reg ID zero",
			Order: &corepb.Order{
				Id:                1,
				RegistrationID:    0,
				Expires:           1,
				CertificateSerial: "",
				V2Authorizations:  []int64{},
				Names:             []string{"example.com"},
				BeganProcessing:   false,
			},
		},
		{
			Name: "Expires 0",
			Order: &corepb.Order{
				Id:                1,
				RegistrationID:    1,
				Expires:           0,
				CertificateSerial: "",
				V2Authorizations:  []int64{},
				Names:             []string{"example.com"},
				BeganProcessing:   false,
			},
		},
		{
			Name: "Names empty",
			Order: &corepb.Order{
				Id:                1,
				RegistrationID:    1,
				Expires:           1,
				CertificateSerial: "",
				V2Authorizations:  []int64{},
				Names:             []string{},
				BeganProcessing:   false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := orderValid(tc.Order)
			test.AssertEquals(t, result, tc.ExpectedValid)
		})
	}
}
