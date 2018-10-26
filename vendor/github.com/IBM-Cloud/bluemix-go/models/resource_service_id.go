package models

type ServiceID struct {
	UUID        string `json:"uuid,omitempty"`
	IAMID       string `json:"iam_id,omitempty"`
	CRN         string `json:"crn,omitempty"`
	Version     string `json:"version,omitempty"`
	BoundTo     string `json:"boundTo,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	ModifiedAt  string `json:"modifiedAt,omitempty"`
	Locked      bool   `json:"locked,omitempty"`
}
