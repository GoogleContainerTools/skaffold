/*
Copyright 2019 The Knative Authors

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

package apis

import (
	"encoding/json"
	"fmt"
	"net/url"

	"k8s.io/apimachinery/pkg/api/equality"
)

// URL is an alias of url.URL.
// It has custom json marshal methods that enable it to be used in K8s CRDs
// such that the CRD resource will have the URL but operator code can can work with url.URL struct
type URL url.URL

// ParseURL attempts to parse the given string as a URL.
// Compatible with net/url.Parse except in the case of an empty string, where
// the resulting *URL will be nil with no error.
func ParseURL(u string) (*URL, error) {
	if u == "" {
		return nil, nil
	}
	pu, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	return (*URL)(pu), nil
}

// HTTP creates an http:// URL pointing to a known domain.
func HTTP(domain string) *URL {
	return &URL{
		Scheme: "http",
		Host:   domain,
	}
}

// HTTPS creates an https:// URL pointing to a known domain.
func HTTPS(domain string) *URL {
	return &URL{
		Scheme: "https",
		Host:   domain,
	}
}

// IsEmpty returns true if the URL is `nil` or represents an empty URL.
func (u *URL) IsEmpty() bool {
	if u == nil {
		return true
	}
	return *u == URL{}
}

// MarshalJSON implements a custom json marshal method used when this type is
// marshaled using json.Marshal.
// json.Marshaler impl
func (u URL) MarshalJSON() ([]byte, error) {
	b := fmt.Sprintf("%q", u.String())
	return []byte(b), nil
}

// UnmarshalJSON implements the json unmarshal method used when this type is
// unmarsheled using json.Unmarshal.
// json.Unmarshaler impl
func (u *URL) UnmarshalJSON(b []byte) error {
	var ref string
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	if r, err := ParseURL(ref); err != nil {
		return err
	} else if r != nil {
		*u = *r
	} else {
		*u = URL{}
	}

	return nil
}

// String returns the full string representation of the URL.
func (u *URL) String() string {
	if u == nil {
		return ""
	}
	uu := url.URL(*u)
	return uu.String()
}

// URL returns the URL as a url.URL.
func (u *URL) URL() *url.URL {
	if u == nil {
		return &url.URL{}
	}
	url := url.URL(*u)
	return &url
}

// ResolveReference calls the underlying ResolveReference method
// and returns an apis.URL
func (u *URL) ResolveReference(ref *URL) *URL {
	if ref == nil {
		return u
	}
	// Turn both u / ref to url.URL
	uRef := url.URL(*ref)
	uu := url.URL(*u)

	newU := uu.ResolveReference(&uRef)

	// Turn new back to apis.URL
	ret := URL(*newU)
	return &ret
}

func init() {
	equality.Semantic.AddFunc(
		// url.URL has an unexported type (UserInfo) which causes semantic
		// equality to panic unless we add a custom equality function
		func(a, b URL) bool {
			return a.String() == b.String()
		},
	)
}
