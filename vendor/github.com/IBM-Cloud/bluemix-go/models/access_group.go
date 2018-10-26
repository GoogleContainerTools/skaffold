package models

// AccessGroup represents the access group of IAM UUM
type AccessGroup struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}
