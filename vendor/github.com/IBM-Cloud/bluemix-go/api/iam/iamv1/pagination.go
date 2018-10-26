package iamv1

import (
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
)

const _PageTokenQuery = "pagetoken"

type IAMPaginatedResourcesHandler struct {
	resourceType reflect.Type
}

func NewIAMPaginatedResources(resource interface{}) IAMPaginatedResourcesHandler {
	return IAMPaginatedResourcesHandler{
		resourceType: reflect.TypeOf(resource),
	}
}

func (pr IAMPaginatedResourcesHandler) Resources(bytes []byte, curPath string) ([]interface{}, string, error) {
	var paginatedResources = struct {
		NextPageToken  string          `json:"nextPageToken"`
		ResourcesBytes json.RawMessage `json:"items"`
	}{}

	err := json.Unmarshal(bytes, &paginatedResources)

	var nextPath string
	if paginatedResources.NextPageToken != "" {
		u, err := url.Parse(curPath)
		if err == nil {
			q := u.Query()
			q.Set(_PageTokenQuery, paginatedResources.NextPageToken)
			u.RawQuery = q.Encode()
			nextPath = u.String()
		}
	}

	slicePtr := reflect.New(reflect.SliceOf(pr.resourceType))
	dc := json.NewDecoder(strings.NewReader(string(paginatedResources.ResourcesBytes)))
	dc.UseNumber()
	err = dc.Decode(slicePtr.Interface())
	slice := reflect.Indirect(slicePtr)

	contents := make([]interface{}, 0, slice.Len())
	for i := 0; i < slice.Len(); i++ {
		contents = append(contents, slice.Index(i).Interface())
	}
	return contents, nextPath, err
}
