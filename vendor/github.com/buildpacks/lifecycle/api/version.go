package api

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

var regex = regexp.MustCompile(`^v?(\d+)\.(\d*)$`)

type Version struct {
	major,
	minor uint64
}

func MustParse(v string) *Version {
	version, err := NewVersion(v)
	if err != nil {
		panic(err)
	}

	return version
}

func NewVersion(v string) (*Version, error) {
	matches := regex.FindAllStringSubmatch(v, -1)
	if len(matches) == 0 {
		return nil, errors.Errorf("could not parse '%s' as version", v)
	}

	var (
		major, minor uint64
		err          error
	)
	if len(matches[0]) == 3 {
		major, err = strconv.ParseUint(matches[0][1], 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing major '%s'", matches[0][1])
		}

		minor, err = strconv.ParseUint(matches[0][2], 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing minor '%s'", matches[0][2])
		}
	} else {
		return nil, errors.Errorf("could not parse version '%s'", v)
	}

	return &Version{major: major, minor: minor}, nil
}

func (v *Version) String() string {
	return fmt.Sprintf("%d.%d", v.major, v.minor)
}

// MarshalText makes Version satisfy the encoding.TextMarshaler interface.
func (v *Version) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

// UnmarshalText makes Version satisfy the encoding.TextUnmarshaler interface.
func (v *Version) UnmarshalText(text []byte) error {
	s := string(text)

	parsedVersion, err := NewVersion(s)
	if err != nil {
		return errors.Wrapf(err, "invalid api version '%s'", s)
	}

	v.major = parsedVersion.major
	v.minor = parsedVersion.minor

	return nil
}

func (v *Version) Equal(o *Version) bool {
	return v.Compare(o) == 0
}

func (v *Version) Compare(o *Version) int {
	if v.major != o.major {
		if v.major < o.major {
			return -1
		}

		if v.major > o.major {
			return 1
		}
	}

	if v.minor != o.minor {
		if v.minor < o.minor {
			return -1
		}

		if v.minor > o.minor {
			return 1
		}
	}

	return 0
}

// IsAPICompatible determines if the lifecycle's API version is compatible with another's API version.
//
// Example Usage Pseudocode:
//
//	IsAPICompatible(Platform API from Lifecycle, Platform API from Platform)
//	IsAPICompatible(Buildpack API from Lifecycle, Buildpack API from Buildpack)
//
func IsAPICompatible(apiFromLifecycle, apiFromOther *Version) bool {
	if apiFromLifecycle.Equal(apiFromOther) {
		return true
	}

	if apiFromLifecycle.major != 0 {
		return apiFromLifecycle.major == apiFromOther.major && apiFromLifecycle.minor >= apiFromOther.minor
	}

	return false
}
