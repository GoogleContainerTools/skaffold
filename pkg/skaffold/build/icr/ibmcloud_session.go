/*
Copyright 2018 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package icr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/user"

	ibmcloud "github.com/IBM-Cloud/bluemix-go"
	"github.com/IBM-Cloud/bluemix-go/api/container/registryv1"
	"github.com/IBM-Cloud/bluemix-go/api/iam/iamv1"
	"github.com/IBM-Cloud/bluemix-go/endpoints"
	"github.com/IBM-Cloud/bluemix-go/session"
	"github.com/pkg/errors"

	"strings"
)

// IBMRegistrySession structure
type IBMRegistrySession struct {
	Registry          string
	Builds            registryv1.Builds
	BuildTargetHeader registryv1.BuildTargetHeader
}

type configJSON struct {
	Region          string `json:"Region"`
	IAMToken        string `json:"IAMToken"`
	IAMRefreshToken string `json:"IAMRefreshToken"`
	Account         struct {
		GUID string `json:"GUID"`
	} `json:"Account"`
	SSLDisabled bool `json:"SSLDisabled"`
}

// NewRegistryClient Authenticates with IBM Cloud using provided API Key
// Fixes the image name if the registry name isn't part of it
func (b Builder) NewRegistryClient(imageName string) (*IBMRegistrySession, string, error) {
	var (
		c = &ibmcloud.Config{
			Region:        b.DefaultRegion,
			BluemixAPIKey: b.APIKey,
		}

		authSession *session.Session
		endpoint    *string
		account     string
		iamAPI      iamv1.IAMServiceAPI
		registryAPI registryv1.RegistryServiceAPI
		userInfo    *iamv1.UserInfo
		err         error
	)

	if c.BluemixAPIKey == "" {
		account, err = configFromJSON(c)
	}
	authSession, err = session.New(c)
	if err != nil {
		return nil, imageName, errors.Wrap(err, "IBM Cloud configuration error.")
	}
	iamAPI, err = iamv1.New(authSession)
	if err != nil {
		return nil, imageName, errors.Wrap(err, "IBM Cloud auth error.")
	}

	if account == "" {
		userInfo, err = iamAPI.Identity().UserInfo()
		if err != nil {
			return nil, imageName, errors.Wrap(err, "IBM Cloud fetching user account error.")
		}
		account = userInfo.Account.Bss
	}
	endpoint = getRegistryEndpoint(imageName)
	if endpoint == nil {
		var registry string

		registry, err = endpoints.NewEndpointLocator(b.DefaultRegion).ContainerRegistryEndpoint()
		if err != nil {
			return nil, imageName, errors.Wrap(err, "Unsupported IBM Cloud default region")
		}
		endpoint = &registry
		imageName, err = addRegistry(registry, imageName)
		if err != nil {
			return nil, imageName, err
		}
	}
	c.Endpoint = endpoint
	registryAPI, err = registryv1.New(authSession)
	if err != nil {
		return nil, imageName, errors.Wrap(err, "IBM Cloud auth error.")
	}

	return &IBMRegistrySession{
		BuildTargetHeader: registryv1.BuildTargetHeader{
			AccountID: account,
		},
		Builds: registryAPI.Builds(),
	}, imageName, nil
}

func getRegistryEndpoint(imageName string) *string {
	var segments []string
	var endpoint string

	segments = strings.Split(imageName, "/")
	if len(segments) > 0 && len(imageName) > 0 {
		if !strings.Contains(segments[0], ".") {
			return nil
		}
	}
	endpoint = fmt.Sprintf("https://%s", segments[0])
	return &endpoint
}

// addRegistry for adding the default registry if no registry is in image
func addRegistry(endpoint string, imageName string) (string, error) {
	var (
		registryURL *url.URL
		segments    []string
		err         error
	)

	registryURL, err = url.Parse(endpoint)
	if err != nil {
		return "", errors.Wrap(err, "Bad registry URL for IBM Cloud default region")
	}
	segments = strings.Split(imageName, "/")
	if len(segments) > 0 && len(imageName) > 0 {
		if strings.Contains(segments[0], ".") {
			return imageName, nil
		}
		tempName := imageName
		if strings.HasPrefix(imageName, "/") {
			tempName = imageName[1:len(imageName)]
		}
		return fmt.Sprintf("%s/%s", registryURL.Hostname(), tempName), nil
	}
	return imageName, nil
}

// If the authenticated with IBM Cloud using CLI
func configFromJSON(icconfig *ibmcloud.Config) (accountID string, err error) {
	var (
		config    *configJSON
		jsonFile  *os.File
		usr       *user.User
		byteValue []byte
	)

	config = new(configJSON)
	usr, err = user.Current()
	if err == nil {
		jsonFile, err = os.Open(usr.HomeDir + "/.bluemix/config.json")
		defer jsonFile.Close()
		if err == nil {
			byteValue, err = ioutil.ReadAll(jsonFile)
			if err == nil {
				err = json.Unmarshal(byteValue, config)
				if err == nil {
					icconfig.Region = config.Region
					icconfig.IAMAccessToken = config.IAMToken
					icconfig.IAMRefreshToken = config.IAMRefreshToken
					icconfig.SSLDisable = config.SSLDisabled
					icconfig.BluemixAPIKey = "n/a"
					icconfig.IBMID = "n/a"
					icconfig.IBMIDPassword = "n/a"
					accountID = config.Account.GUID
				}
			}
		}
	}
	return accountID, err
}
