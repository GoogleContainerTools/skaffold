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

package acr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"

	cr "github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2018-09-01/containerregistry"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

type AzureAccountConfig struct {
	AccessToken  string `json:"accessToken"`
	Subscription string `json:"subscription"`
}

// simple autorest.Authorizer which adds the bearer token as a header to the
// autorest requests
type BearerAuthorizer struct {
	bearerToken string
}

func (b BearerAuthorizer) WithAuthorization() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				return autorest.Prepare(r, autorest.WithHeader("Authorization", fmt.Sprintf("Bearer %s", b.bearerToken)))
			}
			return r, err
		})
	}
}

// creates a new RegistriesClient with either the credentials from the skaffold.yaml
// or the bearer token provided by the Azure CLI
func (b Builder) NewRegistriesClient() (*cr.RegistriesClient, error) {
	if b.SubscriptionID != "" && b.TenantID != "" && b.ClientSecret != "" && b.ClientID != "" {
		client := cr.NewRegistriesClient(b.SubscriptionID)

		authorizer, err := auth.NewClientCredentialsConfig(b.ClientID, b.ClientSecret, b.TenantID).Authorizer()
		if err != nil {
			return nil, errors.Wrap(err, "create authorizer with credentials")
		}

		client.Authorizer = authorizer
		return &client, nil
	}

	cmd := exec.Command("az", "account", "get-access-token")
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't find azure cli.")
	}

	cfg := AzureAccountConfig{}
	err = json.Unmarshal(stdout, &cfg)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal azure account config")
	}
	b.SubscriptionID = cfg.Subscription //we'll need the subscriptionId for the RunsClient

	client := cr.NewRegistriesClient(cfg.Subscription)
	client.Authorizer = BearerAuthorizer{bearerToken: cfg.AccessToken}

	return &client, nil
}
