package registry

import (
	"fmt"
	"strings"

	ggcrname "github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
)

// Buildpack contains information about a buildpack stored in a Registry
type Buildpack struct {
	Namespace string `json:"ns"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Yanked    bool   `json:"yanked"`
	Address   string `json:"addr,omitempty"`
}

// Validate that a buildpack reference contains required information
func Validate(b Buildpack) error {
	if b.Address == "" {
		return errors.New("invalid entry: address is a required field")
	}
	_, err := ggcrname.NewDigest(b.Address)
	if err != nil {
		return fmt.Errorf("invalid entry: '%s' is not a digest reference", b.Address)
	}

	return nil
}

func ParseNamespaceName(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid id %s does not contain a namespace", style.Symbol(id))
	} else if len(parts) > 2 {
		return "", "", fmt.Errorf("invalid id %s contains unexpected characters", style.Symbol(id))
	}

	return parts[0], parts[1], nil
}
