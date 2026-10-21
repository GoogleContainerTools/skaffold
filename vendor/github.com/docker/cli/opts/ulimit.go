// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.26

package opts

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/container"
)

// UlimitOpt defines a map of Ulimits
type UlimitOpt struct {
	values *map[string]*container.Ulimit
}

// NewUlimitOpt creates a new UlimitOpt. Ulimits are not validated.
func NewUlimitOpt(ref *map[string]*container.Ulimit) *UlimitOpt {
	// TODO(thaJeztah): why do we need a map with pointers here?
	if ref == nil {
		ref = &map[string]*container.Ulimit{}
	}
	return &UlimitOpt{ref}
}

// Set validates a Ulimit and sets its name as a key in UlimitOpt
func (o *UlimitOpt) Set(val string) error {
	// FIXME(thaJeztah): these functions also need to be moved over from go-units.
	l, err := units.ParseUlimit(val)
	if err != nil {
		return err
	}

	(*o.values)[l.Name] = l

	return nil
}

// String returns Ulimit values as a string. Values are sorted by name.
func (o *UlimitOpt) String() string {
	out := make([]string, 0, len(*o.values))
	for _, v := range *o.values {
		out = append(out, v.String())
	}
	slices.Sort(out)
	return fmt.Sprint(out)
}

// GetList returns a slice of pointers to Ulimits. Values are sorted by name.
func (o *UlimitOpt) GetList() []*container.Ulimit {
	return slices.SortedFunc(maps.Values(*o.values), func(a, b *container.Ulimit) int {
		return strings.Compare(a.Name, b.Name)
	})
}

// Type returns the option type
func (*UlimitOpt) Type() string {
	return "ulimit"
}
