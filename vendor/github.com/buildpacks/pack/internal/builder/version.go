package builder

import (
	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
)

// Version is an extension to semver.Version to make it marshalable.
type Version struct {
	semver.Version
}

// VersionMustParse parses a string into a Version
func VersionMustParse(v string) *Version {
	return &Version{Version: *semver.MustParse(v)}
}

// String returns the string value of the Version
func (v *Version) String() string {
	return v.Version.String()
}

// Equal compares two Versions
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
