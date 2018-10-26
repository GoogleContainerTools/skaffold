package authentication

import (
	"encoding/base64"
	"fmt"

	bluemix "github.com/IBM-Cloud/bluemix-go"
	"github.com/IBM-Cloud/bluemix-go/bmxerror"
	"github.com/IBM-Cloud/bluemix-go/rest"
)

//UAAError ...
type UAAError struct {
	ErrorCode   string `json:"error"`
	Description string `json:"error_description"`
}

//UAATokenResponse ...
type UAATokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
}

//UAARepository ...
type UAARepository struct {
	config   *bluemix.Config
	client   *rest.Client
	endpoint string
}

//NewUAARepository ...
func NewUAARepository(config *bluemix.Config, client *rest.Client) (*UAARepository, error) {
	var endpoint string

	if config.TokenProviderEndpoint != nil {
		endpoint = *config.TokenProviderEndpoint
	} else {
		var err error
		endpoint, err = config.EndpointLocator.UAAEndpoint()
		if err != nil {
			return nil, err
		}
	}
	return &UAARepository{
		config:   config,
		client:   client,
		endpoint: endpoint,
	}, nil
}

//AuthenticatePassword ...
func (auth *UAARepository) AuthenticatePassword(username string, password string) error {
	return auth.getToken(map[string]string{
		"grant_type": "password",
		"username":   username,
		"password":   password,
	})
}

//AuthenticateSSO ...
func (auth *UAARepository) AuthenticateSSO(passcode string) error {
	return auth.getToken(map[string]string{
		"grant_type": "password",
		"passcode":   passcode,
	})
}

//AuthenticateAPIKey ...
func (auth *UAARepository) AuthenticateAPIKey(apiKey string) error {
	return auth.AuthenticatePassword("apikey", apiKey)
}

//RefreshToken ...
func (auth *UAARepository) RefreshToken() (string, error) {
	err := auth.getToken(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": auth.config.UAARefreshToken,
	})
	if err != nil {
		return "", err
	}

	return auth.config.UAAAccessToken, nil
}

func (auth *UAARepository) getToken(data map[string]string) error {
	request := rest.PostRequest(auth.endpoint+"/oauth/token").
		Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("cf:"))).
		Field("scope", "")

	for k, v := range data {
		request.Field(k, v)
	}

	var tokens UAATokenResponse
	var apiErr UAAError

	resp, err := auth.client.Do(request, &tokens, &apiErr)
	if err != nil {
		return err
	}
	if apiErr.ErrorCode != "" {
		if apiErr.ErrorCode == "invalid-token" {
			return bmxerror.NewInvalidTokenError(apiErr.Description)
		}
		return bmxerror.NewRequestFailure(apiErr.ErrorCode, apiErr.Description, resp.StatusCode)
	}

	auth.config.UAAAccessToken = fmt.Sprintf("%s %s", tokens.TokenType, tokens.AccessToken)
	auth.config.UAARefreshToken = tokens.RefreshToken
	return nil
}
