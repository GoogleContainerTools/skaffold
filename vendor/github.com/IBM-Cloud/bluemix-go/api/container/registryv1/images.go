package registryv1

import (
	"fmt"
	"strconv"
	"time"

	"github.com/IBM-Cloud/bluemix-go/client"
	"github.com/IBM-Cloud/bluemix-go/helpers"
	"github.com/IBM-Cloud/bluemix-go/rest"
)

type ImageTargetHeader struct {
	AccountID string
}

//ToMap ...
func (c ImageTargetHeader) ToMap() map[string]string {
	m := make(map[string]string, 1)
	m[accountIDHeader] = c.AccountID
	return m
}

//Subnets interface
type Images interface {
	GetImages(params GetImageRequest, target ImageTargetHeader) (*GetImagesResponse, error)
	InspectImage(imageName string, target ImageTargetHeader) (*ImageInspectResponse, error)
	DeleteImage(imageName string, target ImageTargetHeader) (*DeleteImageResponse, error)
	ImageVulnerabilities(imageName string, param ImageVulnerabilitiesRequest, target ImageTargetHeader) (*ImageVulnerabilitiesResponse, error)
}

type images struct {
	client *client.Client
}

func newImageAPI(c *client.Client) Images {
	return &images{
		client: c,
	}
}

type Digesttags struct {
	Tags map[string][]string
}

type Labels struct {
	Labels map[string][]string
}

type GetImagesResponse []struct {
	ID                      string              `json:"Id"`
	ParentID                string              `json:"ParentId"`
	DigestTags              map[string][]string `json:"DigestTags"`
	RepoTags                []string            `json:"RepoTags"`
	RepoDigests             []string            `json:"RepoDigests"`
	Created                 int                 `json:"Created"`
	Size                    int64               `json:"Size"`
	VirtualSize             int64               `json:"VirtualSize"`
	Labels                  map[string]string   `json:"Labels"`
	Vulnerable              string              `json:"Vulnerable"`
	VulnerabilityCount      int                 `json:"VulnerabilityCount"`
	ConfigurationIssueCount int                 `json:"ConfigurationIssueCount"`
	IssueCount              int                 `json:"IssueCount"`
	ExemptIssueCount        int                 `json:"ExemptIssueCount"`
}
type ImageInspectResponse struct {
	ID              string    `json:"Id"`
	Parent          string    `json:"Parent"`
	Comment         string    `json:"Comment"`
	Created         time.Time `json:"Created"`
	Container       string    `json:"Container"`
	ContainerConfig struct {
		Hostname     string                 `json:"Hostname"`
		Domainname   string                 `json:"Domainname"`
		User         string                 `json:"User"`
		AttachStdin  bool                   `json:"AttachStdin"`
		AttachStdout bool                   `json:"AttachStdout"`
		AttachStderr bool                   `json:"AttachStderr"`
		ExposedPorts map[string]interface{} `json:"ExposedPorts"`
		Tty          bool                   `json:"Tty"`
		OpenStdin    bool                   `json:"OpenStdin"`
		StdinOnce    bool                   `json:"StdinOnce"`
		Env          []string               `json:"Env"`
		Cmd          []string               `json:"Cmd"`
		ArgsEscaped  bool                   `json:"ArgsEscaped"`
		Image        string                 `json:"Image"`
		Volumes      map[string]interface{} `json:"Volumes"`
		WorkingDir   string                 `json:"WorkingDir"`
		Entrypoint   []string               `json:"Entrypoint"`
		OnBuild      []string               `json:"OnBuild"`
		Labels       map[string]string      `json:"Labels"`
	} `json:"ContainerConfig"`
	DockerVersion string `json:"DockerVersion"`
	Author        string `json:"Author"`
	Config        struct {
		Hostname     string                 `json:"Hostname"`
		Domainname   string                 `json:"Domainname"`
		User         string                 `json:"User"`
		AttachStdin  bool                   `json:"AttachStdin"`
		AttachStdout bool                   `json:"AttachStdout"`
		AttachStderr bool                   `json:"AttachStderr"`
		ExposedPorts map[string]interface{} `json:"ExposedPorts"`
		Tty          bool                   `json:"Tty"`
		OpenStdin    bool                   `json:"OpenStdin"`
		StdinOnce    bool                   `json:"StdinOnce"`
		Env          []string               `json:"Env"`
		Cmd          []string               `json:"Cmd"`
		ArgsEscaped  bool                   `json:"ArgsEscaped"`
		Image        string                 `json:"Image"`
		Volumes      map[string]interface{} `json:"Volumes"`
		WorkingDir   string                 `json:"WorkingDir"`
		Entrypoint   []string               `json:"Entrypoint"`
		OnBuild      []string               `json:"OnBuild"`
		Labels       map[string]string      `json:"Labels"`
	} `json:"Config"`
	Architecture string `json:"Architecture"`
	Os           string `json:"Os"`
	Size         int64  `json:"Size"`
	VirtualSize  int64  `json:"VirtualSize"`
	RootFS       struct {
		Type   string   `json:"Type"`
		Layers []string `json:"Layers"`
	} `json:"RootFS"`
}

type DeleteImageResponse struct {
	Untagged string `json:"Untagged"`
}

type ImageVulnerabilitiesResponse struct {
	Metadata struct {
		Namespace   string    `json:"namespace"`
		Complete    bool      `json:"complete"`
		CrawledTime time.Time `json:"crawled_time"`
		OsSupported bool      `json:"os_supported"`
	} `json:"metadata"`
	Summary struct {
		Malware struct {
			Compliant bool   `json:"compliant"`
			Reason    string `json:"reason"`
		} `json:"malware"`
		Compliance struct {
			ComplianceViolations int    `json:"compliance_violations"`
			Reason               string `json:"reason"`
			Compliant            bool   `json:"compliant"`
			TotalComplianceRules int    `json:"total_compliance_rules"`
			ExecutionStatus      string `json:"execution_status"`
		} `json:"compliance"`
		Secureconfig struct {
			Misconfigured   int `json:"misconfigured"`
			CorrectOutput   int `json:"correct_output"`
			TotalOutputDocs int `json:"total_output_docs"`
		} `json:"secureconfig"`
		Vulnerability struct {
			TotalPackages      int `json:"total_packages"`
			TotalUsnsForDistro int `json:"total_usns_for_distro"`
			VulnerableUsns     int `json:"vulnerable_usns"`
			VulnerablePackages int `json:"vulnerable_packages"`
		} `json:"vulnerability"`
	} `json:"summary"`
	Detail struct {
		Compliance []struct {
			Reason         string `json:"reason"`
			Compliant      bool   `json:"compliant"`
			Description    string `json:"description"`
			PolicyMandated bool   `json:"policy_mandated"`
		} `json:"compliance"`
		Vulnerability []struct {
			PackageName     string `json:"package_name"`
			Vulnerabilities []struct {
				URL     string   `json:"url"`
				Cveid   []string `json:"cveid"`
				Summary string   `json:"summary"`
			} `json:"vulnerabilities"`
		} `json:"vulnerability"`
	} `json:"detail"`
}

/*GetImageRequest contains all the parameters to send to the API endpoint
for the image list operation typically these are written to a http.Request
*/
type GetImageRequest struct {
	/*IncludeIBM
	  Includes IBM-provided public images in the list of images. If this option is not specified, private images are listed only. If this option is specified more than once, the last parsed setting is the setting that is used.
	*/
	IncludeIBM bool
	/*IncludePrivate
	  Includes private images in the list of images. If this option is not specified, private images are listed. If this option is specified more than once, the last parsed setting is the setting that is used.
	*/
	IncludePrivate bool
	/*Namespace
	  Lists images that are stored in the specified namespace only. Query multiple namespaces by specifying this option for each namespace. If this option is not specified, images from all namespaces in the specified IBM Cloud account are listed.
	*/
	Namespace string
	/*Repository
	  Lists images that are stored in the specified repository, under your namespaces. Query multiple repositories by specifying this option for each repository. If this option is not specified, images from all repos are listed.
	*/
	Repository string
	/*Vulnerabilities
	  Displays Vulnerability Advisor status for the listed images. If this option is specified more than once, the last parsed setting is the setting that is used.
	*/
	Vulnerabilities bool
}

type ImageVulnerabilitiesRequest struct {

	/*Advisory
	  Specifies to include advisory compliance checks in the report.
	*/
	Advisory bool
	/*All
	  Specifies to include all checks in the report. If not specified or false, only failing checks are returned.
	*/
	All bool
}

func DefaultGetImageRequest() *GetImageRequest {
	return &GetImageRequest{
		IncludeIBM:      false,
		IncludePrivate:  true,
		Namespace:       "",
		Repository:      "",
		Vulnerabilities: true,
	}
}

func DefaultImageVulnerabilitiesRequest() *ImageVulnerabilitiesRequest {
	return &ImageVulnerabilitiesRequest{
		Advisory: false,
		All:      false,
	}
}

func (r *images) GetImages(params GetImageRequest, target ImageTargetHeader) (*GetImagesResponse, error) {

	var retVal GetImagesResponse
	req := rest.GetRequest(helpers.GetFullURL(*r.client.Config.Endpoint, "/api/v1/images")).
		Query("includeIBM", strconv.FormatBool(params.IncludeIBM)).
		Query("includePrivate", strconv.FormatBool(params.IncludePrivate)).
		Query("vulnerabilities", strconv.FormatBool(params.Vulnerabilities))
	if params.Namespace != "" {
		req = req.Query("namespace", params.Namespace)
	}
	if params.Repository != "repository" {
		req = req.Query("repository", params.Repository)
	}
	for key, value := range target.ToMap() {
		req.Set(key, value)
	}

	_, err := r.client.SendRequest(req, &retVal)
	if err != nil {
		return nil, err
	}
	return &retVal, err
}

func (r *images) InspectImage(imageName string, target ImageTargetHeader) (*ImageInspectResponse, error) {

	var retVal ImageInspectResponse
	req := rest.GetRequest(helpers.GetFullURL(*r.client.Config.Endpoint, fmt.Sprintf("/api/v1/images/%s/json", imageName)))

	for key, value := range target.ToMap() {
		req.Set(key, value)
	}

	_, err := r.client.SendRequest(req, &retVal)
	if err != nil {
		return nil, err
	}
	return &retVal, err
}

func (r *images) DeleteImage(imageName string, target ImageTargetHeader) (*DeleteImageResponse, error) {

	var retVal DeleteImageResponse
	req := rest.DeleteRequest(helpers.GetFullURL(*r.client.Config.Endpoint, fmt.Sprintf("/api/v1/images/%s", imageName)))

	for key, value := range target.ToMap() {
		req.Set(key, value)
	}

	_, err := r.client.SendRequest(req, &retVal)
	if err != nil {
		return nil, err
	}
	return &retVal, err
}

func (r *images) ImageVulnerabilities(imageName string, params ImageVulnerabilitiesRequest, target ImageTargetHeader) (*ImageVulnerabilitiesResponse, error) {

	var retVal ImageVulnerabilitiesResponse
	req := rest.GetRequest(helpers.GetFullURL(*r.client.Config.Endpoint, fmt.Sprintf("/api/v1/images/%s/vulnerabilities", imageName))).
		Query("all", strconv.FormatBool(params.All)).
		Query("advisory", strconv.FormatBool(params.Advisory))

	for key, value := range target.ToMap() {
		req.Set(key, value)
	}

	_, err := r.client.SendRequest(req, &retVal)
	if err != nil {
		return nil, err
	}
	return &retVal, err
}
