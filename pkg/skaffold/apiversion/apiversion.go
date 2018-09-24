/*
Copyright 2018 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package apiversion

import (
	"fmt"
	"regexp"
	"strconv"
)

type ReleaseTrack int

const (
	alpha ReleaseTrack = 0
	beta  ReleaseTrack = 1
	ga    ReleaseTrack = 2
)

type Version struct {
	Major   int
	Minor   int
	Release ReleaseTrack
}

var re = regexp.MustCompile(`^skaffold/v(\d)(?:(alpha|beta)(\d))?$`)

// ParseAPIVersion parses a string into a Version.
func ParseVersion(v string) (*Version, error) {
	res := re.FindStringSubmatch(v)
	if len(res) == 0 {
		return nil, fmt.Errorf("%s is an invalid api version", v)
	}

	major, err := strconv.Atoi(res[1])
	if err != nil {
		return nil, fmt.Errorf("%s is an invalid major version number", res[1])
	}

	track := ga
	switch res[2] {
	case "alpha":
		track = alpha
	case "beta":
		track = beta
	}

	av := Version{
		Major:   major,
		Release: track,
	}

	if track != ga {
		minor, err := strconv.Atoi(res[3])
		if err != nil {
			return nil, fmt.Errorf("%s is an invalid major version number", res[1])
		}
		av.Minor = minor
	}
	return &av, nil
}

// MustParseVersion parses the version and panics if there is an error.
func MustParseVersion(v string) *Version {
	av, err := ParseVersion(v)
	if err != nil {
		panic(err)
	}
	return av
}

// Compare compares the Version to another Version, and returns -1 if v is less than ov, 0 if they are equal and 1 if v is greater than ov.
func (v *Version) Compare(ov *Version) int {
	// GA is always higher than beta, beta is always higher than alpha
	if v.Release != ov.Release {
		if v.Release > ov.Release {
			return 1
		}
		return -1
	}

	// v2alpha > v1beta
	if v.Major != ov.Major {
		if v.Major > ov.Major {
			return 1
		}
		return -1
	}

	if v.Minor > ov.Minor {
		return 1
	}
	if v.Minor < ov.Minor {
		return -1
	}
	return 0
}

// LT returns true if v is less than ov
func (v *Version) LT(ov *Version) bool {
	return v.Compare(ov) == -1
}

// GT returns true if v is greather than ov
func (v *Version) GT(ov *Version) bool {
	return v.Compare(ov) == 1
}
