package models

import (
	"encoding/json"

	"github.com/IBM-Cloud/bluemix-go/crn"
)

type Service struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CatalogCRN string `json:"catalog_crn"`
	URL        string `json:"url"`
	Kind       string `json:"kind"`

	Metadata json.RawMessage `json:"metadata"`
	Children []Service       `json:"children"`
	Active   bool            `json:"active"`
}

func (c Service) GetMetadata() ServiceMetadata {
	if len(c.Metadata) == 0 {
		return nil
	}

	var metadata ServiceMetadata
	switch c.Kind {
	case "runtime":
		metadata = &RuntimeResourceMetadata{}
	case "service", "iaas":
		metadata = &ServiceResourceMetadata{}
	case "platform_service":
		metadata = &PlatformServiceResourceMetadata{}
	case "template":
		metadata = &TemplateResourceMetadata{}
	default:
		return nil
	}
	err := json.Unmarshal(c.Metadata, metadata)
	if err != nil {
		return nil
	}
	return metadata
}

type ServicePlan struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CatalogCRN string `json:"catalog_crn"`
	URL        string `json:"url"`
	Kind       string `json:"kind"`
}

type ServiceDeployment struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	CatalogCRN string             `json:"catalog_crn"`
	Metadata   DeploymentMetaData `json:"metadata,omitempty"`
}

type ServiceDeploymentAlias struct {
	Metadata DeploymentMetaData `json:"metadata,omitempty"`
}

type DeploymentMetaData struct {
	RCCompatible  bool                       `json:"rc_compatible"`
	IAMCompatible bool                       `json:"iam_compatible"`
	Deployment    MetadataDeploymentFragment `json:"deployment,omitempty"`
	Service       MetadataServiceFragment    `json:"service,omitempty"`
}

type MetadataDeploymentFragment struct {
	DeploymentID string  `json:"deployment_id,omitempty"`
	TargetCrn    crn.CRN `json:"target_crn"`
	Location     string  `json:"location"`
}

type ServiceMetadata interface{}

type ServiceResourceMetadata struct {
	Service MetadataServiceFragment `json:"service"`
}

type MetadataServiceFragment struct {
	Bindable            bool   `json:"bindable"`
	IAMCompatible       bool   `json:"iam_compatible"`
	RCProvisionable     bool   `json:"rc_provisionable"`
	PlanUpdateable      bool   `json:"plan_updateable"`
	ServiceCheckEnabled bool   `json:"service_check_enabled"`
	ServiceKeySupported bool   `json:"service_key_supported"`
	State               string `json:"state"`
	TestCheckInterval   int    `json:"test_check_interval"`
	UniqueAPIKey        bool   `json:"unique_api_key"`

	// CF properties
	ServiceBrokerGUID string `json:"service_broker_guid"`
}

type PlatformServiceResourceMetadata struct {
}

type TemplateResourceMetadata struct {
}

type RuntimeResourceMetadata struct {
}
