package registry

import (
	"fmt"

	ggcrname "github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
)

// Buildpack contains information about a buildpack stored in a Registry
type Buildpack struct {
	Namespace string `json:"ns"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Yanked    bool   `json:"yanked"`
	Address   string `json:"addr"`
}

// Validate that a buildpack reference contains required information
func (b *Buildpack) Validate() error {
	if b.Address == "" {
		return errors.New("invalid entry: address is a required field")
	}
	_, err := ggcrname.NewDigest(b.Address)
	if err != nil {
		return fmt.Errorf("invalid entry: '%s' is not a digest reference", b.Address)
	}

	return nil
}
