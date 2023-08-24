package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"

	ecr "github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	"github.com/chrismellard/docker-credential-acr-env/pkg/credhelper"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
)

const EnvRegistryAuth = "CNB_REGISTRY_AUTH"

var (
	amazonKeychain = authn.NewKeychainFromHelper(ecr.NewECRHelper(ecr.WithLogger(io.Discard)))
	azureKeychain  = authn.NewKeychainFromHelper(credhelper.NewACRCredentialsHelper())
)

// DefaultKeychain returns a keychain containing authentication configuration for the given images
// from the following sources, if they exist, in order of precedence:
// the provided environment variable
// the docker config.json file
// credential helpers for Amazon and Azure
func DefaultKeychain(images ...string) (authn.Keychain, error) {
	envKeychain, err := NewEnvKeychain(EnvRegistryAuth)
	if err != nil {
		return nil, err
	}

	return authn.NewMultiKeychain(
		envKeychain,
		NewResolvedKeychain(authn.DefaultKeychain, images...),
		NewResolvedKeychain(amazonKeychain, images...),
		NewResolvedKeychain(azureKeychain, images...),
	), nil
}

// NewEnvKeychain returns an authn.Keychain that uses the provided environment variable as a source of credentials.
// The value of the environment variable should be a JSON object that maps OCI registry hostnames to Authorization headers.
func NewEnvKeychain(envVar string) (authn.Keychain, error) {
	authHeaders, err := ReadEnvVar(envVar)
	if err != nil {
		return nil, errors.Wrap(err, "reading auth env var")
	}
	return &EnvKeychain{AuthHeaders: authHeaders}, nil
}

// EnvKeychain is an implementation of authn.Keychain that stores credentials as auth headers.
type EnvKeychain struct {
	AuthHeaders map[string]string
}

func (k *EnvKeychain) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	header, ok := k.AuthHeaders[resource.RegistryStr()]
	if ok {
		authConfig, err := authHeaderToConfig(header)
		if err != nil {
			return nil, errors.Wrap(err, "parsing auth header")
		}
		return &providedAuth{config: authConfig}, nil
	}
	return authn.Anonymous, nil
}

var (
	basicAuthRegExp     = regexp.MustCompile("(?i)^basic (.*)$")
	bearerAuthRegExp    = regexp.MustCompile("(?i)^bearer (.*)$")
	identityTokenRegExp = regexp.MustCompile("(?i)^x-identity (.*)$")
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

	if matches := identityTokenRegExp.FindAllStringSubmatch(header, -1); len(matches) != 0 {
		return &authn.AuthConfig{
			IdentityToken: matches[0][1],
		}, nil
	}

	return nil, errors.New("unknown auth type from header")
}

type providedAuth struct {
	config *authn.AuthConfig
}

func (p *providedAuth) Authorization() (*authn.AuthConfig, error) {
	return p.config, nil
}

// NewResolvedKeychain resolves credentials for the given images from the given keychain and returns a new keychain
// that stores the pre-resolved credentials in memory and returns them on demand. This is useful in cases where the
// backing credential store may become inaccessible in the future.
func NewResolvedKeychain(keychain authn.Keychain, images ...string) authn.Keychain {
	return &ResolvedKeychain{
		AuthConfigs: buildAuthConfigs(keychain, images...),
	}
}

func buildAuthConfigs(keychain authn.Keychain, images ...string) map[string]*authn.AuthConfig {
	registryAuths := map[string]*authn.AuthConfig{}
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
		if *authConfig == (authn.AuthConfig{}) {
			continue
		}
		registryAuths[reference.Context().Registry.Name()] = authConfig
	}
	return registryAuths
}

// ResolvedKeychain is an implementation of authn.Keychain that stores credentials in memory.
type ResolvedKeychain struct {
	AuthConfigs map[string]*authn.AuthConfig
}

func (k *ResolvedKeychain) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	authConfig, ok := k.AuthConfigs[resource.RegistryStr()]
	if ok {
		return &providedAuth{config: authConfig}, nil
	}
	return authn.Anonymous, nil
}

// ReadEnvVar parses an environment variable to produce a map of 'registry url' to 'authorization header'.
//
// Complementary to `BuildEnvVar`.
//
// Example Input:
//
//	{"gcr.io": "Bearer asdf=", "docker.io": "Basic qwerty="}
//
// Example Output:
//
//	gcr.io -> Bearer asdf=
//	docker.io -> Basic qwerty=
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
	registryAuths := buildAuthHeaders(keychain, images...)

	authData, err := json.Marshal(registryAuths)
	if err != nil {
		return "", err
	}
	return string(authData), nil
}

func buildAuthHeaders(keychain authn.Keychain, images ...string) map[string]string {
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

// authConfigToHeader accepts an authn.AuthConfig and returns an Authorization header,
// or an error if the config cannot be processed.
// Note that when resolving credentials, the header is simply used to reconstruct the originally provided authn.AuthConfig,
// making it essentially a stringification (the actual value is unimportant as long as it is consistent and contains
// all the necessary information).
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

	if config.IdentityToken != "" {
		// There isn't an Authorization header for identity tokens, but we just need a way to represent the data.
		return fmt.Sprintf("X-Identity %s", config.IdentityToken), nil
	}

	return "", errors.New("failed to find authorization information")
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
