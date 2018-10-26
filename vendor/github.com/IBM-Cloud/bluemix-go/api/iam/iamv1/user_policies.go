package iamv1

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/IBM-Cloud/bluemix-go/client"
	"github.com/IBM-Cloud/bluemix-go/models"
	"github.com/IBM-Cloud/bluemix-go/rest"
)

const (
	_UserPoliciesPathTemplate = "/acms/v1/scopes/%s/users/%s/policies"
	_UserPolicyPathTemplate   = "/acms/v1/scopes/%s/users/%s/policies/%s"
)

//go:generate counterfeiter . UserPolicyRepository
type UserPolicyRepository interface {
	List(scope string, ibmUniqueID string) ([]models.Policy, error)
	Get(scope string, ibmUniqueID string, policyID string) (models.Policy, error)
	Create(scope string, ibmUniqueID string, policy models.Policy) (models.Policy, error)
	Update(scope string, ibmUniqueID string, policyID string, policy models.Policy, version string) (models.Policy, error)
	Delete(scope string, ibmUniqueID string, policyID string) error
}

type userPolicyRepository struct {
	client *client.Client
}

func NewUserPolicyRepository(c *client.Client) UserPolicyRepository {
	return &userPolicyRepository{
		client: c,
	}
}

type PoliciesQueryResult struct {
	Policies []models.Policy `json:"policies"`
}

func (r *userPolicyRepository) List(scope string, ibmUniqueID string) ([]models.Policy, error) {
	result := PoliciesQueryResult{}
	resp, err := r.client.Get(r.generateURLPath(_UserPoliciesPathTemplate, scope, ibmUniqueID), &result)

	if resp.StatusCode == http.StatusNotFound {
		return []models.Policy{}, nil
	}

	if err != nil {
		return nil, err
	}

	return result.Policies, nil
}

func (r *userPolicyRepository) Get(scope string, ibmUniqueID string, policyID string) (models.Policy, error) {
	policy := models.Policy{}
	resp, err := r.client.Get(r.generateURLPath(_UserPolicyPathTemplate, scope, ibmUniqueID, policyID), &policy)
	if err != nil {
		return models.Policy{}, err
	}
	policy.Version = resp.Header.Get("Etag")
	return policy, nil
}

func (r *userPolicyRepository) Create(scope string, ibmUniqueID string, policy models.Policy) (models.Policy, error) {
	policyCreated := models.Policy{}
	resp, err := r.client.Post(r.generateURLPath(_UserPoliciesPathTemplate, scope, ibmUniqueID), &policy, &policyCreated)
	if err != nil {
		return models.Policy{}, err
	}
	policyCreated.Version = resp.Header.Get("Etag")
	return policyCreated, nil
}

func (r *userPolicyRepository) Update(scope string, ibmUniqueID string, policyID string, policy models.Policy, version string) (models.Policy, error) {
	policyUpdated := models.Policy{}

	request := rest.PutRequest(*r.client.Config.Endpoint + r.generateURLPath(_UserPolicyPathTemplate, scope, ibmUniqueID, policyID))
	request = request.Set("If-Match", version).Body(&policy)

	resp, err := r.client.SendRequest(request, &policyUpdated)
	if err != nil {
		return models.Policy{}, err
	}
	policyUpdated.Version = resp.Header.Get("Etag")
	return policyUpdated, nil
}

func (r *userPolicyRepository) Delete(scope string, ibmUniqueID string, policyID string) error {
	_, err := r.client.Delete(r.generateURLPath(_UserPolicyPathTemplate, scope, ibmUniqueID, policyID))
	return err
}

func (r *userPolicyRepository) generateURLPath(template string, parameters ...string) string {
	escaped := []interface{}{}
	for _, parameter := range parameters {
		escaped = append(escaped, url.QueryEscape(parameter))
	}
	return fmt.Sprintf(template, escaped...)
}
