//
// Copyright 2022 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package intoto

import (
	"bytes"
	"context"
	"crypto"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/secure-systems-lab/go-securesystemslib/dsse"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"

	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/sigstore/rekor/pkg/log"
	"github.com/sigstore/rekor/pkg/pki"
	"github.com/sigstore/rekor/pkg/pki/x509"
	"github.com/sigstore/rekor/pkg/types"
	"github.com/sigstore/rekor/pkg/types/intoto"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/options"
)

const (
	APIVERSION = "0.0.2"
)

func init() {
	if err := intoto.VersionMap.SetEntryFactory(APIVERSION, NewEntry); err != nil {
		log.Logger.Panic(err)
	}
}

type V002Entry struct {
	IntotoObj models.IntotoV002Schema
	env       dsse.Envelope
}

func (v V002Entry) APIVersion() string {
	return APIVERSION
}

func NewEntry() types.EntryImpl {
	return &V002Entry{}
}

func (v V002Entry) IndexKeys() ([]string, error) {
	var result []string

	if v.IntotoObj.Content == nil || v.IntotoObj.Content.Envelope == nil {
		log.Logger.Info("IntotoObj content or dsse envelope is nil")
		return result, nil
	}

	for _, sig := range v.IntotoObj.Content.Envelope.Signatures {
		if sig == nil || sig.PublicKey == nil {
			return result, errors.New("malformed or missing signature")
		}
		keyObj, err := x509.NewPublicKey(bytes.NewReader(*sig.PublicKey))
		if err != nil {
			return result, err
		}

		canonKey, err := keyObj.CanonicalValue()
		if err != nil {
			return result, fmt.Errorf("could not canonicize key: %w", err)
		}

		keyHash := sha256.Sum256(canonKey)
		result = append(result, "sha256:"+hex.EncodeToString(keyHash[:]))

		result = append(result, keyObj.Subjects()...)
	}

	payloadKey := strings.ToLower(fmt.Sprintf("%s:%s", *v.IntotoObj.Content.PayloadHash.Algorithm, *v.IntotoObj.Content.PayloadHash.Value))
	result = append(result, payloadKey)

	// since we can't deterministically calculate this server-side (due to public keys being added inline, and also canonicalization being potentially different),
	// we'll just skip adding this index key
	// hashkey := strings.ToLower(fmt.Sprintf("%s:%s", *v.IntotoObj.Content.Hash.Algorithm, *v.IntotoObj.Content.Hash.Value))
	// result = append(result, hashkey)

	switch *v.IntotoObj.Content.Envelope.PayloadType {
	case in_toto.PayloadType:

		if v.IntotoObj.Content.Envelope.Payload == nil {
			log.Logger.Info("IntotoObj DSSE payload is empty")
			return result, nil
		}
		decodedPayload, err := base64.StdEncoding.DecodeString(string(v.IntotoObj.Content.Envelope.Payload))
		if err != nil {
			return result, fmt.Errorf("could not decode envelope payload: %w", err)
		}
		statement, err := parseStatement(decodedPayload)
		if err != nil {
			return result, err
		}
		for _, s := range statement.Subject {
			for alg, ds := range s.Digest {
				result = append(result, alg+":"+ds)
			}
		}
		// Not all in-toto statements will contain a SLSA provenance predicate.
		// See https://github.com/in-toto/attestation/blob/main/spec/README.md#predicate
		// for other predicates.
		if predicate, err := parseSlsaPredicate(decodedPayload); err == nil {
			if predicate.Predicate.Materials != nil {
				for _, s := range predicate.Predicate.Materials {
					for alg, ds := range s.Digest {
						result = append(result, alg+":"+ds)
					}
				}
			}
		}
	default:
		log.Logger.Infof("Unknown in_toto DSSE envelope Type: %s", *v.IntotoObj.Content.Envelope.PayloadType)
	}
	return result, nil
}

func parseStatement(p []byte) (*in_toto.Statement, error) {
	ps := in_toto.Statement{}
	if err := json.Unmarshal(p, &ps); err != nil {
		return nil, err
	}
	return &ps, nil
}

func parseSlsaPredicate(p []byte) (*in_toto.ProvenanceStatement, error) {
	predicate := in_toto.ProvenanceStatement{}
	if err := json.Unmarshal(p, &predicate); err != nil {
		return nil, err
	}
	return &predicate, nil
}

func (v *V002Entry) Unmarshal(pe models.ProposedEntry) error {
	it, ok := pe.(*models.Intoto)
	if !ok {
		return errors.New("cannot unmarshal non Intoto v0.0.2 type")
	}

	var err error
	if err := types.DecodeEntry(it.Spec, &v.IntotoObj); err != nil {
		return err
	}

	// field validation
	if err := v.IntotoObj.Validate(strfmt.Default); err != nil {
		return err
	}

	if string(v.IntotoObj.Content.Envelope.Payload) == "" {
		return nil
	}

	env := &dsse.Envelope{
		Payload:     string(v.IntotoObj.Content.Envelope.Payload),
		PayloadType: *v.IntotoObj.Content.Envelope.PayloadType,
	}

	allPubKeyBytes := make([][]byte, 0)
	for i, sig := range v.IntotoObj.Content.Envelope.Signatures {
		if sig == nil {
			v.IntotoObj.Content.Envelope.Signatures = slices.Delete(v.IntotoObj.Content.Envelope.Signatures, i, i)
			continue
		}
		env.Signatures = append(env.Signatures, dsse.Signature{
			KeyID: sig.Keyid,
			Sig:   string(*sig.Sig),
		})

		allPubKeyBytes = append(allPubKeyBytes, *sig.PublicKey)
	}

	if _, err := verifyEnvelope(allPubKeyBytes, env); err != nil {
		return err
	}

	v.env = *env

	decodedPayload, err := base64.StdEncoding.DecodeString(string(v.IntotoObj.Content.Envelope.Payload))
	if err != nil {
		return fmt.Errorf("could not decode envelope payload: %w", err)
	}

	h := sha256.Sum256(decodedPayload)
	v.IntotoObj.Content.PayloadHash = &models.IntotoV002SchemaContentPayloadHash{
		Algorithm: swag.String(models.IntotoV002SchemaContentPayloadHashAlgorithmSha256),
		Value:     swag.String(hex.EncodeToString(h[:])),
	}

	return nil
}

func (v *V002Entry) Canonicalize(_ context.Context) ([]byte, error) {
	if err := v.IntotoObj.Validate(strfmt.Default); err != nil {
		return nil, err
	}

	if v.IntotoObj.Content.Hash == nil {
		return nil, errors.New("missing envelope digest")
	}

	if err := v.IntotoObj.Content.Hash.Validate(strfmt.Default); err != nil {
		return nil, fmt.Errorf("error validating envelope digest: %w", err)
	}

	if v.IntotoObj.Content.PayloadHash == nil {
		return nil, errors.New("missing payload digest")
	}

	if err := v.IntotoObj.Content.PayloadHash.Validate(strfmt.Default); err != nil {
		return nil, fmt.Errorf("error validating payload digest: %w", err)
	}

	if len(v.IntotoObj.Content.Envelope.Signatures) == 0 {
		return nil, errors.New("missing signatures")
	}

	canonicalEntry := models.IntotoV002Schema{
		Content: &models.IntotoV002SchemaContent{
			Envelope: &models.IntotoV002SchemaContentEnvelope{
				PayloadType: v.IntotoObj.Content.Envelope.PayloadType,
				Signatures:  v.IntotoObj.Content.Envelope.Signatures,
			},
			Hash:        v.IntotoObj.Content.Hash,
			PayloadHash: v.IntotoObj.Content.PayloadHash,
		},
	}
	itObj := models.Intoto{}
	itObj.APIVersion = swag.String(APIVERSION)
	itObj.Spec = &canonicalEntry

	return json.Marshal(&itObj)
}

// AttestationKey returns the digest of the attestation that was uploaded, to be used to lookup the attestation from storage
func (v *V002Entry) AttestationKey() string {
	if v.IntotoObj.Content != nil && v.IntotoObj.Content.PayloadHash != nil {
		return fmt.Sprintf("%s:%s", *v.IntotoObj.Content.PayloadHash.Algorithm, *v.IntotoObj.Content.PayloadHash.Value)
	}
	return ""
}

// AttestationKeyValue returns both the key and value to be persisted into attestation storage
func (v *V002Entry) AttestationKeyValue() (string, []byte) {
	storageSize := base64.StdEncoding.DecodedLen(len(v.env.Payload))
	if storageSize > viper.GetInt("max_attestation_size") {
		log.Logger.Infof("Skipping attestation storage, size %d is greater than max %d", storageSize, viper.GetInt("max_attestation_size"))
		return "", nil
	}
	attBytes, err := base64.StdEncoding.DecodeString(v.env.Payload)
	if err != nil {
		log.Logger.Infof("could not decode envelope payload: %w", err)
		return "", nil
	}
	return v.AttestationKey(), attBytes
}

type verifier struct {
	s signature.Signer
	v signature.Verifier
}

func (v *verifier) KeyID() (string, error) {
	return "", nil
}

func (v *verifier) Public() crypto.PublicKey {
	// the dsse library uses this to generate a key ID if the KeyID function returns an empty string
	// as well for the AcceptedKey return value.  Unfortunately since key ids can be arbitrary, we don't
	// know how to generate a matching id for the key id on the envelope's signature...
	// dsse verify will skip verifiers whose key id doesn't match the signature's key id, unless it fails
	// to generate one from the public key... so we trick it by returning nil ¯\_(ツ)_/¯
	return nil
}

func (v *verifier) Sign(_ context.Context, data []byte) (sig []byte, err error) {
	if v.s == nil {
		return nil, errors.New("nil signer")
	}
	sig, err = v.s.SignMessage(bytes.NewReader(data), options.WithCryptoSignerOpts(crypto.SHA256))
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func (v *verifier) Verify(_ context.Context, data, sig []byte) error {
	if v.v == nil {
		return errors.New("nil verifier")
	}
	return v.v.VerifySignature(bytes.NewReader(sig), bytes.NewReader(data))
}

func (v V002Entry) CreateFromArtifactProperties(_ context.Context, props types.ArtifactProperties) (models.ProposedEntry, error) {
	returnVal := models.Intoto{}
	re := V002Entry{
		IntotoObj: models.IntotoV002Schema{
			Content: &models.IntotoV002SchemaContent{
				Envelope: &models.IntotoV002SchemaContentEnvelope{},
			},
		}}
	var err error
	artifactBytes := props.ArtifactBytes
	if artifactBytes == nil {
		if props.ArtifactPath == nil {
			return nil, errors.New("path to artifact file must be specified")
		}
		if props.ArtifactPath.IsAbs() {
			return nil, errors.New("intoto envelopes cannot be fetched over HTTP(S)")
		}
		artifactBytes, err = os.ReadFile(filepath.Clean(props.ArtifactPath.Path))
		if err != nil {
			return nil, err
		}
	}

	env := dsse.Envelope{}
	if err := json.Unmarshal(artifactBytes, &env); err != nil {
		return nil, fmt.Errorf("payload must be a valid dsse envelope: %w", err)
	}

	allPubKeyBytes := make([][]byte, 0)
	if len(props.PublicKeyBytes) > 0 {
		allPubKeyBytes = append(allPubKeyBytes, props.PublicKeyBytes...)
	}

	if len(props.PublicKeyPaths) > 0 {
		for _, path := range props.PublicKeyPaths {
			if path.IsAbs() {
				return nil, errors.New("dsse public keys cannot be fetched over HTTP(S)")
			}

			publicKeyBytes, err := os.ReadFile(filepath.Clean(path.Path))
			if err != nil {
				return nil, fmt.Errorf("error reading public key file: %w", err)
			}

			allPubKeyBytes = append(allPubKeyBytes, publicKeyBytes)
		}
	}

	keysBySig, err := verifyEnvelope(allPubKeyBytes, &env)
	if err != nil {
		return nil, err
	}

	b64 := strfmt.Base64([]byte(env.Payload))
	re.IntotoObj.Content.Envelope.Payload = b64
	re.IntotoObj.Content.Envelope.PayloadType = &env.PayloadType

	for _, sig := range env.Signatures {
		key, ok := keysBySig[sig.Sig]
		if !ok {
			return nil, errors.New("all signatures must have a key that verifies it")
		}

		canonKey, err := key.CanonicalValue()
		if err != nil {
			return nil, fmt.Errorf("could not canonicize key: %w", err)
		}

		keyBytes := strfmt.Base64(canonKey)
		sigBytes := strfmt.Base64([]byte(sig.Sig))
		re.IntotoObj.Content.Envelope.Signatures = append(re.IntotoObj.Content.Envelope.Signatures, &models.IntotoV002SchemaContentEnvelopeSignaturesItems0{
			Keyid:     sig.KeyID,
			Sig:       &sigBytes,
			PublicKey: &keyBytes,
		})
	}

	h := sha256.Sum256([]byte(artifactBytes))
	re.IntotoObj.Content.Hash = &models.IntotoV002SchemaContentHash{
		Algorithm: swag.String(models.IntotoV001SchemaContentHashAlgorithmSha256),
		Value:     swag.String(hex.EncodeToString(h[:])),
	}

	returnVal.Spec = re.IntotoObj
	returnVal.APIVersion = swag.String(re.APIVersion())

	return &returnVal, nil
}

// verifyEnvelope takes in an array of possible key bytes and attempts to parse them as x509 public keys.
// it then uses these to verify the envelope and makes sure that every signature on the envelope is verified.
// it returns a map of verifiers indexed by the signature the verifier corresponds to.
func verifyEnvelope(allPubKeyBytes [][]byte, env *dsse.Envelope) (map[string]*x509.PublicKey, error) {
	// generate a fake id for these keys so we can get back to the key bytes and match them to their corresponding signature
	verifierBySig := make(map[string]*x509.PublicKey)
	allSigs := make(map[string]struct{})
	for _, sig := range env.Signatures {
		allSigs[sig.Sig] = struct{}{}
	}

	for _, pubKeyBytes := range allPubKeyBytes {
		key, err := x509.NewPublicKey(bytes.NewReader(pubKeyBytes))
		if err != nil {
			return nil, fmt.Errorf("could not parse public key as x509: %w", err)
		}

		vfr, err := signature.LoadVerifier(key.CryptoPubKey(), crypto.SHA256)
		if err != nil {
			return nil, fmt.Errorf("could not load verifier: %w", err)
		}

		dsseVfr, err := dsse.NewEnvelopeVerifier(&verifier{
			v: vfr,
		})

		if err != nil {
			return nil, fmt.Errorf("could not use public key as a dsse verifier: %w", err)
		}

		accepted, err := dsseVfr.Verify(context.Background(), env)
		if err != nil {
			return nil, fmt.Errorf("could not verify envelope: %w", err)
		}

		for _, accept := range accepted {
			delete(allSigs, accept.Sig.Sig)
			verifierBySig[accept.Sig.Sig] = key
		}
	}

	if len(allSigs) > 0 {
		return nil, errors.New("all signatures must have a key that verifies it")
	}

	return verifierBySig, nil
}

func (v V002Entry) Verifiers() ([]pki.PublicKey, error) {
	if v.IntotoObj.Content == nil || v.IntotoObj.Content.Envelope == nil {
		return nil, errors.New("intoto v0.0.2 entry not initialized")
	}

	sigs := v.IntotoObj.Content.Envelope.Signatures
	if len(sigs) == 0 {
		return nil, errors.New("no signatures found on intoto entry")
	}

	var keys []pki.PublicKey
	for _, s := range v.IntotoObj.Content.Envelope.Signatures {
		key, err := x509.NewPublicKey(bytes.NewReader(*s.PublicKey))
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (v V002Entry) ArtifactHash() (string, error) {
	if v.IntotoObj.Content == nil || v.IntotoObj.Content.PayloadHash == nil || v.IntotoObj.Content.PayloadHash.Algorithm == nil || v.IntotoObj.Content.PayloadHash.Value == nil {
		return "", errors.New("intoto v0.0.2 entry not initialized")
	}
	return strings.ToLower(fmt.Sprintf("%s:%s", *v.IntotoObj.Content.PayloadHash.Algorithm, *v.IntotoObj.Content.PayloadHash.Value)), nil
}

func (v V002Entry) Insertable() (bool, error) {
	if v.IntotoObj.Content == nil {
		return false, errors.New("missing content property")
	}
	if v.IntotoObj.Content.Envelope == nil {
		return false, errors.New("missing envelope property")
	}
	if len(v.IntotoObj.Content.Envelope.Payload) == 0 {
		return false, errors.New("missing envelope content")
	}

	if v.IntotoObj.Content.Envelope.PayloadType == nil || len(*v.IntotoObj.Content.Envelope.PayloadType) == 0 {
		return false, errors.New("missing payloadType content")
	}

	if len(v.IntotoObj.Content.Envelope.Signatures) == 0 {
		return false, errors.New("missing signatures content")
	}
	for _, sig := range v.IntotoObj.Content.Envelope.Signatures {
		if sig == nil {
			return false, errors.New("missing signature entry")
		}
		if sig.Sig == nil || len(*sig.Sig) == 0 {
			return false, errors.New("missing signature content")
		}
		if sig.PublicKey == nil || len(*sig.PublicKey) == 0 {
			return false, errors.New("missing publicKey content")
		}
	}

	if v.env.Payload == "" || v.env.PayloadType == "" || len(v.env.Signatures) == 0 {
		return false, errors.New("invalid DSSE envelope")
	}

	return true, nil
}
