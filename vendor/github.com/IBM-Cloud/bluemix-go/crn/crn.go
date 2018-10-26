package crn

import (
	"encoding/json"
	"errors"
	"strings"
)

// CRN spec: https://github.ibm.com/ibmcloud/builders-guide/tree/master/specifications/crn

const (
	crn     = "crn"
	version = "v1"

	crnSeparator   = ":"
	scopeSeparator = "/"
)

var (
	ErrMalformedCRN   = errors.New("malformed CRN")
	ErrMalformedScope = errors.New("malformed scope in CRN")
)

const (
	ServiceBluemix = "bluemix"
	ServiceIAM     = "iam"
	// more services ...

	ScopeAccount      = "a"
	ScopeOrganization = "o"
	ScopeSpace        = "s"
	ScopeProject      = "p"

	ResourceTypeCFSpace   = "cf-space"
	ResourceTypeCFApp     = "cf-application"
	ResourceTypeCFService = "cf-service-instance"
	ResourceTypeRole      = "role"
	// more resources ...
)

type CRN struct {
	Scheme          string
	Version         string
	CName           string
	CType           string
	ServiceName     string
	Region          string
	ScopeType       string
	Scope           string
	ServiceInstance string
	ResourceType    string
	Resource        string
}

func New(cloudName string, cloudType string) CRN {
	return CRN{
		Scheme:  crn,
		Version: version,
		CName:   cloudName,
		CType:   cloudType,
	}
}

func (c *CRN) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	*c, err = Parse(s)
	return err
}

func (c CRN) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

func Parse(s string) (CRN, error) {
	if s == "" {
		return CRN{}, nil
	}

	segments := strings.Split(s, crnSeparator)
	if len(segments) != 10 || segments[0] != crn {
		return CRN{}, ErrMalformedCRN
	}

	crn := CRN{
		Scheme:          segments[0],
		Version:         segments[1],
		CName:           segments[2],
		CType:           segments[3],
		ServiceName:     segments[4],
		Region:          segments[5],
		ServiceInstance: segments[7],
		ResourceType:    segments[8],
		Resource:        segments[9],
	}

	scopeSegments := segments[6]
	if scopeSegments != "" {
		scopeParts := strings.Split(scopeSegments, scopeSeparator)
		if len(scopeParts) != 2 {
			return CRN{}, ErrMalformedScope
		}
		crn.ScopeType, crn.Scope = scopeParts[0], scopeParts[1]
	}

	return crn, nil
}

func (c CRN) String() string {
	return strings.Join([]string{
		c.Scheme,
		c.Version,
		c.CName,
		c.CType,
		c.ServiceName,
		c.Region,
		c.ScopeSegment(),
		c.ServiceInstance,
		c.ResourceType,
		c.Resource,
	}, crnSeparator)
}

func (c CRN) ScopeSegment() string {
	if c.ScopeType == "" && c.Scope == "" {
		return ""
	}
	return c.ScopeType + scopeSeparator + c.Scope
}
