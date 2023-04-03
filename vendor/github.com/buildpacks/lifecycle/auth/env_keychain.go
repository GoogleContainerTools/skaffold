package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
)

const EnvRegistryAuth = "CNB_REGISTRY_AUTH"

// DefaultKeychain returns a keychain containing authentication configuration for the given images
// from the following sources, if they exist, in order of precedence:
// the provided environment variable
// the docker config.json file
func DefaultKeychain(images ...string) (authn.Keychain, error) {
	envKeychain, err := EnvKeychain(EnvRegistryAuth)
	if err != nil {
		return nil, err
	}

	return authn.NewMultiKeychain(
		envKeychain,
		InMemoryKeychain(authn.DefaultKeychain, images...),
	), nil
}

// ResolvedKeychain is an implementation of authn.Keychain that stores credentials in memory.
type ResolvedKeychain struct {
	Auths map[string]string
}

// EnvKeychain returns an authn.Keychain that uses the provided environment variable as a source of credentials.
// The value of the environment variable should be a JSON object that maps OCI registry hostnames to Authorization headers.
func EnvKeychain(envVar string) (authn.Keychain, error) {
	authHeaders, err := ReadEnvVar(envVar)
	if err != nil {
		return nil, errors.Wrap(err, "reading auth env var")
	}
	return &ResolvedKeychain{Auths: authHeaders}, nil
}

// InMemoryKeychain resolves credentials for the given images from the given keychain and returns a new keychain
// that stores the pre-resolved credentials in memory and returns them on demand. This is useful in cases where the
// backing credential store may become inaccessible in the the future.
func InMemoryKeychain(keychain authn.Keychain, images ...string) authn.Keychain {
	return &ResolvedKeychain{
		Auths: buildAuthMap(keychain, images...),
	}
}

func (k *ResolvedKeychain) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	header, ok := k.Auths[resource.RegistryStr()]
	if ok {
		authConfig, err := authHeaderToConfig(header)
		if err != nil {
			return nil, errors.Wrap(err, "parsing auth header")
		}

		return &providedAuth{config: authConfig}, nil
	}

	return authn.Anonymous, nil
}

type providedAuth struct {
	config *authn.AuthConfig
}

func (p *providedAuth) Authorization() (*authn.AuthConfig, error) {
	return p.config, nil
}

// ReadEnvVar parses an environment variable to produce a map of 'registry url' to 'authorization header'.
//
// Complementary to `BuildEnvVar`.
//
// Example Input:
// 	{"gcr.io": "Bearer asdf=", "docker.io": "Basic qwerty="}
//
// Example Output:
//  gcr.io -> Bearer asdf=
//  docker.io -> Basic qwerty=
func ReadEnvVar(envVar string) (map[string]string, error) {
	authMap := map[string]string{}

	env := os.Getenv(envVar)
	if env != "" {
		err := json.Unmarshal([]byte(env), &authMap)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %s value", envVar)
		}
	}

	return authMap, nil
}

func buildAuthMap(keychain authn.Keychain, images ...string) map[string]string {
	registryAuths := map[string]string{}

	for _, image := range images {
		reference, authenticator, err := ReferenceForRepoName(keychain, image)
		if err != nil {
			continue
		}

		if authenticator == authn.Anonymous {
			continue
		}

		authConfig, err := authenticator.Authorization()
		if err != nil {
			continue
		}

		header, err := authConfigToHeader(authConfig)
		if err != nil {
			continue
		}
		registryAuths[reference.Context().Registry.Name()] = header
	}

	return registryAuths
}

// BuildEnvVar creates the contents to use for authentication environment variable.
//
// Complementary to `ReadEnvVar`.
func BuildEnvVar(keychain authn.Keychain, images ...string) (string, error) {
	registryAuths := buildAuthMap(keychain, images...)

	authData, err := json.Marshal(registryAuths)
	if err != nil {
		return "", err
	}
	return string(authData), nil
}

func authConfigToHeader(config *authn.AuthConfig) (string, error) {
	if config.Auth != "" {
		return fmt.Sprintf("Basic %s", config.Auth), nil
	}

	if config.RegistryToken != "" {
		return fmt.Sprintf("Bearer %s", config.RegistryToken), nil
	}

	if config.Username != "" && config.Password != "" {
		delimited := fmt.Sprintf("%s:%s", config.Username, config.Password)
		encoded := base64.StdEncoding.EncodeToString([]byte(delimited))
		return fmt.Sprintf("Basic %s", encoded), nil
	}

	return "", nil
}

var (
	basicAuthRegExp  = regexp.MustCompile("(?i)^basic (.*)$")
	bearerAuthRegExp = regexp.MustCompile("(?i)^bearer (.*)$")
)

func authHeaderToConfig(header string) (*authn.AuthConfig, error) {
	if matches := basicAuthRegExp.FindAllStringSubmatch(header, -1); len(matches) != 0 {
		return &authn.AuthConfig{
			Auth: matches[0][1],
		}, nil
	}

	if matches := bearerAuthRegExp.FindAllStringSubmatch(header, -1); len(matches) != 0 {
		return &authn.AuthConfig{
			RegistryToken: matches[0][1],
		}, nil
	}

	return nil, errors.New("unknown auth type from header")
}

// ReferenceForRepoName returns a reference and an authenticator for a given image name and keychain.
func ReferenceForRepoName(keychain authn.Keychain, ref string) (name.Reference, authn.Authenticator, error) {
	var auth authn.Authenticator
	r, err := name.ParseReference(ref, name.WeakValidation)
	if err != nil {
		return nil, nil, err
	}

	auth, err = keychain.Resolve(r.Context().Registry)
	if err != nil {
		return nil, nil, err
	}
	return r, auth, nil
}
