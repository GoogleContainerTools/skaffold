package registryv1

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/IBM-Cloud/bluemix-go/client"
	"github.com/IBM-Cloud/bluemix-go/helpers"
	"github.com/IBM-Cloud/bluemix-go/rest"
)

type TokenTargetHeader struct {
	AccountID string
}

//ToMap ...
func (c TokenTargetHeader) ToMap() map[string]string {
	m := make(map[string]string, 1)
	m[accountIDHeader] = c.AccountID
	return m
}

//Subnets interface
type Tokens interface {
	GetToken(tokenUUID string, target TokenTargetHeader) (*TokenResponse, error)
	GetTokens(target TokenTargetHeader) (*GetTokensResponse, error)
	DeleteToken(tokenUUID string, target TokenTargetHeader) error
	DeleteTokenByDescription(tokenDescription string, target TokenTargetHeader) error
	IssueToken(params IssueTokenRequest, target TokenTargetHeader) (*TokenResponse, error)
}

type tokens struct {
	client *client.Client
}

func newTokenAPI(c *client.Client) Tokens {
	return &tokens{
		client: c,
	}
}

type GetTokensResponse struct {
	Tokens []struct {
		ID             string `json:"_id,omitempty"`
		Owner          string `json:"owner,omitempty"`
		Token          string `json:"token,omitempty"`
		Description    string `json:"secondary_owner,omitempty"`
		Readonly       bool   `json:"readonly,omitempty"`
		Revoked        bool   `json:"revoked,omitempty"`
		Expiry         int64  `json:"expiry,omitempty"`
		EncryptedToken string `json:"encrypted_token,omitempty"`
	} `json:"tokens,omitempty"`
}
type TokenResponse struct {
	ID    string
	Token string `json:"token,omitempty"`
}

/*TokenIssueParams contains all the parameters to send to the API endpoint
for the token issue operation typically these are written to a http.Request
*/
type IssueTokenRequest struct {
	/*Description
	  Specifies a description for the token so it can be more easily identified. If this option is specified more than once, the last parsed setting is the setting that is used.
	*/
	Description string
	/*Permanent
	  When specified, the access token does not expire. If this option is specified more than once, the last parsed setting is the setting that is used.
	*/
	Permanent bool
	/*Write
	  When specified, the token provides write access to registry namespaces in your IBM Cloud account. If this option is not specified, or is set to false, the token provides read-only access. If this option is specified more than once, the last parsed setting is the setting that is used.
	*/
	Write bool
}

func DefaultIssueTokenRequest() *IssueTokenRequest {
	return &IssueTokenRequest{
		Description: "",
		Permanent:   false,
		Write:       false,
	}
}

//GetTokens ...
func (r *tokens) GetTokens(target TokenTargetHeader) (*GetTokensResponse, error) {

	var retVal GetTokensResponse
	req := rest.GetRequest(helpers.GetFullURL(*r.client.Config.Endpoint, "/api/v1/tokens"))

	for key, value := range target.ToMap() {
		req.Set(key, value)
	}

	_, err := r.client.SendRequest(req, &retVal)
	if err != nil {
		return nil, err
	}
	return &retVal, err
}

func getTokID(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("Corrupt Token, not enough parts")
	}
	decodedBytes, decerr := base64.RawStdEncoding.DecodeString(parts[1])
	if decerr != nil {
		return "", fmt.Errorf("Corrupt Token, could not decode, error: %v", decerr)
	}
	jwtID := struct {
		ID string `json:"jti"`
	}{""}
	jerr := json.Unmarshal(decodedBytes, &jwtID)
	if jerr != nil {
		return "", fmt.Errorf("Corrupt Token, could not decode, error: %v", jerr)
	}
	return jwtID.ID, nil
}

//GetToken ...
func (r *tokens) GetToken(tokenUUID string, target TokenTargetHeader) (*TokenResponse, error) {

	var retVal TokenResponse
	req := rest.GetRequest(helpers.GetFullURL(*r.client.Config.Endpoint, fmt.Sprintf("/api/v1/tokens/%s", tokenUUID)))

	for key, value := range target.ToMap() {
		req.Set(key, value)
	}

	_, err := r.client.SendRequest(req, &retVal)
	if err == nil {
		retVal.ID = tokenUUID
	} else {
		return nil, err
	}
	return &retVal, err
}

//Add ...
func (r *tokens) IssueToken(params IssueTokenRequest, target TokenTargetHeader) (*TokenResponse, error) {

	var retVal TokenResponse
	req := rest.PostRequest(helpers.GetFullURL(*r.client.Config.Endpoint, "/api/v1/tokens")).
		Query("description", params.Description).
		Query("permanent", strconv.FormatBool(params.Permanent)).
		Query("write", strconv.FormatBool(params.Permanent))

	for key, value := range target.ToMap() {
		req.Set(key, value)
	}

	_, err := r.client.SendRequest(req, &retVal)
	if err == nil {
		retVal.ID, err = getTokID(retVal.Token)
	} else {
		return nil, err
	}
	return &retVal, err
}

//Delete...
func (r *tokens) DeleteToken(tokenUUID string, target TokenTargetHeader) error {
	req := rest.DeleteRequest(helpers.GetFullURL(*r.client.Config.Endpoint, fmt.Sprintf("/api/v1/tokens/%s", tokenUUID)))

	for key, value := range target.ToMap() {
		req.Set(key, value)
	}

	_, err := r.client.SendRequest(req, nil)
	return err
}

//Delete By Description
func (r *tokens) DeleteTokenByDescription(tokenDescription string, target TokenTargetHeader) error {
	req := rest.DeleteRequest(helpers.GetFullURL(*r.client.Config.Endpoint, "/api/v1/tokens")).
		Query("secondaryOwner", tokenDescription)

	for key, value := range target.ToMap() {
		req.Set(key, value)
	}

	_, err := r.client.SendRequest(req, nil)
	return err
}
