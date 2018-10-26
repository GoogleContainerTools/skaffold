package iamv1

import (
	"github.com/IBM-Cloud/bluemix-go/client"
)

type AccountInfo struct {
	Bss string `json:"bss"`
	Ims string `json:"ims"`
}

type UserInfo struct {
	Active     bool        `json:"active"`
	RealmID    string      `json:"realmId"`
	Identifier string      `json:"identifier"`
	IamID      string      `json:"iam_id"`
	GivenName  string      `json:"given_name"`
	FamilyName string      `json:"family_name"`
	Name       string      `json:"name"`
	Email      string      `json:"email"`
	Sub        string      `json:"sub"`
	Account    AccountInfo `json:"account"`
	Iat        int         `json:"iat"`
	Exp        int         `json:"exp"`
	Iss        string      `json:"iss"`
	GrantType  string      `json:"grant_type"`
	ClientID   string      `json:"client_id"`
	Scope      string      `json:"scope"`
	Acr        int         `json:"acr"`
	Amr        []string    `json:"amr"`
}

type Identity interface {
	UserInfo() (*UserInfo, error)
}

type identity struct {
	client *client.Client
}

func NewIdentity(c *client.Client) Identity {
	return &identity{
		client: c,
	}
}

func (r *identity) UserInfo() (*UserInfo, error) {
	userInfo := UserInfo{}
	_, err := r.client.Get("/identity/userinfo", &userInfo)
	if err != nil {
		return nil, err
	}
	return &userInfo, nil
}
