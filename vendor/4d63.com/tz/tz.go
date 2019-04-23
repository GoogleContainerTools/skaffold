// Package tz contains time zone info so that you can predictably load
// Locations regardless of the locations available on the locally running
// operating system.
//
// The stdlib time.LoadLocation function loads timezone data from the operating
// system or from zoneinfo.zip in a local Go installation. Both of these are
// often missing from some operating systems, especially Windows.
//
// This package has the zoneinfo.zip from Go embedded into the package so that
// queries to load a location always return the same data regardless of
// operating system.
//
// This package exists because of https://github.com/golang/go/issues/21881.
package tz // import "4d63.com/tz"

import (
	"errors"
	"time"
)

//go:generate rm -fr zoneinfo
//go:generate unzip -q $GOROOT/lib/time/zoneinfo.zip -d zoneinfo/
//go:generate go get 4d63.com/embedfiles
//go:generate embedfiles -out=zoneinfo.go -pkg=tz zoneinfo/

func tzData(name string) ([]byte, bool) {
	data, ok := files["zoneinfo/"+name]
	return data, ok
}

func LoadLocation(name string) (*time.Location, error) {
	if name == "" || name == "UTC" || name == "Local" {
		return time.LoadLocation(name)
	}
	if tzdata, ok := tzData(name); ok {
		return time.LoadLocationFromTZData(name, tzdata)
	}
	return nil, errors.New("unknown location " + name)
}
