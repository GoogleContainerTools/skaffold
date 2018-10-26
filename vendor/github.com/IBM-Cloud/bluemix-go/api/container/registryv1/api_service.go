package registryv1

import (
	gohttp "net/http"

	ibmcloud "github.com/IBM-Cloud/bluemix-go"
	"github.com/IBM-Cloud/bluemix-go/authentication"
	"github.com/IBM-Cloud/bluemix-go/client"
	"github.com/IBM-Cloud/bluemix-go/http"
	"github.com/IBM-Cloud/bluemix-go/rest"
	"github.com/IBM-Cloud/bluemix-go/session"
)

//ErrCodeAPICreation ...
const ErrCodeAPICreation = "APICreationError"

const (
	accountIDHeader = "Account"
)

//RegistryServiceAPI is the IBM Cloud Registry client ...
type RegistryServiceAPI interface {
	Builds() Builds
	Namespaces() Namespaces
	Tokens() Tokens
	Images() Images
	/*Auth() Auth
	Messages() Messages
	Plans() Plans
	Quotas() Quotas
	*/
}

//RegistryService holds the client
type rsService struct {
	*client.Client
}

func addToRequestHeader(h interface{}, r *rest.Request) {
	switch v := h.(type) {
	case map[string]string:
		for key, value := range v {
			r.Set(key, value)
		}
	}
}

//New ...
func New(sess *session.Session) (RegistryServiceAPI, error) {
	config := sess.Config.Copy()
	err := config.ValidateConfigForService(ibmcloud.ContainerRegistryService)
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
		ep, err := config.EndpointLocator.ContainerRegistryEndpoint()
		if err != nil {
			return nil, err
		}
		config.Endpoint = &ep
	}

	return &rsService{
		Client: client.New(config, ibmcloud.ContainerRegistryService, tokenRefreher),
	}, nil
}

//Builds implements builds API
func (c *rsService) Builds() Builds {
	return newBuildAPI(c.Client)
}

//Namespaces implements Namespaces API
func (c *rsService) Namespaces() Namespaces {
	return newNamespaceAPI(c.Client)
}

//Tokens implements Tokens API
func (c *rsService) Tokens() Tokens {
	return newTokenAPI(c.Client)
}

//Images implements Images API
func (c *rsService) Images() Images {
	return newImageAPI(c.Client)
}

/*
//Auth implement auth API
func (c *csService) Auth() Auth {
	return newAuthAPI(c.Client)
}


//Messages implements Messages API
func (c *csService) Messages() Messages {
	return newMessageAPI(c.Client)
}



//Plans implements Plans API
func (c *csService) Plans() Plans {
	return newPlanAPI(c.Client)
}

//Quotas implements Quotas API
func (c *csService) Quotas() Quotas {
	return newQuotaAPI(c.Client)
}


*/
