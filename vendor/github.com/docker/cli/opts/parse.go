package opts

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/docker/cli/pkg/kvfile"
	"github.com/moby/moby/api/types/container"
)

// ReadKVStrings reads a file of line terminated key=value pairs, and overrides any keys
// present in the file with additional pairs specified in the override parameter
func ReadKVStrings(files []string, override []string) ([]string, error) {
	return readKVStrings(files, override, nil)
}

// ReadKVEnvStrings reads a file of line terminated key=value pairs, and overrides any keys
// present in the file with additional pairs specified in the override parameter.
// If a key has no value, it will get the value from the environment.
func ReadKVEnvStrings(files []string, override []string) ([]string, error) {
	return readKVStrings(files, override, os.LookupEnv)
}

func readKVStrings(files []string, override []string, emptyFn func(string) (string, bool)) ([]string, error) {
	var variables []string
	for _, ef := range files {
		parsedVars, err := kvfile.Parse(ef, emptyFn)
		if err != nil {
			return nil, err
		}
		variables = append(variables, parsedVars...)
	}
	// parse the '-e' and '--env' after, to allow override
	variables = append(variables, override...)

	return variables, nil
}

// ConvertKVStringsToMap converts ["key=value"] to {"key":"value"}
func ConvertKVStringsToMap(values []string) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		k, v, _ := strings.Cut(value, "=")
		result[k] = v
	}

	return result
}

// ConvertKVStringsToMapWithNil converts ["key=value"] to {"key":"value"}
// but set unset keys to nil - meaning the ones with no "=" in them.
// We use this in cases where we need to distinguish between
//
//	FOO=  and FOO
//
// where the latter case just means FOO was mentioned but not given a value
func ConvertKVStringsToMapWithNil(values []string) map[string]*string {
	result := make(map[string]*string, len(values))
	for _, value := range values {
		k, v, ok := strings.Cut(value, "=")
		if !ok {
			result[k] = nil
		} else {
			result[k] = &v
		}
	}

	return result
}

// ParseRestartPolicy parses a restart policy string ("name[:max-retries]")
// into a [container.RestartPolicy].
//
// Parsing is syntactic only; semantic validation is deferred to the daemon/API.
// An empty input returns a zero-value policy for backward compatibility. The
// retry count, if set, must be an integer (negative values are allowed here
// but may be rejected by the daemon).
func ParseRestartPolicy(policy string) (container.RestartPolicy, error) {
	if policy == "" {
		// For backward compatibility, do not set an explicit default ("no"),
		// as older daemons may not support it.
		return container.RestartPolicy{}, nil
	}

	name, count, ok := strings.Cut(policy, ":")
	if ok && name == "" {
		return container.RestartPolicy{}, errors.New("invalid restart policy format: no policy provided before colon")
	}

	var retryCount int
	if count != "" {
		c, err := strconv.Atoi(count)
		if err != nil {
			return container.RestartPolicy{}, errors.New("invalid restart policy format: maximum retry count must be an integer")
		}
		retryCount = c
	}

	return container.RestartPolicy{
		Name:              container.RestartPolicyMode(name),
		MaximumRetryCount: retryCount,
	}, nil
}
