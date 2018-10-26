package endpoints

import (
	"fmt"

	"github.com/IBM-Cloud/bluemix-go/bmxerror"
	"github.com/IBM-Cloud/bluemix-go/helpers"
)

//EndpointLocator ...
type EndpointLocator interface {
	AccountManagementEndpoint() (string, error)
	CFAPIEndpoint() (string, error)
	MCCPAPIEndpoint() (string, error)
	ContainerEndpoint() (string, error)
	IAMEndpoint() (string, error)
	IAMPAPEndpoint() (string, error)
	ResourceManagementEndpoint() (string, error)
	ResourceControllerEndpoint() (string, error)
	ResourceCatalogEndpoint() (string, error)
	UAAEndpoint() (string, error)
	ContainerRegistryEndpoint() (string, error)
}

const (
	//ErrCodeServiceEndpoint ...
	ErrCodeServiceEndpoint = "ServiceEndpointDoesnotExist"
)

var regionToEndpoint = map[string]map[string]string{
	"cf": {
		"us-south": "https://api.ng.bluemix.net",
		"us-east":  "https://api.us-east.bluemix.net",
		"eu-gb":    "https://api.eu-gb.bluemix.net",
		"au-syd":   "https://api.au-syd.bluemix.net",
		"eu-de":    "https://api.eu-de.bluemix.net",
	},
	"mccp": {
		"us-south": "https://mccp.ng.bluemix.net",
		"us-east":  "https://mccp.us-east.bluemix.net",
		"eu-gb":    "https://mccp.eu-gb.bluemix.net",
		"au-syd":   "https://mccp.au-syd.bluemix.net",
		"eu-de":    "https://mccp.eu-de.bluemix.net",
	},
	"iam": {
		"us-south": "https://iam.bluemix.net",
		"us-east":  "https://iam.bluemix.net",
		"eu-gb":    "https://iam.bluemix.net",
		"au-syd":   "https://iam.bluemix.net",
		"eu-de":    "https://iam.bluemix.net",
		"jp-tok":   "https://iam.bluemix.net",
	},
	"iampap": {
		"us-south": "https://iam.bluemix.net",
		"us-east":  "https://iam.bluemix.net",
		"eu-gb":    "https://iam.bluemix.net",
		"au-syd":   "https://iam.bluemix.net",
		"eu-de":    "https://iam.bluemix.net",
		"jp-tok":   "https://iam.bluemix.net",
	},
	"uaa": {
		"us-south": "https://login.ng.bluemix.net/UAALoginServerWAR",
		"us-east":  "https://login.us-east.bluemix.net/UAALoginServerWAR",
		"eu-gb":    "https://login.eu-gb.bluemix.net/UAALoginServerWAR",
		"au-syd":   "https://login.au-syd.bluemix.net/UAALoginServerWAR",
		"eu-de":    "https://login.eu-de.bluemix.net/UAALoginServerWAR",
	},
	"account": {
		"us-south": "https://accountmanagement.ng.bluemix.net",
		"us-east":  "https://accountmanagement.us-east.bluemix.net",
		"eu-gb":    "https://accountmanagement.eu-gb.bluemix.net",
		"au-syd":   "https://accountmanagement.au-syd.bluemix.net",
		"eu-de":    "https://accountmanagement.eu-de.bluemix.net",
	},
	"cs": {
		"us-south": "https://containers.bluemix.net",
		"us-east":  "https://containers.bluemix.net",
		"eu-de":    "https://containers.bluemix.net",
		"au-syd":   "https://containers.bluemix.net",
		"eu-gb":    "https://containers.bluemix.net",
		"jp-tok":   "https://containers.bluemix.net",
	},

	"resource-manager": {
		"us-south": "https://resource-manager.bluemix.net",
		"us-east":  "https://resource-manager.bluemix.net",
		"eu-de":    "https://resource-manager.bluemix.net",
		"au-syd":   "https://resource-manager.bluemix.net",
		"eu-gb":    "https://resource-manager.bluemix.net",
	},
	"resource-catalog": {
		"us-south": "https://resource-catalog.bluemix.net",
		"us-east":  "https://resource-catalog.bluemix.net",
		"eu-de":    "https://resource-catalog.bluemix.net",
		"au-syd":   "https://resource-catalog.bluemix.net",
		"eu-gb":    "https://resource-catalog.bluemix.net",
	},
	"resource-controller": {
		"us-south": "https://resource-controller.bluemix.net",
		"us-east":  "https://resource-controller.bluemix.net",
		"eu-de":    "https://resource-controller.bluemix.net",
		"au-syd":   "https://resource-controller.bluemix.net",
		"eu-gb":    "https://resource-controller.bluemix.net",
	},
	"cr": {
		"us-south": "https://registry.ng.bluemix.net",
		"us-east":  "https://registry.ng.bluemix.net",
		"eu-de":    "https://registry.eu-de.bluemix.net",
		"au-syd":   "https://registry.au-syd.bluemix.net",
		"eu-gb":    "https://registry.eu-gb.bluemix.net",
	},
}

func init() {
	//TODO populate the endpoints which can be retrieved from given endpoints dynamically
	//Example - UAA can be found from the CF endpoint
}

type endpointLocator struct {
	region string
}

//NewEndpointLocator ...
func NewEndpointLocator(region string) EndpointLocator {
	return &endpointLocator{region: region}
}

func (e *endpointLocator) CFAPIEndpoint() (string, error) {
	if ep, ok := regionToEndpoint["cf"][e.region]; ok {
		//As the current list of regionToEndpoint above is not exhaustive we allow to read endpoints from the env
		return helpers.EnvFallBack([]string{"IBMCLOUD_CF_API_ENDPOINT"}, ep), nil

	}
	return "", bmxerror.New(ErrCodeServiceEndpoint, fmt.Sprintf("Cloud Foundry endpoint doesn't exist for region: %q", e.region))
}

func (e *endpointLocator) MCCPAPIEndpoint() (string, error) {
	if ep, ok := regionToEndpoint["mccp"][e.region]; ok {
		//As the current list of regionToEndpoint above is not exhaustive we allow to read endpoints from the env
		return helpers.EnvFallBack([]string{"IBMCLOUD_MCCP_API_ENDPOINT"}, ep), nil

	}
	return "", bmxerror.New(ErrCodeServiceEndpoint, fmt.Sprintf("MCCP API endpoint doesn't exist for region: %q", e.region))
}

func (e *endpointLocator) UAAEndpoint() (string, error) {
	if ep, ok := regionToEndpoint["uaa"][e.region]; ok {
		//As the current list of regionToEndpoint above is not exhaustive we allow to read endpoints from the env
		return helpers.EnvFallBack([]string{"IBMCLOUD_UAA_ENDPOINT"}, ep), nil

	}
	return "", bmxerror.New(ErrCodeServiceEndpoint, fmt.Sprintf("UAA endpoint doesn't exist for region: %q", e.region))
}

func (e *endpointLocator) AccountManagementEndpoint() (string, error) {
	if ep, ok := regionToEndpoint["account"][e.region]; ok {
		//As the current list of regionToEndpoint above is not exhaustive we allow to read endpoints from the env
		return helpers.EnvFallBack([]string{"IBMCLOUD_ACCOUNT_MANAGEMENT_API_ENDPOINT"}, ep), nil

	}
	return "", bmxerror.New(ErrCodeServiceEndpoint, fmt.Sprintf("Account Management endpoint doesn't exist for region: %q", e.region))
}

func (e *endpointLocator) IAMEndpoint() (string, error) {
	if ep, ok := regionToEndpoint["iam"][e.region]; ok {
		//As the current list of regionToEndpoint above is not exhaustive we allow to read endpoints from the env
		return helpers.EnvFallBack([]string{"IBMCLOUD_IAM_API_ENDPOINT"}, ep), nil

	}
	return "", bmxerror.New(ErrCodeServiceEndpoint, fmt.Sprintf("IAM  endpoint doesn't exist for region: %q", e.region))
}

func (e *endpointLocator) IAMPAPEndpoint() (string, error) {
	if ep, ok := regionToEndpoint["iampap"][e.region]; ok {
		//As the current list of regionToEndpoint above is not exhaustive we allow to read endpoints from the env
		return helpers.EnvFallBack([]string{"IBMCLOUD_IAMPAP_API_ENDPOINT"}, ep), nil

	}
	return "", fmt.Errorf("IAMPAP  endpoint doesn't exist for region: %q", e.region)
}

func (e *endpointLocator) ContainerEndpoint() (string, error) {
	if ep, ok := regionToEndpoint["cs"][e.region]; ok {
		//As the current list of regionToEndpoint above is not exhaustive we allow to read endpoints from the env
		return helpers.EnvFallBack([]string{"IBMCLOUD_CS_API_ENDPOINT"}, ep), nil
	}
	return "", bmxerror.New(ErrCodeServiceEndpoint, fmt.Sprintf("Container Service endpoint doesn't exist for region: %q", e.region))
}

func (e *endpointLocator) ResourceManagementEndpoint() (string, error) {
	if ep, ok := regionToEndpoint["resource-manager"][e.region]; ok {
		//As the current list of regionToEndpoint above is not exhaustive we allow to read endpoints from the env
		return helpers.EnvFallBack([]string{"IBMCLOUD_RESOURCE_MANAGEMENT_API_ENDPOINT"}, ep), nil

	}
	return "", bmxerror.New(ErrCodeServiceEndpoint, fmt.Sprintf("Resource Management endpoint doesn't exist"))
}

func (e *endpointLocator) ResourceControllerEndpoint() (string, error) {
	if ep, ok := regionToEndpoint["resource-controller"][e.region]; ok {
		//As the current list of regionToEndpoint above is not exhaustive we allow to read endpoints from the env
		return helpers.EnvFallBack([]string{"IBMCLOUD_RESOURCE_CONTROLLER_API_ENDPOINT"}, ep), nil

	}
	return "", bmxerror.New(ErrCodeServiceEndpoint, fmt.Sprintf("Resource Controller endpoint doesn't exist"))
}

func (e *endpointLocator) ResourceCatalogEndpoint() (string, error) {
	if ep, ok := regionToEndpoint["resource-catalog"][e.region]; ok {
		//As the current list of regionToEndpoint above is not exhaustive we allow to read endpoints from the env
		return helpers.EnvFallBack([]string{"IBMCLOUD_RESOURCE_CATALOG_API_ENDPOINT"}, ep), nil

	}
	return "", bmxerror.New(ErrCodeServiceEndpoint, fmt.Sprintf("Resource Catalog endpoint doesn't exist"))
}

func (e *endpointLocator) ContainerRegistryEndpoint() (string, error) {
	if ep, ok := regionToEndpoint["cr"][e.region]; ok {
		//As the current list of regionToEndpoint above is not exhaustive we allow to read endpoints from the env
		return helpers.EnvFallBack([]string{"IBMCLOUD_CR_API_ENDPOINT"}, ep), nil
	}
	return "", bmxerror.New(ErrCodeServiceEndpoint, fmt.Sprintf("Container Registry Service endpoint doesn't exist for region: %q", e.region))
}