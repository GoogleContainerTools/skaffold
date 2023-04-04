package sa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/jmhodges/clock"
	"github.com/letsencrypt/boulder/db"
	"github.com/letsencrypt/boulder/grpc"
	"github.com/letsencrypt/boulder/probs"

	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	"github.com/letsencrypt/boulder/test"
)

func TestRegistrationModelToPb(t *testing.T) {
	badCases := []struct {
		name  string
		input regModel
	}{
		{
			name:  "No ID",
			input: regModel{ID: 0, Key: []byte("foo"), InitialIP: []byte("foo")},
		},
		{
			name:  "No Key",
			input: regModel{ID: 1, Key: nil, InitialIP: []byte("foo")},
		},
		{
			name:  "No IP",
			input: regModel{ID: 1, Key: []byte("foo"), InitialIP: nil},
		},
		{
			name:  "Bad IP",
			input: regModel{ID: 1, Key: []byte("foo"), InitialIP: []byte("foo")},
		},
	}
	for _, tc := range badCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := registrationModelToPb(&tc.input)
			test.AssertError(t, err, "Should fail")
		})
	}

	_, err := registrationModelToPb(&regModel{
		ID: 1, Key: []byte("foo"), InitialIP: net.ParseIP("1.2.3.4"),
	})
	test.AssertNotError(t, err, "Should pass")
}

func TestRegistrationPbToModel(t *testing.T) {}

func TestAuthzModel(t *testing.T) {
	authzPB := &corepb.Authorization{
		Id:             "1",
		Identifier:     "example.com",
		RegistrationID: 1,
		Status:         string(core.StatusValid),
		Expires:        1234,
		Challenges: []*corepb.Challenge{
			{
				Type:      string(core.ChallengeTypeHTTP01),
				Status:    string(core.StatusValid),
				Token:     "MTIz",
				Validated: 1234,
				Validationrecords: []*corepb.ValidationRecord{
					{
						Hostname:          "hostname",
						Port:              "port",
						AddressUsed:       []byte("1.2.3.4"),
						Url:               "url",
						AddressesResolved: [][]byte{{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4}},
						AddressesTried:    [][]byte{{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4}},
					},
				},
			},
		},
	}

	model, err := authzPBToModel(authzPB)
	test.AssertNotError(t, err, "authzPBToModel failed")

	authzPBOut, err := modelToAuthzPB(*model)
	test.AssertNotError(t, err, "modelToAuthzPB failed")
	test.AssertDeepEquals(t, authzPB.Challenges, authzPBOut.Challenges)

	validationErr := probs.ConnectionFailure("weewoo")
	authzPB.Challenges[0].Status = string(core.StatusInvalid)
	authzPB.Challenges[0].Error, err = grpc.ProblemDetailsToPB(validationErr)
	test.AssertNotError(t, err, "grpc.ProblemDetailsToPB failed")
	model, err = authzPBToModel(authzPB)
	test.AssertNotError(t, err, "authzPBToModel failed")

	authzPBOut, err = modelToAuthzPB(*model)
	test.AssertNotError(t, err, "modelToAuthzPB failed")
	test.AssertDeepEquals(t, authzPB.Challenges, authzPBOut.Challenges)

	authzPB = &corepb.Authorization{
		Id:             "1",
		Identifier:     "example.com",
		RegistrationID: 1,
		Status:         string(core.StatusInvalid),
		Expires:        1234,
		Challenges: []*corepb.Challenge{
			{
				Type:   string(core.ChallengeTypeHTTP01),
				Status: string(core.StatusInvalid),
				Token:  "MTIz",
				Validationrecords: []*corepb.ValidationRecord{
					{
						Hostname:          "hostname",
						Port:              "port",
						AddressUsed:       []byte("1.2.3.4"),
						Url:               "url",
						AddressesResolved: [][]byte{{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4}},
						AddressesTried:    [][]byte{{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4}},
					},
				},
			},
			{
				Type:   string(core.ChallengeTypeDNS01),
				Status: string(core.StatusInvalid),
				Token:  "MTIz",
				Validationrecords: []*corepb.ValidationRecord{
					{
						Hostname:          "hostname",
						Port:              "port",
						AddressUsed:       []byte("1.2.3.4"),
						Url:               "url",
						AddressesResolved: [][]byte{{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4}},
						AddressesTried:    [][]byte{{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4}},
					},
				},
			},
		},
	}
	_, err = authzPBToModel(authzPB)
	test.AssertError(t, err, "authzPBToModel didn't fail with multiple non-pending challenges")
}

// TestModelToOrderBADJSON tests that converting an order model with an invalid
// validation error JSON field to an Order produces the expected bad JSON error.
func TestModelToOrderBadJSON(t *testing.T) {
	badJSON := []byte(`{`)
	_, err := modelToOrder(&orderModel{
		Error: badJSON,
	})
	test.AssertError(t, err, "expected error from modelToOrder")
	var badJSONErr errBadJSON
	test.AssertErrorWraps(t, err, &badJSONErr)
	test.AssertEquals(t, string(badJSONErr.json), string(badJSON))
}

// TestPopulateAttemptedFieldsBadJSON tests that populating a challenge from an
// authz2 model with an invalid validation error or an invalid validation record
// produces the expected bad JSON error.
func TestPopulateAttemptedFieldsBadJSON(t *testing.T) {
	badJSON := []byte(`{`)

	testCases := []struct {
		Name  string
		Model *authzModel
	}{
		{
			Name: "Bad validation error field",
			Model: &authzModel{
				ValidationError: badJSON,
			},
		},
		{
			Name: "Bad validation record field",
			Model: &authzModel{
				ValidationRecord: badJSON,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := populateAttemptedFields(*tc.Model, &corepb.Challenge{})
			test.AssertError(t, err, "expected error from populateAttemptedFields")
			var badJSONErr errBadJSON
			test.AssertErrorWraps(t, err, &badJSONErr)
			test.AssertEquals(t, string(badJSONErr.json), string(badJSON))
		})
	}
}

func TestCertificatesTableContainsDuplicateSerials(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	serialString := core.SerialToString(big.NewInt(1337))

	// Insert a certificate with a serial of `1337`.
	err := insertCertificate(sa.dbMap, fc, "1337.com", "leet", 1337, 1)
	test.AssertNotError(t, err, "couldn't insert valid certificate")

	// This should return the certificate that we just inserted.
	certA, err := SelectCertificate(sa.dbMap, serialString)
	test.AssertNotError(t, err, "received an error for a valid query")

	// Insert a certificate with a serial of `1337` but for a different
	// hostname.
	err = insertCertificate(sa.dbMap, fc, "1337.net", "leet", 1337, 1)
	test.AssertNotError(t, err, "couldn't insert valid certificate")

	// Despite a duplicate being present, this shouldn't error.
	certB, err := SelectCertificate(sa.dbMap, serialString)
	test.AssertNotError(t, err, "received an error for a valid query")

	// Ensure that `certA` and `certB` are the same.
	test.AssertByteEquals(t, certA.DER, certB.DER)
}

func insertCertificate(dbMap *db.WrappedMap, fc clock.FakeClock, hostname, cn string, serial, regID int64) error {
	serialBigInt := big.NewInt(serial)
	serialString := core.SerialToString(serialBigInt)

	template := x509.Certificate{
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotAfter:     fc.Now().Add(30 * 24 * time.Hour),
		DNSNames:     []string{hostname},
		SerialNumber: serialBigInt,
	}

	testKey := makeKey()
	certDer, _ := x509.CreateCertificate(rand.Reader, &template, &template, &testKey.PublicKey, &testKey)
	cert := &core.Certificate{
		RegistrationID: regID,
		Serial:         serialString,
		Expires:        template.NotAfter,
		DER:            certDer,
	}
	err := dbMap.Insert(cert)
	if err != nil {
		return err
	}
	return nil
}

func bigIntFromB64(b64 string) *big.Int {
	bytes, _ := base64.URLEncoding.DecodeString(b64)
	x := big.NewInt(0)
	x.SetBytes(bytes)
	return x
}

func makeKey() rsa.PrivateKey {
	n := bigIntFromB64("n4EPtAOCc9AlkeQHPzHStgAbgs7bTZLwUBZdR8_KuKPEHLd4rHVTeT-O-XV2jRojdNhxJWTDvNd7nqQ0VEiZQHz_AJmSCpMaJMRBSFKrKb2wqVwGU_NsYOYL-QtiWN2lbzcEe6XC0dApr5ydQLrHqkHHig3RBordaZ6Aj-oBHqFEHYpPe7Tpe-OfVfHd1E6cS6M1FZcD1NNLYD5lFHpPI9bTwJlsde3uhGqC0ZCuEHg8lhzwOHrtIQbS0FVbb9k3-tVTU4fg_3L_vniUFAKwuCLqKnS2BYwdq_mzSnbLY7h_qixoR7jig3__kRhuaxwUkRz5iaiQkqgc5gHdrNP5zw==")
	e := int(bigIntFromB64("AQAB").Int64())
	d := bigIntFromB64("bWUC9B-EFRIo8kpGfh0ZuyGPvMNKvYWNtB_ikiH9k20eT-O1q_I78eiZkpXxXQ0UTEs2LsNRS-8uJbvQ-A1irkwMSMkK1J3XTGgdrhCku9gRldY7sNA_AKZGh-Q661_42rINLRCe8W-nZ34ui_qOfkLnK9QWDDqpaIsA-bMwWWSDFu2MUBYwkHTMEzLYGqOe04noqeq1hExBTHBOBdkMXiuFhUq1BU6l-DqEiWxqg82sXt2h-LMnT3046AOYJoRioz75tSUQfGCshWTBnP5uDjd18kKhyv07lhfSJdrPdM5Plyl21hsFf4L_mHCuoFau7gdsPfHPxxjVOcOpBrQzwQ==")
	p := bigIntFromB64("uKE2dh-cTf6ERF4k4e_jy78GfPYUIaUyoSSJuBzp3Cubk3OCqs6grT8bR_cu0Dm1MZwWmtdqDyI95HrUeq3MP15vMMON8lHTeZu2lmKvwqW7anV5UzhM1iZ7z4yMkuUwFWoBvyY898EXvRD-hdqRxHlSqAZ192zB3pVFJ0s7pFc=")
	q := bigIntFromB64("uKE2dh-cTf6ERF4k4e_jy78GfPYUIaUyoSSJuBzp3Cubk3OCqs6grT8bR_cu0Dm1MZwWmtdqDyI95HrUeq3MP15vMMON8lHTeZu2lmKvwqW7anV5UzhM1iZ7z4yMkuUwFWoBvyY898EXvRD-hdqRxHlSqAZ192zB3pVFJ0s7pFc=")
	return rsa.PrivateKey{PublicKey: rsa.PublicKey{N: n, E: e}, D: d, Primes: []*big.Int{p, q}}
}
