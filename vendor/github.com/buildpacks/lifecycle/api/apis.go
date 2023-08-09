package api

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

var (
	Platform  = newApisMustParse([]string{"0.3", "0.4", "0.5", "0.6", "0.7", "0.8"}, nil)
	Buildpack = newApisMustParse([]string{"0.2", "0.3", "0.4", "0.5", "0.6", "0.7"}, nil)
)

type APIs struct {
	Supported  List
	Deprecated List
}

type List []*Version

func (l List) String() string {
	var els []string
	for _, el := range l {
		els = append(els, fmt.Sprintf("%q", el.String()))
	}
	return "[" + strings.Join(els, ", ") + "]"
}

// newApisMustParse calls NewApis and panics on error
func newApisMustParse(supported []string, deprecated []string) APIs {
	apis, err := NewAPIs(supported, deprecated)
	if err != nil {
		panic(err)
	}
	return apis
}

// NewApis constructs an instance of APIs
//  supported must be a superset of deprecated
//  deprecated APIs greater than 1.0 should should not include minor versions
//  supported APIs should always include minor versions
//  Examples:
//     deprecated API 1 implies all 1.x APIs are deprecated
//     supported API 1 implies only 1.0 is supported
func NewAPIs(supported []string, deprecated []string) (APIs, error) {
	apis := APIs{}
	for _, api := range supported {
		apis.Supported = append(apis.Supported, MustParse(api))
	}
	for _, d := range deprecated {
		dAPI := MustParse(d)
		if err := validateDeprecated(apis, dAPI); err != nil {
			return APIs{}, errors.Wrapf(err, "invalid deprecated API '%s'", d)
		}
		apis.Deprecated = append(apis.Deprecated, dAPI)
	}
	return apis, nil
}

func validateDeprecated(apis APIs, deprecated *Version) error {
	if !apis.IsSupported(deprecated) {
		return errors.New("all deprecated APIs must also be supported")
	}
	if deprecated.Major != 0 && deprecated.Minor != 0 {
		return errors.New("deprecated APIs may only contain 0.x or a major version")
	}
	return nil
}

// IsSupported returns true or false depending on whether the target API is supported
func (a APIs) IsSupported(target *Version) bool {
	for _, sAPI := range a.Supported {
		if sAPI.IsSupersetOf(target) {
			return true
		}
	}
	return false
}

// IsDeprecated returns true or false depending on whether the target API is deprecated
func (a APIs) IsDeprecated(target *Version) bool {
	for _, dAPI := range a.Deprecated {
		if target.IsSupersetOf(dAPI) {
			return true
		}
	}
	return false
}

// Latest returns the latest API that is supported
func (a APIs) Latest() *Version {
	latest := a.Supported[0]
	for _, sAPI := range a.Supported {
		if sAPI.Compare(latest) > 0 {
			latest = sAPI
		}
	}
	return latest
}
