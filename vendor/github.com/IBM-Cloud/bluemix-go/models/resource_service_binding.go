package models

import "github.com/IBM-Cloud/bluemix-go/crn"

type ServiceBinding struct {
	*MetadataType
	SourceCrn         crn.CRN                `json:"source_crn"`
	TargetCrn         crn.CRN                `json:"target_crn"`
	Parameters        map[string]interface{} `json:"parameters,omitempty"`
	Crn               crn.CRN                `json:"crn"`
	RegionBindingID   string                 `json:"region_binding_id"`
	AccountID         string                 `json:"account_id"`
	State             string                 `json:"state"`
	Credentials       map[string]interface{} `json:"credentials"`
	ServiceAliasesUrl string                 `json:"resource_aliases_url"`
	TargetName        string
}
