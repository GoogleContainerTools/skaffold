package models

import "github.com/IBM-Cloud/bluemix-go/crn"

type ServiceKey struct {
	MetadataType
	Name        string                 `json:"name"`
	SourceCrn   crn.CRN                `json:"source_crn"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Crn         crn.CRN                `json:"crn"`
	State       string                 `json:"state"`
	AccountID   string                 `json:"account_id"`
	Credentials map[string]interface{} `json:"credentials"`
}
