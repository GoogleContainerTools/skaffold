package authentication

import (
	"errors"

	bluemix "github.com/IBM-Cloud/bluemix-go"
	"github.com/IBM-Cloud/bluemix-go/client"
)

const (
	//ErrCodeInvalidToken  ...
	ErrCodeInvalidToken = "InvalidToken"
)

//PopulateTokens populate the relevant tokens in the bluemix Config using the token provider
func PopulateTokens(tokenProvider client.TokenProvider, c *bluemix.Config) error {
	if c.IBMID != "" && c.IBMIDPassword != "" {
		err := tokenProvider.AuthenticatePassword(c.IBMID, c.IBMIDPassword)
		return err
	}
	if c.BluemixAPIKey != "" {
		err := tokenProvider.AuthenticateAPIKey(c.BluemixAPIKey)
		return err
	}
	return errors.New("Insufficient credentials, need IBMID/IBMIDPassword or Bluemix API Key")
}
