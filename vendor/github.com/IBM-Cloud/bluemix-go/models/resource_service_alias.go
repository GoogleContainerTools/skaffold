package models

import (
	"github.com/IBM-Cloud/bluemix-go/crn"
)

type ServiceAlias struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	ServiceInstanceID string                 `json:"resource_instance_id"`
	ScopeCRN          crn.CRN                `json:"scope_crn"`
	CRN               crn.CRN                `json:"crn"`
	Tags              []string               `json:"tags,omitempty"`
	Parameters        map[string]interface{} `json:"parameters,omitempty"` // TODO: check whether the response contains the field
	State             string                 `json:"state"`
}

func (a ServiceAlias) ScopeSpaceID() string {
	if a.ScopeCRN.ResourceType == crn.ResourceTypeCFSpace {
		return a.ScopeCRN.Resource
	}
	return ""
}
