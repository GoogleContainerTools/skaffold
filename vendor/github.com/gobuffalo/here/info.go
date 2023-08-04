package here

import (
	"encoding/json"
)

// Info represents details about the directory/package
type Info struct {
	Dir         string
	ImportPath  string
	Name        string
	Doc         string
	Target      string
	Root        string
	Match       []string
	Stale       bool
	StaleReason string
	GoFiles     []string
	Imports     []string
	Deps        []string
	TestGoFiles []string
	TestImports []string
	Module      Module
}

// IsZero checks if the type has been filled
// with rich chocolately data goodness
func (i Info) IsZero() bool {
	return i.String() == Info{}.String()
}

func (i Info) String() string {
	b, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return err.Error()
	}
	s := string(b)
	return s
}
