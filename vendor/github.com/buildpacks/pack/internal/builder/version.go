package builder

import (
	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
)

// Version is an extension to semver.Version to make it marshalable.
type Version struct {
	semver.Version
}

func VersionMustParse(v string) *Version {
	return &Version{Version: *semver.MustParse(v)}
}

func (v *Version) String() string {
	return v.Version.String()
}

func (v *Version) Equal(other *Version) bool {
	if other != nil {
		return v.Version.Equal(&other.Version)
	}

	return false
}

// MarshalText makes Version satisfy the encoding.TextMarshaler interface.
func (v *Version) MarshalText() ([]byte, error) {
	return []byte(v.Version.Original()), nil
}

// UnmarshalText makes Version satisfy the encoding.TextUnmarshaler interface.
func (v *Version) UnmarshalText(text []byte) error {
	s := string(text)
	w, err := semver.NewVersion(s)
	if err != nil {
		return errors.Wrapf(err, "invalid semantic version %s", s)
	}

	v.Version = *w
	return nil
}
