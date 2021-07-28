/*
Copyright 2020 The Skaffold Authors

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

package kpt

import (
	"fmt"

	deployerr "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/error"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

// sourceErr raises the user message when the given path cannot be used as `kpt fn source`. This could be caused by
// different reasons and need users to take an action.
func sourceErr(err error, path string) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("unable to read the manifests as kpt fn source: %s", err),
			ErrCode: proto.StatusCode_DEPLOY_KPT_SOURCE_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_INVALID_KPT_MANIFESTS,
					Action: fmt.Sprintf("please make sure the content is valid kubernetes yaml or "+
						"kpt fn configuration: %v", path),
				},
			},
		})
}

// pkgInitErr raises when skaffold fails to run `kpt pkg init` which creates the Kptfile under the given dir.
func pkgInitErr(err error, path string) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("fail to run `kpt pkg init`: %s", err),
			ErrCode: proto.StatusCode_DEPLOY_KPTFILE_INIT_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_KPTFILE_MANUAL_INIT,
					Action:         fmt.Sprintf("suggest manually run `kpt pkg init %v`", path),
				},
			},
		})
}

// liveInitErr raises when skaffold fails to init the "inventory" field in the Kptfile.
// The Kptfile should have already exist.
func liveInitErr(err error, path string) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("fail to run `kpt live init`: %s", err),
			ErrCode: proto.StatusCode_DEPLOY_KPTFILE_INIT_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_KPTFILE_MANUAL_INIT,
					Action: fmt.Sprintf("please check the %v and make sure the \"inventory\" field "+
						"does not exist and manually run `kpt live init %v`", path, path),
				},
			},
		})
}

// liveApplyErr raises when skaffold fails to live apply the manifests to the cluster. The Action gives the solutions
// for a very common case for new kpt users.
func liveApplyErr(err error, path string) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("fail to run `kpt live apply %v`: %s", path, err),
			ErrCode: proto.StatusCode_DEPLOY_KPT_APPLY_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_ALIGN_KPT_INVENTORY,
					Action: fmt.Sprintln("if you encounter an inventory mismatch (or can't adopt) error, it indicates " +
						"the manifests have been deployed before and may not be properly cleaned up. We provide two solutions:\n " +
						"#1: Update your skaffold.yaml by adding the `--inventory-policy=adopt` in the " +
						"`.deploy.kpt.flags`. This will override the resources' inventory.\n " +
						"#2: Find the existing inventory from your cluster resource's annotation \"cli-utils.sigs.k8s.io/inventory-id\", " +
						" then use this inventory-id to look for the inventory name and namespace by running " +
						"`kubectl get resourcegroups.kpt.dev -oyaml | grep <YOUR_INVENTORY_ID> -C20 | grep \"name: inventory-\" -C1` " +
						"and update the `.deploy.kpt.name`, `.deploy.kpt.namespace` and `.deploy.kpt.inventoryID` in your skaffold.yaml"),
				},
			},
		})
}

// liveDestroyErr raises when skaffold fails to run `kpt live destroy`, which is expected to delete the resource on
// the cluster side.
func liveDestroyErr(err error, path string) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("fail to run `kpt live destroy`: %s", err),
			ErrCode: proto.StatusCode_DEPLOY_CLEANUP_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_KPTFILE_MANUAL_INIT,
					Action: fmt.Sprintf("the kubernetes configurations are not deleted from the cluster! "+
						"please run `kpt live status %v` to see what goes wrong.", path),
				},
			},
		})
}

// openFileErr raises when the given file path (Kptfile) cannot be opened. The error message is normally self-explained.
func openFileErr(err error, path string) error {
	return deployerr.UserError(err, proto.StatusCode_UNKNOWN_ERROR)
}

// parseFileErr raises when the Kptfile is invalid.
func parseFileErr(err error, path string) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("unable to parse Kptfile %v: %s", path, err),
			ErrCode: proto.StatusCode_DEPLOY_KPTFILE_INVALID_YAML_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_KPTFILE_CHECK_YAML,
					Action: fmt.Sprintf("please check if the Kptfile is correct and " +
						"the `apiVersion` is greater than `v1alpha2`"),
				},
			},
		})
}
