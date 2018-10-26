package iamv1

import (
	"net/http"
	"net/url"

	"github.com/IBM-Cloud/bluemix-go/client"
	"github.com/IBM-Cloud/bluemix-go/models"
	"github.com/IBM-Cloud/bluemix-go/rest"
)

type APIKeyResource struct {
	Metadata APIKeyMetadata `json:"metadata"`
	Entity   APIKeyEntity   `json:"entity"`
}

type APIKeyMetadata struct {
	UUID       string `json:"uuid"`
	Version    string `json:"version"`
	Crn        string `json:"crn"`
	CreatedAt  string `json:"createdAt"`
	ModifiedAt string `json:"modifiedAt"`
}

type APIKeyEntity struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	BoundTo      string `json:"boundTo"`
	Format       string `json:"format"`
	APIKey       string `json:"apiKey"`
	APIKeyID     string `json:"apiKeyId"`
	APIKeySecret string `json:"apiKeySecret"`
}

func (r APIKeyResource) ToModel() models.APIKey {
	meta := r.Metadata
	entity := r.Entity

	return models.APIKey{
		UUID:       meta.UUID,
		Version:    meta.Version,
		Crn:        meta.Crn,
		CreatedAt:  meta.CreatedAt,
		ModifiedAt: meta.ModifiedAt,

		Name:         entity.Name,
		Description:  entity.Description,
		BoundTo:      entity.BoundTo,
		Format:       entity.Format,
		APIKey:       entity.APIKey,
		APIKeyID:     entity.APIKeyID,
		APIKeySecret: entity.APIKeySecret,
	}
}

const (
	_API_Key_Operation_Path_Root = "/apikeys/"
)

type APIKeyRepository interface {
	Get(uuid string) (*models.APIKey, error)
	List(boundTo string) ([]models.APIKey, error)
	FindByName(name string, boundTo string) ([]models.APIKey, error)
	Create(key models.APIKey) (*models.APIKey, error)
	Delete(uuid string) error
	Update(uuid string, version string, key models.APIKey) (*models.APIKey, error)
}

type apiKeyRepository struct {
	client *client.Client
}

func NewAPIKeyRepository(c *client.Client) APIKeyRepository {
	return &apiKeyRepository{
		client: c,
	}
}

func (r *apiKeyRepository) Get(uuid string) (*models.APIKey, error) {
	key := APIKeyResource{}
	_, err := r.client.Get(_API_Key_Operation_Path_Root+uuid, &key)
	if err != nil {
		return nil, err
	}
	result := key.ToModel()
	return &result, nil
}

func (r *apiKeyRepository) List(boundTo string) ([]models.APIKey, error) {
	var keys []models.APIKey
	resp, err := r.client.GetPaginated("/apikeys?boundTo="+url.QueryEscape(boundTo), NewIAMPaginatedResources(APIKeyResource{}), func(resource interface{}) bool {
		if apiKeyResource, ok := resource.(APIKeyResource); ok {
			keys = append(keys, apiKeyResource.ToModel())
			return true
		}
		return false
	})

	if resp.StatusCode == http.StatusNotFound {
		return []models.APIKey{}, nil
	}

	return keys, err
}

func (r *apiKeyRepository) FindByName(name string, boundTo string) ([]models.APIKey, error) {
	var keys []models.APIKey
	resp, err := r.client.GetPaginated("/apikeys?boundTo="+url.QueryEscape(boundTo), NewIAMPaginatedResources(APIKeyResource{}), func(resource interface{}) bool {
		if apiKeyResource, ok := resource.(APIKeyResource); ok {
			if apiKeyResource.Entity.Name == name {
				keys = append(keys, apiKeyResource.ToModel())
			}
			return true
		}
		return false
	})

	if resp.StatusCode == http.StatusNotFound {
		return []models.APIKey{}, nil
	}

	return keys, err
}

func (r *apiKeyRepository) Create(key models.APIKey) (*models.APIKey, error) {
	var keyCreated APIKeyResource
	_, err := r.client.Post("/apikeys", &key, &keyCreated)
	if err != nil {
		return nil, err
	}
	keyToReturn := keyCreated.ToModel()
	return &keyToReturn, err
}

func (r *apiKeyRepository) Delete(uuid string) error {
	_, err := r.client.Delete("/apikeys/" + uuid)
	return err
}

func (r *apiKeyRepository) Update(uuid string, version string, key models.APIKey) (*models.APIKey, error) {
	req := rest.PutRequest(*r.client.Config.Endpoint + "/apikeys/" + uuid).Body(&key)
	req.Set("If-Match", version)

	var keyUpdated APIKeyResource
	_, err := r.client.SendRequest(req, &keyUpdated)
	if err != nil {
		return nil, err
	}
	keyToReturn := keyUpdated.ToModel()
	return &keyToReturn, nil
}
