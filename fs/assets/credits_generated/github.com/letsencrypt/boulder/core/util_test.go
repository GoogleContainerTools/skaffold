package core

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"gopkg.in/go-jose/go-jose.v2"

	"github.com/letsencrypt/boulder/test"
)

// challenges.go
func TestNewToken(t *testing.T) {
	token := NewToken()
	fmt.Println(token)
	tokenLength := int(math.Ceil(32 * 8 / 6.0)) // 32 bytes, b64 encoded
	if len(token) != tokenLength {
		t.Fatalf("Expected token of length %d, got %d", tokenLength, len(token))
	}
	collider := map[string]bool{}
	// Test for very blatant RNG failures:
	// Try 2^20 birthdays in a 2^72 search space...
	// our naive collision probability here is  2^-32...
	for i := 0; i < 1000000; i++ {
		token = NewToken()[:12] // just sample a portion
		test.Assert(t, !collider[token], "Token collision!")
		collider[token] = true
	}
}

func TestLooksLikeAToken(t *testing.T) {
	test.Assert(t, !LooksLikeAToken("R-UL_7MrV3tUUjO9v5ym2srK3dGGCwlxbVyKBdwLOS"), "Accepted short token")
	test.Assert(t, !LooksLikeAToken("R-UL_7MrV3tUUjO9v5ym2srK3dGGCwlxbVyKBdwLOS%"), "Accepted invalid token")
	test.Assert(t, LooksLikeAToken("R-UL_7MrV3tUUjO9v5ym2srK3dGGCwlxbVyKBdwLOSU"), "Rejected valid token")
}

func TestSerialUtils(t *testing.T) {
	serial := SerialToString(big.NewInt(100000000000000000))
	test.AssertEquals(t, serial, "00000000000000000000016345785d8a0000")

	serialNum, err := StringToSerial("00000000000000000000016345785d8a0000")
	test.AssertNotError(t, err, "Couldn't convert serial number to *big.Int")
	if serialNum.Cmp(big.NewInt(100000000000000000)) != 0 {
		t.Fatalf("Incorrect conversion, got %d", serialNum)
	}

	badSerial, err := StringToSerial("doop!!!!000")
	test.AssertEquals(t, fmt.Sprintf("%v", err), "Invalid serial number")
	fmt.Println(badSerial)
}

func TestBuildID(t *testing.T) {
	test.AssertEquals(t, Unspecified, GetBuildID())
}

const JWK1JSON = `{
  "kty": "RSA",
  "n": "vuc785P8lBj3fUxyZchF_uZw6WtbxcorqgTyq-qapF5lrO1U82Tp93rpXlmctj6fyFHBVVB5aXnUHJ7LZeVPod7Wnfl8p5OyhlHQHC8BnzdzCqCMKmWZNX5DtETDId0qzU7dPzh0LP0idt5buU7L9QNaabChw3nnaL47iu_1Di5Wp264p2TwACeedv2hfRDjDlJmaQXuS8Rtv9GnRWyC9JBu7XmGvGDziumnJH7Hyzh3VNu-kSPQD3vuAFgMZS6uUzOztCkT0fpOalZI6hqxtWLvXUMj-crXrn-Maavz8qRhpAyp5kcYk3jiHGgQIi7QSK2JIdRJ8APyX9HlmTN5AQ",
  "e": "AQAB"
}`
const JWK1Digest = `ul04Iq07ulKnnrebv2hv3yxCGgVvoHs8hjq2tVKx3mc=`
const JWK2JSON = `{
  "kty":"RSA",
  "n":"yTsLkI8n4lg9UuSKNRC0UPHsVjNdCYk8rGXIqeb_rRYaEev3D9-kxXY8HrYfGkVt5CiIVJ-n2t50BKT8oBEMuilmypSQqJw0pCgtUm-e6Z0Eg3Ly6DMXFlycyikegiZ0b-rVX7i5OCEZRDkENAYwFNX4G7NNCwEZcH7HUMUmty9dchAqDS9YWzPh_dde1A9oy9JMH07nRGDcOzIh1rCPwc71nwfPPYeeS4tTvkjanjeigOYBFkBLQuv7iBB4LPozsGF1XdoKiIIi-8ye44McdhOTPDcQp3xKxj89aO02pQhBECv61rmbPinvjMG9DYxJmZvjsKF4bN2oy0DxdC1jDw",
  "e":"AQAB"
}`

func TestKeyDigest(t *testing.T) {
	// Test with JWK (value, reference, and direct)
	var jwk jose.JSONWebKey
	err := json.Unmarshal([]byte(JWK1JSON), &jwk)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := KeyDigestB64(jwk)
	test.Assert(t, err == nil && digest == JWK1Digest, "Failed to digest JWK by value")
	digest, err = KeyDigestB64(&jwk)
	test.Assert(t, err == nil && digest == JWK1Digest, "Failed to digest JWK by reference")
	digest, err = KeyDigestB64(jwk.Key)
	test.Assert(t, err == nil && digest == JWK1Digest, "Failed to digest bare key")

	// Test with unknown key type
	_, err = KeyDigestB64(struct{}{})
	test.Assert(t, err != nil, "Should have rejected unknown key type")
}

func TestKeyDigestEquals(t *testing.T) {
	var jwk1, jwk2 jose.JSONWebKey
	err := json.Unmarshal([]byte(JWK1JSON), &jwk1)
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal([]byte(JWK2JSON), &jwk2)
	if err != nil {
		t.Fatal(err)
	}

	test.Assert(t, KeyDigestEquals(jwk1, jwk1), "Key digests for same key should match")
	test.Assert(t, !KeyDigestEquals(jwk1, jwk2), "Key digests for different keys should not match")
	test.Assert(t, !KeyDigestEquals(jwk1, struct{}{}), "Unknown key types should not match anything")
	test.Assert(t, !KeyDigestEquals(struct{}{}, struct{}{}), "Unknown key types should not match anything")
}

func TestIsAnyNilOrZero(t *testing.T) {
	test.Assert(t, IsAnyNilOrZero(nil), "Nil seen as non-zero")

	test.Assert(t, IsAnyNilOrZero(false), "False bool seen as non-zero")
	test.Assert(t, !IsAnyNilOrZero(true), "True bool seen as zero")

	test.Assert(t, IsAnyNilOrZero(0), "Zero num seen as non-zero")
	test.Assert(t, !IsAnyNilOrZero(uint32(5)), "Non-zero num seen as zero")
	test.Assert(t, !IsAnyNilOrZero(-12.345), "Non-zero num seen as zero")

	test.Assert(t, IsAnyNilOrZero(""), "Empty string seen as non-zero")
	test.Assert(t, !IsAnyNilOrZero("string"), "Non-empty string seen as zero")

	test.Assert(t, IsAnyNilOrZero([]byte{}), "Empty byte slice seen as non-zero")
	test.Assert(t, !IsAnyNilOrZero([]byte("byte")), "Non-empty byte slice seen as zero")

	type Foo struct {
		foo int
	}
	test.Assert(t, IsAnyNilOrZero(Foo{}), "Empty struct seen as non-zero")
	test.Assert(t, !IsAnyNilOrZero(Foo{5}), "Non-empty struct seen as zero")
	var f *Foo
	test.Assert(t, IsAnyNilOrZero(f), "Pointer to uninitialized struct seen as non-zero")

	test.Assert(t, IsAnyNilOrZero(1, ""), "Mixed values seen as non-zero")
	test.Assert(t, IsAnyNilOrZero("", 1), "Mixed values seen as non-zero")
}

func TestUniqueLowerNames(t *testing.T) {
	u := UniqueLowerNames([]string{"foobar.com", "fooBAR.com", "baz.com", "foobar.com", "bar.com", "bar.com", "a.com"})
	sort.Strings(u)
	test.AssertDeepEquals(t, []string{"a.com", "bar.com", "baz.com", "foobar.com"}, u)
}

func TestValidSerial(t *testing.T) {
	notLength32Or36 := "A"
	length32 := strings.Repeat("A", 32)
	length36 := strings.Repeat("A", 36)
	isValidSerial := ValidSerial(notLength32Or36)
	test.AssertEquals(t, isValidSerial, false)
	isValidSerial = ValidSerial(length32)
	test.AssertEquals(t, isValidSerial, true)
	isValidSerial = ValidSerial(length36)
	test.AssertEquals(t, isValidSerial, true)
}

func TestLoadCert(t *testing.T) {
	var osPathErr *os.PathError
	_, err := LoadCert("")
	test.AssertError(t, err, "Loading empty path did not error")
	test.AssertErrorWraps(t, err, &osPathErr)

	_, err = LoadCert("totally/fake/path")
	test.AssertError(t, err, "Loading nonexistent path did not error")
	test.AssertErrorWraps(t, err, &osPathErr)

	_, err = LoadCert("../test/test-ca.der")
	test.AssertError(t, err, "Loading non-PEM file did not error")
	test.AssertEquals(t, err.Error(), "No data in cert PEM file ../test/test-ca.der")

	_, err = LoadCert("../test/test-ca.key")
	test.AssertError(t, err, "Loading non-cert file did not error")
	test.AssertEquals(t, err.Error(), "x509: malformed tbs certificate")

	cert, err := LoadCert("../test/test-ca.pem")
	test.AssertNotError(t, err, "Failed to load cert file")
	test.AssertEquals(t, cert.Subject.CommonName, "happy hacker fake CA")
}

func TestRetryBackoff(t *testing.T) {
	assertBetween := func(a, b, c float64) {
		t.Helper()
		if a < b || a > c {
			t.Fatalf("%f is not between %f and %f", a, b, c)
		}
	}

	factor := 1.5
	base := time.Minute
	max := 10 * time.Minute

	expected := base
	backoff := RetryBackoff(1, base, max, factor)
	assertBetween(float64(backoff), float64(expected)*0.8, float64(expected)*1.2)
	expected = time.Second * 90
	backoff = RetryBackoff(2, base, max, factor)
	assertBetween(float64(backoff), float64(expected)*0.8, float64(expected)*1.2)
	expected = time.Minute * 10
	// should be truncated
	backoff = RetryBackoff(7, base, max, factor)
	assertBetween(float64(backoff), float64(expected)*0.8, float64(expected)*1.2)

}
