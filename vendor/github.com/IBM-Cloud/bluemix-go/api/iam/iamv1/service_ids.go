package iamv1

import (
	"net/http"
	"net/url"

	"github.com/IBM-Cloud/bluemix-go/client"
	"github.com/IBM-Cloud/bluemix-go/helpers"
	"github.com/IBM-Cloud/bluemix-go/models"
	"github.com/IBM-Cloud/bluemix-go/rest"
)

type ServiceIDResource struct {
	Metadata IAMMetadata     `json:"metadata"`
	Entity   ServiceIDEntity `json:"entity"`
}

type ServiceIDEntity struct {
	BoundTo     string `json:"boundTo"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r *ServiceIDResource) ToModel() models.ServiceID {
	return models.ServiceID{
		UUID:        r.Metadata.UUID,
		IAMID:       r.Metadata.IAMID,
		CRN:         r.Metadata.CRN,
		BoundTo:     r.Entity.BoundTo,
		Name:        r.Entity.Name,
		Description: r.Entity.Description,
		Version:     r.Metadata.Version,
		CreatedAt:   r.Metadata.CreatedAt,
		ModifiedAt:  r.Metadata.ModifiedAt,
	}
}

type IAMMetadata struct {
	UUID       string `json:"uuid"`
	IAMID      string `json:"iam_id"`
	Version    string `json:"version"`
	CRN        string `json:"crn"`
	CreatedAt  string `json:"createdAt"`
	ModifiedAt string `json:"modifiedAt"`
}

const (
	_IAM_App          = "iam"
	_IAM_ENDPOINT_ENV = "IAM_ENDPOINT"
	_SERVICE_ID_PATH  = "/serviceids/"
	_BoundToQuery     = "boundTo"
)

//go:generate counterfeiter . ServiceIDRepository
type ServiceIDRepository interface {
	Get(uuid string) (models.ServiceID, error)
	List(boundTo string) ([]models.ServiceID, error)
	FindByName(boundTo string, name string) ([]models.ServiceID, error)
	Create(serviceId models.ServiceID) (models.ServiceID, error)
	Update(uuid string, serviceId models.ServiceID, version string) (models.ServiceID, error)
	Delete(uuid string) error
}

type serviceIDRepository struct {
	client *client.Client
}

func NewServiceIDRepository(c *client.Client) ServiceIDRepository {
	return &serviceIDRepository{
		client: c,
	}
}

type IAMResponseContext struct {
	RequestID   string `json:"requestId"`
	RequestType string `json:"requestType"`
	UserAgent   string `json:"userAgent"`
	ClientIP    string `json:"clientIp"`
	InstanceID  string `json:"instanceId"`
	ThreadID    string `json:"threadId"`
	Host        string `json:"host"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	ElapsedTime string `json:"elapsedTime"`
	Locale      string `json:"locale"`
}

type ServiceIDSearchResults struct {
	Context    IAMResponseContext  `json:"context"`
	ServiceIDs []ServiceIDResource `json:"items"`
}

func (r *serviceIDRepository) List(boundTo string) ([]models.ServiceID, error) {
	var serviceIDs []models.ServiceID
	_, err := r.client.GetPaginated("/serviceids?boundTo="+url.QueryEscape(boundTo), NewIAMPaginatedResources(ServiceIDResource{}), func(r interface{}) bool {
		if idResource, ok := r.(ServiceIDResource); ok {
			serviceIDs = append(serviceIDs, idResource.ToModel())
			return true
		}
		return false
	})

	if err != nil {
		return []models.ServiceID{}, err
	}

	return serviceIDs, nil
}

func (r *serviceIDRepository) FindByName(boundTo string, name string) ([]models.ServiceID, error) {
	var serviceIDs []models.ServiceID
	resp, err := r.client.GetPaginated("/serviceids?boundTo="+url.QueryEscape(boundTo), NewIAMPaginatedResources(ServiceIDResource{}), func(r interface{}) bool {
		if idResource, ok := r.(ServiceIDResource); ok {
			if idResource.Entity.Name == name {
				serviceIDs = append(serviceIDs, idResource.ToModel())
			}
			return true
		}
		return false
	})

	if resp.StatusCode == http.StatusNotFound {
		return []models.ServiceID{}, nil
	}

	return serviceIDs, err
}

type ServiceIDResponse struct {
	IAMResponseContext
	ServiceIDResource
}

func (r *serviceIDRepository) Create(serviceId models.ServiceID) (models.ServiceID, error) {
	createdId := ServiceIDResponse{}
	_, err := r.client.Post(_SERVICE_ID_PATH, &serviceId, &createdId)
	if err != nil {
		return models.ServiceID{}, err
	}
	return createdId.ToModel(), err
}

func (r *serviceIDRepository) Update(uuid string, serviceId models.ServiceID, version string) (models.ServiceID, error) {
	updatedId := ServiceIDResponse{}
	request := rest.PutRequest(helpers.GetFullURL(*r.client.Config.Endpoint, _SERVICE_ID_PATH+uuid)).Add("If-Match", version).Body(&serviceId)
	_, err := r.client.SendRequest(request, &updatedId)
	if err != nil {
		return models.ServiceID{}, err
	}
	return updatedId.ToModel(), err
}

func (r *serviceIDRepository) Delete(uuid string) error {
	_, err := r.client.Delete(_SERVICE_ID_PATH + uuid)
	return err
}

func (r *serviceIDRepository) Get(uuid string) (models.ServiceID, error) {
	serviceID := ServiceIDResponse{}
	_, err := r.client.Get(_SERVICE_ID_PATH+uuid, &serviceID)
	if err != nil {
		return models.ServiceID{}, err
	}
	return serviceID.ToModel(), nil
}
