package iamv1

import (
	"fmt"
	"net/url"

	"github.com/IBM-Cloud/bluemix-go/client"
	"github.com/IBM-Cloud/bluemix-go/helpers"
	"github.com/IBM-Cloud/bluemix-go/models"
	"github.com/IBM-Cloud/bluemix-go/rest"
)

const (
	_ServicePoliciesEndpointTemplate = "/acms/v1/scopes/%s/service_ids/%s/policies"
	_ServicePolicyEndpointTemplate   = "/acms/v1/scopes/%s/service_ids/%s/policies/%s"
)

// identifier to specify the exact service policy
// following hierarchy "scope/service ID/policy ID"
type ServicePolicyIdentifier struct {
	Scope    string
	IAMID    string
	PolicyID string
}

//go:generate counterfeiter . ServicePolicyRepository
type ServicePolicyRepository interface {
	List(scope string, serviceID string) ([]models.Policy, error)
	Get(scope string, serviceID string, policyID string) (models.Policy, error)
	Create(scope string, serviceID string, policy models.Policy) (models.Policy, error)
	Update(identifier ServicePolicyIdentifier, policy models.Policy, version string) (models.Policy, error)
	Delete(identifier ServicePolicyIdentifier) error
}

type servicePolicyRepository struct {
	client *client.Client
}

func NewServicePolicyRepository(c *client.Client) ServicePolicyRepository {
	return &servicePolicyRepository{
		client: c,
	}
}

type ServicePolicyQueryResult struct {
	Policies []models.Policy `json:"policies"`
}

func (r *servicePolicyRepository) List(scope string, serviceID string) ([]models.Policy, error) {
	response := ServicePolicyQueryResult{}
	_, err := r.client.Get(r.generateURLPath(_ServicePoliciesEndpointTemplate, scope, serviceID), &response)
	if err != nil {
		return []models.Policy{}, err
	}
	return response.Policies, nil
}

func (r *servicePolicyRepository) Get(scope string, serviceID string, policyID string) (models.Policy, error) {
	response := models.Policy{}
	resp, err := r.client.Get(r.generateURLPath(_ServicePolicyEndpointTemplate, scope, serviceID, policyID), &response)
	if err != nil {
		return models.Policy{}, err
	}
	response.Version = resp.Header.Get("Etag")
	return response, nil
}

func (r *servicePolicyRepository) Create(scope string, serviceID string, policy models.Policy) (models.Policy, error) {
	policyCreated := models.Policy{}
	resp, err := r.client.Post(r.generateURLPath(_ServicePoliciesEndpointTemplate, scope, serviceID), &policy, &policyCreated)
	if err != nil {
		return models.Policy{}, err
	}
	policyCreated.Version = resp.Header.Get("Etag")
	return policyCreated, nil
}

func (r *servicePolicyRepository) Update(identifier ServicePolicyIdentifier, policy models.Policy, version string) (models.Policy, error) {
	policyUpdated := models.Policy{}
	request := rest.PutRequest(helpers.GetFullURL(*r.client.Config.Endpoint,
		r.generateURLPath(_ServicePolicyEndpointTemplate, identifier.Scope, identifier.IAMID, identifier.PolicyID))).Body(&policy).Set("If-Match", version)
	resp, err := r.client.SendRequest(request, &policyUpdated)
	if err != nil {
		return models.Policy{}, err
	}
	policyUpdated.Version = resp.Header.Get("Etag")
	return policyUpdated, nil
}

func (r *servicePolicyRepository) Delete(identifier ServicePolicyIdentifier) error {
	_, err := r.client.Delete(r.generateURLPath(_ServicePolicyEndpointTemplate, identifier.Scope, identifier.IAMID, identifier.PolicyID))
	return err
}

func (r *servicePolicyRepository) generateURLPath(template string, parameters ...string) string {
	// TODO: need a URL generator to auto escape parameters
	escaped := []interface{}{}
	for _, parameter := range parameters {
		escaped = append(escaped, url.PathEscape(parameter))
	}
	return fmt.Sprintf(template, escaped...)
}
