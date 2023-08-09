package here

import (
	"encoding/json"
	"fmt"
)

// Package attempts to gather info for the requested package.
//
// From the `go help list` docs:
//	The -find flag causes list to identify the named packages but not
//	resolve their dependencies: the Imports and Deps lists will be empty.
//
// A workaround for this issue is to use the `Dir` field in the
// returned `Info` value and pass it to the `Dir(string) (Info, error)`
// function to return the complete data.
func (h Here) Package(p string) (Info, error) {
	i, err := h.cache(p, func(p string) (Info, error) {
		var i Info
		if len(p) == 0 || p == "." {
			return i, fmt.Errorf("missing package name")
		}
		b, err := run("go", "list", "-json", "-find", p)
		if err != nil {
			return i, err
		}
		if err := json.Unmarshal(b, &i); err != nil {
			return i, err
		}

		return i, nil
	})

	if err != nil {
		return i, err
	}

	h.cache(i.Dir, func(p string) (Info, error) {
		return i, nil
	})

	return i, nil

}

// Package attempts to gather info for the requested package.
//
// From the `go help list` docs:
//	The -find flag causes list to identify the named packages but not
//	resolve their dependencies: the Imports and Deps lists will be empty.
//
// A workaround for this issue is to use the `Dir` field in the
// returned `Info` value and pass it to the `Dir(string) (Info, error)`
// function to return the complete data.
func Package(p string) (Info, error) {
	return New().Package(p)
}
