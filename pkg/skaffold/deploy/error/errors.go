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

package error

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/types"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	proto "github.com/GoogleContainerTools/skaffold/proto/v1"
)

const (
	executableNotFound     = "executable file not found"
	notFound               = "%s not found"
	defaultMinikubeProfile = "minikube"
)

var (
	clusterInternalSystemErr = regexp.MustCompile(".*Internal Server Error")

	// for testing
	internalSystemErrSuggestion = internalSystemErrSuggestionFunc
)

// DebugHelperRetrieveErr is thrown when debug helpers could not be retrieved.
// This error occurs in skaffold debug command when transforming the manifest before deploying.
func DebugHelperRetrieveErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_DEBUG_HELPER_RETRIEVE_ERR,
		})
}

// CleanupErr represents error during deploy clean up.
// This error could happen in the skaffold clean up phase or
// if `wait-for-deletions` is specified on command line.
func CleanupErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_CLEANUP_ERR,
		})
}

// MissingToolErr returns a concise error if error is due to deploy tool executable not found.
func MissingToolErr(toolName string, err error) string {
	if strings.Contains(err.Error(), executableNotFound) {
		return fmt.Sprintf(notFound, toolName)
	}
	return err.Error()
}

func UserError(err error, sc proto.StatusCode) error {
	if sErrors.IsSkaffoldErr(err) {
		return err
	}
	if clusterInternalSystemErr.MatchString(err.Error()) {
		return sErrors.NewProblem(
			func(err error) string {
				return fmt.Sprintf("Deploy Failed. %v", err)
			},
			proto.StatusCode_DEPLOY_CLUSTER_INTERNAL_SYSTEM_ERR,
			internalSystemErrSuggestion,
			err)
	}
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: err.Error(),
			ErrCode: sc,
		})
}

func CheckMinikubeStatusSuggestion(cfg types.Config) *proto.Suggestion {
	return &proto.Suggestion{
		SuggestionCode: proto.SuggestionCode_CHECK_MINIKUBE_STATUS,
		Action: fmt.Sprintf("Check if minikube is running using %q command and try again",
			getMinikubeStatusCommand(cfg.GetKubeContext())),
	}
}

func getMinikubeStatusCommand(p string) string {
	if p == defaultMinikubeProfile {
		return "minikube status"
	}
	return fmt.Sprintf("minikube status -p %s", p)
}

func internalSystemErrSuggestionFunc(cfg interface{}) []*proto.Suggestion {
	deployCfg, ok := cfg.(types.Config)
	if !ok {
		return nil
	}
	if deployCfg.MinikubeProfile() != "" {
		return []*proto.Suggestion{
			CheckMinikubeStatusSuggestion(deployCfg),
			{
				SuggestionCode: proto.SuggestionCode_OPEN_ISSUE,
				// TODO: show tip to run minikube logs command and attach logs.
				Action: fmt.Sprintf("open an issue at %s", constants.GithubIssueLink),
			}}
	}
	return []*proto.Suggestion{{
		SuggestionCode: proto.SuggestionCode_OPEN_ISSUE,
		Action:         fmt.Sprintf("Something went wrong with your cluster %q. Try again.\nIf this keeps happening please open an issue at %s", deployCfg.GetKubeContext(), constants.GithubIssueLink),
	}}
}
