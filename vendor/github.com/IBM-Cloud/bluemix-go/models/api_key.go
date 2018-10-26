package models

type APIKey struct {
	UUID       string `json:"uuid,omitempty"`
	Version    string `json:"version,omitempty"`
	Crn        string `json:"crn,omitempty"`
	CreatedAt  string `json:"createdAt,omitempty"`
	ModifiedAt string `json:"modifiedAt,omitempty"`

	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	Format       string `json:"format,omitempty"`
	BoundTo      string `json:"boundTo,omitempty"`
	APIKey       string `json:"apiKey,omitempty"`
	APIKeyID     string `json:"apiKeyId,omitempty"`
	APIKeySecret string `json:"apiKeySecret,omitempty"`
	Locked       bool   `json:"locked,omitempty"`
}
