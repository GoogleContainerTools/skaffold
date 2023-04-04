package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"text/template"

	"github.com/letsencrypt/boulder/cmd"
)

// createSlot initializes a SoftHSM slot and token. SoftHSM chooses the highest empty
// slot, initializes it, and then assigns it a new randomly chosen slot ID. Since we can't
// predict this ID we need to parse out the new ID so that we can use it in the ceremony
// configs.
func createSlot(label string) (string, error) {
	output, err := exec.Command("softhsm2-util", "--init-token", "--free", "--label", label, "--pin", "1234", "--so-pin", "5678").CombinedOutput()
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(`to slot (\d+)`)
	matches := re.FindSubmatch(output)
	if len(matches) != 2 {
		return "", errors.New("unexpected number of slot matches")
	}
	return string(matches[1]), nil
}

// genKey is used to run a key ceremony with a given config, replacing SlotID in
// the YAML with a specific slot ID.
func genKey(path string, inSlot string) error {
	tmpPath, err := rewriteConfig(path, map[string]string{"SlotID": inSlot})
	if err != nil {
		return err
	}
	output, err := exec.Command("bin/ceremony", "-config", tmpPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running ceremony for %s: %s:\n%s", tmpPath, err, string(output))
	}
	return nil
}

// rewriteConfig creates a temporary config based on the template at path
// using the variables in rewrites.
func rewriteConfig(path string, rewrites map[string]string) (string, error) {
	tmplBytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(os.TempDir(), "ceremony-config")
	if err != nil {
		return "", err
	}
	defer tmp.Close()
	tmpl, err := template.New("config").Parse(string(tmplBytes))
	if err != nil {
		return "", err
	}
	err = tmpl.Execute(tmp, rewrites)
	if err != nil {
		return "", err
	}
	return tmp.Name(), nil
}

// genCert is used to run ceremony when we don't actually care about,
// any of the output and only want to verify it exits cleanly
func genCert(path string) error {
	return exec.Command("bin/ceremony", "-config", path).Run()
}

func main() {
	// If one of the output files already exists, assume this ran once
	// already for the container and don't re-run.
	outputFile := "/hierarchy/root-signing-pub-rsa.pem"
	if loc, err := os.Stat(outputFile); err == nil && loc.Mode().IsRegular() {
		fmt.Println("skipping certificate generation: already exists")
		return
	} else if err == nil && !loc.Mode().IsRegular() {
		log.Fatalf("statting %q: not a regular file", outputFile)
	} else if err != nil && !os.IsNotExist(err) {
		log.Fatalf("statting %q: %s", outputFile, err)
	}
	// Create SoftHSM slots for the root signing keys
	rsaRootKeySlot, err := createSlot("root signing key (rsa)")
	cmd.FailOnError(err, "failed creating softhsm2 slot for RSA root key")
	ecdsaRootKeySlot, err := createSlot("root signing key (ecdsa)")
	cmd.FailOnError(err, "failed creating softhsm2 slot for root key")

	// Generate the root signing keys and certificates
	err = genKey("test/cert-ceremonies/root-ceremony-rsa.yaml", rsaRootKeySlot)
	cmd.FailOnError(err, "failed to generate RSA root key + root cert")
	err = genKey("test/cert-ceremonies/root-ceremony-ecdsa.yaml", ecdsaRootKeySlot)
	cmd.FailOnError(err, "failed to generate ECDSA root key + root cert")

	// Create SoftHSM slots for the intermediate signing keys
	rsaIntermediateKeySlot, err := createSlot("intermediate signing key (rsa)")
	cmd.FailOnError(err, "failed to create softhsm2 slot for RSA intermediate key")
	ecdsaIntermediateKeySlot, err := createSlot("intermediate signing key (ecdsa)")
	cmd.FailOnError(err, "failed to create softhsm2 slot for ECDSA intermediate key")

	// Generate the intermediate signing keys
	err = genKey("test/cert-ceremonies/intermediate-key-ceremony-rsa.yaml", rsaIntermediateKeySlot)
	cmd.FailOnError(err, "failed to generate RSA intermediate key")
	err = genKey("test/cert-ceremonies/intermediate-key-ceremony-ecdsa.yaml", ecdsaIntermediateKeySlot)
	cmd.FailOnError(err, "failed to generate ECDSA intermediate key")

	// Create the A intermediate ceremony config files with the root
	// signing key slots and IDs
	rsaTmpIntermediateA, err := rewriteConfig("test/cert-ceremonies/intermediate-ceremony-rsa.yaml", map[string]string{
		"SlotID":     rsaRootKeySlot,
		"CertPath":   "/hierarchy/intermediate-cert-rsa-a.pem",
		"CommonName": "CA intermediate (RSA) A",
	})
	cmd.FailOnError(err, "failed to rewrite RSA intermediate cert config with key ID")
	ecdsaTmpIntermediateA, err := rewriteConfig("test/cert-ceremonies/intermediate-ceremony-ecdsa.yaml", map[string]string{
		"SlotID":     ecdsaRootKeySlot,
		"CertPath":   "/hierarchy/intermediate-cert-ecdsa-a.pem",
		"CommonName": "CA intermediate (ECDSA) A",
	})
	cmd.FailOnError(err, "failed to rewrite ECDSA intermediate cert config with key ID")

	// Create the A intermediate certificates
	err = genCert(rsaTmpIntermediateA)
	cmd.FailOnError(err, "failed to generate RSA intermediate cert")
	err = genCert(ecdsaTmpIntermediateA)
	cmd.FailOnError(err, "failed to generate ECDSA intermediate cert")

	// Create the B intermediate ceremony config files with the root
	// signing key slots and IDs
	rsaTmpIntermediateB, err := rewriteConfig("test/cert-ceremonies/intermediate-ceremony-rsa.yaml", map[string]string{
		"SlotID":     rsaRootKeySlot,
		"CertPath":   "/hierarchy/intermediate-cert-rsa-b.pem",
		"CommonName": "CA intermediate (RSA) B",
	})
	cmd.FailOnError(err, "failed to rewrite RSA intermediate cert config with key ID")
	ecdsaTmpIntermediateB, err := rewriteConfig("test/cert-ceremonies/intermediate-ceremony-ecdsa.yaml", map[string]string{
		"SlotID":     ecdsaRootKeySlot,
		"CertPath":   "/hierarchy/intermediate-cert-ecdsa-b.pem",
		"CommonName": "CA intermediate (ECDSA) B",
	})
	cmd.FailOnError(err, "failed to rewrite ECDSA intermediate cert config with key ID")

	// Create the B intermediate certificates
	err = genCert(rsaTmpIntermediateB)
	cmd.FailOnError(err, "failed to generate RSA intermediate cert")
	err = genCert(ecdsaTmpIntermediateB)
	cmd.FailOnError(err, "failed to generate ECDSA intermediate cert")

	// Rewrite OCSP configs and generate OCSP responses for the A intermediates
	rsaTmpOCSPConfig, err := rewriteConfig("test/cert-ceremonies/intermediate-ocsp-rsa.yaml", map[string]string{
		"SlotID": rsaRootKeySlot,
	})
	cmd.FailOnError(err, "failed to rewrite RSA intermediate OCSP config with key ID")
	err = genCert(rsaTmpOCSPConfig)
	cmd.FailOnError(err, "failed to generate RSA intermediate OCSP response")
	// We do not generate OCSP for the ECDSA intermediates, as our new issuers
	// only use CRLs, not OCSP.
}
