package wfe2

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jmhodges/clock"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/ocsp"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	jose "gopkg.in/go-jose/go-jose.v2"

	capb "github.com/letsencrypt/boulder/ca/proto"
	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	berrors "github.com/letsencrypt/boulder/errors"
	"github.com/letsencrypt/boulder/features"
	"github.com/letsencrypt/boulder/goodkey"
	"github.com/letsencrypt/boulder/identifier"
	"github.com/letsencrypt/boulder/issuance"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/mocks"
	"github.com/letsencrypt/boulder/nonce"
	noncepb "github.com/letsencrypt/boulder/nonce/proto"
	"github.com/letsencrypt/boulder/probs"
	rapb "github.com/letsencrypt/boulder/ra/proto"
	"github.com/letsencrypt/boulder/revocation"
	sapb "github.com/letsencrypt/boulder/sa/proto"
	"github.com/letsencrypt/boulder/test"
	inmemnonce "github.com/letsencrypt/boulder/test/inmem/nonce"
	"github.com/letsencrypt/boulder/web"
)

const (
	agreementURL = "http://example.invalid/terms"

	test1KeyPublicJSON = `
	{
		"kty":"RSA",
		"n":"yNWVhtYEKJR21y9xsHV-PD_bYwbXSeNuFal46xYxVfRL5mqha7vttvjB_vc7Xg2RvgCxHPCqoxgMPTzHrZT75LjCwIW2K_klBYN8oYvTwwmeSkAz6ut7ZxPv-nZaT5TJhGk0NT2kh_zSpdriEJ_3vW-mqxYbbBmpvHqsa1_zx9fSuHYctAZJWzxzUZXykbWMWQZpEiE0J4ajj51fInEzVn7VxV-mzfMyboQjujPh7aNJxAWSq4oQEJJDgWwSh9leyoJoPpONHxh5nEE5AjE01FkGICSxjpZsF-w8hOTI3XXohUdu29Se26k2B0PolDSuj0GIQU6-W9TdLXSjBb2SpQ",
		"e":"AQAB"
	}`

	test1KeyPrivatePEM = `
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAyNWVhtYEKJR21y9xsHV+PD/bYwbXSeNuFal46xYxVfRL5mqh
a7vttvjB/vc7Xg2RvgCxHPCqoxgMPTzHrZT75LjCwIW2K/klBYN8oYvTwwmeSkAz
6ut7ZxPv+nZaT5TJhGk0NT2kh/zSpdriEJ/3vW+mqxYbbBmpvHqsa1/zx9fSuHYc
tAZJWzxzUZXykbWMWQZpEiE0J4ajj51fInEzVn7VxV+mzfMyboQjujPh7aNJxAWS
q4oQEJJDgWwSh9leyoJoPpONHxh5nEE5AjE01FkGICSxjpZsF+w8hOTI3XXohUdu
29Se26k2B0PolDSuj0GIQU6+W9TdLXSjBb2SpQIDAQABAoIBAHw58SXYV/Yp72Cn
jjFSW+U0sqWMY7rmnP91NsBjl9zNIe3C41pagm39bTIjB2vkBNR8ZRG7pDEB/QAc
Cn9Keo094+lmTArjL407ien7Ld+koW7YS8TyKADYikZo0vAK3qOy14JfQNiFAF9r
Bw61hG5/E58cK5YwQZe+YcyBK6/erM8fLrJEyw4CV49wWdq/QqmNYU1dx4OExAkl
KMfvYXpjzpvyyTnZuS4RONfHsO8+JTyJVm+lUv2x+bTce6R4W++UhQY38HakJ0x3
XRfXooRv1Bletu5OFlpXfTSGz/5gqsfemLSr5UHncsCcFMgoFBsk2t/5BVukBgC7
PnHrAjkCgYEA887PRr7zu3OnaXKxylW5U5t4LzdMQLpslVW7cLPD4Y08Rye6fF5s
O/jK1DNFXIoUB7iS30qR7HtaOnveW6H8/kTmMv/YAhLO7PAbRPCKxxcKtniEmP1x
ADH0tF2g5uHB/zeZhCo9qJiF0QaJynvSyvSyJFmY6lLvYZsAW+C+PesCgYEA0uCi
Q8rXLzLpfH2NKlLwlJTi5JjE+xjbabgja0YySwsKzSlmvYJqdnE2Xk+FHj7TCnSK
KUzQKR7+rEk5flwEAf+aCCNh3W4+Hp9MmrdAcCn8ZsKmEW/o7oDzwiAkRCmLw/ck
RSFJZpvFoxEg15riT37EjOJ4LBZ6SwedsoGA/a8CgYEA2Ve4sdGSR73/NOKZGc23
q4/B4R2DrYRDPhEySnMGoPCeFrSU6z/lbsUIU4jtQWSaHJPu4n2AfncsZUx9WeSb
OzTCnh4zOw33R4N4W8mvfXHODAJ9+kCc1tax1YRN5uTEYzb2dLqPQtfNGxygA1DF
BkaC9CKnTeTnH3TlKgK8tUcCgYB7J1lcgh+9ntwhKinBKAL8ox8HJfkUM+YgDbwR
sEM69E3wl1c7IekPFvsLhSFXEpWpq3nsuMFw4nsVHwaGtzJYAHByhEdpTDLXK21P
heoKF1sioFbgJB1C/Ohe3OqRLDpFzhXOkawOUrbPjvdBM2Erz/r11GUeSlpNazs7
vsoYXQKBgFwFM1IHmqOf8a2wEFa/a++2y/WT7ZG9nNw1W36S3P04K4lGRNRS2Y/S
snYiqxD9nL7pVqQP2Qbqbn0yD6d3G5/7r86F7Wu2pihM8g6oyMZ3qZvvRIBvKfWo
eROL1ve1vmQF3kjrMPhhK2kr6qdWnTE5XlPllVSZFQenSTzj98AO
-----END RSA PRIVATE KEY-----
`

	test2KeyPublicJSON = `{
		"kty":"RSA",
		"n":"qnARLrT7Xz4gRcKyLdydmCr-ey9OuPImX4X40thk3on26FkMznR3fRjs66eLK7mmPcBZ6uOJseURU6wAaZNmemoYx1dMvqvWWIyiQleHSD7Q8vBrhR6uIoO4jAzJZR-ChzZuSDt7iHN-3xUVspu5XGwXU_MVJZshTwp4TaFx5elHIT_ObnTvTOU3Xhish07AbgZKmWsVbXh5s-CrIicU4OexJPgunWZ_YJJueOKmTvnLlTV4MzKR2oZlBKZ27S0-SfdV_QDx_ydle5oMAyKVtlAV35cyPMIsYNwgUGBCdY_2Uzi5eX0lTc7MPRwz6qR1kip-i59VcGcUQgqHV6Fyqw",
		"e":"AQAB"
	}`

	test2KeyPrivatePEM = `
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAqnARLrT7Xz4gRcKyLdydmCr+ey9OuPImX4X40thk3on26FkM
znR3fRjs66eLK7mmPcBZ6uOJseURU6wAaZNmemoYx1dMvqvWWIyiQleHSD7Q8vBr
hR6uIoO4jAzJZR+ChzZuSDt7iHN+3xUVspu5XGwXU/MVJZshTwp4TaFx5elHIT/O
bnTvTOU3Xhish07AbgZKmWsVbXh5s+CrIicU4OexJPgunWZ/YJJueOKmTvnLlTV4
MzKR2oZlBKZ27S0+SfdV/QDx/ydle5oMAyKVtlAV35cyPMIsYNwgUGBCdY/2Uzi5
eX0lTc7MPRwz6qR1kip+i59VcGcUQgqHV6FyqwIDAQABAoIBAG5m8Xpj2YC0aYtG
tsxmX9812mpJFqFOmfS+f5N0gMJ2c+3F4TnKz6vE/ZMYkFnehAT0GErC4WrOiw68
F/hLdtJM74gQ0LGh9dKeJmz67bKqngcAHWW5nerVkDGIBtzuMEsNwxofDcIxrjkr
G0b7AHMRwXqrt0MI3eapTYxby7+08Yxm40mxpSsW87FSaI61LDxUDpeVkn7kolSN
WifVat7CpZb/D2BfGAQDxiU79YzgztpKhbynPdGc/OyyU+CNgk9S5MgUX2m9Elh3
aXrWh2bT2xzF+3KgZdNkJQcdIYVoGq/YRBxlGXPYcG4Do3xKhBmH79Io2BizevZv
nHkbUGECgYEAydjb4rl7wYrElDqAYpoVwKDCZAgC6o3AKSGXfPX1Jd2CXgGR5Hkl
ywP0jdSLbn2v/jgKQSAdRbYuEiP7VdroMb5M6BkBhSY619cH8etoRoLzFo1GxcE8
Y7B598VXMq8TT+TQqw/XRvM18aL3YDZ3LSsR7Gl2jF/sl6VwQAaZToUCgYEA2Cn4
fG58ME+M4IzlZLgAIJ83PlLb9ip6MeHEhUq2Dd0In89nss7Acu0IVg8ES88glJZy
4SjDLGSiuQuoQVo9UBq/E5YghdMJFp5ovwVfEaJ+ruWqOeujvWzzzPVyIWSLXRQa
N4kedtfrlqldMIXywxVru66Q1NOGvhDHm/Q8+28CgYEAkhLCbn3VNed7A9qidrkT
7OdqRoIVujEDU8DfpKtK0jBP3EA+mJ2j4Bvoq4uZrEiBSPS9VwwqovyIstAfX66g
Qv95IK6YDwfvpawUL9sxB3ZU/YkYIp0JWwun+Mtzo1ZYH4V0DZfVL59q9of9hj9k
V+fHfNOF22jAC67KYUtlPxECgYEAwF6hj4L3rDqvQYrB/p8tJdrrW+B7dhgZRNkJ
fiGd4LqLGUWHoH4UkHJXT9bvWNPMx88YDz6qapBoq8svAnHfTLFwyGp7KP1FAkcZ
Kp4KG/SDTvx+QCtvPX1/fjAUUJlc2QmxxyiU3uiK9Tpl/2/FOk2O4aiZpX1VVUIz
kZuKxasCgYBiVRkEBk2W4Ia0B7dDkr2VBrz4m23Y7B9cQLpNAapiijz/0uHrrCl8
TkLlEeVOuQfxTadw05gzKX0jKkMC4igGxvEeilYc6NR6a4nvRulG84Q8VV9Sy9Ie
wk6Oiadty3eQqSBJv0HnpmiEdQVffIK5Pg4M8Dd+aOBnEkbopAJOuA==
-----END RSA PRIVATE KEY-----
`
	test3KeyPrivatePEM = `
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAuTQER6vUA1RDixS8xsfCRiKUNGRzzyIK0MhbS2biClShbb0h
Sx2mPP7gBvis2lizZ9r+y9hL57kNQoYCKndOBg0FYsHzrQ3O9AcoV1z2Mq+XhHZb
FrVYaXI0M3oY9BJCWog0dyi3XC0x8AxC1npd1U61cToHx+3uSvgZOuQA5ffEn5L3
8Dz1Ti7OV3E4XahnRJvejadUmTkki7phLBUXm5MnnyFm0CPpf6ApV7zhLjN5W+nV
0WL17o7v8aDgV/t9nIdi1Y26c3PlCEtiVHZcebDH5F1Deta3oLLg9+g6rWnTqPbY
3knffhp4m0scLD6e33k8MtzxDX/D7vHsg0/X1wIDAQABAoIBAQCnFJpX3lhiuH5G
1uqHmmdVxpRVv9oKn/eJ63cRSzvZfgg0bE/A6Hq0xGtvXqDySttvck4zsGqqHnQr
86G4lfE53D1jnv4qvS5bUKnARwmFKIxU4EHE9s1QM8uMNTaV2nMqIX7TkVP6QHuw
yB70R2inq15dS7EBWVGFKNX6HwAAdj8pFuF6o2vIwmAfee20aFzpWWf81jOH9Ai6
hyJyV3NqrU1JzIwlXaeX67R1VroFdhN/lapp+2b0ZEcJJtFlcYFl99NjkQeVZyik
izNv0GZZNWizc57wU0/8cv+jQ2f26ltvyrPz3QNK61bFfzy+/tfMvLq7sdCmztKJ
tMxCBJOBAoGBAPKnIVQIS2nTvC/qZ8ajw1FP1rkvYblIiixegjgfFhM32HehQ+nu
3TELi3I3LngLYi9o6YSqtNBmdBJB+DUAzIXp0TdOihOweGiv5dAEWwY9rjCzMT5S
GP7dCWiJwoMUHrOs1Po3dwcjj/YsoAW+FC0jSvach2Ln2CvPgr5FP0ARAoGBAMNj
64qUCzgeXiSyPKK69bCCGtHlTYUndwHQAZmABjbmxAXZNYgp/kBezFpKOwmICE8R
kK8YALRrL0VWXl/yj85b0HAZGkquNFHPUDd1e6iiP5TrY+Hy4oqtlYApjH6f85CE
lWjQ1iyUL7aT6fcSgzq65ZWD2hUzvNtWbTt6zQFnAoGAWS/EuDY0QblpOdNWQVR/
vasyqO4ZZRiccKJsCmSioH2uOoozhBAfjJ9JqblOgyDr/bD546E6xD5j+zH0IMci
ZTYDh+h+J659Ez1Topl3O1wAYjX6q4VRWpuzkZDQxYznm/KydSVdwmn3x+uvBW1P
zSdjrjDqMhg1BCVJUNXy4YECgYEAjX1z+dwO68qB3gz7/9NnSzRL+6cTJdNYSIW6
QtAEsAkX9iw+qaXPKgn77X5HljVd3vQXU9QL3pqnloxetxhNrt+p5yMmeOIBnSSF
MEPxEkK7zDlRETPzfP0Kf86WoLNviz2XfFmOXqXIj2w5RuOvB/6DdmwOpr/aiPLj
EulwPw0CgYAMSzsWOt6vU+y/G5NyhUCHvY50TdnGOj2btBk9rYVwWGWxCpg2QF0R
pcKXgGzXEVZKFAqB8V1c/mmCo8ojPgmqGM+GzX2Bj4seVBW7PsTeZUjrHpADshjV
F7o5b7y92NlxO5kwQzRKEAhwS5PbKJdx90iCuG+JlI1YgWlA1VcJMw==
-----END RSA PRIVATE KEY-----
`

	testE1KeyPrivatePEM = `
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIH+p32RUnqT/iICBEGKrLIWFcyButv0S0lU/BLPOyHn2oAoGCCqGSM49
AwEHoUQDQgAEFwvSZpu06i3frSk/mz9HcD9nETn4wf3mQ+zDtG21GapLytH7R1Zr
ycBzDV9u6cX9qNLc9Bn5DAumz7Zp2AuA+Q==
-----END EC PRIVATE KEY-----
`

	testE2KeyPrivatePEM = `
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIFRcPxQ989AY6se2RyIoF1ll9O6gHev4oY15SWJ+Jf5eoAoGCCqGSM49
AwEHoUQDQgAES8FOmrZ3ywj4yyFqt0etAD90U+EnkNaOBSLfQmf7pNi8y+kPKoUN
EeMZ9nWyIM6bktLrE11HnFOnKhAYsM5fZA==
-----END EC PRIVATE KEY-----`
)

type MockRegistrationAuthority struct {
	lastRevocationReason revocation.Reason
}

func (ra *MockRegistrationAuthority) NewRegistration(ctx context.Context, in *corepb.Registration, _ ...grpc.CallOption) (*corepb.Registration, error) {
	in.Id = 1
	in.CreatedAt = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()
	return in, nil
}

func (ra *MockRegistrationAuthority) UpdateRegistration(ctx context.Context, in *rapb.UpdateRegistrationRequest, _ ...grpc.CallOption) (*corepb.Registration, error) {
	if !bytes.Equal(in.Base.Key, in.Update.Key) {
		in.Base.Key = in.Update.Key
	}
	return in.Base, nil
}

func (ra *MockRegistrationAuthority) PerformValidation(context.Context, *rapb.PerformValidationRequest, ...grpc.CallOption) (*corepb.Authorization, error) {
	return &corepb.Authorization{}, nil
}

func (ra *MockRegistrationAuthority) RevokeCertByApplicant(ctx context.Context, in *rapb.RevokeCertByApplicantRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	ra.lastRevocationReason = revocation.Reason(in.Code)
	return &emptypb.Empty{}, nil
}

func (ra *MockRegistrationAuthority) RevokeCertByKey(ctx context.Context, in *rapb.RevokeCertByKeyRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	ra.lastRevocationReason = revocation.Reason(ocsp.KeyCompromise)
	return &emptypb.Empty{}, nil
}

func (ra *MockRegistrationAuthority) GenerateOCSP(ctx context.Context, req *rapb.GenerateOCSPRequest, _ ...grpc.CallOption) (*capb.OCSPResponse, error) {
	return nil, nil
}

func (ra *MockRegistrationAuthority) AdministrativelyRevokeCertificate(context.Context, *rapb.AdministrativelyRevokeCertificateRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (ra *MockRegistrationAuthority) OnValidationUpdate(context.Context, core.Authorization, ...grpc.CallOption) error {
	return nil
}

func (ra *MockRegistrationAuthority) DeactivateAuthorization(context.Context, *corepb.Authorization, ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (ra *MockRegistrationAuthority) DeactivateRegistration(context.Context, *corepb.Registration, ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (ra *MockRegistrationAuthority) NewOrder(ctx context.Context, in *rapb.NewOrderRequest, _ ...grpc.CallOption) (*corepb.Order, error) {
	return &corepb.Order{
		Id:               1,
		RegistrationID:   in.RegistrationID,
		Created:          time.Date(2021, 1, 1, 1, 1, 1, 0, time.UTC).UnixNano(),
		Expires:          time.Date(2021, 2, 1, 1, 1, 1, 0, time.UTC).UnixNano(),
		Names:            in.Names,
		Status:           string(core.StatusPending),
		V2Authorizations: []int64{1},
	}, nil
}

func (ra *MockRegistrationAuthority) FinalizeOrder(ctx context.Context, in *rapb.FinalizeOrderRequest, _ ...grpc.CallOption) (*corepb.Order, error) {
	in.Order.Status = string(core.StatusProcessing)
	return in.Order, nil
}

func makeBody(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

// loadKey loads a private key from PEM/DER-encoded data and returns
// a `crypto.Signer`.
func loadKey(t *testing.T, keyBytes []byte) crypto.Signer {
	// pem.Decode does not return an error as its 2nd arg, but instead the "rest"
	// that was leftover from parsing the PEM block. We only care if the decoded
	// PEM block was empty for this test function.
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		t.Fatal("Unable to decode private key PEM bytes")
	}

	// Try decoding as an RSA private key
	if rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return rsaKey
	}

	// Try decoding as a PKCS8 private key
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		// Determine the key's true type and return it as a crypto.Signer
		switch k := key.(type) {
		case *rsa.PrivateKey:
			return k
		case *ecdsa.PrivateKey:
			return k
		}
	}

	// Try as an ECDSA private key
	if ecdsaKey, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return ecdsaKey
	}

	// Nothing worked! Fail hard.
	t.Fatalf("Unable to decode private key PEM bytes")
	// NOOP - the t.Fatal() call will abort before this return
	return nil
}

var testKeyPolicy = goodkey.KeyPolicy{
	AllowRSA:           true,
	AllowECDSANISTP256: true,
	AllowECDSANISTP384: true,
}

var ctx = context.Background()

func setupWFE(t *testing.T) (WebFrontEndImpl, clock.FakeClock, requestSigner) {
	features.Reset()

	fc := clock.NewFake()
	stats := metrics.NoopRegisterer

	certChains := map[issuance.IssuerNameID][][]byte{}
	issuerCertificates := map[issuance.IssuerNameID]*issuance.Certificate{}
	for _, files := range [][]string{
		{
			"../test/hierarchy/int-r3.cert.pem",
			"../test/hierarchy/root-x1.cert.pem",
		},
		{
			"../test/hierarchy/int-r3-cross.cert.pem",
			"../test/hierarchy/root-dst.cert.pem",
		},
		{
			"../test/hierarchy/int-e1.cert.pem",
			"../test/hierarchy/root-x2.cert.pem",
		},
		{
			"../test/hierarchy/int-e1.cert.pem",
			"../test/hierarchy/root-x2-cross.cert.pem",
			"../test/hierarchy/root-x1-cross.cert.pem",
			"../test/hierarchy/root-dst.cert.pem",
		},
	} {
		certs, err := issuance.LoadChain(files)
		test.AssertNotError(t, err, "Unable to load chain")
		var buf bytes.Buffer
		for _, cert := range certs {
			buf.Write([]byte("\n"))
			buf.Write(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
		}
		id := certs[0].NameID()
		certChains[id] = append(certChains[id], buf.Bytes())
		issuerCertificates[id] = certs[0]
	}

	mockSA := mocks.NewStorageAuthorityReadOnly(fc)
	noncePrefix := "mlem"
	nonceService, err := nonce.NewNonceService(metrics.NoopRegisterer, 100, noncePrefix)
	test.AssertNotError(t, err, "making nonceService")
	nonceGRPCService := &inmemnonce.Service{
		NonceService: nonceService,
	}

	wfe, err := NewWebFrontEndImpl(
		stats,
		fc,
		testKeyPolicy,
		certChains,
		issuerCertificates,
		blog.NewMock(),
		10*time.Second,
		10*time.Second,
		30*24*time.Hour,
		7*24*time.Hour,
		&MockRegistrationAuthority{},
		mockSA,
		nonceGRPCService,
		map[string]nonce.Redeemer{noncePrefix: nonceGRPCService},
		nonceGRPCService,
		"",
		mockSA)
	test.AssertNotError(t, err, "Unable to create WFE")

	wfe.SubscriberAgreementURL = agreementURL

	return wfe, fc, requestSigner{t, nonceGRPCService.AsSource()}
}

// makePostRequestWithPath creates an http.Request for localhost with method
// POST, the provided body, and the correct Content-Length. The path provided
// will be parsed as a URL and used to populate the request URL and RequestURI
func makePostRequestWithPath(path string, body string) *http.Request {
	request := &http.Request{
		Method:     "POST",
		RemoteAddr: "1.1.1.1:7882",
		Header: map[string][]string{
			"Content-Length": {strconv.Itoa(len(body))},
			"Content-Type":   {expectedJWSContentType},
		},
		Body: makeBody(body),
		Host: "localhost",
	}
	url := mustParseURL(path)
	request.URL = url
	request.RequestURI = url.Path
	return request
}

// signAndPost constructs a JWS signed by the account with ID 1, over the given
// payload, with the protected URL set to the provided signedURL. An HTTP
// request constructed to the provided path with the encoded JWS body as the
// POST body is returned.
func signAndPost(signer requestSigner, path, signedURL, payload string) *http.Request {
	_, _, body := signer.byKeyID(1, nil, signedURL, payload)
	return makePostRequestWithPath(path, body)
}

func mustParseURL(s string) *url.URL {
	if u, err := url.Parse(s); err != nil {
		panic("Cannot parse URL " + s)
	} else {
		return u
	}
}

func sortHeader(s string) string {
	a := strings.Split(s, ", ")
	sort.Strings(a)
	return strings.Join(a, ", ")
}

func addHeadIfGet(s []string) []string {
	for _, a := range s {
		if a == "GET" {
			return append(s, "HEAD")
		}
	}
	return s
}

func TestHandleFunc(t *testing.T) {
	wfe, _, _ := setupWFE(t)
	var mux *http.ServeMux
	var rw *httptest.ResponseRecorder
	var stubCalled bool
	runWrappedHandler := func(req *http.Request, pattern string, allowed ...string) {
		mux = http.NewServeMux()
		rw = httptest.NewRecorder()
		stubCalled = false
		wfe.HandleFunc(mux, pattern, func(context.Context, *web.RequestEvent, http.ResponseWriter, *http.Request) {
			stubCalled = true
		}, allowed...)
		req.URL = mustParseURL(pattern)
		mux.ServeHTTP(rw, req)
	}

	// Plain requests (no CORS)
	type testCase struct {
		allowed        []string
		reqMethod      string
		shouldCallStub bool
		shouldSucceed  bool
		pattern        string
	}
	var lastNonce string
	for _, c := range []testCase{
		{[]string{"GET", "POST"}, "GET", true, true, "/test"},
		{[]string{"GET", "POST"}, "GET", true, true, newNoncePath},
		{[]string{"GET", "POST"}, "POST", true, true, "/test"},
		{[]string{"GET"}, "", false, false, "/test"},
		{[]string{"GET"}, "POST", false, false, "/test"},
		{[]string{"GET"}, "OPTIONS", false, true, "/test"},
		{[]string{"GET"}, "MAKE-COFFEE", false, false, "/test"}, // 405, or 418?
		{[]string{"GET"}, "GET", true, true, directoryPath},
	} {
		runWrappedHandler(&http.Request{Method: c.reqMethod}, c.pattern, c.allowed...)
		test.AssertEquals(t, stubCalled, c.shouldCallStub)
		if c.shouldSucceed {
			test.AssertEquals(t, rw.Code, http.StatusOK)
		} else {
			test.AssertEquals(t, rw.Code, http.StatusMethodNotAllowed)
			test.AssertEquals(t, sortHeader(rw.Header().Get("Allow")), sortHeader(strings.Join(addHeadIfGet(c.allowed), ", ")))
			test.AssertUnmarshaledEquals(t,
				rw.Body.String(),
				`{"type":"`+probs.V2ErrorNS+`malformed","detail":"Method not allowed","status":405}`)
		}
		if c.reqMethod == "GET" && c.pattern != newNoncePath {
			nonce := rw.Header().Get("Replay-Nonce")
			test.AssertEquals(t, nonce, "")
		} else {
			nonce := rw.Header().Get("Replay-Nonce")
			test.AssertNotEquals(t, nonce, lastNonce)
			test.AssertNotEquals(t, nonce, "")
			lastNonce = nonce
		}
		linkHeader := rw.Header().Get("Link")
		if c.pattern != directoryPath {
			// If the pattern wasn't the directory there should be a Link header for the index
			test.AssertEquals(t, linkHeader, `<http://localhost/directory>;rel="index"`)
		} else {
			// The directory resource shouldn't get a link header
			test.AssertEquals(t, linkHeader, "")
		}
	}

	// Disallowed method returns error JSON in body
	runWrappedHandler(&http.Request{Method: "PUT"}, "/test", "GET", "POST")
	test.AssertEquals(t, rw.Header().Get("Content-Type"), "application/problem+json")
	test.AssertUnmarshaledEquals(t, rw.Body.String(), `{"type":"`+probs.V2ErrorNS+`malformed","detail":"Method not allowed","status":405}`)
	test.AssertEquals(t, sortHeader(rw.Header().Get("Allow")), "GET, HEAD, POST")

	// Disallowed method special case: response to HEAD has got no body
	runWrappedHandler(&http.Request{Method: "HEAD"}, "/test", "GET", "POST")
	test.AssertEquals(t, stubCalled, true)
	test.AssertEquals(t, rw.Body.String(), "")

	// HEAD doesn't work with POST-only endpoints
	runWrappedHandler(&http.Request{Method: "HEAD"}, "/test", "POST")
	test.AssertEquals(t, stubCalled, false)
	test.AssertEquals(t, rw.Code, http.StatusMethodNotAllowed)
	test.AssertEquals(t, rw.Header().Get("Content-Type"), "application/problem+json")
	test.AssertEquals(t, rw.Header().Get("Allow"), "POST")
	test.AssertUnmarshaledEquals(t, rw.Body.String(), `{"type":"`+probs.V2ErrorNS+`malformed","detail":"Method not allowed","status":405}`)

	wfe.AllowOrigins = []string{"*"}
	testOrigin := "https://example.com"

	// CORS "actual" request for disallowed method
	runWrappedHandler(&http.Request{
		Method: "POST",
		Header: map[string][]string{
			"Origin": {testOrigin},
		},
	}, "/test", "GET")
	test.AssertEquals(t, stubCalled, false)
	test.AssertEquals(t, rw.Code, http.StatusMethodNotAllowed)

	// CORS "actual" request for allowed method
	runWrappedHandler(&http.Request{
		Method: "GET",
		Header: map[string][]string{
			"Origin": {testOrigin},
		},
	}, "/test", "GET", "POST")
	test.AssertEquals(t, stubCalled, true)
	test.AssertEquals(t, rw.Code, http.StatusOK)
	test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Methods"), "")
	test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Origin"), "*")
	test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	test.AssertEquals(t, sortHeader(rw.Header().Get("Access-Control-Expose-Headers")), "Link, Location, Replay-Nonce")

	// CORS preflight request for disallowed method
	runWrappedHandler(&http.Request{
		Method: "OPTIONS",
		Header: map[string][]string{
			"Origin":                        {testOrigin},
			"Access-Control-Request-Method": {"POST"},
		},
	}, "/test", "GET")
	test.AssertEquals(t, stubCalled, false)
	test.AssertEquals(t, rw.Code, http.StatusOK)
	test.AssertEquals(t, rw.Header().Get("Allow"), "GET, HEAD")
	test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Origin"), "")
	test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Headers"), "")

	// CORS preflight request for allowed method
	runWrappedHandler(&http.Request{
		Method: "OPTIONS",
		Header: map[string][]string{
			"Origin":                         {testOrigin},
			"Access-Control-Request-Method":  {"POST"},
			"Access-Control-Request-Headers": {"X-Accept-Header1, X-Accept-Header2", "X-Accept-Header3"},
		},
	}, "/test", "GET", "POST")
	test.AssertEquals(t, rw.Code, http.StatusOK)
	test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Origin"), "*")
	test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	test.AssertEquals(t, rw.Header().Get("Access-Control-Max-Age"), "86400")
	test.AssertEquals(t, sortHeader(rw.Header().Get("Access-Control-Allow-Methods")), "GET, HEAD, POST")
	test.AssertEquals(t, sortHeader(rw.Header().Get("Access-Control-Expose-Headers")), "Link, Location, Replay-Nonce")

	// OPTIONS request without an Origin header (i.e., not a CORS
	// preflight request)
	runWrappedHandler(&http.Request{
		Method: "OPTIONS",
		Header: map[string][]string{
			"Access-Control-Request-Method": {"POST"},
		},
	}, "/test", "GET", "POST")
	test.AssertEquals(t, rw.Code, http.StatusOK)
	test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Origin"), "")
	test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Headers"), "")
	test.AssertEquals(t, sortHeader(rw.Header().Get("Allow")), "GET, HEAD, POST")

	// CORS preflight request missing optional Request-Method
	// header. The "actual" request will be GET.
	for _, allowedMethod := range []string{"GET", "POST"} {
		runWrappedHandler(&http.Request{
			Method: "OPTIONS",
			Header: map[string][]string{
				"Origin": {testOrigin},
			},
		}, "/test", allowedMethod)
		test.AssertEquals(t, rw.Code, http.StatusOK)
		if allowedMethod == "GET" {
			test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Origin"), "*")
			test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
			test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Methods"), "GET, HEAD")
		} else {
			test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Origin"), "")
			test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Headers"), "")
		}
	}

	// No CORS headers are given when configuration does not list
	// "*" or the client-provided origin.
	for _, wfe.AllowOrigins = range [][]string{
		{},
		{"http://example.com", "https://other.example"},
		{""}, // Invalid origin is never matched
	} {
		runWrappedHandler(&http.Request{
			Method: "OPTIONS",
			Header: map[string][]string{
				"Origin":                        {testOrigin},
				"Access-Control-Request-Method": {"POST"},
			},
		}, "/test", "POST")
		test.AssertEquals(t, rw.Code, http.StatusOK)
		for _, h := range []string{
			"Access-Control-Allow-Methods",
			"Access-Control-Allow-Origin",
			"Access-Control-Allow-Headers",
			"Access-Control-Expose-Headers",
			"Access-Control-Request-Headers",
		} {
			test.AssertEquals(t, rw.Header().Get(h), "")
		}
	}

	// CORS headers are offered when configuration lists "*" or
	// the client-provided origin.
	for _, wfe.AllowOrigins = range [][]string{
		{testOrigin, "http://example.org", "*"},
		{"", "http://example.org", testOrigin}, // Invalid origin is harmless
	} {
		runWrappedHandler(&http.Request{
			Method: "OPTIONS",
			Header: map[string][]string{
				"Origin":                        {testOrigin},
				"Access-Control-Request-Method": {"POST"},
			},
		}, "/test", "POST")
		test.AssertEquals(t, rw.Code, http.StatusOK)
		test.AssertEquals(t, rw.Header().Get("Access-Control-Allow-Origin"), testOrigin)
		// http://www.w3.org/TR/cors/ section 6.4:
		test.AssertEquals(t, rw.Header().Get("Vary"), "Origin")
	}
}

func TestPOST404(t *testing.T) {
	wfe, _, _ := setupWFE(t)
	responseWriter := httptest.NewRecorder()
	url, _ := url.Parse("/foobar")
	wfe.Index(ctx, newRequestEvent(), responseWriter, &http.Request{
		Method: "POST",
		URL:    url,
	})
	test.AssertEquals(t, responseWriter.Code, http.StatusNotFound)
}

func TestIndex(t *testing.T) {
	wfe, _, _ := setupWFE(t)

	responseWriter := httptest.NewRecorder()

	url, _ := url.Parse("/")
	wfe.Index(ctx, newRequestEvent(), responseWriter, &http.Request{
		Method: "GET",
		URL:    url,
	})
	test.AssertEquals(t, responseWriter.Code, http.StatusOK)
	test.AssertNotEquals(t, responseWriter.Body.String(), "404 page not found\n")
	test.Assert(t, strings.Contains(responseWriter.Body.String(), directoryPath),
		"directory path not found")
	test.AssertEquals(t, responseWriter.Header().Get("Cache-Control"), "public, max-age=0, no-cache")

	responseWriter.Body.Reset()
	responseWriter.Header().Del("Cache-Control")
	url, _ = url.Parse("/foo")
	wfe.Index(ctx, newRequestEvent(), responseWriter, &http.Request{
		URL: url,
	})
	//test.AssertEquals(t, responseWriter.Code, http.StatusNotFound)
	test.AssertEquals(t, responseWriter.Body.String(), "404 page not found\n")
	test.AssertEquals(t, responseWriter.Header().Get("Cache-Control"), "")
}

// randomDirectoryKeyPresent unmarshals the given buf of JSON and returns true
// if `randomDirKeyExplanationLink` appears as the value of a key in the directory
// object.
func randomDirectoryKeyPresent(t *testing.T, buf []byte) bool {
	var dir map[string]interface{}
	err := json.Unmarshal(buf, &dir)
	if err != nil {
		t.Errorf("Failed to unmarshal directory: %s", err)
	}
	for _, v := range dir {
		if v == randomDirKeyExplanationLink {
			return true
		}
	}
	return false
}

type fakeRand struct{}

func (fr fakeRand) Read(p []byte) (int, error) {
	return len(p), nil
}

func TestDirectory(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	mux := wfe.Handler(metrics.NoopRegisterer)
	core.RandReader = fakeRand{}
	defer func() { core.RandReader = rand.Reader }()

	dirURL, _ := url.Parse("/directory")

	getReq := &http.Request{
		Method: http.MethodGet,
		URL:    dirURL,
		Host:   "localhost:4300",
	}

	_, _, jwsBody := signer.byKeyID(1, nil, "http://localhost/directory", "")
	postAsGetReq := makePostRequestWithPath("/directory", jwsBody)

	testCases := []struct {
		name         string
		caaIdent     string
		website      string
		expectedJSON string
		request      *http.Request
	}{
		{
			name:    "standard GET, no CAA ident/website meta",
			request: getReq,
			expectedJSON: `{
  "keyChange": "http://localhost:4300/acme/key-change",
  "meta": {
    "termsOfService": "http://example.invalid/terms"
  },
  "newNonce": "http://localhost:4300/acme/new-nonce",
  "newAccount": "http://localhost:4300/acme/new-acct",
  "newOrder": "http://localhost:4300/acme/new-order",
  "revokeCert": "http://localhost:4300/acme/revoke-cert",
  "AAAAAAAAAAA": "https://community.letsencrypt.org/t/adding-random-entries-to-the-directory/33417"
}`,
		},
		{
			name:     "standard GET, CAA ident/website meta",
			caaIdent: "Radiant Lock",
			website:  "zombo.com",
			request:  getReq,
			expectedJSON: `{
  "AAAAAAAAAAA": "https://community.letsencrypt.org/t/adding-random-entries-to-the-directory/33417",
  "keyChange": "http://localhost:4300/acme/key-change",
  "meta": {
    "caaIdentities": [
      "Radiant Lock"
    ],
    "termsOfService": "http://example.invalid/terms",
    "website": "zombo.com"
  },
  "newAccount": "http://localhost:4300/acme/new-acct",
  "newNonce": "http://localhost:4300/acme/new-nonce",
  "newOrder": "http://localhost:4300/acme/new-order",
  "revokeCert": "http://localhost:4300/acme/revoke-cert"
}`,
		},
		{
			name:     "POST-as-GET, CAA ident/website meta",
			caaIdent: "Radiant Lock",
			website:  "zombo.com",
			request:  postAsGetReq,
			expectedJSON: `{
  "AAAAAAAAAAA": "https://community.letsencrypt.org/t/adding-random-entries-to-the-directory/33417",
  "keyChange": "http://localhost/acme/key-change",
  "meta": {
    "caaIdentities": [
      "Radiant Lock"
    ],
    "termsOfService": "http://example.invalid/terms",
    "website": "zombo.com"
  },
  "newAccount": "http://localhost/acme/new-acct",
  "newNonce": "http://localhost/acme/new-nonce",
  "newOrder": "http://localhost/acme/new-order",
  "revokeCert": "http://localhost/acme/revoke-cert"
}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Configure a caaIdentity and website for the /directory meta based on the tc
			wfe.DirectoryCAAIdentity = tc.caaIdent // "Radiant Lock"
			wfe.DirectoryWebsite = tc.website      //"zombo.com"
			responseWriter := httptest.NewRecorder()
			// Serve the /directory response for this request into a recorder
			mux.ServeHTTP(responseWriter, tc.request)
			// We expect all directory requests to return a json object with a good HTTP status
			test.AssertEquals(t, responseWriter.Header().Get("Content-Type"), "application/json")
			// We expect all requests to return status OK
			test.AssertEquals(t, responseWriter.Code, http.StatusOK)
			// The response should match expected
			test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), tc.expectedJSON)
			// Check that the random directory key is present
			test.AssertEquals(t,
				randomDirectoryKeyPresent(t, responseWriter.Body.Bytes()),
				true)
		})
	}
}

func TestRelativeDirectory(t *testing.T) {
	wfe, _, _ := setupWFE(t)
	mux := wfe.Handler(metrics.NoopRegisterer)
	core.RandReader = fakeRand{}
	defer func() { core.RandReader = rand.Reader }()

	expectedDirectory := func(hostname string) string {
		expected := new(bytes.Buffer)

		fmt.Fprintf(expected, "{")
		fmt.Fprintf(expected, `"keyChange":"%s/acme/key-change",`, hostname)
		fmt.Fprintf(expected, `"newNonce":"%s/acme/new-nonce",`, hostname)
		fmt.Fprintf(expected, `"newAccount":"%s/acme/new-acct",`, hostname)
		fmt.Fprintf(expected, `"newOrder":"%s/acme/new-order",`, hostname)
		fmt.Fprintf(expected, `"revokeCert":"%s/acme/revoke-cert",`, hostname)
		fmt.Fprintf(expected, `"AAAAAAAAAAA":"https://community.letsencrypt.org/t/adding-random-entries-to-the-directory/33417",`)
		fmt.Fprintf(expected, `"meta":{"termsOfService":"http://example.invalid/terms"}`)
		fmt.Fprintf(expected, "}")
		return expected.String()
	}

	dirTests := []struct {
		host        string
		protoHeader string
		result      string
	}{
		// Test '' (No host header) with no proto header
		{"", "", expectedDirectory("http://localhost")},
		// Test localhost:4300 with no proto header
		{"localhost:4300", "", expectedDirectory("http://localhost:4300")},
		// Test 127.0.0.1:4300 with no proto header
		{"127.0.0.1:4300", "", expectedDirectory("http://127.0.0.1:4300")},
		// Test localhost:4300 with HTTP proto header
		{"localhost:4300", "http", expectedDirectory("http://localhost:4300")},
		// Test localhost:4300 with HTTPS proto header
		{"localhost:4300", "https", expectedDirectory("https://localhost:4300")},
	}

	for _, tt := range dirTests {
		var headers map[string][]string
		responseWriter := httptest.NewRecorder()

		if tt.protoHeader != "" {
			headers = map[string][]string{
				"X-Forwarded-Proto": {tt.protoHeader},
			}
		}

		mux.ServeHTTP(responseWriter, &http.Request{
			Method: "GET",
			Host:   tt.host,
			URL:    mustParseURL(directoryPath),
			Header: headers,
		})
		test.AssertEquals(t, responseWriter.Header().Get("Content-Type"), "application/json")
		test.AssertEquals(t, responseWriter.Code, http.StatusOK)
		test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), tt.result)
	}
}

// TestNonceEndpoint tests requests to the WFE2's new-nonce endpoint
func TestNonceEndpoint(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	mux := wfe.Handler(metrics.NoopRegisterer)

	getReq := &http.Request{
		Method: http.MethodGet,
		URL:    mustParseURL(newNoncePath),
	}
	headReq := &http.Request{
		Method: http.MethodHead,
		URL:    mustParseURL(newNoncePath),
	}

	_, _, jwsBody := signer.byKeyID(1, nil, fmt.Sprintf("http://localhost%s", newNoncePath), "")
	postAsGetReq := makePostRequestWithPath(newNoncePath, jwsBody)

	testCases := []struct {
		name           string
		request        *http.Request
		expectedStatus int
	}{
		{
			name:           "GET new-nonce request",
			request:        getReq,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "HEAD new-nonce request",
			request:        headReq,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST-as-GET new-nonce request",
			request:        postAsGetReq,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			responseWriter := httptest.NewRecorder()
			mux.ServeHTTP(responseWriter, tc.request)
			// The response should have the expected HTTP status code
			test.AssertEquals(t, responseWriter.Code, tc.expectedStatus)
			// And the response should contain a valid nonce in the Replay-Nonce header
			nonce := responseWriter.Header().Get("Replay-Nonce")
			redeemResp, err := wfe.rnc.Redeem(context.Background(), &noncepb.NonceMessage{Nonce: nonce})
			test.AssertNotError(t, err, "redeeming nonce")
			test.AssertEquals(t, redeemResp.Valid, true)
			// The server MUST include a Cache-Control header field with the "no-store"
			// directive in responses for the newNonce resource, in order to prevent
			// caching of this resource.
			cacheControl := responseWriter.Header().Get("Cache-Control")
			test.AssertEquals(t, cacheControl, "no-store")
		})
	}
}

func TestHTTPMethods(t *testing.T) {
	wfe, _, _ := setupWFE(t)
	mux := wfe.Handler(metrics.NoopRegisterer)

	// NOTE: Boulder's muxer treats HEAD as implicitly allowed if GET is specified
	// so we include both here in `getOnly`
	getOnly := map[string]bool{http.MethodGet: true, http.MethodHead: true}
	postOnly := map[string]bool{http.MethodPost: true}
	getOrPost := map[string]bool{http.MethodGet: true, http.MethodHead: true, http.MethodPost: true}

	testCases := []struct {
		Name    string
		Path    string
		Allowed map[string]bool
	}{
		{
			Name:    "Index path should be GET only",
			Path:    "/",
			Allowed: getOnly,
		},
		{
			Name:    "Directory path should be GET or POST only",
			Path:    directoryPath,
			Allowed: getOrPost,
		},
		{
			Name:    "NewAcct path should be POST only",
			Path:    newAcctPath,
			Allowed: postOnly,
		},
		{
			Name:    "Acct path should be POST only",
			Path:    acctPath,
			Allowed: postOnly,
		},
		// TODO(@cpu): Remove GET authz support, support only POST-as-GET
		{
			Name:    "Authz path should be GET or POST only",
			Path:    authzPath,
			Allowed: getOrPost,
		},
		// TODO(@cpu): Remove GET challenge support, support only POST-as-GET
		{
			Name:    "Challenge path should be GET or POST only",
			Path:    challengePath,
			Allowed: getOrPost,
		},
		// TODO(@cpu): Remove GET certificate support, support only POST-as-GET
		{
			Name:    "Certificate path should be GET or POST only",
			Path:    certPath,
			Allowed: getOrPost,
		},
		{
			Name:    "RevokeCert path should be POST only",
			Path:    revokeCertPath,
			Allowed: postOnly,
		},
		{
			Name:    "Build ID path should be GET only",
			Path:    buildIDPath,
			Allowed: getOnly,
		},
		{
			Name:    "Rollover path should be POST only",
			Path:    rolloverPath,
			Allowed: postOnly,
		},
		{
			Name:    "New order path should be POST only",
			Path:    newOrderPath,
			Allowed: postOnly,
		},
		// TODO(@cpu): Remove GET order support, support only POST-as-GET
		{
			Name:    "Order path should be GET or POST only",
			Path:    orderPath,
			Allowed: getOrPost,
		},
		{
			Name:    "Nonce path should be GET or POST only",
			Path:    newNoncePath,
			Allowed: getOrPost,
		},
	}

	// NOTE: We omit http.MethodOptions because all requests with this method are
	// redirected to a special endpoint for CORS headers
	allMethods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodTrace,
	}

	responseWriter := httptest.NewRecorder()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// For every possible HTTP method check what the mux serves for the test
			// case path
			for _, method := range allMethods {
				responseWriter.Body.Reset()
				mux.ServeHTTP(responseWriter, &http.Request{
					Method: method,
					URL:    mustParseURL(tc.Path),
				})
				// If the method isn't one that is intended to be allowed by the path,
				// check that the response was the not allowed response
				if _, ok := tc.Allowed[method]; !ok {
					var prob probs.ProblemDetails
					// Unmarshal the body into a problem
					body := responseWriter.Body.String()
					err := json.Unmarshal([]byte(body), &prob)
					test.AssertNotError(t, err, fmt.Sprintf("Error unmarshalling resp body: %q", body))
					// TODO(@cpu): It seems like the mux should be returning
					// http.StatusMethodNotAllowed here, but instead it returns StatusOK
					// with a problem that has a StatusMethodNotAllowed HTTPStatus. Is
					// this a bug?
					test.AssertEquals(t, responseWriter.Code, http.StatusOK)
					test.AssertEquals(t, prob.HTTPStatus, http.StatusMethodNotAllowed)
					test.AssertEquals(t, prob.Detail, "Method not allowed")
				} else {
					// Otherwise if it was an allowed method, ensure that the response was
					// *not* StatusMethodNotAllowed
					test.AssertNotEquals(t, responseWriter.Code, http.StatusMethodNotAllowed)
				}
			}
		})
	}
}

func TestGetChallenge(t *testing.T) {
	wfe, _, _ := setupWFE(t)

	challengeURL := "http://localhost/acme/chall-v3/1/-ZfxEw"

	for _, method := range []string{"GET", "HEAD"} {
		resp := httptest.NewRecorder()

		req, err := http.NewRequest(method, challengeURL, nil)
		req.URL.Path = "1/-ZfxEw"
		test.AssertNotError(t, err, "Could not make NewRequest")

		wfe.Challenge(ctx, newRequestEvent(), resp, req)
		test.AssertEquals(t,
			resp.Code,
			http.StatusOK)
		test.AssertEquals(t,
			resp.Header().Get("Location"),
			challengeURL)
		test.AssertEquals(t,
			resp.Header().Get("Content-Type"),
			"application/json")
		test.AssertEquals(t,
			resp.Header().Get("Link"),
			`<http://localhost/acme/authz-v3/1>;rel="up"`)
		// Body is only relevant for GET. For HEAD, body will
		// be discarded by HandleFunc() anyway, so it doesn't
		// matter what Challenge() writes to it.
		if method == "GET" {
			test.AssertUnmarshaledEquals(
				t, resp.Body.String(),
				`{"status": "pending", "type":"dns","token":"token","url":"http://localhost/acme/chall-v3/1/-ZfxEw"}`)
		}
	}
}

func TestChallenge(t *testing.T) {
	wfe, _, signer := setupWFE(t)

	post := func(path string) *http.Request {
		signedURL := fmt.Sprintf("http://localhost/%s", path)
		_, _, jwsBody := signer.byKeyID(1, nil, signedURL, `{}`)
		return makePostRequestWithPath(path, jwsBody)
	}
	postAsGet := func(keyID int64, path, body string) *http.Request {
		_, _, jwsBody := signer.byKeyID(keyID, nil, fmt.Sprintf("http://localhost/%s", path), body)
		return makePostRequestWithPath(path, jwsBody)
	}

	testCases := []struct {
		Name            string
		Request         *http.Request
		ExpectedStatus  int
		ExpectedHeaders map[string]string
		ExpectedBody    string
	}{
		{
			Name:           "Valid challenge",
			Request:        post("1/-ZfxEw"),
			ExpectedStatus: http.StatusOK,
			ExpectedHeaders: map[string]string{
				"Location": "http://localhost/acme/chall-v3/1/-ZfxEw",
				"Link":     `<http://localhost/acme/authz-v3/1>;rel="up"`,
			},
			ExpectedBody: `{"status": "pending", "type":"dns","token":"token","url":"http://localhost/acme/chall-v3/1/-ZfxEw"}`,
		},
		{
			Name:           "Expired challenge",
			Request:        post("3/-ZfxEw"),
			ExpectedStatus: http.StatusNotFound,
			ExpectedBody:   `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Expired authorization","status":404}`,
		},
		{
			Name:           "Missing challenge",
			Request:        post("1/"),
			ExpectedStatus: http.StatusNotFound,
			ExpectedBody:   `{"type":"` + probs.V2ErrorNS + `malformed","detail":"No such challenge","status":404}`,
		},
		{
			Name:           "Unspecified database error",
			Request:        post("4/-ZfxEw"),
			ExpectedStatus: http.StatusInternalServerError,
			ExpectedBody:   `{"type":"` + probs.V2ErrorNS + `serverInternal","detail":"Problem getting authorization","status":500}`,
		},
		{
			Name:           "POST-as-GET, wrong owner",
			Request:        postAsGet(1, "5/-ZfxEw", ""),
			ExpectedStatus: http.StatusForbidden,
			ExpectedBody:   `{"type":"` + probs.V2ErrorNS + `unauthorized","detail":"User account ID doesn't match account ID in authorization","status":403}`,
		},
		{
			Name:           "Valid POST-as-GET",
			Request:        postAsGet(1, "1/-ZfxEw", ""),
			ExpectedStatus: http.StatusOK,
			ExpectedBody:   `{"status": "pending", "type":"dns", "token":"token", "url": "http://localhost/acme/chall-v3/1/-ZfxEw"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			responseWriter := httptest.NewRecorder()
			wfe.Challenge(ctx, newRequestEvent(), responseWriter, tc.Request)
			// Check the response code, headers and body match expected
			headers := responseWriter.Header()
			body := responseWriter.Body.String()
			test.AssertEquals(t, responseWriter.Code, tc.ExpectedStatus)
			for h, v := range tc.ExpectedHeaders {
				test.AssertEquals(t, headers.Get(h), v)
			}
			test.AssertUnmarshaledEquals(t, body, tc.ExpectedBody)
		})
	}
}

// MockRAPerformValidationError is a mock RA that just returns an error on
// PerformValidation.
type MockRAPerformValidationError struct {
	MockRegistrationAuthority
}

func (ra *MockRAPerformValidationError) PerformValidation(context.Context, *rapb.PerformValidationRequest, ...grpc.CallOption) (*corepb.Authorization, error) {
	return nil, errors.New("broken on purpose")
}

// TestUpdateChallengeFinalizedAuthz tests that POSTing a challenge associated
// with an already valid authorization just returns the challenge without calling
// the RA.
func TestUpdateChallengeFinalizedAuthz(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	wfe.ra = &MockRAPerformValidationError{}
	responseWriter := httptest.NewRecorder()

	signedURL := "http://localhost/1/-ZfxEw"
	_, _, jwsBody := signer.byKeyID(1, nil, signedURL, `{}`)
	request := makePostRequestWithPath("1/-ZfxEw", jwsBody)
	wfe.Challenge(ctx, newRequestEvent(), responseWriter, request)

	body := responseWriter.Body.String()
	test.AssertUnmarshaledEquals(t, body, `{
	  "status": "pending",
		"type": "dns",
		"token":"token",
		"url": "http://localhost/acme/chall-v3/1/-ZfxEw"
	  }`)
}

// TestUpdateChallengeRAError tests that when the RA returns an error from
// PerformValidation that the WFE returns an internal server error as expected
// and does not panic or otherwise bug out.
func TestUpdateChallengeRAError(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	// Mock the RA to always fail PerformValidation
	wfe.ra = &MockRAPerformValidationError{}

	// Update a pending challenge
	signedURL := "http://localhost/2/-ZfxEw"
	_, _, jwsBody := signer.byKeyID(1, nil, signedURL, `{}`)
	responseWriter := httptest.NewRecorder()
	request := makePostRequestWithPath("2/-ZfxEw", jwsBody)

	wfe.Challenge(ctx, newRequestEvent(), responseWriter, request)

	// The result should be an internal server error problem.
	body := responseWriter.Body.String()
	test.AssertUnmarshaledEquals(t, body, `{
		"type": "urn:ietf:params:acme:error:serverInternal",
	  "detail": "Unable to update challenge",
		"status": 500
	}`)
}

func TestBadNonce(t *testing.T) {
	wfe, _, _ := setupWFE(t)

	key := loadKey(t, []byte(test2KeyPrivatePEM))
	rsaKey, ok := key.(*rsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load RSA key")
	// NOTE: We deliberately do not set the NonceSource in the jose.SignerOptions
	// for this test in order to provoke a bad nonce error
	noNonceSigner, err := jose.NewSigner(jose.SigningKey{
		Key:       rsaKey,
		Algorithm: jose.RS256,
	}, &jose.SignerOptions{
		EmbedJWK: true,
	})
	test.AssertNotError(t, err, "Failed to make signer")

	responseWriter := httptest.NewRecorder()
	result, err := noNonceSigner.Sign([]byte(`{"contact":["mailto:person@mail.com"]}`))
	test.AssertNotError(t, err, "Failed to sign body")
	wfe.NewAccount(ctx, newRequestEvent(), responseWriter,
		makePostRequestWithPath("nonce", result.FullSerialize()))
	test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), `{"type":"`+probs.V2ErrorNS+`badNonce","detail":"JWS has no anti-replay nonce","status":400}`)
}

func TestNewECDSAAccount(t *testing.T) {
	wfe, _, signer := setupWFE(t)

	// E1 always exists; E2 never exists
	key := loadKey(t, []byte(testE2KeyPrivatePEM))
	_, ok := key.(*ecdsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load ECDSA key")

	payload := `{"contact":["mailto:person@mail.com"],"termsOfServiceAgreed":true}`
	path := newAcctPath
	signedURL := fmt.Sprintf("http://localhost%s", path)
	_, _, body := signer.embeddedJWK(key, signedURL, payload)
	request := makePostRequestWithPath(path, body)

	responseWriter := httptest.NewRecorder()
	wfe.NewAccount(ctx, newRequestEvent(), responseWriter, request)

	var acct core.Registration
	responseBody := responseWriter.Body.String()
	err := json.Unmarshal([]byte(responseBody), &acct)
	test.AssertNotError(t, err, "Couldn't unmarshal returned account object")
	test.Assert(t, len(*acct.Contact) >= 1, "No contact field in account")
	test.AssertEquals(t, (*acct.Contact)[0], "mailto:person@mail.com")
	test.AssertEquals(t, acct.Agreement, "")
	test.AssertEquals(t, acct.InitialIP.String(), "1.1.1.1")

	test.AssertEquals(t, responseWriter.Header().Get("Location"), "http://localhost/acme/acct/1")

	key = loadKey(t, []byte(testE1KeyPrivatePEM))
	_, ok = key.(*ecdsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load ECDSA key")

	_, _, body = signer.embeddedJWK(key, signedURL, payload)
	request = makePostRequestWithPath(path, body)

	// Reset the body and status code
	responseWriter = httptest.NewRecorder()
	// POST, Valid JSON, Key already in use
	wfe.NewAccount(ctx, newRequestEvent(), responseWriter, request)
	test.AssertUnmarshaledEquals(t, responseWriter.Body.String(),
		`{
		"key": {
			"kty": "EC",
			"crv": "P-256",
			"x": "FwvSZpu06i3frSk_mz9HcD9nETn4wf3mQ-zDtG21Gao",
			"y": "S8rR-0dWa8nAcw1fbunF_ajS3PQZ-QwLps-2adgLgPk"
		},
		"initialIp": "",
		"status": ""
		}`)
	test.AssertEquals(t, responseWriter.Header().Get("Location"), "http://localhost/acme/acct/3")
	test.AssertEquals(t, responseWriter.Code, 200)

	// test3KeyPrivatePEM is a private key corresponding to a deactivated account in the mock SA's GetRegistration test data.
	key = loadKey(t, []byte(test3KeyPrivatePEM))
	_, ok = key.(*rsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load test3 key")

	// Reset the body and status code
	responseWriter = httptest.NewRecorder()

	// Test POST valid JSON with deactivated account
	payload = `{}`
	path = "1"
	signedURL = "http://localhost/1"
	_, _, body = signer.embeddedJWK(key, signedURL, payload)
	request = makePostRequestWithPath(path, body)
	wfe.NewAccount(ctx, newRequestEvent(), responseWriter, request)
	test.AssertEquals(t, responseWriter.Code, http.StatusForbidden)
}

// Test that the WFE handling of the "empty update" POST is correct. The ACME
// spec describes how when clients wish to query the server for information
// about an account an empty account update should be sent, and
// a populated acct object will be returned.
func TestEmptyAccount(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	responseWriter := httptest.NewRecorder()

	// Test Key 1 is mocked in the mock StorageAuthority used in setupWFE to
	// return a populated account for GetRegistrationByKey when test key 1 is
	// used.
	key := loadKey(t, []byte(test1KeyPrivatePEM))
	_, ok := key.(*rsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load RSA key")

	payload := `{}`
	path := "1"
	signedURL := "http://localhost/1"
	_, _, body := signer.byKeyID(1, key, signedURL, payload)
	request := makePostRequestWithPath(path, body)

	// Send an account update with the trivial body
	wfe.Account(
		ctx,
		newRequestEvent(),
		responseWriter,
		request)

	responseBody := responseWriter.Body.String()
	// There should be no error
	test.AssertNotContains(t, responseBody, probs.V2ErrorNS)

	// We should get back a populated Account
	var acct core.Registration
	err := json.Unmarshal([]byte(responseBody), &acct)
	test.AssertNotError(t, err, "Couldn't unmarshal returned account object")
	test.Assert(t, len(*acct.Contact) >= 1, "No contact field in account")
	test.AssertEquals(t, (*acct.Contact)[0], "mailto:person@mail.com")
	test.AssertEquals(t, acct.Agreement, "")
	responseWriter.Body.Reset()
}

func TestNewAccount(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	mux := wfe.Handler(metrics.NoopRegisterer)
	key := loadKey(t, []byte(test2KeyPrivatePEM))
	_, ok := key.(*rsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load test2 key")

	path := newAcctPath
	signedURL := fmt.Sprintf("http://localhost%s", path)

	wrongAgreementAcct := `{"contact":["mailto:person@mail.com"],"termsOfServiceAgreed":false}`
	// An acct with the terms not agreed to
	_, _, wrongAgreementBody := signer.embeddedJWK(key, signedURL, wrongAgreementAcct)

	// A non-JSON payload
	_, _, fooBody := signer.embeddedJWK(key, signedURL, `foo`)

	type newAcctErrorTest struct {
		r        *http.Request
		respBody string
	}

	acctErrTests := []newAcctErrorTest{
		// POST, but no body.
		{
			&http.Request{
				Method: "POST",
				URL:    mustParseURL(newAcctPath),
				Header: map[string][]string{
					"Content-Length": {"0"},
					"Content-Type":   {expectedJWSContentType},
				},
			},
			`{"type":"` + probs.V2ErrorNS + `malformed","detail":"No body on POST","status":400}`,
		},

		// POST, but body that isn't valid JWS
		{
			makePostRequestWithPath(newAcctPath, "hi"),
			`{"type":"` + probs.V2ErrorNS + `malformed","detail":"Parse error reading JWS","status":400}`,
		},

		// POST, Properly JWS-signed, but payload is "foo", not base64-encoded JSON.
		{
			makePostRequestWithPath(newAcctPath, fooBody),
			`{"type":"` + probs.V2ErrorNS + `malformed","detail":"Request payload did not parse as JSON","status":400}`,
		},

		// Same signed body, but payload modified by one byte, breaking signature.
		// should fail JWS verification.
		{
			makePostRequestWithPath(newAcctPath,
				`{"payload":"Zm9x","protected":"eyJhbGciOiJSUzI1NiIsImp3ayI6eyJrdHkiOiJSU0EiLCJuIjoicW5BUkxyVDdYejRnUmNLeUxkeWRtQ3ItZXk5T3VQSW1YNFg0MHRoazNvbjI2RmtNem5SM2ZSanM2NmVMSzdtbVBjQlo2dU9Kc2VVUlU2d0FhWk5tZW1vWXgxZE12cXZXV0l5aVFsZUhTRDdROHZCcmhSNnVJb080akF6SlpSLUNoelp1U0R0N2lITi0zeFVWc3B1NVhHd1hVX01WSlpzaFR3cDRUYUZ4NWVsSElUX09iblR2VE9VM1hoaXNoMDdBYmdaS21Xc1ZiWGg1cy1DcklpY1U0T2V4SlBndW5XWl9ZSkp1ZU9LbVR2bkxsVFY0TXpLUjJvWmxCS1oyN1MwLVNmZFZfUUR4X3lkbGU1b01BeUtWdGxBVjM1Y3lQTUlzWU53Z1VHQkNkWV8yVXppNWVYMGxUYzdNUFJ3ejZxUjFraXAtaTU5VmNHY1VRZ3FIVjZGeXF3IiwiZSI6IkFRQUIifSwia2lkIjoiIiwibm9uY2UiOiJyNHpuenZQQUVwMDlDN1JwZUtYVHhvNkx3SGwxZVBVdmpGeXhOSE1hQnVvIiwidXJsIjoiaHR0cDovL2xvY2FsaG9zdC9hY21lL25ldy1yZWcifQ","signature":"jcTdxSygm_cvD7KbXqsxgnoPApCTSkV4jolToSOd2ciRkg5W7Yl0ZKEEKwOc-dYIbQiwGiDzisyPCicwWsOUA1WSqHylKvZ3nxSMc6KtwJCW2DaOqcf0EEjy5VjiZJUrOt2c-r6b07tbn8sfOJKwlF2lsOeGi4s-rtvvkeQpAU-AWauzl9G4bv2nDUeCviAZjHx_PoUC-f9GmZhYrbDzAvXZ859ktM6RmMeD0OqPN7bhAeju2j9Gl0lnryZMtq2m0J2m1ucenQBL1g4ZkP1JiJvzd2cAz5G7Ftl2YeJJyWhqNd3qq0GVOt1P11s8PTGNaSoM0iR9QfUxT9A6jxARtg"}`),
			`{"type":"` + probs.V2ErrorNS + `malformed","detail":"JWS verification error","status":400}`,
		},
		{
			makePostRequestWithPath(newAcctPath, wrongAgreementBody),
			`{"type":"` + probs.V2ErrorNS + `malformed","detail":"must agree to terms of service","status":400}`,
		},
	}
	for _, rt := range acctErrTests {
		responseWriter := httptest.NewRecorder()
		mux.ServeHTTP(responseWriter, rt.r)
		test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), rt.respBody)
	}

	responseWriter := httptest.NewRecorder()

	payload := `{"contact":["mailto:person@mail.com"],"termsOfServiceAgreed":true}`
	_, _, body := signer.embeddedJWK(key, signedURL, payload)
	request := makePostRequestWithPath(path, body)

	wfe.NewAccount(ctx, newRequestEvent(), responseWriter, request)

	var acct core.Registration
	responseBody := responseWriter.Body.String()
	err := json.Unmarshal([]byte(responseBody), &acct)
	test.AssertNotError(t, err, "Couldn't unmarshal returned account object")
	test.Assert(t, len(*acct.Contact) >= 1, "No contact field in account")
	test.AssertEquals(t, (*acct.Contact)[0], "mailto:person@mail.com")
	test.AssertEquals(t, acct.InitialIP.String(), "1.1.1.1")
	// Agreement is an ACMEv1 field and should not be present
	test.AssertEquals(t, acct.Agreement, "")

	test.AssertEquals(
		t, responseWriter.Header().Get("Location"),
		"http://localhost/acme/acct/1")

	// Load an existing key
	key = loadKey(t, []byte(test1KeyPrivatePEM))
	_, ok = key.(*rsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load test1 key")

	// Reset the body and status code
	responseWriter = httptest.NewRecorder()
	// POST, Valid JSON, Key already in use
	_, _, body = signer.embeddedJWK(key, signedURL, payload)
	request = makePostRequestWithPath(path, body)
	// POST the NewAccount request
	wfe.NewAccount(ctx, newRequestEvent(), responseWriter, request)
	// We expect a Location header and a 200 response with an empty body
	test.AssertEquals(
		t, responseWriter.Header().Get("Location"),
		"http://localhost/acme/acct/1")
	test.AssertEquals(t, responseWriter.Code, 200)
	test.AssertUnmarshaledEquals(t, responseWriter.Body.String(),
		`{
		"key": {
			"kty": "RSA",
			"n": "yNWVhtYEKJR21y9xsHV-PD_bYwbXSeNuFal46xYxVfRL5mqha7vttvjB_vc7Xg2RvgCxHPCqoxgMPTzHrZT75LjCwIW2K_klBYN8oYvTwwmeSkAz6ut7ZxPv-nZaT5TJhGk0NT2kh_zSpdriEJ_3vW-mqxYbbBmpvHqsa1_zx9fSuHYctAZJWzxzUZXykbWMWQZpEiE0J4ajj51fInEzVn7VxV-mzfMyboQjujPh7aNJxAWSq4oQEJJDgWwSh9leyoJoPpONHxh5nEE5AjE01FkGICSxjpZsF-w8hOTI3XXohUdu29Se26k2B0PolDSuj0GIQU6-W9TdLXSjBb2SpQ",
			"e": "AQAB"
		},
		"contact": [
			"mailto:person@mail.com"
		],
		"initialIp": "",
		"status": "valid"
	}`)
}

func TestNewAccountWhenAccountHasBeenDeactivated(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	signedURL := fmt.Sprintf("http://localhost%s", newAcctPath)
	// test3KeyPrivatePEM is a private key corresponding to a deactivated account in the mock SA's GetRegistration test data.
	k := loadKey(t, []byte(test3KeyPrivatePEM))
	_, ok := k.(*rsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load test3 key")

	payload := `{"contact":["mailto:person@mail.com"],"termsOfServiceAgreed":true}`
	_, _, body := signer.embeddedJWK(k, signedURL, payload)
	request := makePostRequestWithPath(newAcctPath, body)

	responseWriter := httptest.NewRecorder()
	wfe.NewAccount(ctx, newRequestEvent(), responseWriter, request)

	test.AssertEquals(t, responseWriter.Code, http.StatusForbidden)
}

func TestNewAccountNoID(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	key := loadKey(t, []byte(test2KeyPrivatePEM))
	_, ok := key.(*rsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load test2 key")
	path := newAcctPath
	signedURL := fmt.Sprintf("http://localhost%s", path)

	payload := `{"contact":["mailto:person@mail.com"],"termsOfServiceAgreed":true}`
	_, _, body := signer.embeddedJWK(key, signedURL, payload)
	request := makePostRequestWithPath(path, body)

	responseWriter := httptest.NewRecorder()
	wfe.NewAccount(ctx, newRequestEvent(), responseWriter, request)

	responseBody := responseWriter.Body.String()
	test.AssertUnmarshaledEquals(t, responseBody, `{
		"key": {
			"kty": "RSA",
			"n": "qnARLrT7Xz4gRcKyLdydmCr-ey9OuPImX4X40thk3on26FkMznR3fRjs66eLK7mmPcBZ6uOJseURU6wAaZNmemoYx1dMvqvWWIyiQleHSD7Q8vBrhR6uIoO4jAzJZR-ChzZuSDt7iHN-3xUVspu5XGwXU_MVJZshTwp4TaFx5elHIT_ObnTvTOU3Xhish07AbgZKmWsVbXh5s-CrIicU4OexJPgunWZ_YJJueOKmTvnLlTV4MzKR2oZlBKZ27S0-SfdV_QDx_ydle5oMAyKVtlAV35cyPMIsYNwgUGBCdY_2Uzi5eX0lTc7MPRwz6qR1kip-i59VcGcUQgqHV6Fyqw",
			"e": "AQAB"
		},
		"contact": [
			"mailto:person@mail.com"
		],
		"initialIp": "1.1.1.1",
		"createdAt": "2021-01-01T00:00:00Z",
		"status": ""
	}`)
}

func TestGetAuthorization(t *testing.T) {
	wfe, _, signer := setupWFE(t)

	// Expired authorizations should be inaccessible
	authzURL := "3"
	responseWriter := httptest.NewRecorder()
	wfe.Authorization(ctx, newRequestEvent(), responseWriter, &http.Request{
		Method: "GET",
		URL:    mustParseURL(authzURL),
	})
	test.AssertEquals(t, responseWriter.Code, http.StatusNotFound)
	test.AssertUnmarshaledEquals(t, responseWriter.Body.String(),
		`{"type":"`+probs.V2ErrorNS+`malformed","detail":"Expired authorization","status":404}`)
	responseWriter.Body.Reset()

	// Ensure that a valid authorization can't be reached with an invalid URL
	wfe.Authorization(ctx, newRequestEvent(), responseWriter, &http.Request{
		URL:    mustParseURL("1d"),
		Method: "GET",
	})
	test.AssertUnmarshaledEquals(t, responseWriter.Body.String(),
		`{"type":"`+probs.V2ErrorNS+`malformed","detail":"Invalid authorization ID","status":400}`)

	_, _, jwsBody := signer.byKeyID(1, nil, "http://localhost/1", "")
	postAsGet := makePostRequestWithPath("1", jwsBody)

	responseWriter = httptest.NewRecorder()
	// Ensure that a POST-as-GET to an authorization works
	wfe.Authorization(ctx, newRequestEvent(), responseWriter, postAsGet)
	test.AssertEquals(t, responseWriter.Code, http.StatusOK)
	body := responseWriter.Body.String()
	test.AssertUnmarshaledEquals(t, body, `
	{
		"identifier": {
			"type": "dns",
			"value": "not-an-example.com"
		},
		"status": "valid",
		"expires": "2070-01-01T00:00:00Z",
		"challenges": [
			{
			  "status": "pending",
				"type": "dns",
				"token":"token",
				"url": "http://localhost/acme/chall-v3/1/-ZfxEw"
			}
		]
	}`)
}

// TestAuthorization500 tests that internal errors on GetAuthorization result in
// a 500.
func TestAuthorization500(t *testing.T) {
	wfe, _, _ := setupWFE(t)

	responseWriter := httptest.NewRecorder()
	wfe.Authorization(ctx, newRequestEvent(), responseWriter, &http.Request{
		Method: "GET",
		URL:    mustParseURL("4"),
	})
	expected := `{
         "type": "urn:ietf:params:acme:error:serverInternal",
				 "detail": "Problem getting authorization",
				 "status": 500
  }`
	test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), expected)
}

// TestAuthorizationChallengeNamespace tests that the runtime prefixing of
// Challenge Problem Types works as expected
func TestAuthorizationChallengeNamespace(t *testing.T) {
	wfe, clk, _ := setupWFE(t)

	mockSA := &mocks.SAWithFailedChallenges{Clk: clk}
	wfe.sa = mockSA

	// For "oldNS" the SA mock returns an authorization with a failed challenge
	// that has an error with the type already prefixed by the v1 error NS
	responseWriter := httptest.NewRecorder()
	wfe.Authorization(ctx, newRequestEvent(), responseWriter, &http.Request{
		Method: "GET",
		URL:    mustParseURL("55"),
	})

	var authz core.Authorization
	err := json.Unmarshal(responseWriter.Body.Bytes(), &authz)
	test.AssertNotError(t, err, "Couldn't unmarshal returned authorization object")
	test.AssertEquals(t, len(authz.Challenges), 1)
	// The Challenge Error Type should have its prefix unmodified
	test.AssertEquals(t, string(authz.Challenges[0].Error.Type), probs.V1ErrorNS+"things:are:whack")

	// For "failed" the SA mock returns an authorization with a failed challenge
	// that has an error with the type not prefixed by an error namespace.
	responseWriter = httptest.NewRecorder()
	wfe.Authorization(ctx, newRequestEvent(), responseWriter, &http.Request{
		Method: "GET",
		URL:    mustParseURL("56"),
	})

	err = json.Unmarshal(responseWriter.Body.Bytes(), &authz)
	test.AssertNotError(t, err, "Couldn't unmarshal returned authorization object")
	test.AssertEquals(t, len(authz.Challenges), 1)
	// The Challenge Error Type should have had the probs.V2ErrorNS prefix added
	test.AssertEquals(t, string(authz.Challenges[0].Error.Type), probs.V2ErrorNS+"things:are:whack")
	responseWriter.Body.Reset()
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func TestAccount(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	mux := wfe.Handler(metrics.NoopRegisterer)
	responseWriter := httptest.NewRecorder()

	// Test GET proper entry returns 405
	mux.ServeHTTP(responseWriter, &http.Request{
		Method: "GET",
		URL:    mustParseURL(acctPath),
	})
	test.AssertUnmarshaledEquals(t,
		responseWriter.Body.String(),
		`{"type":"`+probs.V2ErrorNS+`malformed","detail":"Method not allowed","status":405}`)
	responseWriter.Body.Reset()

	// Test POST invalid JSON
	wfe.Account(ctx, newRequestEvent(), responseWriter, makePostRequestWithPath("2", "invalid"))
	test.AssertUnmarshaledEquals(t,
		responseWriter.Body.String(),
		`{"type":"`+probs.V2ErrorNS+`malformed","detail":"Parse error reading JWS","status":400}`)
	responseWriter.Body.Reset()

	key := loadKey(t, []byte(test2KeyPrivatePEM))
	_, ok := key.(*rsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load RSA key")

	signedURL := fmt.Sprintf("http://localhost%s%d", acctPath, 102)
	path := fmt.Sprintf("%s%d", acctPath, 102)
	payload := `{}`
	// ID 102 is used by the mock for missing acct
	_, _, body := signer.byKeyID(102, nil, signedURL, payload)
	request := makePostRequestWithPath(path, body)

	// Test POST valid JSON but key is not registered
	wfe.Account(ctx, newRequestEvent(), responseWriter, request)
	test.AssertUnmarshaledEquals(t,
		responseWriter.Body.String(),
		`{"type":"`+probs.V2ErrorNS+`accountDoesNotExist","detail":"Account \"http://localhost/acme/acct/102\" not found","status":400}`)
	responseWriter.Body.Reset()

	key = loadKey(t, []byte(test1KeyPrivatePEM))
	_, ok = key.(*rsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load RSA key")

	// Test POST valid JSON with account up in the mock
	payload = `{}`
	path = "1"
	signedURL = "http://localhost/1"
	_, _, body = signer.byKeyID(1, nil, signedURL, payload)
	request = makePostRequestWithPath(path, body)

	wfe.Account(ctx, newRequestEvent(), responseWriter, request)
	test.AssertNotContains(t, responseWriter.Body.String(), probs.V2ErrorNS)
	links := responseWriter.Header()["Link"]
	test.AssertEquals(t, contains(links, "<"+agreementURL+">;rel=\"terms-of-service\""), true)
	responseWriter.Body.Reset()

	// Test POST valid JSON with garbage in URL but valid account ID
	payload = `{}`
	signedURL = "http://localhost/a/bunch/of/garbage/1"
	_, _, body = signer.byKeyID(1, nil, signedURL, payload)
	request = makePostRequestWithPath("/a/bunch/of/garbage/1", body)

	wfe.Account(ctx, newRequestEvent(), responseWriter, request)
	test.AssertContains(t, responseWriter.Body.String(), "400")
	test.AssertContains(t, responseWriter.Body.String(), probs.V2ErrorNS+"malformed")
	responseWriter.Body.Reset()

	// Test valid POST-as-GET request
	responseWriter = httptest.NewRecorder()
	_, _, body = signer.byKeyID(1, nil, "http://localhost/1", "")
	request = makePostRequestWithPath("1", body)
	wfe.Account(ctx, newRequestEvent(), responseWriter, request)
	// It should not error
	test.AssertNotContains(t, responseWriter.Body.String(), probs.V2ErrorNS)
	test.AssertEquals(t, responseWriter.Code, http.StatusOK)

	altKey := loadKey(t, []byte(test2KeyPrivatePEM))
	_, ok = altKey.(*rsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load altKey RSA key")

	// Test POST-as-GET request signed with wrong account key
	responseWriter = httptest.NewRecorder()
	_, _, body = signer.byKeyID(2, altKey, "http://localhost/1", "")
	request = makePostRequestWithPath("1", body)
	wfe.Account(ctx, newRequestEvent(), responseWriter, request)
	// It should error
	test.AssertEquals(t, responseWriter.Code, http.StatusForbidden)
	test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), `{
		"type": "urn:ietf:params:acme:error:unauthorized",
		"detail": "Request signing key did not match account key",
		"status": 403
	}`)
}

type mockSAWithCert struct {
	sapb.StorageAuthorityReadOnlyClient
	cert   *x509.Certificate
	status core.OCSPStatus
}

func newMockSAWithCert(t *testing.T, sa sapb.StorageAuthorityReadOnlyClient, zeroNotBefore bool) *mockSAWithCert {
	cert, err := core.LoadCert("../test/hierarchy/ee-r3.cert.pem")
	if zeroNotBefore {
		// Just for the sake of TestGetAPIAndMandatoryPOSTAsGET, we set the
		// Issued timestamp of this certificate to be very old (the year 0).
		cert.NotBefore = time.Time{}
	}
	test.AssertNotError(t, err, "Failed to load test cert")
	return &mockSAWithCert{sa, cert, core.OCSPStatusGood}
}

// GetCertificate returns the mock SA's hard-coded certificate, issued by the
// account with regID 1, if the given serial matches. Otherwise, returns not found.
func (sa *mockSAWithCert) GetCertificate(_ context.Context, req *sapb.Serial, _ ...grpc.CallOption) (*corepb.Certificate, error) {
	if req.Serial != core.SerialToString(sa.cert.SerialNumber) {
		return nil, berrors.NotFoundError("Certificate with serial %q not found", req.Serial)
	}

	return &corepb.Certificate{
		RegistrationID: 1,
		Serial:         core.SerialToString(sa.cert.SerialNumber),
		Issued:         sa.cert.NotBefore.UnixNano(),
		Expires:        sa.cert.NotAfter.UnixNano(),
		Der:            sa.cert.Raw,
	}, nil
}

// GetCertificateStatus returns the mock SA's status, if the given serial matches.
// Otherwise, returns not found.
func (sa *mockSAWithCert) GetCertificateStatus(_ context.Context, req *sapb.Serial, _ ...grpc.CallOption) (*corepb.CertificateStatus, error) {
	if req.Serial != core.SerialToString(sa.cert.SerialNumber) {
		return nil, berrors.NotFoundError("Status for certificate with serial %q not found", req.Serial)
	}

	return &corepb.CertificateStatus{
		Serial: core.SerialToString(sa.cert.SerialNumber),
		Status: string(sa.status),
	}, nil
}

type mockSAWithIncident struct {
	sapb.StorageAuthorityReadOnlyClient
	incidents map[string]*sapb.Incidents
}

// newMockSAWithIncident returns a mock SA with an enabled (ongoing) incident
// for each of the provided serials.
func newMockSAWithIncident(sa sapb.StorageAuthorityReadOnlyClient, serial []string) *mockSAWithIncident {
	incidents := make(map[string]*sapb.Incidents)
	for _, s := range serial {
		incidents[s] = &sapb.Incidents{
			Incidents: []*sapb.Incident{
				{
					Id:          0,
					SerialTable: "incident_foo",
					Url:         agreementURL,
					RenewBy:     0,
					Enabled:     true,
				},
			},
		}
	}
	return &mockSAWithIncident{sa, incidents}
}

func (sa *mockSAWithIncident) IncidentsForSerial(_ context.Context, req *sapb.Serial, _ ...grpc.CallOption) (*sapb.Incidents, error) {
	incidents, ok := sa.incidents[req.Serial]
	if ok {
		return incidents, nil
	}
	return &sapb.Incidents{}, nil
}

func TestGetCertificate(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	wfe.sa = newMockSAWithCert(t, wfe.sa, false)
	mux := wfe.Handler(metrics.NoopRegisterer)

	makeGet := func(path string) *http.Request {
		return &http.Request{URL: &url.URL{Path: path}, Method: "GET"}
	}

	makePost := func(keyID int64, key interface{}, path, body string) *http.Request {
		_, _, jwsBody := signer.byKeyID(keyID, key, fmt.Sprintf("http://localhost%s", path), body)
		return makePostRequestWithPath(path, jwsBody)
	}

	altKey := loadKey(t, []byte(test2KeyPrivatePEM))
	_, ok := altKey.(*rsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load RSA key")

	certPemBytes, _ := os.ReadFile("../test/hierarchy/ee-r3.cert.pem")
	cert, err := core.LoadCert("../test/hierarchy/ee-r3.cert.pem")
	test.AssertNotError(t, err, "failed to load test certificate")

	chainPemBytes, err := os.ReadFile("../test/hierarchy/int-r3.cert.pem")
	test.AssertNotError(t, err, "Error reading ../test/hierarchy/int-r3.cert.pem")

	chainCrossPemBytes, err := os.ReadFile("../test/hierarchy/int-r3-cross.cert.pem")
	test.AssertNotError(t, err, "Error reading ../test/hierarchy/int-r3-cross.cert.pem")

	reqPath := fmt.Sprintf("/acme/cert/%s", core.SerialToString(cert.SerialNumber))
	pkixContent := "application/pem-certificate-chain"
	noCache := "public, max-age=0, no-cache"
	notFound := `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Certificate not found","status":404}`

	testCases := []struct {
		Name            string
		Request         *http.Request
		ExpectedStatus  int
		ExpectedHeaders map[string]string
		ExpectedLink    string
		ExpectedBody    string
		ExpectedCert    []byte
		AnyCert         bool
	}{
		{
			Name:           "Valid serial",
			Request:        makeGet(reqPath),
			ExpectedStatus: http.StatusOK,
			ExpectedHeaders: map[string]string{
				"Content-Type": pkixContent,
			},
			ExpectedCert: append(certPemBytes, append([]byte("\n"), chainPemBytes...)...),
			ExpectedLink: fmt.Sprintf(`<http://localhost%s/1>;rel="alternate"`, reqPath),
		},
		{
			Name:           "Valid serial, POST-as-GET",
			Request:        makePost(1, nil, reqPath, ""),
			ExpectedStatus: http.StatusOK,
			ExpectedHeaders: map[string]string{
				"Content-Type": pkixContent,
			},
			ExpectedCert: append(certPemBytes, append([]byte("\n"), chainPemBytes...)...),
		},
		{
			Name:           "Valid serial, bad POST-as-GET",
			Request:        makePost(1, nil, reqPath, "{}"),
			ExpectedStatus: http.StatusBadRequest,
			ExpectedBody: `{
				"type": "urn:ietf:params:acme:error:malformed",
				"status": 400,
				"detail": "POST-as-GET requests must have an empty payload"
			}`,
		},
		{
			Name:           "Valid serial, POST-as-GET from wrong account",
			Request:        makePost(2, altKey, reqPath, ""),
			ExpectedStatus: http.StatusForbidden,
			ExpectedBody: `{
				"type": "urn:ietf:params:acme:error:unauthorized",
				"status": 403,
				"detail": "Account in use did not issue specified certificate"
			}`,
		},
		{
			Name:           "Unused serial, no cache",
			Request:        makeGet("/acme/cert/000000000000000000000000000000000001"),
			ExpectedStatus: http.StatusNotFound,
			ExpectedBody:   notFound,
		},
		{
			Name:           "Invalid serial, no cache",
			Request:        makeGet("/acme/cert/nothex"),
			ExpectedStatus: http.StatusNotFound,
			ExpectedBody:   notFound,
		},
		{
			Name:           "Another invalid serial, no cache",
			Request:        makeGet("/acme/cert/00000000000000"),
			ExpectedStatus: http.StatusNotFound,
			ExpectedBody:   notFound,
		},
		{
			Name:           "Valid serial (explicit default chain)",
			Request:        makeGet(reqPath + "/0"),
			ExpectedStatus: http.StatusOK,
			ExpectedHeaders: map[string]string{
				"Content-Type": pkixContent,
			},
			ExpectedLink: fmt.Sprintf(`<http://localhost%s/1>;rel="alternate"`, reqPath),
			ExpectedCert: append(certPemBytes, append([]byte("\n"), chainPemBytes...)...),
		},
		{
			Name:           "Valid serial (explicit alternate chain)",
			Request:        makeGet(reqPath + "/1"),
			ExpectedStatus: http.StatusOK,
			ExpectedHeaders: map[string]string{
				"Content-Type": pkixContent,
			},
			ExpectedLink: fmt.Sprintf(`<http://localhost%s/0>;rel="alternate"`, reqPath),
			ExpectedCert: append(certPemBytes, append([]byte("\n"), chainCrossPemBytes...)...),
		},
		{
			Name:           "Valid serial (explicit non-existent alternate chain)",
			Request:        makeGet(reqPath + "/2"),
			ExpectedStatus: http.StatusNotFound,
			ExpectedBody:   `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Unknown issuance chain","status":404}`,
		},
		{
			Name:           "Valid serial (explicit negative alternate chain)",
			Request:        makeGet(reqPath + "/-1"),
			ExpectedStatus: http.StatusBadRequest,
			ExpectedBody:   `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Chain ID must be a non-negative integer","status":400}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			responseWriter := httptest.NewRecorder()
			mockLog := wfe.log.(*blog.Mock)
			mockLog.Clear()

			// Mux a request for a certificate
			mux.ServeHTTP(responseWriter, tc.Request)
			headers := responseWriter.Header()

			// Assert that the status code written is as expected
			test.AssertEquals(t, responseWriter.Code, tc.ExpectedStatus)

			// All of the responses should have the correct cache control header
			test.AssertEquals(t, headers.Get("Cache-Control"), noCache)

			// If the test cases expects additional headers, check those too
			for h, v := range tc.ExpectedHeaders {
				test.AssertEquals(t, headers.Get(h), v)
			}

			if tc.ExpectedLink != "" {
				found := false
				links := headers["Link"]
				for _, link := range links {
					if link == tc.ExpectedLink {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected link '%s', but did not find it in (%v)",
						tc.ExpectedLink, links)
				}
			}

			if tc.AnyCert { // Certificate is randomly generated, don't match it
				return
			}

			if len(tc.ExpectedCert) > 0 {
				// If the expectation was to return a certificate, check that it was the one expected
				bodyBytes := responseWriter.Body.Bytes()
				test.Assert(t, bytes.Equal(bodyBytes, tc.ExpectedCert), "Certificates don't match")

				// Successful requests should be logged as such
				reqlogs := mockLog.GetAllMatching(`INFO: [^ ]+ [^ ]+ [^ ]+ 200 .*`)
				if len(reqlogs) != 1 {
					t.Errorf("Didn't find info logs with code 200. Instead got:\n%s\n",
						strings.Join(mockLog.GetAllMatching(`.*`), "\n"))
				}
			} else {
				// Otherwise if the expectation wasn't a certificate, check that the body matches the expected
				body := responseWriter.Body.String()
				test.AssertUnmarshaledEquals(t, body, tc.ExpectedBody)

				// Unsuccessful requests should be logged as such
				reqlogs := mockLog.GetAllMatching(fmt.Sprintf(`INFO: [^ ]+ [^ ]+ [^ ]+ %d .*`, tc.ExpectedStatus))
				if len(reqlogs) != 1 {
					t.Errorf("Didn't find info logs with code %d. Instead got:\n%s\n",
						tc.ExpectedStatus, strings.Join(mockLog.GetAllMatching(`.*`), "\n"))
				}
			}
		})
	}
}

type mockSAWithNewCert struct {
	sapb.StorageAuthorityReadOnlyClient
	clk clock.Clock
}

func (sa *mockSAWithNewCert) GetCertificate(_ context.Context, req *sapb.Serial, _ ...grpc.CallOption) (*corepb.Certificate, error) {
	issuer, err := core.LoadCert("../test/hierarchy/int-e1.cert.pem")
	if err != nil {
		return nil, fmt.Errorf("failed to load test issuer cert: %w", err)
	}

	issuerKeyPem, err := os.ReadFile("../test/hierarchy/int-e1.key.pem")
	if err != nil {
		return nil, fmt.Errorf("failed to load test issuer key: %w", err)
	}
	issuerKey := loadKey(&testing.T{}, issuerKeyPem)

	newKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create test key: %w", err)
	}

	sn, err := core.StringToSerial(req.Serial)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: sn,
		DNSNames:     []string{"new.ee.boulder.test"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, issuer, &newKey.PublicKey, issuerKey)
	if err != nil {
		return nil, fmt.Errorf("failed to issue test cert: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test cert: %w", err)
	}

	return &corepb.Certificate{
		RegistrationID: 1,
		Serial:         core.SerialToString(cert.SerialNumber),
		Issued:         sa.clk.Now().Add(-1 * time.Second).UnixNano(),
		Der:            cert.Raw,
	}, nil
}

// TestGetCertificateNew tests for the case when the certificate is new (by
// dynamically generating it at test time), and therefore isn't served by the
// GET api.
func TestGetCertificateNew(t *testing.T) {
	wfe, fc, signer := setupWFE(t)
	wfe.sa = &mockSAWithNewCert{wfe.sa, fc}
	mux := wfe.Handler(metrics.NoopRegisterer)

	makeGet := func(path string) *http.Request {
		return &http.Request{URL: &url.URL{Path: path}, Method: "GET"}
	}

	makePost := func(keyID int64, key interface{}, path, body string) *http.Request {
		_, _, jwsBody := signer.byKeyID(keyID, key, fmt.Sprintf("http://localhost%s", path), body)
		return makePostRequestWithPath(path, jwsBody)
	}

	altKey := loadKey(t, []byte(test2KeyPrivatePEM))
	_, ok := altKey.(*rsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load RSA key")

	pkixContent := "application/pem-certificate-chain"
	noCache := "public, max-age=0, no-cache"

	testCases := []struct {
		Name            string
		Request         *http.Request
		ExpectedStatus  int
		ExpectedHeaders map[string]string
		ExpectedBody    string
	}{
		{
			Name:           "Get",
			Request:        makeGet("/get/cert/000000000000000000000000000000000001"),
			ExpectedStatus: http.StatusForbidden,
			ExpectedBody: `{
				"type": "` + probs.V2ErrorNS + `unauthorized",
				"detail": "Certificate is too new for GET API. You should only use this non-standard API to access resources created more than 10s ago",
				"status": 403
			}`,
		},
		{
			Name:           "ACME Get",
			Request:        makeGet("/acme/cert/000000000000000000000000000000000002"),
			ExpectedStatus: http.StatusOK,
			ExpectedHeaders: map[string]string{
				"Content-Type": pkixContent,
			},
		},
		{
			Name:           "ACME POST-as-GET",
			Request:        makePost(1, nil, "/acme/cert/000000000000000000000000000000000003", ""),
			ExpectedStatus: http.StatusOK,
			ExpectedHeaders: map[string]string{
				"Content-Type": pkixContent,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			responseWriter := httptest.NewRecorder()
			mockLog := wfe.log.(*blog.Mock)
			mockLog.Clear()

			// Mux a request for a certificate
			mux.ServeHTTP(responseWriter, tc.Request)
			headers := responseWriter.Header()

			// Assert that the status code written is as expected
			test.AssertEquals(t, responseWriter.Code, tc.ExpectedStatus)

			// All of the responses should have the correct cache control header
			test.AssertEquals(t, headers.Get("Cache-Control"), noCache)

			// If the test cases expects additional headers, check those too
			for h, v := range tc.ExpectedHeaders {
				test.AssertEquals(t, headers.Get(h), v)
			}

			// If we're expecting a particular body (because of an error), check that.
			if tc.ExpectedBody != "" {
				body := responseWriter.Body.String()
				test.AssertUnmarshaledEquals(t, body, tc.ExpectedBody)

				// Unsuccessful requests should be logged as such
				reqlogs := mockLog.GetAllMatching(fmt.Sprintf(`INFO: [^ ]+ [^ ]+ [^ ]+ %d .*`, tc.ExpectedStatus))
				if len(reqlogs) != 1 {
					t.Errorf("Didn't find info logs with code %d. Instead got:\n%s\n",
						tc.ExpectedStatus, strings.Join(mockLog.GetAllMatching(`.*`), "\n"))
				}
			}
		})
	}
}

// This uses httptest.NewServer because ServeMux.ServeHTTP won't prevent the
// body from being sent like the net/http Server's actually do.
func TestGetCertificateHEADHasCorrectBodyLength(t *testing.T) {
	wfe, _, _ := setupWFE(t)
	wfe.sa = newMockSAWithCert(t, wfe.sa, false)

	certPemBytes, _ := os.ReadFile("../test/hierarchy/ee-r3.cert.pem")
	cert, err := core.LoadCert("../test/hierarchy/ee-r3.cert.pem")
	test.AssertNotError(t, err, "failed to load test certificate")

	chainPemBytes, err := os.ReadFile("../test/hierarchy/int-r3.cert.pem")
	test.AssertNotError(t, err, "Error reading ../test/hierarchy/int-r3.cert.pem")
	chain := fmt.Sprintf("%s\n%s", string(certPemBytes), string(chainPemBytes))
	chainLen := strconv.Itoa(len(chain))

	mockLog := wfe.log.(*blog.Mock)
	mockLog.Clear()

	mux := wfe.Handler(metrics.NoopRegisterer)
	s := httptest.NewServer(mux)
	defer s.Close()
	req, _ := http.NewRequest(
		"HEAD", fmt.Sprintf("%s/acme/cert/%s", s.URL, core.SerialToString(cert.SerialNumber)), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		test.AssertNotError(t, err, "do error")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		test.AssertNotEquals(t, err, "readall error")
	}
	err = resp.Body.Close()
	if err != nil {
		test.AssertNotEquals(t, err, "readall error")
	}
	test.AssertEquals(t, resp.StatusCode, 200)
	test.AssertEquals(t, chainLen, resp.Header.Get("Content-Length"))
	test.AssertEquals(t, 0, len(body))
}

type mockSAWithError struct {
	sapb.StorageAuthorityReadOnlyClient
}

func (sa *mockSAWithError) GetCertificate(_ context.Context, req *sapb.Serial, _ ...grpc.CallOption) (*corepb.Certificate, error) {
	return nil, errors.New("Oops")
}

func TestGetCertificateServerError(t *testing.T) {
	// TODO: add tests for failure to parse the retrieved cert, a cert whose
	// IssuerNameID is unknown, and a cert whose signature can't be verified.
	wfe, _, _ := setupWFE(t)
	wfe.sa = &mockSAWithError{wfe.sa}
	mux := wfe.Handler(metrics.NoopRegisterer)

	cert, err := core.LoadCert("../test/hierarchy/ee-r3.cert.pem")
	test.AssertNotError(t, err, "failed to load test certificate")

	reqPath := fmt.Sprintf("/acme/cert/%s", core.SerialToString(cert.SerialNumber))
	req := &http.Request{URL: &url.URL{Path: reqPath}, Method: "GET"}

	// Mux a request for a certificate
	responseWriter := httptest.NewRecorder()
	mux.ServeHTTP(responseWriter, req)

	test.AssertEquals(t, responseWriter.Code, http.StatusInternalServerError)

	noCache := "public, max-age=0, no-cache"
	test.AssertEquals(t, responseWriter.Header().Get("Cache-Control"), noCache)

	body := `{
		"type": "urn:ietf:params:acme:error:serverInternal",
		"status": 500,
		"detail": "Failed to retrieve certificate"
	}`
	test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), body)
}

func newRequestEvent() *web.RequestEvent {
	return &web.RequestEvent{Extra: make(map[string]interface{})}
}

func TestHeaderBoulderRequester(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	mux := wfe.Handler(metrics.NoopRegisterer)
	responseWriter := httptest.NewRecorder()

	key := loadKey(t, []byte(test1KeyPrivatePEM))
	_, ok := key.(*rsa.PrivateKey)
	test.Assert(t, ok, "Failed to load test 1 RSA key")

	payload := `{}`
	path := fmt.Sprintf("%s%d", acctPath, 1)
	signedURL := fmt.Sprintf("http://localhost%s", path)
	_, _, body := signer.byKeyID(1, nil, signedURL, payload)
	request := makePostRequestWithPath(path, body)

	mux.ServeHTTP(responseWriter, request)
	test.AssertEquals(t, responseWriter.Header().Get("Boulder-Requester"), "1")

	// requests that do call sendError() also should have the requester header
	payload = `{"agreement":"https://letsencrypt.org/im-bad"}`
	_, _, body = signer.byKeyID(1, nil, signedURL, payload)
	request = makePostRequestWithPath(path, body)
	mux.ServeHTTP(responseWriter, request)
	test.AssertEquals(t, responseWriter.Header().Get("Boulder-Requester"), "1")
}

func TestDeactivateAuthorization(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	responseWriter := httptest.NewRecorder()

	responseWriter.Body.Reset()

	payload := `{"status":""}`
	_, _, body := signer.byKeyID(1, nil, "http://localhost/1", payload)
	request := makePostRequestWithPath("1", body)

	wfe.Authorization(ctx, newRequestEvent(), responseWriter, request)
	test.AssertUnmarshaledEquals(t,
		responseWriter.Body.String(),
		`{"type": "`+probs.V2ErrorNS+`malformed","detail": "Invalid status value","status": 400}`)

	responseWriter.Body.Reset()
	payload = `{"status":"deactivated"}`
	_, _, body = signer.byKeyID(1, nil, "http://localhost/1", payload)
	request = makePostRequestWithPath("1", body)

	wfe.Authorization(ctx, newRequestEvent(), responseWriter, request)
	test.AssertUnmarshaledEquals(t,
		responseWriter.Body.String(),
		`{
		  "identifier": {
		    "type": "dns",
		    "value": "not-an-example.com"
		  },
		  "status": "deactivated",
		  "expires": "2070-01-01T00:00:00Z",
		  "challenges": [
		    {
				"status": "pending",
			  "type": "dns",
			  "token":"token",
		      "url": "http://localhost/acme/chall-v3/1/-ZfxEw"
		    }
		  ]
		}`)
}

func TestDeactivateAccount(t *testing.T) {
	responseWriter := httptest.NewRecorder()
	wfe, _, signer := setupWFE(t)

	responseWriter.Body.Reset()
	payload := `{"status":"asd"}`
	signedURL := "http://localhost/1"
	path := "1"
	_, _, body := signer.byKeyID(1, nil, signedURL, payload)
	request := makePostRequestWithPath(path, body)

	wfe.Account(ctx, newRequestEvent(), responseWriter, request)
	test.AssertUnmarshaledEquals(t,
		responseWriter.Body.String(),
		`{"type": "`+probs.V2ErrorNS+`malformed","detail": "Invalid value provided for status field","status": 400}`)

	responseWriter.Body.Reset()
	payload = `{"status":"deactivated"}`
	_, _, body = signer.byKeyID(1, nil, signedURL, payload)
	request = makePostRequestWithPath(path, body)

	wfe.Account(ctx, newRequestEvent(), responseWriter, request)
	test.AssertUnmarshaledEquals(t,
		responseWriter.Body.String(),
		`{
		  "key": {
		    "kty": "RSA",
		    "n": "yNWVhtYEKJR21y9xsHV-PD_bYwbXSeNuFal46xYxVfRL5mqha7vttvjB_vc7Xg2RvgCxHPCqoxgMPTzHrZT75LjCwIW2K_klBYN8oYvTwwmeSkAz6ut7ZxPv-nZaT5TJhGk0NT2kh_zSpdriEJ_3vW-mqxYbbBmpvHqsa1_zx9fSuHYctAZJWzxzUZXykbWMWQZpEiE0J4ajj51fInEzVn7VxV-mzfMyboQjujPh7aNJxAWSq4oQEJJDgWwSh9leyoJoPpONHxh5nEE5AjE01FkGICSxjpZsF-w8hOTI3XXohUdu29Se26k2B0PolDSuj0GIQU6-W9TdLXSjBb2SpQ",
		    "e": "AQAB"
		  },
		  "contact": [
		    "mailto:person@mail.com"
		  ],
		  "initialIp": "",
		  "status": "deactivated"
		}`)

	responseWriter.Body.Reset()
	payload = `{"status":"deactivated", "contact":[]}`
	_, _, body = signer.byKeyID(1, nil, signedURL, payload)
	request = makePostRequestWithPath(path, body)
	wfe.Account(ctx, newRequestEvent(), responseWriter, request)
	test.AssertUnmarshaledEquals(t,
		responseWriter.Body.String(),
		`{
		  "key": {
		    "kty": "RSA",
		    "n": "yNWVhtYEKJR21y9xsHV-PD_bYwbXSeNuFal46xYxVfRL5mqha7vttvjB_vc7Xg2RvgCxHPCqoxgMPTzHrZT75LjCwIW2K_klBYN8oYvTwwmeSkAz6ut7ZxPv-nZaT5TJhGk0NT2kh_zSpdriEJ_3vW-mqxYbbBmpvHqsa1_zx9fSuHYctAZJWzxzUZXykbWMWQZpEiE0J4ajj51fInEzVn7VxV-mzfMyboQjujPh7aNJxAWSq4oQEJJDgWwSh9leyoJoPpONHxh5nEE5AjE01FkGICSxjpZsF-w8hOTI3XXohUdu29Se26k2B0PolDSuj0GIQU6-W9TdLXSjBb2SpQ",
		    "e": "AQAB"
		  },
		  "contact": [
		    "mailto:person@mail.com"
		  ],
		  "initialIp": "",
		  "status": "deactivated"
		}`)

	responseWriter.Body.Reset()
	key := loadKey(t, []byte(test3KeyPrivatePEM))
	_, ok := key.(*rsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load test3 RSA key")

	payload = `{"status":"deactivated"}`
	path = "3"
	signedURL = "http://localhost/3"
	_, _, body = signer.byKeyID(3, key, signedURL, payload)
	request = makePostRequestWithPath(path, body)

	wfe.Account(ctx, newRequestEvent(), responseWriter, request)

	test.AssertUnmarshaledEquals(t,
		responseWriter.Body.String(),
		`{
		  "type": "`+probs.V2ErrorNS+`unauthorized",
		  "detail": "Account is not valid, has status \"deactivated\"",
		  "status": 403
		}`)
}

func TestNewOrder(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	responseWriter := httptest.NewRecorder()

	targetHost := "localhost"
	targetPath := "new-order"
	signedURL := fmt.Sprintf("http://%s/%s", targetHost, targetPath)

	nonDNSIdentifierBody := `
	{
		"Identifiers": [
		  {"type": "dns",    "value": "not-example.com"},
			{"type": "dns",    "value": "www.not-example.com"},
			{"type": "fakeID", "value": "www.i-am-21.com"}
		]
	}
	`

	validOrderBody := `
	{
		"Identifiers": [
		  {"type": "dns", "value": "not-example.com"},
			{"type": "dns", "value": "www.not-example.com"}
		]
	}`

	// Body with a SAN that is longer than 64 bytes. This one is 65 bytes.
	tooLongCNBody := `
	{
		"Identifiers": [
			{
				"type": "dns",
				"value": "thisreallylongexampledomainisabytelongerthanthemaxcnbytelimit.com"
			}
		]
	}`

	oneLongOneShortCNBody := `
	{
		"Identifiers": [
			{
				"type": "dns",
				"value": "thisreallylongexampledomainisabytelongerthanthemaxcnbytelimit.com"
			},
			{
				"type": "dns",
				"value": "not-example.com"
			}
		]
	}`

	testCases := []struct {
		Name            string
		Request         *http.Request
		ExpectedBody    string
		ExpectedHeaders map[string]string
	}{
		{
			Name: "POST, but no body",
			Request: &http.Request{
				Method: "POST",
				Header: map[string][]string{
					"Content-Length": {"0"},
					"Content-Type":   {expectedJWSContentType},
				},
			},
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"No body on POST","status":400}`,
		},
		{
			Name:         "POST, with an invalid JWS body",
			Request:      makePostRequestWithPath("hi", "hi"),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Parse error reading JWS","status":400}`,
		},
		{
			Name:         "POST, properly signed JWS, payload isn't valid",
			Request:      signAndPost(signer, targetPath, signedURL, "foo"),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Request payload did not parse as JSON","status":400}`,
		},
		{
			Name:         "POST, empty domain name identifier",
			Request:      signAndPost(signer, targetPath, signedURL, `{"identifiers":[{"type":"dns","value":""}]}`),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"NewOrder request included empty domain name","status":400}`,
		},
		{
			Name:         "POST, no identifiers in payload",
			Request:      signAndPost(signer, targetPath, signedURL, "{}"),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"NewOrder request did not specify any identifiers","status":400}`,
		},
		{
			Name:         "POST, invalid identifier in payload",
			Request:      signAndPost(signer, targetPath, signedURL, nonDNSIdentifierBody),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"NewOrder request included invalid non-DNS type identifier: type \"fakeID\", value \"www.i-am-21.com\"","status":400}`,
		},
		{
			Name:         "POST, notAfter and notBefore in payload",
			Request:      signAndPost(signer, targetPath, signedURL, `{"identifiers":[{"type": "dns", "value": "not-example.com"}], "notBefore":"now", "notAfter": "later"}`),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"NotBefore and NotAfter are not supported","status":400}`,
		},
		{
			Name:         "POST, no potential CNs 64 bytes or smaller",
			Request:      signAndPost(signer, targetPath, signedURL, tooLongCNBody),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `rejectedIdentifier","detail":"NewOrder request did not include a SAN short enough to fit in CN","status":400}`,
		},
		{
			Name:    "POST, good payload, one potential CNs less than 64 bytes and one longer",
			Request: signAndPost(signer, targetPath, signedURL, oneLongOneShortCNBody),
			ExpectedBody: `
			{
				"status": "pending",
				"expires": "2021-02-01T01:01:01Z",
				"identifiers": [
					{ "type": "dns", "value": "thisreallylongexampledomainisabytelongerthanthemaxcnbytelimit.com"},
					{ "type": "dns", "value": "not-example.com"}
				],
				"authorizations": [
					"http://localhost/acme/authz-v3/1"
				],
				"finalize": "http://localhost/acme/finalize/1/1"
			}`,
		},
		{
			Name:    "POST, good payload",
			Request: signAndPost(signer, targetPath, signedURL, validOrderBody),
			ExpectedBody: `
					{
						"status": "pending",
						"expires": "2021-02-01T01:01:01Z",
						"identifiers": [
							{ "type": "dns", "value": "not-example.com"},
							{ "type": "dns", "value": "www.not-example.com"}
						],
						"authorizations": [
							"http://localhost/acme/authz-v3/1"
						],
						"finalize": "http://localhost/acme/finalize/1/1"
					}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			responseWriter.Body.Reset()

			wfe.NewOrder(ctx, newRequestEvent(), responseWriter, tc.Request)
			test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), tc.ExpectedBody)

			headers := responseWriter.Header()
			for k, v := range tc.ExpectedHeaders {
				test.AssertEquals(t, headers.Get(k), v)
			}
		})
	}

	// Test that we log the "Created" field.
	responseWriter.Body.Reset()
	request := signAndPost(signer, targetPath, signedURL, validOrderBody)
	requestEvent := newRequestEvent()
	wfe.NewOrder(ctx, requestEvent, responseWriter, request)

	if requestEvent.Created != "1" {
		t.Errorf("Expected to log Created field when creating Order: %#v", requestEvent)
	}
}

func TestFinalizeOrder(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	responseWriter := httptest.NewRecorder()

	targetHost := "localhost"
	targetPath := "1/1"
	signedURL := fmt.Sprintf("http://%s/%s", targetHost, targetPath)

	// openssl req -outform der -new -nodes -key wfe/test/178.key -subj /CN=not-an-example.com | b64url
	// a valid CSR
	goodCertCSRPayload := `{
		"csr": "MIICYjCCAUoCAQAwHTEbMBkGA1UEAwwSbm90LWFuLWV4YW1wbGUuY29tMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAmqs7nue5oFxKBk2WaFZJAma2nm1oFyPIq19gYEAdQN4mWvaJ8RjzHFkDMYUrlIrGxCYuFJDHFUk9dh19Na1MIY-NVLgcSbyNcOML3bLbLEwGmvXPbbEOflBA9mxUS9TLMgXW5ghf_qbt4vmSGKloIim41QXt55QFW6O-84s8Kd2OE6df0wTsEwLhZB3j5pDU-t7j5vTMv4Tc7EptaPkOdfQn-68viUJjlYM_4yIBVRhWCdexFdylCKVLg0obsghQEwULKYCUjdg6F0VJUI115DU49tzscXU_3FS3CyY8rchunuYszBNkdmgpAwViHNWuP7ESdEd_emrj1xuioSe6PwIDAQABoAAwDQYJKoZIhvcNAQELBQADggEBAE_T1nWU38XVYL28hNVSXU0rW5IBUKtbvr0qAkD4kda4HmQRTYkt-LNSuvxoZCC9lxijjgtJi-OJe_DCTdZZpYzewlVvcKToWSYHYQ6Wm1-fxxD_XzphvZOujpmBySchdiz7QSVWJmVZu34XD5RJbIcrmj_cjRt42J1hiTFjNMzQu9U6_HwIMmliDL-soFY2RTvvZf-dAFvOUQ-Wbxt97eM1PbbmxJNWRhbAmgEpe9PWDPTpqV5AK56VAa991cQ1P8ZVmPss5hvwGWhOtpnpTZVHN3toGNYFKqxWPboirqushQlfKiFqT9rpRgM3-mFjOHidGqsKEkTdmfSVlVEk3oo="
	}`

	egUrl := mustParseURL("1/1")

	testCases := []struct {
		Name            string
		Request         *http.Request
		ExpectedHeaders map[string]string
		ExpectedBody    string
	}{
		{
			Name: "POST, but no body",
			Request: &http.Request{
				URL:        egUrl,
				RequestURI: targetPath,
				Method:     "POST",
				Header: map[string][]string{
					"Content-Length": {"0"},
					"Content-Type":   {expectedJWSContentType},
				},
			},
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"No body on POST","status":400}`,
		},
		{
			Name:         "POST, with an invalid JWS body",
			Request:      makePostRequestWithPath(targetPath, "hi"),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Parse error reading JWS","status":400}`,
		},
		{
			Name:         "POST, properly signed JWS, payload isn't valid",
			Request:      signAndPost(signer, targetPath, signedURL, "foo"),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Request payload did not parse as JSON","status":400}`,
		},
		{
			Name:         "Invalid path",
			Request:      signAndPost(signer, "1", "http://localhost/1", "{}"),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Invalid request path","status":404}`,
		},
		{
			Name:         "Bad acct ID in path",
			Request:      signAndPost(signer, "a/1", "http://localhost/a/1", "{}"),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Invalid account ID","status":400}`,
		},
		{
			Name: "Mismatched acct ID in path/JWS",
			// Note(@cpu): We use "http://localhost/2/1" here not
			// "http://localhost/order/2/1" because we are calling the Order
			// handler directly and it normally has the initial path component
			// stripped by the global WFE2 handler. We need the JWS URL to match the request
			// URL so we fudge both such that the finalize-order prefix has been removed.
			Request:      signAndPost(signer, "2/1", "http://localhost/2/1", "{}"),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"No order found for account ID 2","status":404}`,
		},
		{
			Name:         "Order ID is invalid",
			Request:      signAndPost(signer, "1/okwhatever/finalize-order", "http://localhost/1/okwhatever/finalize-order", "{}"),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Invalid order ID","status":400}`,
		},
		{
			Name: "Order doesn't exist",
			// mocks/mocks.go's StorageAuthority's GetOrder mock treats ID 2 as missing
			Request:      signAndPost(signer, "1/2", "http://localhost/1/2", "{}"),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"No order for ID 2","status":404}`,
		},
		{
			Name: "Order is already finalized",
			// mocks/mocks.go's StorageAuthority's GetOrder mock treats ID 1 as an Order with a Serial
			Request:      signAndPost(signer, "1/1", "http://localhost/1/1", goodCertCSRPayload),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `orderNotReady","detail":"Order's status (\"valid\") is not acceptable for finalization","status":403}`,
		},
		{
			Name: "Order is expired",
			// mocks/mocks.go's StorageAuthority's GetOrder mock treats ID 7 as an Order that has already expired
			Request:      signAndPost(signer, "1/7", "http://localhost/1/7", goodCertCSRPayload),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Order 7 is expired","status":404}`,
		},
		{
			Name:         "Good CSR, Pending Order",
			Request:      signAndPost(signer, "1/4", "http://localhost/1/4", goodCertCSRPayload),
			ExpectedBody: `{"type":"` + probs.V2ErrorNS + `orderNotReady","detail":"Order's status (\"pending\") is not acceptable for finalization","status":403}`,
		},
		{
			Name:            "Good CSR, Ready Order",
			Request:         signAndPost(signer, "1/8", "http://localhost/1/8", goodCertCSRPayload),
			ExpectedHeaders: map[string]string{"Location": "http://localhost/acme/order/1/8"},
			ExpectedBody: `
{
  "status": "processing",
  "expires": "1970-01-01T00:00:00.9466848Z",
  "identifiers": [
    {"type":"dns","value":"example.com"}
  ],
  "authorizations": [
    "http://localhost/acme/authz-v3/1"
  ],
  "finalize": "http://localhost/acme/finalize/1/8"
}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			responseWriter.Body.Reset()
			wfe.FinalizeOrder(ctx, newRequestEvent(), responseWriter, tc.Request)
			for k, v := range tc.ExpectedHeaders {
				got := responseWriter.Header().Get(k)
				if v != got {
					t.Errorf("Header %q: Expected %q, got %q", k, v, got)
				}
			}
			test.AssertUnmarshaledEquals(t, responseWriter.Body.String(),
				tc.ExpectedBody)
		})
	}

	// Check a bad CSR request separately from the above testcases. We don't want
	// to match the whole response body because the "detail" of a bad CSR problem
	// contains a verbose Go error message that can change between versions (e.g.
	// Go 1.10.4 to 1.11 changed the expected format)
	badCSRReq := signAndPost(signer, "1/8", "http://localhost/1/8", `{"CSR": "ABCD"}`)
	responseWriter.Body.Reset()
	wfe.FinalizeOrder(ctx, newRequestEvent(), responseWriter, badCSRReq)
	responseBody := responseWriter.Body.String()
	test.AssertContains(t, responseBody, "Error parsing certificate request")
}

func TestKeyRollover(t *testing.T) {
	responseWriter := httptest.NewRecorder()
	wfe, _, signer := setupWFE(t)

	existingKey, err := rsa.GenerateKey(rand.Reader, 2048)
	test.AssertNotError(t, err, "Error creating random 2048 RSA key")

	newKeyBytes, err := os.ReadFile("../test/test-key-5.der")
	test.AssertNotError(t, err, "Failed to read ../test/test-key-5.der")
	newKeyPriv, err := x509.ParsePKCS1PrivateKey(newKeyBytes)
	test.AssertNotError(t, err, "Failed parsing private key")
	newJWKJSON, err := jose.JSONWebKey{Key: newKeyPriv.Public()}.MarshalJSON()
	test.AssertNotError(t, err, "Failed to marshal JWK JSON")

	wfe.KeyRollover(ctx, newRequestEvent(), responseWriter, makePostRequestWithPath("", "{}"))
	test.AssertUnmarshaledEquals(t,
		responseWriter.Body.String(),
		`{
		  "type": "`+probs.V2ErrorNS+`malformed",
		  "detail": "Parse error reading JWS",
		  "status": 400
		}`)

	testCases := []struct {
		Name             string
		Payload          string
		ExpectedResponse string
		NewKey           crypto.Signer
		ErrorStatType    string
	}{
		{
			Name:    "Missing account URL",
			Payload: `{"oldKey":` + test1KeyPublicJSON + `}`,
			ExpectedResponse: `{
		     "type": "` + probs.V2ErrorNS + `malformed",
		     "detail": "Inner key rollover request specified Account \"\", but outer JWS has Key ID \"http://localhost/acme/acct/1\"",
		     "status": 400
		   }`,
			NewKey:        newKeyPriv,
			ErrorStatType: "KeyRolloverMismatchedAccount",
		},
		{
			Name:    "incorrect old key",
			Payload: `{"oldKey":` + string(newJWKJSON) + `,"account":"http://localhost/acme/acct/1"}`,
			ExpectedResponse: `{
		     "type": "` + probs.V2ErrorNS + `malformed",
		     "detail": "Inner JWS does not contain old key field matching current account key",
		     "status": 400
		   }`,
			NewKey:        newKeyPriv,
			ErrorStatType: "KeyRolloverWrongOldKey",
		},
		{
			Name:    "Valid key rollover request, key exists",
			Payload: `{"oldKey":` + test1KeyPublicJSON + `,"account":"http://localhost/acme/acct/1"}`,
			ExpectedResponse: `{
                          "type": "urn:ietf:params:acme:error:malformed",
                          "detail": "New key is already in use for a different account",
                          "status": 409
                        }`,
			NewKey: existingKey,
		},
		{
			Name:    "Valid key rollover request",
			Payload: `{"oldKey":` + test1KeyPublicJSON + `,"account":"http://localhost/acme/acct/1"}`,
			ExpectedResponse: `{
		     "key": ` + string(newJWKJSON) + `,
		     "contact": [
		       "mailto:person@mail.com"
		     ],
		     "initialIp": "",
		     "status": "valid"
		   }`,
			NewKey: newKeyPriv,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			wfe.stats.joseErrorCount.Reset()
			responseWriter.Body.Reset()
			_, _, inner := signer.embeddedJWK(tc.NewKey, "http://localhost/key-change", tc.Payload)
			_, _, outer := signer.byKeyID(1, nil, "http://localhost/key-change", inner)
			wfe.KeyRollover(ctx, newRequestEvent(), responseWriter, makePostRequestWithPath("key-change", outer))
			test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), tc.ExpectedResponse)
			if tc.ErrorStatType != "" {
				test.AssertMetricWithLabelsEquals(
					t, wfe.stats.joseErrorCount, prometheus.Labels{"type": tc.ErrorStatType}, 1)
			}
		})
	}
}

func TestKeyRolloverMismatchedJWSURLs(t *testing.T) {
	responseWriter := httptest.NewRecorder()
	wfe, _, signer := setupWFE(t)

	newKeyBytes, err := os.ReadFile("../test/test-key-5.der")
	test.AssertNotError(t, err, "Failed to read ../test/test-key-5.der")
	newKeyPriv, err := x509.ParsePKCS1PrivateKey(newKeyBytes)
	test.AssertNotError(t, err, "Failed parsing private key")

	_, _, inner := signer.embeddedJWK(newKeyPriv, "http://localhost/wrong-url", "{}")
	_, _, outer := signer.byKeyID(1, nil, "http://localhost/key-change", inner)
	wfe.KeyRollover(ctx, newRequestEvent(), responseWriter, makePostRequestWithPath("key-change", outer))
	test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), `
		{
			"type": "urn:ietf:params:acme:error:malformed",
			"detail": "Outer JWS 'url' value \"http://localhost/key-change\" does not match inner JWS 'url' value \"http://localhost/wrong-url\"",
			"status": 400
		}`)
}

func TestGetOrder(t *testing.T) {
	wfe, _, signer := setupWFE(t)

	makeGet := func(path string) *http.Request {
		return &http.Request{URL: &url.URL{Path: path}, Method: "GET"}
	}

	makePost := func(keyID int64, path, body string) *http.Request {
		_, _, jwsBody := signer.byKeyID(keyID, nil, fmt.Sprintf("http://localhost/%s", path), body)
		return makePostRequestWithPath(path, jwsBody)
	}

	testCases := []struct {
		Name     string
		Request  *http.Request
		Response string
		Endpoint string
	}{
		{
			Name:     "Good request",
			Request:  makeGet("1/1"),
			Response: `{"status": "valid","expires": "1970-01-01T00:00:00.9466848Z","identifiers":[{"type":"dns", "value":"example.com"}], "authorizations":["http://localhost/acme/authz-v3/1"],"finalize":"http://localhost/acme/finalize/1/1","certificate":"http://localhost/acme/cert/serial"}`,
		},
		{
			Name:     "404 request",
			Request:  makeGet("1/2"),
			Response: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"No order for ID 2", "status":404}`,
		},
		{
			Name:     "Invalid request path",
			Request:  makeGet("asd"),
			Response: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Invalid request path","status":404}`,
		},
		{
			Name:     "Invalid account ID",
			Request:  makeGet("asd/asd"),
			Response: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Invalid account ID","status":400}`,
		},
		{
			Name:     "Invalid order ID",
			Request:  makeGet("1/asd"),
			Response: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"Invalid order ID","status":400}`,
		},
		{
			Name:     "Real request, wrong account",
			Request:  makeGet("2/1"),
			Response: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"No order found for account ID 2", "status":404}`,
		},
		{
			Name:     "Internal error request",
			Request:  makeGet("1/3"),
			Response: `{"type":"` + probs.V2ErrorNS + `serverInternal","detail":"Failed to retrieve order for ID 3","status":500}`,
		},
		{
			Name:     "Invalid POST-as-GET",
			Request:  makePost(1, "1/1", "{}"),
			Response: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"POST-as-GET requests must have an empty payload", "status":400}`,
		},
		{
			Name:     "Valid POST-as-GET, wrong account",
			Request:  makePost(1, "2/1", ""),
			Response: `{"type":"` + probs.V2ErrorNS + `malformed","detail":"No order found for account ID 2", "status":404}`,
		},
		{
			Name:     "Valid POST-as-GET",
			Request:  makePost(1, "1/1", ""),
			Response: `{"status": "valid","expires": "1970-01-01T00:00:00.9466848Z","identifiers":[{"type":"dns", "value":"example.com"}], "authorizations":["http://localhost/acme/authz-v3/1"],"finalize":"http://localhost/acme/finalize/1/1","certificate":"http://localhost/acme/cert/serial"}`,
		},
		{
			Name:     "GET new order",
			Request:  makeGet("1/9"),
			Response: `{"type":"` + probs.V2ErrorNS + `unauthorized","detail":"Order is too new for GET API. You should only use this non-standard API to access resources created more than 10s ago","status":403}`,
			Endpoint: "/get/order/",
		},
		{
			Name:     "GET new order from old endpoint",
			Request:  makeGet("1/9"),
			Response: `{"status": "valid","expires": "1970-01-01T00:00:00.9466848Z","identifiers":[{"type":"dns", "value":"example.com"}], "authorizations":["http://localhost/acme/authz-v3/1"],"finalize":"http://localhost/acme/finalize/1/9","certificate":"http://localhost/acme/cert/serial"}`,
		},
		{
			Name:     "POST-as-GET new order",
			Request:  makePost(1, "1/9", ""),
			Response: `{"status": "valid","expires": "1970-01-01T00:00:00.9466848Z","identifiers":[{"type":"dns", "value":"example.com"}], "authorizations":["http://localhost/acme/authz-v3/1"],"finalize":"http://localhost/acme/finalize/1/9","certificate":"http://localhost/acme/cert/serial"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			responseWriter := httptest.NewRecorder()
			if tc.Endpoint != "" {
				wfe.GetOrder(ctx, &web.RequestEvent{Extra: make(map[string]interface{}), Endpoint: tc.Endpoint}, responseWriter, tc.Request)
			} else {
				wfe.GetOrder(ctx, newRequestEvent(), responseWriter, tc.Request)
			}
			test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), tc.Response)
		})
	}
}

func makeRevokeRequestJSON(reason *revocation.Reason) ([]byte, error) {
	certPemBytes, err := os.ReadFile("../test/hierarchy/ee-r3.cert.pem")
	if err != nil {
		return nil, err
	}
	certBlock, _ := pem.Decode(certPemBytes)
	if err != nil {
		return nil, err
	}
	return makeRevokeRequestJSONForCert(certBlock.Bytes, reason)
}

func makeRevokeRequestJSONForCert(der []byte, reason *revocation.Reason) ([]byte, error) {
	revokeRequest := struct {
		CertificateDER core.JSONBuffer    `json:"certificate"`
		Reason         *revocation.Reason `json:"reason"`
	}{
		CertificateDER: der,
		Reason:         reason,
	}
	revokeRequestJSON, err := json.Marshal(revokeRequest)
	if err != nil {
		return nil, err
	}
	return revokeRequestJSON, nil
}

// Valid revocation request for existing, non-revoked cert, signed using the
// issuing account key.
func TestRevokeCertificateByApplicantValid(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	wfe.sa = newMockSAWithCert(t, wfe.sa, false)

	mockLog := wfe.log.(*blog.Mock)
	mockLog.Clear()

	revokeRequestJSON, err := makeRevokeRequestJSON(nil)
	test.AssertNotError(t, err, "Failed to make revokeRequestJSON")
	_, _, jwsBody := signer.byKeyID(1, nil, "http://localhost/revoke-cert", string(revokeRequestJSON))

	responseWriter := httptest.NewRecorder()
	wfe.RevokeCertificate(ctx, newRequestEvent(), responseWriter,
		makePostRequestWithPath("revoke-cert", jwsBody))

	test.AssertEquals(t, responseWriter.Code, 200)
	test.AssertEquals(t, responseWriter.Body.String(), "")
	test.AssertDeepEquals(t, mockLog.GetAllMatching("Authenticated revocation"), []string{
		`INFO: [AUDIT] Authenticated revocation JSON={"Serial":"000000000000000000001d72443db5189821","Reason":0,"RegID":1,"Method":"applicant"}`,
	})
}

// Valid revocation request for existing, non-revoked cert, signed using the
// certificate private key.
func TestRevokeCertificateByKeyValid(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	wfe.sa = newMockSAWithCert(t, wfe.sa, false)

	mockLog := wfe.log.(*blog.Mock)
	mockLog.Clear()

	keyPemBytes, err := os.ReadFile("../test/hierarchy/ee-r3.key.pem")
	test.AssertNotError(t, err, "Failed to load key")
	key := loadKey(t, keyPemBytes)

	revocationReason := revocation.Reason(ocsp.KeyCompromise)
	revokeRequestJSON, err := makeRevokeRequestJSON(&revocationReason)
	test.AssertNotError(t, err, "Failed to make revokeRequestJSON")
	_, _, jwsBody := signer.embeddedJWK(key, "http://localhost/revoke-cert", string(revokeRequestJSON))

	responseWriter := httptest.NewRecorder()
	wfe.RevokeCertificate(ctx, newRequestEvent(), responseWriter,
		makePostRequestWithPath("revoke-cert", jwsBody))

	test.AssertEquals(t, responseWriter.Code, 200)
	test.AssertEquals(t, responseWriter.Body.String(), "")
	test.AssertDeepEquals(t, mockLog.GetAllMatching("Authenticated revocation"), []string{
		`INFO: [AUDIT] Authenticated revocation JSON={"Serial":"000000000000000000001d72443db5189821","Reason":1,"RegID":0,"Method":"privkey"}`,
	})
}

// Invalid revocation request: although signed with the cert key, the cert
// wasn't issued by any issuer the Boulder is aware of.
func TestRevokeCertificateNotIssued(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	wfe.sa = newMockSAWithCert(t, wfe.sa, false)

	// Make a self-signed junk certificate
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	test.AssertNotError(t, err, "unexpected error making random private key")
	// Use a known serial from the mockSAWithValidCert mock.
	// This ensures that any failures here are due to the certificate's issuer
	// not matching up with issuers known by the mock, rather than due to the
	// certificate's serial not matching up with serials known by the mock.
	knownCert, err := core.LoadCert("../test/hierarchy/ee-r3.cert.pem")
	test.AssertNotError(t, err, "Unexpected error loading test cert")
	template := &x509.Certificate{
		SerialNumber: knownCert.SerialNumber,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, k.Public(), k)
	test.AssertNotError(t, err, "Unexpected error creating self-signed junk cert")

	keyPemBytes, err := os.ReadFile("../test/hierarchy/ee-r3.key.pem")
	test.AssertNotError(t, err, "Failed to load key")
	key := loadKey(t, keyPemBytes)

	revokeRequestJSON, err := makeRevokeRequestJSONForCert(certDER, nil)
	test.AssertNotError(t, err, "Failed to make revokeRequestJSON for certDER")
	_, _, jwsBody := signer.embeddedJWK(key, "http://localhost/revoke-cert", string(revokeRequestJSON))

	responseWriter := httptest.NewRecorder()
	wfe.RevokeCertificate(ctx, newRequestEvent(), responseWriter,
		makePostRequestWithPath("revoke-cert", jwsBody))
	// It should result in a 404 response with a problem body
	test.AssertEquals(t, responseWriter.Code, 404)
	test.AssertEquals(t, responseWriter.Body.String(), "{\n  \"type\": \"urn:ietf:params:acme:error:malformed\",\n  \"detail\": \"Certificate from unrecognized issuer\",\n  \"status\": 404\n}")
}

func TestRevokeCertificateExpired(t *testing.T) {
	wfe, fc, signer := setupWFE(t)
	wfe.sa = newMockSAWithCert(t, wfe.sa, false)

	keyPemBytes, err := os.ReadFile("../test/hierarchy/ee-r3.key.pem")
	test.AssertNotError(t, err, "Failed to load key")
	key := loadKey(t, keyPemBytes)

	revokeRequestJSON, err := makeRevokeRequestJSON(nil)
	test.AssertNotError(t, err, "Failed to make revokeRequestJSON")

	_, _, jwsBody := signer.embeddedJWK(key, "http://localhost/revoke-cert", string(revokeRequestJSON))

	cert, err := core.LoadCert("../test/hierarchy/ee-r3.cert.pem")
	test.AssertNotError(t, err, "Failed to load test certificate")

	fc.Set(cert.NotAfter.Add(time.Hour))

	responseWriter := httptest.NewRecorder()
	wfe.RevokeCertificate(ctx, newRequestEvent(), responseWriter,
		makePostRequestWithPath("revoke-cert", jwsBody))
	test.AssertEquals(t, responseWriter.Code, 403)
	test.AssertEquals(t, responseWriter.Body.String(), "{\n  \"type\": \"urn:ietf:params:acme:error:unauthorized\",\n  \"detail\": \"Certificate is expired\",\n  \"status\": 403\n}")
}

func TestRevokeCertificateReasons(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	wfe.sa = newMockSAWithCert(t, wfe.sa, false)
	ra := wfe.ra.(*MockRegistrationAuthority)

	reason0 := revocation.Reason(ocsp.Unspecified)
	reason1 := revocation.Reason(ocsp.KeyCompromise)
	reason2 := revocation.Reason(ocsp.CACompromise)
	reason100 := revocation.Reason(100)

	testCases := []struct {
		Name             string
		Reason           *revocation.Reason
		ExpectedHTTPCode int
		ExpectedBody     string
		ExpectedReason   *revocation.Reason
	}{
		{
			Name:             "Valid reason",
			Reason:           &reason1,
			ExpectedHTTPCode: http.StatusOK,
			ExpectedReason:   &reason1,
		},
		{
			Name:             "No reason",
			ExpectedHTTPCode: http.StatusOK,
			ExpectedReason:   &reason0,
		},
		{
			Name:             "Unsupported reason",
			Reason:           &reason2,
			ExpectedHTTPCode: http.StatusBadRequest,
			ExpectedBody:     `{"type":"` + probs.V2ErrorNS + `badRevocationReason","detail":"unsupported revocation reason code provided: cACompromise (2). Supported reasons: unspecified (0), keyCompromise (1), superseded (4), cessationOfOperation (5)","status":400}`,
		},
		{
			Name:             "Non-existent reason",
			Reason:           &reason100,
			ExpectedHTTPCode: http.StatusBadRequest,
			ExpectedBody:     `{"type":"` + probs.V2ErrorNS + `badRevocationReason","detail":"unsupported revocation reason code provided: unknown (100). Supported reasons: unspecified (0), keyCompromise (1), superseded (4), cessationOfOperation (5)","status":400}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			revokeRequestJSON, err := makeRevokeRequestJSON(tc.Reason)
			test.AssertNotError(t, err, "Failed to make revokeRequestJSON")
			_, _, jwsBody := signer.byKeyID(1, nil, "http://localhost/revoke-cert", string(revokeRequestJSON))

			responseWriter := httptest.NewRecorder()
			wfe.RevokeCertificate(ctx, newRequestEvent(), responseWriter,
				makePostRequestWithPath("revoke-cert", jwsBody))

			test.AssertEquals(t, responseWriter.Code, tc.ExpectedHTTPCode)
			if tc.ExpectedBody != "" {
				test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), tc.ExpectedBody)
			} else {
				test.AssertEquals(t, responseWriter.Body.String(), tc.ExpectedBody)
			}
			if tc.ExpectedReason != nil {
				test.AssertEquals(t, ra.lastRevocationReason, *tc.ExpectedReason)
			}
		})
	}
}

// A revocation request signed by an incorrect certificate private key.
func TestRevokeCertificateWrongCertificateKey(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	wfe.sa = newMockSAWithCert(t, wfe.sa, false)

	keyPemBytes, err := os.ReadFile("../test/hierarchy/ee-e1.key.pem")
	test.AssertNotError(t, err, "Failed to load key")
	key := loadKey(t, keyPemBytes)

	revocationReason := revocation.Reason(ocsp.KeyCompromise)
	revokeRequestJSON, err := makeRevokeRequestJSON(&revocationReason)
	test.AssertNotError(t, err, "Failed to make revokeRequestJSON")
	_, _, jwsBody := signer.embeddedJWK(key, "http://localhost/revoke-cert", string(revokeRequestJSON))

	responseWriter := httptest.NewRecorder()
	wfe.RevokeCertificate(ctx, newRequestEvent(), responseWriter,
		makePostRequestWithPath("revoke-cert", jwsBody))
	test.AssertEquals(t, responseWriter.Code, 403)
	test.AssertUnmarshaledEquals(t, responseWriter.Body.String(),
		`{"type":"`+probs.V2ErrorNS+`unauthorized","detail":"JWK embedded in revocation request must be the same public key as the cert to be revoked","status":403}`)
}

type mockSAGetRegByKeyFails struct {
	sapb.StorageAuthorityReadOnlyClient
}

func (sa *mockSAGetRegByKeyFails) GetRegistrationByKey(_ context.Context, req *sapb.JSONWebKey, _ ...grpc.CallOption) (*corepb.Registration, error) {
	return nil, fmt.Errorf("whoops")
}

// When SA.GetRegistrationByKey errors (e.g. gRPC timeout), NewAccount should
// return internal server errors.
func TestNewAccountWhenGetRegByKeyFails(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	wfe.sa = &mockSAGetRegByKeyFails{wfe.sa}
	key := loadKey(t, []byte(testE2KeyPrivatePEM))
	_, ok := key.(*ecdsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load ECDSA key")
	payload := `{"contact":["mailto:person@mail.com"],"agreement":"` + agreementURL + `"}`
	responseWriter := httptest.NewRecorder()
	_, _, body := signer.embeddedJWK(key, "http://localhost/new-account", payload)
	wfe.NewAccount(ctx, newRequestEvent(), responseWriter, makePostRequestWithPath("/new-account", body))
	if responseWriter.Code != 500 {
		t.Fatalf("Wrong response code %d for NewAccount with failing GetRegByKey (wanted 500)", responseWriter.Code)
	}
	var prob probs.ProblemDetails
	err := json.Unmarshal(responseWriter.Body.Bytes(), &prob)
	test.AssertNotError(t, err, "unmarshalling response")
	if prob.Type != probs.V2ErrorNS+probs.ServerInternalProblem {
		t.Errorf("Wrong type for returned problem: %#v", prob.Type)
	}
}

type mockSAGetRegByKeyNotFound struct {
	sapb.StorageAuthorityReadOnlyClient
}

func (sa *mockSAGetRegByKeyNotFound) GetRegistrationByKey(_ context.Context, req *sapb.JSONWebKey, _ ...grpc.CallOption) (*corepb.Registration, error) {
	return nil, berrors.NotFoundError("not found")
}

func TestNewAccountWhenGetRegByKeyNotFound(t *testing.T) {
	wfe, _, signer := setupWFE(t)
	wfe.sa = &mockSAGetRegByKeyNotFound{wfe.sa}
	key := loadKey(t, []byte(testE2KeyPrivatePEM))
	_, ok := key.(*ecdsa.PrivateKey)
	test.Assert(t, ok, "Couldn't load ECDSA key")
	// When SA.GetRegistrationByKey returns NotFound, and no onlyReturnExisting
	// field is sent, NewAccount should succeed.
	payload := `{"contact":["mailto:person@mail.com"],"termsOfServiceAgreed":true}`
	signedURL := "http://localhost/new-account"
	responseWriter := httptest.NewRecorder()
	_, _, body := signer.embeddedJWK(key, signedURL, payload)
	wfe.NewAccount(ctx, newRequestEvent(), responseWriter, makePostRequestWithPath("/new-account", body))
	if responseWriter.Code != http.StatusCreated {
		t.Errorf("Bad response to NewRegistration: %d, %s", responseWriter.Code, responseWriter.Body)
	}

	// When SA.GetRegistrationByKey returns NotFound, and onlyReturnExisting
	// field **is** sent, NewAccount should fail with the expected error.
	payload = `{"contact":["mailto:person@mail.com"],"termsOfServiceAgreed":true,"onlyReturnExisting":true}`
	responseWriter = httptest.NewRecorder()
	_, _, body = signer.embeddedJWK(key, signedURL, payload)
	// Process the new account request
	wfe.NewAccount(ctx, newRequestEvent(), responseWriter, makePostRequestWithPath("/new-account", body))
	test.AssertEquals(t, responseWriter.Code, http.StatusBadRequest)
	test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), `
	{
		"type": "urn:ietf:params:acme:error:accountDoesNotExist",
		"detail": "No account exists with the provided key",
		"status": 400
	}`)
}

func TestPrepAuthzForDisplay(t *testing.T) {
	wfe, _, _ := setupWFE(t)

	// Make an authz for a wildcard identifier
	authz := &core.Authorization{
		ID:             "12345",
		Status:         core.StatusPending,
		RegistrationID: 1,
		Identifier:     identifier.DNSIdentifier("*.example.com"),
		Challenges: []core.Challenge{
			{
				Type:                     "dns",
				ProvidedKeyAuthorization: "	🔑",
			},
		},
	}

	// Prep the wildcard authz for display
	wfe.prepAuthorizationForDisplay(&http.Request{Host: "localhost"}, authz)

	// The authz should not have a wildcard prefix in the identifier value
	test.AssertEquals(t, strings.HasPrefix(authz.Identifier.Value, "*."), false)
	// The authz should be marked as corresponding to a wildcard name
	test.AssertEquals(t, authz.Wildcard, true)

	// We expect the authz challenge has its URL set and the URI emptied.
	authz.ID = "12345"
	wfe.prepAuthorizationForDisplay(&http.Request{Host: "localhost"}, authz)
	chal := authz.Challenges[0]
	test.AssertEquals(t, chal.URL, "http://localhost/acme/chall-v3/12345/po1V2w")
	test.AssertEquals(t, chal.URI, "")
	test.AssertEquals(t, chal.ProvidedKeyAuthorization, "")
}

// noSCTMockRA is a mock RA that always returns a `berrors.MissingSCTsError` from `FinalizeOrder`
type noSCTMockRA struct {
	MockRegistrationAuthority
}

func (ra *noSCTMockRA) FinalizeOrder(context.Context, *rapb.FinalizeOrderRequest, ...grpc.CallOption) (*corepb.Order, error) {
	return nil, berrors.MissingSCTsError("noSCTMockRA missing scts error")
}

func TestFinalizeSCTError(t *testing.T) {
	wfe, _, signer := setupWFE(t)

	// Set up an RA mock that always returns a berrors.MissingSCTsError from
	// `FinalizeOrder`
	wfe.ra = &noSCTMockRA{}

	// Create a response writer to capture the WFE response
	responseWriter := httptest.NewRecorder()

	// Example CSR payload taken from `TestFinalizeOrder`
	// openssl req -outform der -new -nodes -key wfe/test/178.key -subj /CN=not-an-example.com | b64url
	// a valid CSR
	goodCertCSRPayload := `{
		"csr": "MIICYjCCAUoCAQAwHTEbMBkGA1UEAwwSbm90LWFuLWV4YW1wbGUuY29tMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAmqs7nue5oFxKBk2WaFZJAma2nm1oFyPIq19gYEAdQN4mWvaJ8RjzHFkDMYUrlIrGxCYuFJDHFUk9dh19Na1MIY-NVLgcSbyNcOML3bLbLEwGmvXPbbEOflBA9mxUS9TLMgXW5ghf_qbt4vmSGKloIim41QXt55QFW6O-84s8Kd2OE6df0wTsEwLhZB3j5pDU-t7j5vTMv4Tc7EptaPkOdfQn-68viUJjlYM_4yIBVRhWCdexFdylCKVLg0obsghQEwULKYCUjdg6F0VJUI115DU49tzscXU_3FS3CyY8rchunuYszBNkdmgpAwViHNWuP7ESdEd_emrj1xuioSe6PwIDAQABoAAwDQYJKoZIhvcNAQELBQADggEBAE_T1nWU38XVYL28hNVSXU0rW5IBUKtbvr0qAkD4kda4HmQRTYkt-LNSuvxoZCC9lxijjgtJi-OJe_DCTdZZpYzewlVvcKToWSYHYQ6Wm1-fxxD_XzphvZOujpmBySchdiz7QSVWJmVZu34XD5RJbIcrmj_cjRt42J1hiTFjNMzQu9U6_HwIMmliDL-soFY2RTvvZf-dAFvOUQ-Wbxt97eM1PbbmxJNWRhbAmgEpe9PWDPTpqV5AK56VAa991cQ1P8ZVmPss5hvwGWhOtpnpTZVHN3toGNYFKqxWPboirqushQlfKiFqT9rpRgM3-mFjOHidGqsKEkTdmfSVlVEk3oo="
	}`

	// Create a finalization request with the above payload
	request := signAndPost(signer, "1/8", "http://localhost/1/8", goodCertCSRPayload)

	// POST the finalize order request.
	wfe.FinalizeOrder(ctx, newRequestEvent(), responseWriter, request)

	// We expect the berrors.MissingSCTsError error to have been converted into
	// a serverInternal error with the right message.
	test.AssertUnmarshaledEquals(t,
		responseWriter.Body.String(),
		`{"type":"`+probs.V2ErrorNS+`serverInternal","detail":"Error finalizing order :: Unable to meet CA SCT embedding requirements","status":500}`)
}

func TestOrderToOrderJSONV2Authorizations(t *testing.T) {
	wfe, fc, _ := setupWFE(t)

	orderJSON := wfe.orderToOrderJSON(&http.Request{}, &corepb.Order{
		Id:               1,
		RegistrationID:   1,
		Names:            []string{"a"},
		Status:           string(core.StatusPending),
		Expires:          fc.Now().UnixNano(),
		V2Authorizations: []int64{1, 2},
	})
	test.AssertDeepEquals(t, orderJSON.Authorizations, []string{
		"http://localhost/acme/authz-v3/1",
		"http://localhost/acme/authz-v3/2",
	})
}

// TestMandatoryPOSTAsGET tests that the MandatoryPOSTAsGET feature flag
// correctly causes unauthenticated GET requests to ACME resources to be
// forbidden.
func TestMandatoryPOSTAsGET(t *testing.T) {
	wfe, _, _ := setupWFE(t)

	_ = features.Set(map[string]bool{"MandatoryPOSTAsGET": true})
	defer features.Reset()

	// CheckProblem matches a HTTP response body to a Method Not Allowed problem.
	checkProblem := func(actual []byte) {
		var prob probs.ProblemDetails
		err := json.Unmarshal(actual, &prob)
		test.AssertNotError(t, err, "error unmarshaling HTTP response body as problem")
		test.AssertEquals(t, string(prob.Type), "urn:ietf:params:acme:error:malformed")
		test.AssertEquals(t, prob.Detail, "Method not allowed")
		test.AssertEquals(t, prob.HTTPStatus, http.StatusMethodNotAllowed)
	}

	testCases := []struct {
		name    string
		path    string
		handler web.WFEHandlerFunc
	}{
		{
			// GET requests to a mocked order path should return an error
			name:    "GET Order",
			path:    "1/1",
			handler: wfe.GetOrder,
		},
		{
			// GET requests to a mocked authorization path should return an error
			name:    "GET Authz",
			path:    "1",
			handler: wfe.Authorization,
		},
		{
			// GET requests to a mocked challenge path should return an error
			name:    "GET Chall",
			path:    "1/-ZfxEw",
			handler: wfe.Challenge,
		},
		{
			// GET requests to a mocked certificate serial path should return an error
			name:    "GET Cert",
			path:    "acme/cert/0000000000000000000000000000000000b2",
			handler: wfe.Certificate,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			responseWriter := httptest.NewRecorder()
			req := &http.Request{URL: &url.URL{Path: tc.path}, Method: "GET"}
			tc.handler(ctx, newRequestEvent(), responseWriter, req)
			checkProblem(responseWriter.Body.Bytes())
		})
	}
}

func TestGetChallengeUpRel(t *testing.T) {
	wfe, _, _ := setupWFE(t)

	challengeURL := "http://localhost/acme/chall-v3/1/-ZfxEw"
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", challengeURL, nil)
	test.AssertNotError(t, err, "Could not make NewRequest")
	req.URL.Path = "1/-ZfxEw"

	wfe.Challenge(ctx, newRequestEvent(), resp, req)
	test.AssertEquals(t,
		resp.Code,
		http.StatusOK)
	test.AssertEquals(t,
		resp.Header().Get("Link"),
		`<http://localhost/acme/authz-v3/1>;rel="up"`)
}

func TestPrepAccountForDisplay(t *testing.T) {
	acct := &core.Registration{
		ID:        1987,
		Agreement: "disagreement",
	}

	// Prep the account for display.
	prepAccountForDisplay(acct)

	// The Agreement should always be cleared.
	test.AssertEquals(t, acct.Agreement, "")
	// The ID field should be zeroed.
	test.AssertEquals(t, acct.ID, int64(0))
}

func TestGETAPIAuthz(t *testing.T) {
	wfe, _, _ := setupWFE(t)
	makeGet := func(path, endpoint string) (*http.Request, *web.RequestEvent) {
		return &http.Request{URL: &url.URL{Path: path}, Method: "GET"},
			&web.RequestEvent{Endpoint: endpoint}
	}

	testCases := []struct {
		name              string
		path              string
		expectTooFreshErr bool
	}{
		{
			name:              "fresh authz",
			path:              "1",
			expectTooFreshErr: true,
		},
		{
			name:              "old authz",
			path:              "2",
			expectTooFreshErr: false,
		},
	}

	tooFreshErr := `{"type":"` + probs.V2ErrorNS + `unauthorized","detail":"Authorization is too new for GET API. You should only use this non-standard API to access resources created more than 10s ago","status":403}`
	for _, tc := range testCases {
		responseWriter := httptest.NewRecorder()
		req, logEvent := makeGet(tc.path, getAuthzPath)
		wfe.Authorization(context.Background(), logEvent, responseWriter, req)

		if responseWriter.Code == http.StatusOK && tc.expectTooFreshErr {
			t.Errorf("expected too fresh error, got http.StatusOK")
		} else {
			test.AssertEquals(t, responseWriter.Code, http.StatusForbidden)
			test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), tooFreshErr)
		}
	}
}

func TestGETAPIChallenge(t *testing.T) {
	wfe, _, _ := setupWFE(t)
	makeGet := func(path, endpoint string) (*http.Request, *web.RequestEvent) {
		return &http.Request{URL: &url.URL{Path: path}, Method: "GET"},
			&web.RequestEvent{Endpoint: endpoint}
	}

	testCases := []struct {
		name              string
		path              string
		expectTooFreshErr bool
	}{
		{
			name:              "fresh authz challenge",
			path:              "1/-ZfxEw",
			expectTooFreshErr: true,
		},
		{
			name:              "old authz challenge",
			path:              "2/-ZfxEw",
			expectTooFreshErr: false,
		},
	}

	tooFreshErr := `{"type":"` + probs.V2ErrorNS + `unauthorized","detail":"Authorization is too new for GET API. You should only use this non-standard API to access resources created more than 10s ago","status":403}`
	for _, tc := range testCases {
		responseWriter := httptest.NewRecorder()
		req, logEvent := makeGet(tc.path, getAuthzPath)
		wfe.Challenge(context.Background(), logEvent, responseWriter, req)

		if responseWriter.Code == http.StatusOK && tc.expectTooFreshErr {
			t.Errorf("expected too fresh error, got http.StatusOK")
		} else {
			test.AssertEquals(t, responseWriter.Code, http.StatusForbidden)
			test.AssertUnmarshaledEquals(t, responseWriter.Body.String(), tooFreshErr)
		}
	}
}

// TestGet404 tests that a 404 is served and that the expected endpoint of
// "/" is logged when an unknown path is requested. This will test the
// codepath to the wfe.Index() handler which handles "/" and all non-api
// endpoint requests to make sure the endpoint is set properly in the logs.
func TestIndexGet404(t *testing.T) {
	// Setup
	wfe, _, _ := setupWFE(t)
	path := "/nopathhere/nope/nofilehere"
	req := &http.Request{URL: &url.URL{Path: path}, Method: "GET"}
	logEvent := &web.RequestEvent{}
	responseWriter := httptest.NewRecorder()

	// Send a request to wfe.Index()
	wfe.Index(context.Background(), logEvent, responseWriter, req)

	// Test that a 404 is received as expected
	test.AssertEquals(t, responseWriter.Code, http.StatusNotFound)
	// Test that we logged the "/" endpoint
	test.AssertEquals(t, logEvent.Endpoint, "/")
	// Test that the rest of the path is logged as the slug
	test.AssertEquals(t, logEvent.Slug, path[1:])
}

// TestGetAPIAndMandatoryPOSTAsGet that, even when MandatoryPOSTAsGet is on,
// we are still willing to allow a GET request for a certificate that is old
// enough that we're no longer worried about stale info being cached.
func TestGetAPIAndMandatoryPOSTAsGET(t *testing.T) {
	wfe, _, _ := setupWFE(t)
	wfe.sa = newMockSAWithCert(t, wfe.sa, true)

	makeGet := func(path, endpoint string) (*http.Request, *web.RequestEvent) {
		return &http.Request{URL: &url.URL{Path: path}, Method: "GET"},
			&web.RequestEvent{Endpoint: endpoint, Extra: map[string]interface{}{}}
	}
	_ = features.Set(map[string]bool{"MandatoryPOSTAsGET": true})
	defer features.Reset()

	cert, err := core.LoadCert("../test/hierarchy/ee-r3.cert.pem")
	test.AssertNotError(t, err, "failed to load test certificate")

	req, event := makeGet(core.SerialToString(cert.SerialNumber), getCertPath)
	resp := httptest.NewRecorder()
	wfe.Certificate(context.Background(), event, resp, req)
	test.AssertEquals(t, resp.Code, 200)
}

// TestARI tests that requests for real certs result in renewal info, while
// requests for certs that don't exist result in errors.
func TestARI(t *testing.T) {
	wfe, _, _ := setupWFE(t)
	msa := newMockSAWithCert(t, wfe.sa, false)
	wfe.sa = msa

	makeGet := func(path, endpoint string) (*http.Request, *web.RequestEvent) {
		return &http.Request{URL: &url.URL{Path: path}, Method: "GET"},
			&web.RequestEvent{Endpoint: endpoint, Extra: map[string]interface{}{}}
	}
	_ = features.Set(map[string]bool{"ServeRenewalInfo": true})
	defer features.Reset()

	// Load the certificate and its issuer.
	cert, err := core.LoadCert("../test/hierarchy/ee-r3.cert.pem")
	test.AssertNotError(t, err, "failed to load test certificate")
	issuer, err := core.LoadCert("../test/hierarchy/int-r3.cert.pem")
	test.AssertNotError(t, err, "failed to load test issuer")

	// Take advantage of OCSP to build the issuer hashes.
	ocspReqBytes, err := ocsp.CreateRequest(cert, issuer, &ocsp.RequestOptions{Hash: crypto.SHA256})
	test.AssertNotError(t, err, "failed to create ocsp request")
	ocspReq, err := ocsp.ParseRequest(ocspReqBytes)
	test.AssertNotError(t, err, "failed to parse ocsp request")

	// Ensure that a correct query results in a 200.
	idBytes, err := asn1.Marshal(certID{
		pkix.AlgorithmIdentifier{ // SHA256
			Algorithm:  asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1},
			Parameters: asn1.RawValue{Tag: 5 /* ASN.1 NULL */},
		},
		ocspReq.IssuerNameHash,
		ocspReq.IssuerKeyHash,
		cert.SerialNumber,
	})
	test.AssertNotError(t, err, "failed to marshal certID")
	req, event := makeGet(base64.RawURLEncoding.EncodeToString(idBytes), renewalInfoPath)
	resp := httptest.NewRecorder()
	wfe.RenewalInfo(context.Background(), event, resp, req)
	test.AssertEquals(t, resp.Code, http.StatusOK)
	test.AssertEquals(t, resp.Header().Get("Retry-After"), "21600")
	var ri core.RenewalInfo
	err = json.Unmarshal(resp.Body.Bytes(), &ri)
	test.AssertNotError(t, err, "unmarshalling renewal info")
	test.Assert(t, ri.SuggestedWindow.Start.After(cert.NotBefore), "suggested window begins before cert issuance")
	test.Assert(t, ri.SuggestedWindow.End.Before(cert.NotAfter), "suggested window ends after cert expiry")

	// Ensure that a correct query for a revoked cert results in a renewal window
	// in the past.
	msa.status = core.OCSPStatusRevoked
	req, event = makeGet(base64.RawURLEncoding.EncodeToString(idBytes), renewalInfoPath)
	resp = httptest.NewRecorder()
	wfe.RenewalInfo(context.Background(), event, resp, req)
	test.AssertEquals(t, resp.Code, http.StatusOK)
	test.AssertEquals(t, resp.Header().Get("Retry-After"), "21600")
	err = json.Unmarshal(resp.Body.Bytes(), &ri)
	test.AssertNotError(t, err, "unmarshalling renewal info")
	test.Assert(t, ri.SuggestedWindow.End.Before(wfe.clk.Now()), "suggested window should end in the past")
	test.Assert(t, ri.SuggestedWindow.Start.Before(ri.SuggestedWindow.End), "suggested window should start before it ends")

	// Ensure that a query for a non-existent serial results in a 404.
	idBytes, err = asn1.Marshal(certID{
		pkix.AlgorithmIdentifier{ // SHA256
			Algorithm:  asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1},
			Parameters: asn1.RawValue{Tag: 5 /* ASN.1 NULL */},
		},
		ocspReq.IssuerNameHash,
		ocspReq.IssuerKeyHash,
		big.NewInt(0).Add(cert.SerialNumber, big.NewInt(1)),
	})
	test.AssertNotError(t, err, "failed to marshal certID")
	req, event = makeGet(base64.RawURLEncoding.EncodeToString(idBytes), renewalInfoPath)
	resp = httptest.NewRecorder()
	wfe.RenewalInfo(context.Background(), event, resp, req)
	test.AssertEquals(t, resp.Code, http.StatusNotFound)
	test.AssertEquals(t, resp.Header().Get("Retry-After"), "")

	// Ensure that a query with a bad hash algorithm fails.
	idBytes, err = asn1.Marshal(certID{
		pkix.AlgorithmIdentifier{ // SHA-1
			Algorithm:  asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26},
			Parameters: asn1.RawValue{Tag: 5 /* ASN.1 NULL */},
		},
		ocspReq.IssuerNameHash,
		ocspReq.IssuerKeyHash,
		big.NewInt(0).Add(cert.SerialNumber, big.NewInt(1)),
	})
	test.AssertNotError(t, err, "failed to marshal certID")
	req, event = makeGet(base64.RawURLEncoding.EncodeToString(idBytes), renewalInfoPath)
	resp = httptest.NewRecorder()
	wfe.RenewalInfo(context.Background(), event, resp, req)
	test.AssertEquals(t, resp.Code, http.StatusBadRequest)
	test.AssertContains(t, resp.Body.String(), "Request used hash algorithm other than SHA-256")

	// Ensure that a query with a non-CertID path fails.
	req, event = makeGet(base64.RawURLEncoding.EncodeToString(ocspReqBytes), renewalInfoPath)
	resp = httptest.NewRecorder()
	wfe.RenewalInfo(context.Background(), event, resp, req)
	test.AssertEquals(t, resp.Code, http.StatusBadRequest)
	test.AssertContains(t, resp.Body.String(), "Path was not a DER-encoded CertID sequence")

	// Ensure that a query with a non-Base64URL path (including one in the old
	// request path style, which included slashes) fails.
	req, event = makeGet(
		fmt.Sprintf(
			"%s/%s/%s",
			base64.RawURLEncoding.EncodeToString(ocspReq.IssuerNameHash),
			base64.RawURLEncoding.EncodeToString(ocspReq.IssuerKeyHash),
			base64.RawURLEncoding.EncodeToString(cert.SerialNumber.Bytes()),
		),
		renewalInfoPath)
	resp = httptest.NewRecorder()
	wfe.RenewalInfo(context.Background(), event, resp, req)
	test.AssertEquals(t, resp.Code, http.StatusBadRequest)
	test.AssertContains(t, resp.Body.String(), "Path was not base64url-encoded")

	// Ensure that a query with no path slug at all bails out early.
	req, event = makeGet("", renewalInfoPath)
	resp = httptest.NewRecorder()
	wfe.RenewalInfo(context.Background(), event, resp, req)
	test.AssertEquals(t, resp.Code, http.StatusNotFound)
	test.AssertContains(t, resp.Body.String(), "Must specify a request path")
}

// TestIncidentARI tests that requests certs impacted by an ongoing revocation
// incident result in a 200 with a retry-after header and a suggested retry
// window in the past.
func TestIncidentARI(t *testing.T) {
	wfe, _, _ := setupWFE(t)
	expectSerial := big.NewInt(12345)
	expectSerialString := core.SerialToString(big.NewInt(12345))
	wfe.sa = newMockSAWithIncident(wfe.sa, []string{expectSerialString})

	makeGet := func(path, endpoint string) (*http.Request, *web.RequestEvent) {
		return &http.Request{URL: &url.URL{Path: path}, Method: "GET"},
			&web.RequestEvent{Endpoint: endpoint, Extra: map[string]interface{}{}}
	}
	_ = features.Set(map[string]bool{"ServeRenewalInfo": true})
	defer features.Reset()

	idBytes, err := asn1.Marshal(certID{
		pkix.AlgorithmIdentifier{ // SHA256
			Algorithm:  asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1},
			Parameters: asn1.RawValue{Tag: 5 /* ASN.1 NULL */},
		},
		[]byte("foo"),
		[]byte("baz"),
		expectSerial,
	})
	test.AssertNotError(t, err, "failed to marshal certID")

	req, event := makeGet(base64.RawURLEncoding.EncodeToString(idBytes), renewalInfoPath)
	resp := httptest.NewRecorder()
	wfe.RenewalInfo(context.Background(), event, resp, req)
	test.AssertEquals(t, resp.Code, 200)
	test.AssertEquals(t, resp.Header().Get("Retry-After"), "21600")
	var ri core.RenewalInfo
	err = json.Unmarshal(resp.Body.Bytes(), &ri)
	test.AssertNotError(t, err, "unmarshalling renewal info")
	// The start of the window should be in the past.
	test.AssertEquals(t, ri.SuggestedWindow.Start.Before(wfe.clk.Now()), true)
	// The end of the window should be after the start.
	test.AssertEquals(t, ri.SuggestedWindow.End.After(ri.SuggestedWindow.Start), true)
	// The end of the window should also be in the past.
	test.AssertEquals(t, ri.SuggestedWindow.End.Before(wfe.clk.Now()), true)
}

func TestOldTLSInbound(t *testing.T) {
	wfe, _, _ := setupWFE(t)
	req := &http.Request{
		URL:    &url.URL{Path: "/directory"},
		Method: "GET",
		Header: http.Header(map[string][]string{
			http.CanonicalHeaderKey("TLS-Version"): {"TLSv1"},
		}),
	}

	responseWriter := httptest.NewRecorder()
	wfe.Handler(metrics.NoopRegisterer).ServeHTTP(responseWriter, req)
	test.AssertEquals(t, responseWriter.Code, http.StatusBadRequest)
}

func Test_sendError(t *testing.T) {
	features.Reset()
	wfe, _, _ := setupWFE(t)
	testResponse := httptest.NewRecorder()

	testErr := berrors.RateLimitError(0, "test")
	wfe.sendError(testResponse, &web.RequestEvent{Endpoint: "test"}, probs.RateLimited("test"), testErr)
	// Ensure a 0 value RetryAfter results in no Retry-After header.
	test.AssertEquals(t, testResponse.Header().Get("Retry-After"), "")
	// Ensure the Link header isn't populatsed.
	test.AssertEquals(t, testResponse.Header().Get("Link"), "")

	testErr = berrors.RateLimitError(time.Millisecond*500, "test")
	wfe.sendError(testResponse, &web.RequestEvent{Endpoint: "test"}, probs.RateLimited("test"), testErr)
	// Ensure a 500ms RetryAfter is rounded up to a 1s Retry-After header.
	test.AssertEquals(t, testResponse.Header().Get("Retry-After"), "1")
	// Ensure the Link header is populated.
	test.AssertEquals(t, testResponse.Header().Get("Link"), "<https://letsencrypt.org/docs/rate-limits>;rel=\"help\"")

	// Clear headers for the next test.
	testResponse.Header().Del("Retry-After")
	testResponse.Header().Del("Link")

	testErr = berrors.RateLimitError(time.Millisecond*499, "test")
	wfe.sendError(testResponse, &web.RequestEvent{Endpoint: "test"}, probs.RateLimited("test"), testErr)
	// Ensure a 499ms RetryAfter results in no Retry-After header.
	test.AssertEquals(t, testResponse.Header().Get("Retry-After"), "")
	// Ensure the Link header isn't populatsed.
	test.AssertEquals(t, testResponse.Header().Get("Link"), "")
}
