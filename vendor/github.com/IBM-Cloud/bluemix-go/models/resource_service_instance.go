package models

import (
	"time"

	"github.com/IBM-Cloud/bluemix-go/crn"
)

type MetadataType struct {
	ID string `json:"id"`
	//Guid      string     `json:"guid"`
	Url       string     `json:"url"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}
type ServiceInstance struct {
	*MetadataType
	Name                string `json:"name"`
	RegionID            string `json:"region_id"`
	AccountID           string `json:"account_id"`
	ServicePlanID       string `json:"resource_plan_id"`
	ServicePlanName     string
	ResourceGroupID     string `json:"resource_group_id"`
	ResourceGroupName   string
	Crn                 crn.CRN                `json:"crn,omitempty"`
	Tags                []string               `json:"tags,omitempty"`
	Parameters          map[string]interface{} `json:"parameters,omitempty"`
	CreateTime          int64                  `json:"create_time"`
	State               string                 `json:"state"`
	Type                string                 `json:"type"`
	ServiceID           string                 `json:"resource_id"`
	ServiceName         string
	DashboardUrl        *string            `json:"dashboard_url"`
	LastOperation       *LastOperationType `json:"last_operation"`
	AccountUrl          string             `json:"account_url"`
	ResourcePlanUrl     string             `json:"resource_plan_url"`
	ResourceBindingsUrl string             `json:"resource_bindings_url"`
	ResourceAliasesUrl  string             `json:"resource_aliases_url"`
	SiblingsUrl         string             `json:"siblings_url"`
	TargetCrn           crn.CRN            `json:"target_crn"`
}

type LastOperationType struct {
	Type        string     `json:"type"`
	State       string     `json:"state"`
	Description *string    `json:"description"`
	UpdatedAt   *time.Time `json:"updated_at"`
}
