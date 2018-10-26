package iamv1

import (
	gohttp "net/http"

	bluemix "github.com/IBM-Cloud/bluemix-go"
	"github.com/IBM-Cloud/bluemix-go/authentication"
	"github.com/IBM-Cloud/bluemix-go/client"
	"github.com/IBM-Cloud/bluemix-go/http"
	"github.com/IBM-Cloud/bluemix-go/rest"
	"github.com/IBM-Cloud/bluemix-go/session"
)

//IAMServiceAPI is the resource client ...
type IAMServiceAPI interface {
	ServiceRoles() ServiceRoleRepository
	ServiceIds() ServiceIDRepository
	APIKeys() APIKeyRepository
	ServicePolicies() ServicePolicyRepository
	UserPolicies() UserPolicyRepository
	Identity() Identity
}

//ErrCodeAPICreation ...
const ErrCodeAPICreation = "APICreationError"

//iamService holds the client
type iamService struct {
	*client.Client
}

//New ...
func New(sess *session.Session) (IAMServiceAPI, error) {
	config := sess.Config.Copy()
	err := config.ValidateConfigForService(bluemix.IAMService)
	if err != nil {
		return nil, err
	}
	if config.HTTPClient == nil {
		config.HTTPClient = http.NewHTTPClient(config)
	}
	tokenRefreher, err := authentication.NewIAMAuthRepository(config, &rest.Client{
		DefaultHeader: gohttp.Header{
			"User-Agent": []string{http.UserAgent()},
		},
		HTTPClient: config.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	if config.IAMAccessToken == "" {
		err := authentication.PopulateTokens(tokenRefreher, config)
		if err != nil {
			return nil, err
		}
	}
	if config.Endpoint == nil {
		ep, err := config.EndpointLocator.IAMEndpoint()
		if err != nil {
			return nil, err
		}
		config.Endpoint = &ep
	}

	return &iamService{
		Client: client.New(config, bluemix.IAMService, tokenRefreher),
	}, nil
}

//ServiceRoles API
func (a *iamService) ServiceRoles() ServiceRoleRepository {
	return NewServiceRoleRepository(a.Client)
}

//ServiceIdsAPI
func (a *iamService) ServiceIds() ServiceIDRepository {
	return NewServiceIDRepository(a.Client)
}

//APIkeys
func (a *iamService) APIKeys() APIKeyRepository {
	return NewAPIKeyRepository(a.Client)
}

//ServicePolicyAPI
func (a *iamService) ServicePolicies() ServicePolicyRepository {
	return NewServicePolicyRepository(a.Client)
}

//UserPoliciesAPI
func (a *iamService) UserPolicies() UserPolicyRepository {
	return NewUserPolicyRepository(a.Client)
}

//IdentityAPI
func (a *iamService) Identity() Identity {
	return NewIdentity(a.Client)
}
