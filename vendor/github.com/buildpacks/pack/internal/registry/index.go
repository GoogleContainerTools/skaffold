package registry

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
)

var (
	validCharsPattern = "[a-z0-9\\-.]+"
	validCharsRegexp  = regexp.MustCompile(fmt.Sprintf("^%s$", validCharsPattern))
)

// IndexPath resolves the path for a specific namespace and name of buildpack
func IndexPath(rootDir, ns, name string) (string, error) {
	if err := validateField("namespace", ns); err != nil {
		return "", err
	}

	if err := validateField("name", name); err != nil {
		return "", err
	}

	var indexDir string
	switch {
	case len(name) == 1:
		indexDir = "1"
	case len(name) == 2:
		indexDir = "2"
	case len(name) == 3:
		indexDir = filepath.Join("3", name[:2])
	default:
		indexDir = filepath.Join(name[:2], name[2:4])
	}

	return filepath.Join(rootDir, indexDir, fmt.Sprintf("%s_%s", ns, name)), nil
}

func validateField(field, value string) error {
	length := len(value)
	switch {
	case length == 0:
		return errors.Errorf("%s cannot be empty", style.Symbol(field))
	case length > 253:
		return errors.Errorf("%s too long (max 253 chars)", style.Symbol(field))
	}

	if !validCharsRegexp.MatchString(value) {
		return errors.Errorf("%s contains illegal characters (must match %s)", style.Symbol(field), style.Symbol(validCharsPattern))
	}

	return nil
}
