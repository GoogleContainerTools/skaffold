//go:build integration

package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/letsencrypt/boulder/issuance"
	"github.com/letsencrypt/pkcs11key/v4"
)

var template = `[AUDIT] Failed RPC to store at SA, orphaning precertificate: serial=[%x], cert=[%x], issuerID=[1], regID=[1], orderID=[1], err=[sa.StorageAuthority.AddPrecertificate timed out after 5000 ms]
[AUDIT] Failed RPC to store at SA, orphaning certificate: serial=[%x], cert=[%x], issuerID=[1], regID=[1], orderID=[1], err=[sa.StorageAuthority.AddCertificate timed out after 5000 ms]`

// TestOrphanFinder runs the orphan-finder with an example input file. This must
// be run after other tests so the account ID 1 exists (since the inserted
// certificates will be associated with that account).
func TestOrphanFinder(t *testing.T) {
	t.Parallel()
	precert, err := makeFakeCert(true)
	if err != nil {
		log.Fatal(err)
	}
	cert, err := makeFakeCert(false)
	if err != nil {
		log.Fatal(err)
	}
	f, _ := os.CreateTemp("", "orphaned.log")
	io.WriteString(f, fmt.Sprintf(template, precert.SerialNumber.Bytes(),
		precert.Raw, cert.SerialNumber.Bytes(), cert.Raw))
	f.Close()
	cmd := exec.Command("./bin/boulder", "orphan-finder", "parse-ca-log",
		"--config", "./"+os.Getenv("BOULDER_CONFIG_DIR")+"/orphan-finder.json",
		"--log-file", f.Name())
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("orphan finder failed (%s). Output was: %s", err, out)
	}
	if !strings.Contains(string(out), "Found 1 precertificate orphans and added 1 to the database") {
		t.Fatalf("Failed to insert orphaned precertificate. orphan-finder output was: %s", out)
	}
	if !strings.Contains(string(out), "Found 1 certificate orphans and added 1 to the database") {
		t.Fatalf("Failed to insert orphaned certificate. orphan-finder output was: %s", out)
	}
}

// makeFakeCert a unique fake cert for each run of TestOrphanFinder to avoid duplicate
// errors. This fake cert will have its issuer equal to the issuer we use in the
// general integration test setup, and will be signed by that issuer key.
// Otherwise, the request orphan-finder makes to sign OCSP would be rejected.
func makeFakeCert(precert bool) (*x509.Certificate, error) {
	serialBytes := make([]byte, 18)
	_, err := rand.Read(serialBytes[:])
	if err != nil {
		return nil, err
	}
	serial := big.NewInt(0)
	serial.SetBytes(serialBytes)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	pubKeyBytes, err := os.ReadFile("/hierarchy/intermediate-signing-pub-rsa.pem")
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pubKeyBytes)
	if block == nil {
		return nil, fmt.Errorf("no PEM found")
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing issuer public key: %s", err)
	}
	var pkcs11Config pkcs11key.Config
	contents, err := os.ReadFile("test/test-ca.key-pkcs11.json")
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(contents, &pkcs11Config); err != nil {
		return nil, err
	}
	signer, err := pkcs11key.NewPool(1, pkcs11Config.Module,
		pkcs11Config.TokenLabel, pkcs11Config.PIN, pubKey)
	if err != nil {
		return nil, err
	}
	issuer, err := issuance.LoadCertificate("/hierarchy/intermediate-cert-rsa-a.pem")
	if err != nil {
		return nil, err
	}
	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName: "fake cert for TestOrphanFinder",
		},
		SerialNumber: serial,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 90, 0),
		DNSNames:     []string{"fakecert.example.com"},
	}
	if precert {
		template.ExtraExtensions = []pkix.Extension{
			{
				Id:       OIDExtensionCTPoison,
				Critical: true,
				Value:    []byte{5, 0},
			},
		}
	}

	der, err := x509.CreateCertificate(rand.Reader, template, issuer.Certificate, key.Public(), signer)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return cert, err
}
