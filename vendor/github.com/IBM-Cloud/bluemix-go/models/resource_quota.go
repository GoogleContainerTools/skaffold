package models

type QuotaDefinition struct {
	ID                        string         `json:"_id,omitempty"`
	Revision                  string         `json:"_rev,omitempty"`
	Name                      string         `json:"name,omitmempty"`
	Type                      string         `json:"type,omitempty"`
	ServiceInstanceCountLimit int            `json:"number_of_service_instances,omitempty"`
	AppCountLimit             int            `json:"number_of_apps,omitempty"`
	AppInstanceCountLimit     int            `json:"instances_per_app,omitempty"`
	AppInstanceMemoryLimit    string         `json:"instance_memory,omitempty"`
	TotalAppMemoryLimit       string         `json:"total_app_memory,omitempty"`
	VSICountLimit             int            `json:"vsi_limit,omitempty"`
	ServiceQuotas             []ServiceQuota `json:"service_quotas,omitempty"`
	CreatedAt                 string         `json:"created_at,omitempty"`
	UpdatedAt                 string         `json:"updated_at,omitempty"`
}

type ServiceQuota struct {
	ID        string `json:"_id,omitempty"`
	ServiceID string `json:"service_id,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}
