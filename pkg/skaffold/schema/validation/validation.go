/*
Copyright 2019 The Skaffold Authors

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

package validation

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yamltags"
	"github.com/GoogleContainerTools/skaffold/proto"
)

var (
	// for testing
	validateYamltags       = yamltags.ValidateStruct
	dependencyAliasPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// Process checks if the Skaffold pipeline is valid and returns all encountered errors as a concatenated string
func Process(configs []*latest.SkaffoldConfig) error {
	var errs []error
	for _, config := range configs {
		errs = visitStructs(config, validateYamltags)
		errs = append(errs, validateImageNames(config.Build.Artifacts)...)
		errs = append(errs, validateDockerNetworkMode(config.Build.Artifacts)...)
		errs = append(errs, validateCustomDependencies(config.Build.Artifacts)...)
		errs = append(errs, validateSyncRules(config.Build.Artifacts)...)
		errs = append(errs, validatePortForwardResources(config.PortForward)...)
		errs = append(errs, validateJibPluginTypes(config.Build.Artifacts)...)
		errs = append(errs, validateLogPrefix(config.Deploy.Logs)...)
		errs = append(errs, validateArtifactTypes(config.Build)...)
		errs = append(errs, validateTaggingPolicy(config.Build)...)
	}
	errs = append(errs, validateArtifactDependencies(configs)...)

	if len(errs) == 0 {
		return nil
	}

	var messages []string
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	return fmt.Errorf(strings.Join(messages, " | "))
}

// ProcessWithRunContext checks if the Skaffold pipeline is valid when a RunContext is required.
// It returns all encountered errors as a concatenated string.
func ProcessWithRunContext(runCtx *runcontext.RunContext) error {
	var errs []error
	errs = append(errs, validateDockerNetworkContainerExists(runCtx.Artifacts(), runCtx)...)

	if len(errs) == 0 {
		return nil
	}
	var messages []string
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	return fmt.Errorf(strings.Join(messages, " | "))
}

// validateTaggingPolicy checks that the tagging policy is valid in combination with other options.
func validateTaggingPolicy(bc latest.BuildConfig) (errs []error) {
	if bc.LocalBuild != nil {
		// sha256 just uses `latest` tag, so tryImportMissing will virtually always succeed (#4889)
		if bc.LocalBuild.TryImportMissing && bc.TagPolicy.ShaTagger != nil {
			errs = append(errs, fmt.Errorf("tagging policy 'sha256' can not be used when 'tryImportMissing' is enabled"))
		}
	}
	return
}

// validateImageNames makes sure the artifact image names are valid base names,
// without tags nor digests.
func validateImageNames(artifacts []*latest.Artifact) (errs []error) {
	for _, a := range artifacts {
		parsed, err := docker.ParseReference(a.ImageName)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid imageName '%s': %v", a.ImageName, err))
			continue
		}

		if parsed.Tag != "" {
			errs = append(errs, fmt.Errorf("invalid imageName '%s': no tag should be specified. Use taggers instead: https://skaffold.dev/docs/how-tos/taggers/", a.ImageName))
		}

		if parsed.Digest != "" {
			errs = append(errs, fmt.Errorf("invalid imageName '%s': no digest should be specified. Use taggers instead: https://skaffold.dev/docs/how-tos/taggers/", a.ImageName))
		}
	}
	return
}

func validateArtifactDependencies(configs []*latest.SkaffoldConfig) (errs []error) {
	var artifacts []*latest.Artifact
	for _, c := range configs {
		artifacts = append(artifacts, c.Build.Artifacts...)
	}
	errs = append(errs, validateUniqueDependencyAliases(artifacts)...)
	errs = append(errs, validateAcyclicDependencies(artifacts)...)
	errs = append(errs, validateValidDependencyAliases(artifacts)...)
	return
}

// validateAcyclicDependencies makes sure all artifact dependencies are found and don't have cyclic references
func validateAcyclicDependencies(artifacts []*latest.Artifact) (errs []error) {
	m := make(map[string]*latest.Artifact)
	for _, artifact := range artifacts {
		m[artifact.ImageName] = artifact
	}
	visited := make(map[string]bool)
	for _, artifact := range artifacts {
		if err := dfs(artifact, visited, make(map[string]bool), m); err != nil {
			errs = append(errs, err)
			return
		}
	}
	return
}

// dfs runs a Depth First Search algorithm for cycle detection in a directed graph
func dfs(artifact *latest.Artifact, visited, marked map[string]bool, artifacts map[string]*latest.Artifact) error {
	if marked[artifact.ImageName] {
		return fmt.Errorf("cycle detected in build dependencies involving %q", artifact.ImageName)
	}
	marked[artifact.ImageName] = true
	defer func() {
		marked[artifact.ImageName] = false
	}()
	if visited[artifact.ImageName] {
		return nil
	}
	visited[artifact.ImageName] = true

	for _, dep := range artifact.Dependencies {
		d, found := artifacts[dep.ImageName]
		if !found {
			return fmt.Errorf("unknown build dependency %q for artifact %q", dep.ImageName, artifact.ImageName)
		}
		if err := dfs(d, visited, marked, artifacts); err != nil {
			return err
		}
	}
	return nil
}

// validateValidDependencyAliases makes sure that artifact dependency aliases are valid.
// docker and custom builders require aliases match [a-zA-Z_][a-zA-Z0-9_]* pattern
func validateValidDependencyAliases(artifacts []*latest.Artifact) (errs []error) {
	for _, a := range artifacts {
		if a.DockerArtifact == nil && a.CustomArtifact == nil {
			continue
		}
		for _, d := range a.Dependencies {
			if !dependencyAliasPattern.MatchString(d.Alias) {
				errs = append(errs, fmt.Errorf("invalid build dependency for artifact %q: alias %q doesn't match required pattern %q", a.ImageName, d.Alias, dependencyAliasPattern.String()))
			}
		}
	}
	return
}

// validateUniqueDependencyAliases makes sure that artifact dependency aliases are unique for each artifact
func validateUniqueDependencyAliases(artifacts []*latest.Artifact) (errs []error) {
	type State int
	var (
		unseen   State = 0
		seen     State = 1
		recorded State = 2
	)
	for _, a := range artifacts {
		aliasMap := make(map[string]State)
		for _, d := range a.Dependencies {
			if aliasMap[d.Alias] == seen {
				errs = append(errs, fmt.Errorf("invalid build dependency for artifact %q: alias %q repeated", a.ImageName, d.Alias))
				aliasMap[d.Alias] = recorded
			} else if aliasMap[d.Alias] == unseen {
				aliasMap[d.Alias] = seen
			}
		}
	}
	return
}

// validateDockerNetworkMode makes sure that networkMode is one of `bridge`, `none`, `container:<name|id>`, or `host` if set.
func validateDockerNetworkMode(artifacts []*latest.Artifact) (errs []error) {
	for _, a := range artifacts {
		if a.DockerArtifact == nil || a.DockerArtifact.NetworkMode == "" {
			continue
		}
		mode := strings.ToLower(a.DockerArtifact.NetworkMode)
		if mode == "none" || mode == "bridge" || mode == "host" {
			continue
		}
		containerRegExp := regexp.MustCompile("^container:[a-zA-Z0-9][a-zA-Z0-9_.-]*$")
		if containerRegExp.MatchString(mode) {
			continue
		}

		errMsg := fmt.Sprintf("artifact %s has invalid networkMode '%s'", a.ImageName, mode)
		errs = append(errs, sErrors.NewError(fmt.Errorf(errMsg),
			proto.ActionableErr{
				Message: errMsg,
				ErrCode: proto.StatusCode_INIT_DOCKER_NETWORK_INVALID_CONTAINER_NAME,
				Suggestions: []*proto.Suggestion{
					{
						SuggestionCode: proto.SuggestionCode_FIX_DOCKER_NETWORK_CONTAINER_NAME,
						Action:         "Please fix the docker network container name and try again",
					},
				},
			}))
	}
	return
}

// Validates that a Docker Container with a Network Mode "container:<id|name>" points to an actually running container
func validateDockerNetworkContainerExists(artifacts []*latest.Artifact, runCtx docker.Config) []error {
	var errs []error
	apiClient, err := docker.NewAPIClient(runCtx)
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	client := apiClient.RawClient()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()

	for _, a := range artifacts {
		if a.DockerArtifact == nil || a.DockerArtifact.NetworkMode == "" {
			continue
		}
		mode := strings.ToLower(a.DockerArtifact.NetworkMode)
		prefix := "container:"
		if strings.HasPrefix(mode, prefix) {
			id := strings.TrimPrefix(mode, prefix)
			containers, err := client.ContainerList(ctx, types.ContainerListOptions{})
			if err != nil {
				errs = append(errs, sErrors.NewError(err,
					proto.ActionableErr{
						Message: "error retrieving docker containers list",
						ErrCode: proto.StatusCode_INIT_DOCKER_NETWORK_LISTING_CONTAINERS,
						Suggestions: []*proto.Suggestion{
							{
								SuggestionCode: proto.SuggestionCode_CHECK_DOCKER_RUNNING,
								Action:         "Please check docker is running and try again",
							},
						},
					}))
				return errs
			}
			for _, c := range containers {
				// Comparing ID seeking for <id>
				if c.ID == id {
					return errs
				}
				for _, name := range c.Names {
					// c.Names come in form "/<name>"
					if name == "/"+id {
						return errs
					}
				}
			}
			errMsg := fmt.Sprintf("container '%s' not found, required by image '%s' for docker network stack sharing", id, a.ImageName)
			errs = append(errs, sErrors.NewError(fmt.Errorf(errMsg),
				proto.ActionableErr{
					Message: errMsg,
					ErrCode: proto.StatusCode_INIT_DOCKER_NETWORK_CONTAINER_DOES_NOT_EXIST,
					Suggestions: []*proto.Suggestion{
						{
							SuggestionCode: proto.SuggestionCode_CHECK_DOCKER_NETWORK_CONTAINER_RUNNING,
							Action:         "Please fix the docker network container name and try again.",
						},
					},
				}))
		}
	}
	return errs
}

// validateCustomDependencies makes sure that dependencies.ignore is only used in conjunction with dependencies.paths
func validateCustomDependencies(artifacts []*latest.Artifact) (errs []error) {
	for _, a := range artifacts {
		if a.CustomArtifact == nil || a.CustomArtifact.Dependencies == nil || a.CustomArtifact.Dependencies.Ignore == nil {
			continue
		}

		if a.CustomArtifact.Dependencies.Dockerfile != nil || a.CustomArtifact.Dependencies.Command != "" {
			errs = append(errs, fmt.Errorf("artifact %s has invalid dependencies; dependencies.ignore can only be used in conjunction with dependencies.paths", a.ImageName))
		}
	}
	return
}

// visitStructs recursively visits all fields in the config and collects errors found by the visitor
func visitStructs(s interface{}, visitor func(interface{}) error) []error {
	v := reflect.ValueOf(s)
	t := reflect.TypeOf(s)

	switch v.Kind() {
	case reflect.Struct:
		var errs []error
		if err := visitor(v.Interface()); err != nil {
			errs = append(errs, err)
		}

		// also check all fields of the current struct
		for i := 0; i < t.NumField(); i++ {
			if !v.Field(i).CanInterface() {
				continue
			}
			if fieldErrs := visitStructs(v.Field(i).Interface(), visitor); fieldErrs != nil {
				errs = append(errs, fieldErrs...)
			}
		}

		return errs

	case reflect.Slice:
		// for slices check each element
		var errs []error
		for i := 0; i < v.Len(); i++ {
			if elemErrs := visitStructs(v.Index(i).Interface(), visitor); elemErrs != nil {
				errs = append(errs, elemErrs...)
			}
		}
		return errs

	case reflect.Ptr:
		// for pointers check the referenced value
		if v.IsNil() {
			return nil
		}
		return visitStructs(v.Elem().Interface(), visitor)

	default:
		// other values are fine
		return nil
	}
}

// validateSyncRules checks that all manual sync rules have a valid strip prefix
func validateSyncRules(artifacts []*latest.Artifact) []error {
	var errs []error
	for _, a := range artifacts {
		if a.Sync != nil {
			for _, r := range a.Sync.Manual {
				if !strings.HasPrefix(r.Src, r.Strip) {
					err := fmt.Errorf("sync rule pattern '%s' does not have prefix '%s'", r.Src, r.Strip)
					errs = append(errs, err)
				}
			}
		}
	}
	return errs
}

// validatePortForwardResources checks that all user defined port forward resources
// have a valid resourceType
func validatePortForwardResources(pfrs []*latest.PortForwardResource) []error {
	var errs []error
	validResourceTypes := map[string]struct{}{
		"pod":                   {},
		"deployment":            {},
		"service":               {},
		"replicaset":            {},
		"replicationcontroller": {},
		"statefulset":           {},
		"daemonset":             {},
		"cronjob":               {},
		"job":                   {},
	}
	for _, pfr := range pfrs {
		resourceType := strings.ToLower(string(pfr.Type))
		if _, ok := validResourceTypes[resourceType]; !ok {
			errs = append(errs, fmt.Errorf("%s is not a valid resource type for port forwarding", pfr.Type))
		}
	}
	return errs
}

// validateJibPluginTypes makes sure that jib type is one of `maven`, or `gradle` if set.
func validateJibPluginTypes(artifacts []*latest.Artifact) (errs []error) {
	for _, a := range artifacts {
		if a.JibArtifact == nil || a.JibArtifact.Type == "" {
			continue
		}
		t := strings.ToLower(a.JibArtifact.Type)
		if t == "maven" || t == "gradle" {
			continue
		}
		errs = append(errs, fmt.Errorf("artifact %s has invalid Jib plugin type '%s'", a.ImageName, t))
	}
	return
}

// validateArtifactTypes checks that the artifact types are compatible with the specified builder.
func validateArtifactTypes(bc latest.BuildConfig) (errs []error) {
	switch {
	case bc.LocalBuild != nil:
		for _, a := range bc.Artifacts {
			if misc.ArtifactType(a) == misc.Kaniko {
				errs = append(errs, fmt.Errorf("found a '%s' artifact, which is incompatible with the 'local' builder:\n\n%s\n\nTo use the '%s' builder, add the 'cluster' stanza to the 'build' section of your configuration. For information, see https://skaffold.dev/docs/pipeline-stages/builders/", misc.ArtifactType(a), misc.FormatArtifact(a), misc.ArtifactType(a)))
			}
		}
	case bc.GoogleCloudBuild != nil:
		for _, a := range bc.Artifacts {
			at := misc.ArtifactType(a)
			if at != misc.Kaniko && at != misc.Docker && at != misc.Jib && at != misc.Buildpack {
				errs = append(errs, fmt.Errorf("found a '%s' artifact, which is incompatible with the 'gcb' builder:\n\n%s\n\nTo use the '%s' builder, remove the 'googleCloudBuild' stanza from the 'build' section of your configuration. For information, see https://skaffold.dev/docs/pipeline-stages/builders/", misc.ArtifactType(a), misc.FormatArtifact(a), misc.ArtifactType(a)))
			}
		}
	case bc.Cluster != nil:
		for _, a := range bc.Artifacts {
			if misc.ArtifactType(a) != misc.Kaniko && misc.ArtifactType(a) != misc.Custom {
				errs = append(errs, fmt.Errorf("found a '%s' artifact, which is incompatible with the 'cluster' builder:\n\n%s\n\nTo use the '%s' builder, remove the 'cluster' stanza from the 'build' section of your configuration. For information, see https://skaffold.dev/docs/pipeline-stages/builders/", misc.ArtifactType(a), misc.FormatArtifact(a), misc.ArtifactType(a)))
			}
		}
	}
	return
}

// validateLogPrefix checks that logs are configured with a valid prefix.
func validateLogPrefix(lc latest.LogsConfig) []error {
	validPrefixes := []string{"", "auto", "container", "podAndContainer", "none"}

	if !util.StrSliceContains(validPrefixes, lc.Prefix) {
		return []error{fmt.Errorf("invalid log prefix '%s'. Valid values are 'auto', 'container', 'podAndContainer' or 'none'", lc.Prefix)}
	}

	return nil
}
