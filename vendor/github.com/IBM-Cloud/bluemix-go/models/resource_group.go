package models

type ResourceOrigin string

func (o ResourceOrigin) String() string {
	return string(o)
}

type ResourceGroup struct {
	ID              string    `json:"id,omitempty"`
	AccountID       string    `json:"account_id,omitempty"`
	Name            string    `json:"name,omitempty"`
	Default         bool      `json:"default,omitempty"`
	State           string    `json:"state,omitempty"`
	QuotaID         string    `json:"quota_id,omitempty"`
	PaymentMethodID string    `json:"payment_method_id,omitempty"`
	Linkages        []Linkage `json:"resource_linkages,omitempty"`
	CreatedAt       string    `json:"created_at,omitempty"`
	UpdatedAt       string    `json:"updated_at,omitempty"`
}

type Linkage struct {
	ResourceID     string         `json:"resource_id"`
	ResourceOrigin ResourceOrigin `json:"resource_origin"`
}
