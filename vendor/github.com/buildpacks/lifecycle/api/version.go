package api

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

var regex = regexp.MustCompile(`^v?(\d+)\.?(\d*)$`)

type Version struct {
	Major,
	Minor uint64
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
			return nil, errors.Wrapf(err, "parsing Major '%s'", matches[0][1])
		}

		if matches[0][2] == "" {
			minor = 0
		} else {
			minor, err = strconv.ParseUint(matches[0][2], 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "parsing Minor '%s'", matches[0][2])
			}
		}
	} else {
		return nil, errors.Errorf("could not parse version '%s'", v)
	}

	return &Version{Major: major, Minor: minor}, nil
}

func (v *Version) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
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

	v.Major = parsedVersion.Major
	v.Minor = parsedVersion.Minor

	return nil
}

func (v *Version) Equal(o *Version) bool {
	return v.Compare(o) == 0
}

// Compare returns one of the following results
//   -1 is less than *Version o
//    0 is equal to *Version o
//    1 is greater than *Version o
func (v *Version) Compare(o *Version) int {
	if v.Major != o.Major {
		if v.Major < o.Major {
			return -1
		}

		if v.Major > o.Major {
			return 1
		}
	}

	if v.Minor != o.Minor {
		if v.Minor < o.Minor {
			return -1
		}

		if v.Minor > o.Minor {
			return 1
		}
	}

	return 0
}

func (v *Version) IsSupersetOf(o *Version) bool {
	if v.Major == 0 {
		return v.Equal(o)
	}
	return v.Major == o.Major && v.Minor >= o.Minor
}

func (v *Version) LessThan(other string) bool {
	otherVersion := MustParse(other)
	return v.Compare(otherVersion) < 0
}

func (v *Version) AtLeast(other string) bool {
	otherVersion := MustParse(other)
	return v.Compare(otherVersion) >= 0
}
