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

// EnvKeychain creates a keychain augmenting the default keychain with information from an environment variable.
//
// See `ReadEnvVar`.
func EnvKeychain(envVar string) authn.Keychain {
	return authn.NewMultiKeychain(&envKeychain{envVar: envVar}, authn.DefaultKeychain)
}

type envKeychain struct {
	envVar string
}

func (k *envKeychain) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	authHeaders, err := ReadEnvVar(k.envVar)
	if err != nil {
		return nil, errors.Wrap(err, "reading auth env var")
	}

	header, ok := authHeaders[resource.RegistryStr()]
	if ok {
		authConfig, err := authHeaderToConfig(header)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing auth header '%s'", header)
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

// BuildEnvVar creates the contents to use for authentication environment variable.
//
// Complementary to `ReadEnvVar`.
func BuildEnvVar(keychain authn.Keychain, images ...string) (string, error) {
	registryAuths := map[string]string{}

	for _, image := range images {
		reference, authenticator, err := ReferenceForRepoName(keychain, image)
		if err != nil {
			return "", nil
		}
		if authenticator == authn.Anonymous {
			continue
		}

		authConfig, err := authenticator.Authorization()
		if err != nil {
			return "", nil
		}

		registryAuths[reference.Context().Registry.Name()], err = authConfigToHeader(authConfig)
		if err != nil {
			return "", nil
		}
	}

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

	return nil, errors.Errorf("unknown auth type from header: %s", header)
}

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
