package registryv1

import (
	"io"
	"strconv"

	"github.com/IBM-Cloud/bluemix-go/client"
	"github.com/IBM-Cloud/bluemix-go/helpers"
	"github.com/IBM-Cloud/bluemix-go/rest"
)

type BuildTargetHeader struct {
	AccountID string
}

//ToMap ...
func (c BuildTargetHeader) ToMap() map[string]string {
	m := make(map[string]string, 1)
	m[accountIDHeader] = c.AccountID
	return m
}

type ImageBuildRequest struct {
	/*T
	  The full name for the image that you want to build, including the registry URL and namespace.
	*/
	T string
	/*F
	  Specify the location of the Dockerfile relative to the build context. If not specified, the default is 'PATH/Dockerfile', where PATH is the root of the build context.
	*/
	Dockerfile string
	/*Buildargs
	  A JSON key-value structure that contains build arguments. The value of the build arguments are available as environment variables when you specify an `ARG` line which matches the key in your Dockerfile.
	*/
	Buildargs string
	/*Nocache
	  If set to true, cached image layers from previous builds are not used in this build. Use this option if you expect the result of commands that run in the build to change.
	*/
	Nocache bool
	/*Pull
	  If set to true, the base image is pulled even if an image with a matching tag already exists on the build host. The base image is specified by using the FROM keyword in your Dockerfile. Use this option to update the version of the base image on the build host.
	*/
	Pull bool
	/*Quiet
	  If set to true, build output is suppressed unless an error occurs.
	*/
	Quiet bool
	/*Squash
	  If set to true, the filesystem of the built image is reduced to one layer before it is pushed to the registry. Use this option if the number of layers in your image is close to the maximum for your storage driver.
	*/
	Squash bool
}

func DefaultImageBuildRequest() *ImageBuildRequest {
	return &ImageBuildRequest{
		T:          "",
		Dockerfile: "",
		Buildargs:  "",
		Nocache:    false,
		Pull:       false,
		Quiet:      false,
		Squash:     false,
	}
}

// Errordetail
type Errordetail struct {
	Message string `json:"message,omitempty"`
}

// Progressdetail
type Progressdetail struct {
	Current int `json:"current,omitempty"`
	Total   int `json:"total,omitempty"`
}

//ImageBuildResponse
type ImageBuildResponse struct {
	ID             string                 `json:"id,omitempty"`
	Stream         string                 `json:"stream,omitempty"`
	Status         string                 `json:"status,omitempty"`
	ProgressDetail Progressdetail         `json:"progressDetail,omitempty"`
	Error          string                 `json:"error,omitempty"`
	ErrorDetail    Errordetail            `json:"errorDetail,omitempty"`
	Aux            map[string]interface{} `json:"aux"`
}

// Callback function for build response stream
type ImageBuildResponseCallback func(respV ImageBuildResponse) bool

//Subnets interface
type Builds interface {
	ImageBuild(params ImageBuildRequest, buildContext io.Reader, target BuildTargetHeader, out io.Writer) error
	ImageBuildCallback(params ImageBuildRequest, buildContext io.Reader, target BuildTargetHeader, callback ImageBuildResponseCallback) error
}

type builds struct {
	client *client.Client
}

func newBuildAPI(c *client.Client) Builds {
	return &builds{
		client: c,
	}
}

//Create ...
func (r *builds) ImageBuildCallback(params ImageBuildRequest, buildContext io.Reader, target BuildTargetHeader, callback ImageBuildResponseCallback) error {
	req := rest.PostRequest(helpers.GetFullURL(*r.client.Config.Endpoint, "/api/v1/builds")).
		Query("t", params.T).
		Query("dockerfile", params.Dockerfile).
		Query("buildarg", params.Buildargs).
		Query("nocache", strconv.FormatBool(params.Nocache)).
		Query("pull", strconv.FormatBool(params.Pull)).
		Query("quiet", strconv.FormatBool(params.Quiet)).
		Query("squash", strconv.FormatBool(params.Squash)).
		Body(buildContext)

	for key, value := range target.ToMap() {
		req.Set(key, value)
	}

	_, err := r.client.SendRequest(req, callback)
	return err
}

//Create ...
func (r *builds) ImageBuild(params ImageBuildRequest, buildContext io.Reader, target BuildTargetHeader, out io.Writer) error {
	req := rest.PostRequest(helpers.GetFullURL(*r.client.Config.Endpoint, "/api/v1/builds")).
		Query("t", params.T).
		Query("dockerfile", params.Dockerfile).
		Query("buildarg", params.Buildargs).
		Query("nocache", strconv.FormatBool(params.Nocache)).
		Query("pull", strconv.FormatBool(params.Pull)).
		Query("quiet", strconv.FormatBool(params.Quiet)).
		Query("squash", strconv.FormatBool(params.Squash)).
		Body(buildContext)

	for key, value := range target.ToMap() {
		req.Set(key, value)
	}

	_, err := r.client.SendRequest(req, out)
	return err
}
