package models

type Region struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	DisplayName     string `json:"display_name"`
	Domain          string `json:"domain"`
	APIEndpoint     string `json:"cf_api"`
	ConsoleEndpoint string `json:"console_url"`
	MCCPEndpoint    string `json:"mccp_api"`
	Type            string `json:"type"`
	Geolocation     `json:"geo"`
	Customer        `json:"customer"`
	Deployment      `json:"deployment"`
	IsHome          bool `json:"home"`
}

type Geolocation struct {
	Name        string
	DisplayName string `json:"display_name"`
}

type Customer struct {
	Name        string
	DisplayName string `json:"display_name"`
}

type Deployment struct {
	Name        string
	DisplayName string `json:"display_name"`
}
