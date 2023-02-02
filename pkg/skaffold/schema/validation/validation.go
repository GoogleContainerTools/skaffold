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
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/parser"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/parser/configlocations"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/kpt"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringslice"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yamltags"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

var (
	// for testing
	validateYamltags       = yamltags.ValidateStruct
	DefaultConfig          = Options{CheckDeploySource: true}
	dependencyAliasPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	gcbWorkerPoolPattern   = regexp.MustCompile(`projects/[^\/]*/locations/[^\/]*/workerPools/[^\/]*`)
)

type Options struct {
	CheckDeploySource bool
}

type ErrorWithLocation struct {
	Error    error
	Location *configlocations.Location
}

func GetValidationOpts(opts config.SkaffoldOptions) Options {
	switch opts.Mode() {
	case config.RunModes.Dev, config.RunModes.Deploy, config.RunModes.Run, config.RunModes.Debug, config.RunModes.Render:
		return Options{CheckDeploySource: true}
	default:
		return Options{}
	}
}

// ProcessToErrorWithLocation checks if the Skaffold pipeline is valid and returns all encountered errors as ErrorWithLocation objects
func ProcessToErrorWithLocation(configs parser.SkaffoldConfigSet, validateConfig Options) []ErrorWithLocation {
	var errs = validateImageNames(configs)
	for _, config := range configs {
		errs = append(errs, visitStructs(config, reflect.ValueOf(config.SkaffoldConfig), validateYamltags)...)
		errs = append(errs, validateDockerNetworkMode(config, config.Build.Artifacts)...)
		errs = append(errs, validateCustomDependencies(config, config.Build.Artifacts)...)
		errs = append(errs, validateSyncRules(config, config.Build.Artifacts)...)
		errs = append(errs, validatePortForwardResources(config, config.PortForward)...)
		errs = append(errs, validateJibPluginTypes(config, config.Build.Artifacts)...)
		errs = append(errs, validateKoSync(config, config.Build.Artifacts)...)
		errs = append(errs, validateLogPrefix(config, config.Deploy.Logs)...)
		errs = append(errs, validateArtifactTypes(config, config.Build)...)
		errs = append(errs, validateTaggingPolicy(config, config.Build)...)
		errs = append(errs, validateCustomTest(config, config.Test)...)
		errs = append(errs, validateGCBConfig(config, config.Build)...)
		errs = append(errs, validateKptRendererVersion(config, config.Deploy, config.Render)...)
	}
	errs = append(errs, validateArtifactDependencies(configs)...)
	if validateConfig.CheckDeploySource {
		// TODO(6050) validate for other deploy types - helm, kpt, etc.
		errs = append(errs, validateKubectlManifests(configs)...)
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func validateKptRendererVersion(cfg *parser.SkaffoldConfigEntry, dc latest.DeployConfig, rc latest.RenderConfig) (cfgErrs []ErrorWithLocation) {
	if dc.KptDeploy != nil {
		return
	}

	if rc.Kpt == nil && rc.Transform == nil && rc.Validate == nil { // no kpt renderer created
		return
	}

	if err := kpt.CheckIsProperBinVersion(context.TODO()); err != nil {
		cfgErrs = append(cfgErrs, ErrorWithLocation{
			Error:    err,
			Location: cfg.YAMLInfos.LocateField(cfg, "Render"),
		})
	}

	return
}

// Process checks if the Skaffold pipeline is valid and returns all encountered errors as a concatenated string
func Process(configs parser.SkaffoldConfigSet, validateConfig Options) error {
	errs := ProcessToErrorWithLocation(configs, validateConfig)
	for _, config := range configs {
		errs = append(errs, wrapWithContext(config, errs...)...)
	}
	var messages []string
	for _, err := range errs {
		messages = append(messages, err.Error.Error())
	}
	if len(messages) != 0 {
		return fmt.Errorf(strings.Join(messages, "\n"))
	}
	return nil
}

// ProcessWithRunContext checks if the Skaffold pipeline is valid when a RunContext is required.
// It returns all encountered errors as a concatenated string.
func ProcessWithRunContext(ctx context.Context, runCtx *runcontext.RunContext) error {
	var errs []error
	errs = append(errs, validateDockerNetworkContainerExists(ctx, runCtx.Artifacts(), runCtx)...)
	errs = append(errs, validateVerifyTestsExistOnVerifyCommand(runCtx)...)
	errs = append(errs, validateVerifyTests(runCtx)...)
	errs = append(errs, validateLocationSetForCloudRun(runCtx)...)

	if len(errs) == 0 {
		return nil
	}
	var messages []string
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	return fmt.Errorf(strings.Join(messages, " \n "))
}

// validateTaggingPolicy checks that the tagging policy is valid in combination with other options.
func validateTaggingPolicy(cfg *parser.SkaffoldConfigEntry, bc latest.BuildConfig) (cfgErrs []ErrorWithLocation) {
	if bc.LocalBuild != nil {
		// sha256 just uses `latest` tag, so tryImportMissing will virtually always succeed (#4889)
		if bc.LocalBuild.TryImportMissing && bc.TagPolicy.ShaTagger != nil {
			cfgErrs = append(cfgErrs, ErrorWithLocation{
				Error:    fmt.Errorf("tagging policy 'sha256' can not be used when 'tryImportMissing' is enabled"),
				Location: cfg.YAMLInfos.Locate(cfg.Build.TagPolicy.ShaTagger),
			})
		}
	}
	return
}

// validateImageNames makes sure the artifact image names are unique and valid base names,
// without tags nor digests.
func validateImageNames(configs parser.SkaffoldConfigSet) (errs []ErrorWithLocation) {
	seen := make(map[string]string)
	arMap := make(map[string]*latest.Artifact)

	for _, c := range configs {
		for i, a := range c.Build.Artifacts {
			curLines := c.YAMLInfos.Locate(c.Build.Artifacts[i])
			if prevSource, found := seen[a.ImageName]; found {
				prevLines := c.YAMLInfos.Locate(arMap[c.Build.Artifacts[i].ImageName])
				errs = append(errs, ErrorWithLocation{
					Error: fmt.Errorf("duplicate image %q found in sources %s and %s: artifact image names must be unique across all configurations",
						a.ImageName, prevSource, c.SourceFile),
					Location: curLines,
				})
				errs = append(errs, ErrorWithLocation{
					Error: fmt.Errorf("duplicate image %q found in sources %s and %s: artifact image names must be unique across all configurations",
						a.ImageName, prevSource, c.SourceFile),
					Location: prevLines,
				})
				continue
			}

			seen[a.ImageName] = c.SourceFile
			arMap[a.ImageName] = a
			parsed, err := docker.ParseReference(a.ImageName)
			if err != nil {
				errs = append(errs, wrapWithContext(c, ErrorWithLocation{
					Error:    fmt.Errorf("invalid image %q: %w", a.ImageName, err),
					Location: curLines,
				})...)
				continue
			}

			if parsed.Tag != "" {
				errs = append(errs, wrapWithContext(c, ErrorWithLocation{
					Error:    fmt.Errorf("invalid image %q: no tag should be specified. Use taggers instead: https://skaffold.dev/docs/how-tos/taggers/", a.ImageName),
					Location: curLines,
				})...)
			}

			if parsed.Digest != "" {
				errs = append(errs, wrapWithContext(c, ErrorWithLocation{
					Error:    fmt.Errorf("invalid image %q: no digest should be specified. Use taggers instead: https://skaffold.dev/docs/how-tos/taggers/", a.ImageName),
					Location: curLines,
				})...)
			}
		}
	}
	return errs
}

func validateArtifactDependencies(configs parser.SkaffoldConfigSet) (cfgErrs []ErrorWithLocation) {
	var artifacts []*latest.Artifact
	for _, c := range configs {
		artifacts = append(artifacts, c.Build.Artifacts...)
	}
	cfgErrs = append(cfgErrs, validateUniqueDependencyAliases(&configs, artifacts)...)
	cfgErrs = append(cfgErrs, validateAcyclicDependencies(&configs, artifacts)...)
	cfgErrs = append(cfgErrs, validateValidDependencyAliases(&configs, artifacts)...)
	return
}

// validateAcyclicDependencies makes sure all artifact dependencies are found and don't have cyclic references
func validateAcyclicDependencies(cfgs *parser.SkaffoldConfigSet, artifacts []*latest.Artifact) (cfgErrs []ErrorWithLocation) {
	m := make(map[string]*latest.Artifact)
	for _, artifact := range artifacts {
		m[artifact.ImageName] = artifact
	}
	visited := make(map[string]bool)
	for i, artifact := range artifacts {
		if err := dfs(artifact, visited, make(map[string]bool), m); err != nil {
			cfgErrs = append(cfgErrs, ErrorWithLocation{
				Error:    err,
				Location: cfgs.Locate(artifacts[i]),
			})
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
func validateValidDependencyAliases(cfgs *parser.SkaffoldConfigSet, artifacts []*latest.Artifact) (cfgErrs []ErrorWithLocation) {
	for i, a := range artifacts {
		if a.DockerArtifact == nil && a.CustomArtifact == nil {
			continue
		}
		for j, d := range a.Dependencies {
			if !dependencyAliasPattern.MatchString(d.Alias) {
				cfgErrs = append(cfgErrs, ErrorWithLocation{
					Error:    fmt.Errorf("invalid build dependency for artifact %q: alias %q doesn't match required pattern %q", a.ImageName, d.Alias, dependencyAliasPattern.String()),
					Location: cfgs.LocateField(artifacts[i].Dependencies[j], "Alias"),
				})
			}
		}
	}
	return
}

// validateUniqueDependencyAliases makes sure that artifact dependency aliases are unique for each artifact
func validateUniqueDependencyAliases(cfgs *parser.SkaffoldConfigSet, artifacts []*latest.Artifact) (cfgErrs []ErrorWithLocation) {
	type State int
	var (
		unseen   State = 0
		seen     State = 1
		recorded State = 2
	)
	for i, a := range artifacts {
		aliasMap := make(map[string]State)
		for j, d := range a.Dependencies {
			if aliasMap[d.Alias] == seen {
				cfgErrs = append(cfgErrs, ErrorWithLocation{
					Error:    fmt.Errorf("invalid build dependency for artifact %q: alias %q repeated", a.ImageName, d.Alias),
					Location: cfgs.LocateField(artifacts[i].Dependencies[j], "Alias"),
				})
				aliasMap[d.Alias] = recorded
			} else if aliasMap[d.Alias] == unseen {
				aliasMap[d.Alias] = seen
			}
		}
	}
	return
}

// extractContainerNameFromNetworkMode returns the container name even if it comes from an Env Var. Error if the mode isn't valid
// (only container:<id|name> format allowed)
func extractContainerNameFromNetworkMode(mode string) (string, error) {
	if strings.HasPrefix(strings.ToLower(mode), "container:") {
		// Up to this point, we know that we can strip until the colon symbol and keep the second part
		// this is helpful in case someone sends container not in lowercase
		maybeID := strings.SplitN(mode, ":", 2)[1]
		id, err := util.ExpandEnvTemplate(maybeID, map[string]string{})
		if err != nil {
			return "", sErrors.NewError(err,
				&proto.ActionableErr{
					Message: fmt.Sprintf("unable to parse container name %s: %s", mode, err),
					ErrCode: proto.StatusCode_INIT_DOCKER_NETWORK_PARSE_ERR,
					Suggestions: []*proto.Suggestion{
						{
							SuggestionCode: proto.SuggestionCode_FIX_DOCKER_NETWORK_CONTAINER_NAME,
							Action:         fmt.Sprintf("Check the content of the environment variable: %s", maybeID),
						},
					},
				})
		}
		return id, nil
	}
	errMsg := fmt.Sprintf("extracting container name from a non valid container network mode '%s'", mode)
	return "", sErrors.NewError(fmt.Errorf(errMsg),
		&proto.ActionableErr{
			Message: errMsg,
			ErrCode: proto.StatusCode_INIT_DOCKER_NETWORK_INVALID_MODE,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_FIX_DOCKER_NETWORK_MODE_WHEN_EXTRACTING_CONTAINER_NAME,
					Action:         "Only container mode allowed when calling 'extractContainerNameFromNetworkMode'",
				},
			},
		})
}

// validateDockerNetworkModeExpression makes sure that the network mode starts with "container:" followed by a valid container name
func validateDockerNetworkModeExpression(image string, expr string) error {
	id, err := extractContainerNameFromNetworkMode(expr)
	if err != nil {
		return err
	}
	return validateDockerContainerExpression(image, id)
}

// validateDockerContainerExpression makes sure that the container name pass in matches Docker's regular expression for containers
func validateDockerContainerExpression(image string, id string) error {
	containerRegExp := regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9_.-]*$")
	if !containerRegExp.MatchString(id) {
		errMsg := fmt.Sprintf("artifact %s has invalid container name '%s'", image, id)
		return sErrors.NewError(fmt.Errorf(errMsg),
			&proto.ActionableErr{
				Message: errMsg,
				ErrCode: proto.StatusCode_INIT_DOCKER_NETWORK_INVALID_CONTAINER_NAME,
				Suggestions: []*proto.Suggestion{
					{
						SuggestionCode: proto.SuggestionCode_FIX_DOCKER_NETWORK_CONTAINER_NAME,
						Action:         "Please fix the docker network container name and try again",
					},
				},
			})
	}
	return nil
}

// validateDockerNetworkMode makes sure that networkMode is one of `bridge`, `none`, `container:<name|id>`, or `host` if set.
func validateDockerNetworkMode(cfg *parser.SkaffoldConfigEntry, artifacts []*latest.Artifact) (cfgErrs []ErrorWithLocation) {
	for i, a := range artifacts {
		if a.DockerArtifact == nil || a.DockerArtifact.NetworkMode == "" {
			continue
		}
		mode := strings.ToLower(a.DockerArtifact.NetworkMode)
		if mode == "none" || mode == "bridge" || mode == "host" {
			continue
		}
		networkModeErr := validateDockerNetworkModeExpression(a.ImageName, a.DockerArtifact.NetworkMode)
		if networkModeErr == nil {
			continue
		}
		networkModeCfgErr := ErrorWithLocation{
			Error:    networkModeErr,
			Location: cfg.YAMLInfos.LocateField(cfg.Build.Artifacts[i].DockerArtifact, "NetworkMode"),
		}
		cfgErrs = append(cfgErrs, networkModeCfgErr)
	}
	return
}

// Validate that test cases exist when `verify` is called, otherwise Skaffold should error
func validateVerifyTestsExistOnVerifyCommand(runCtx *runcontext.RunContext) []error {
	var errs []error
	tcs := []*latest.VerifyTestCase{}
	for _, pipeline := range runCtx.GetPipelines() {
		tcs = append(tcs, pipeline.Verify...)
	}
	if len(tcs) == 0 && runCtx.Opts.Command == "verify" {
		errs = append(errs, fmt.Errorf("verify command expects non-zero number of test cases"))
	}
	return errs
}

// Validates that a Docker Container with a Network Mode "container:<id|name>" points to an actually running container
func validateDockerNetworkContainerExists(ctx context.Context, artifacts []*latest.Artifact, runCtx docker.Config) []error {
	var errs []error
	apiClient, err := docker.NewAPIClient(ctx, runCtx)
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	client := apiClient.RawClient()
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(5*time.Second))
	defer cancel()

	for _, a := range artifacts {
		if a.DockerArtifact == nil || a.DockerArtifact.NetworkMode == "" {
			continue
		}
		mode := strings.ToLower(a.DockerArtifact.NetworkMode)
		prefix := "container:"
		if strings.HasPrefix(mode, prefix) {
			// We've already validated the container's name in validateDockerNetworkMode.
			// We can just extract it and check whether it exists
			id, err := extractContainerNameFromNetworkMode(a.DockerArtifact.NetworkMode)
			if err != nil {
				errs = append(errs, err)
				return errs
			}
			containers, err := client.ContainerList(ctx, types.ContainerListOptions{})
			if err != nil {
				errs = append(errs, sErrors.NewError(err,
					&proto.ActionableErr{
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
				if strings.HasPrefix(c.ID, id) {
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
				&proto.ActionableErr{
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
func validateCustomDependencies(cfg *parser.SkaffoldConfigEntry, artifacts []*latest.Artifact) (cfgErrs []ErrorWithLocation) {
	for i, a := range artifacts {
		if a.CustomArtifact == nil || a.CustomArtifact.Dependencies == nil || a.CustomArtifact.Dependencies.Ignore == nil {
			continue
		}

		if a.CustomArtifact.Dependencies.Dockerfile != nil || a.CustomArtifact.Dependencies.Command != "" {
			cfgErrs = append(cfgErrs, ErrorWithLocation{
				Error:    fmt.Errorf("artifact %s has invalid dependencies; dependencies.ignore can only be used in conjunction with dependencies.paths", a.ImageName),
				Location: cfg.YAMLInfos.LocateField(cfg.Build.Artifacts[i], "ImageName"),
			})
		}
	}
	return
}

// visitStructs recursively visits all fields in the config and collects errors found by the visitor
func visitStructs(cfg *parser.SkaffoldConfigEntry, v reflect.Value, visitor func(interface{}) error) []ErrorWithLocation {
	switch v.Kind() {
	case reflect.Struct:
		var cfgErrs []ErrorWithLocation
		if err := visitor(v.Interface()); err != nil {
			var cfgErr ErrorWithLocation
			if v.CanAddr() {
				cfgErr = ErrorWithLocation{
					Error:    err,
					Location: cfg.YAMLInfos.LocateByPointer(v.Addr().Pointer()),
				}
			} else {
				log.Entry(context.TODO()).Debugf("unexpected issue - unable to get pointer to struct in visitStruct")
				cfgErr = ErrorWithLocation{
					Error:    err,
					Location: configlocations.MissingLocation(),
				}
			}
			cfgErrs = append(cfgErrs, cfgErr)
		}

		// also check all fields of the current struct
		for i := 0; i < v.Type().NumField(); i++ {
			if !v.Field(i).CanInterface() {
				continue
			}
			if fieldErrs := visitStructs(cfg, v.Field(i), visitor); fieldErrs != nil {
				cfgErrs = append(cfgErrs, fieldErrs...)
			}
		}

		return cfgErrs

	case reflect.Slice:
		// for slices check each element
		var cfgErrs []ErrorWithLocation
		for i := 0; i < v.Len(); i++ {
			if elemErrs := visitStructs(cfg, v.Index(i), visitor); elemErrs != nil {
				cfgErrs = append(cfgErrs, elemErrs...)
			}
		}
		return cfgErrs

	case reflect.Ptr:
		// for pointers check the referenced value
		if v.IsNil() {
			return nil
		}
		return visitStructs(cfg, v.Elem(), visitor)

	default:
		// other values are fine
		return nil
	}
}

// validateSyncRules checks that all manual sync rules have a valid strip prefix
func validateSyncRules(cfg *parser.SkaffoldConfigEntry, artifacts []*latest.Artifact) []ErrorWithLocation {
	var cfgErrs []ErrorWithLocation
	for i, a := range artifacts {
		if a.Sync != nil {
			for _, r := range a.Sync.Manual {
				if !strings.HasPrefix(r.Src, r.Strip) {
					err := fmt.Errorf("sync rule pattern '%s' does not have prefix '%s'", r.Src, r.Strip)
					cfgErrs = append(cfgErrs, ErrorWithLocation{
						Error:    err,
						Location: cfg.YAMLInfos.LocateField(cfg.Build.Artifacts[i], "Sync"),
					})
				}
			}
		}
	}
	return cfgErrs
}

// validatePortForwardResources checks that all user defined port forward resources
// have a valid resourceType
func validatePortForwardResources(cfg *parser.SkaffoldConfigEntry, pfrs []*latest.PortForwardResource) []ErrorWithLocation {
	var errs []ErrorWithLocation
	validResourceTypes := map[string]struct{}{
		"container":             {},
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
	for i, pfr := range pfrs {
		resourceType := strings.ToLower(string(pfr.Type))
		if _, ok := validResourceTypes[resourceType]; !ok {
			errs = append(errs, ErrorWithLocation{
				Error:    fmt.Errorf("%s is not a valid resource type for port forwarding", pfr.Type),
				Location: cfg.YAMLInfos.Locate(pfrs[i]),
			})
		}
	}
	return errs
}

// validateJibPluginTypes makes sure that jib type is one of `maven`, or `gradle` if set.
func validateJibPluginTypes(cfg *parser.SkaffoldConfigEntry, artifacts []*latest.Artifact) (cfgErrs []ErrorWithLocation) {
	for i, a := range artifacts {
		if a.JibArtifact == nil || a.JibArtifact.Type == "" {
			continue
		}
		t := strings.ToLower(a.JibArtifact.Type)
		if t == "maven" || t == "gradle" {
			continue
		}
		cfgErrs = append(cfgErrs, ErrorWithLocation{
			Error:    fmt.Errorf("artifact %s has invalid Jib plugin type '%s'", a.ImageName, t),
			Location: cfg.YAMLInfos.LocateField(cfg.Build.Artifacts[i].JibArtifact, "Type"),
		})
	}
	return
}

// validateKoSync ensures that infer sync patterns contain the `kodata` string, since infer sync for the ko builder only supports static assets.
func validateKoSync(cfg *parser.SkaffoldConfigEntry, artifacts []*latest.Artifact) []ErrorWithLocation {
	var cfgErrs []ErrorWithLocation
	for i, a := range artifacts {
		if a.KoArtifact == nil || a.Sync == nil {
			continue
		}
		if len(a.Sync.Infer) > 0 && strings.Contains(a.KoArtifact.Main, "...") {
			cfgErrs = append(cfgErrs, ErrorWithLocation{
				Error:    fmt.Errorf("artifact %s cannot use inferred file sync when the ko.main field contains the '...' wildcard. Instead, specify the path to the main package without using wildcards", a.ImageName),
				Location: cfg.YAMLInfos.LocateField(cfg.Build.Artifacts[i].KoArtifact, "Main"),
			})
		}
		for _, pattern := range a.Sync.Infer {
			if !strings.Contains(pattern, "kodata") {
				cfgErrs = append(cfgErrs, ErrorWithLocation{
					Error:    fmt.Errorf("artifact %s has an invalid pattern %s for inferred file sync with the ko builder. The pattern must specify the 'kodata' directory. For instance, if you want to sync all static content, and your main package is in the workspace directory, you can use the pattern 'kodata/**/*'", a.ImageName, pattern),
					Location: cfg.YAMLInfos.LocateField(cfg.Build.Artifacts[i].Sync, "Infer"),
				})
			}
		}
	}
	return cfgErrs
}

// validateArtifactTypes checks that the artifact types are compatible with the specified builder.
func validateArtifactTypes(cfg *parser.SkaffoldConfigEntry, bc latest.BuildConfig) []ErrorWithLocation {
	cfgErrs := []ErrorWithLocation{}
	switch {
	case bc.LocalBuild != nil:
		for i, a := range bc.Artifacts {
			if misc.ArtifactType(a) == misc.Kaniko {
				cfgErrs = append(cfgErrs, ErrorWithLocation{
					Error:    fmt.Errorf("found a '%s' artifact, which is incompatible with the 'local' builder:\n\n%s\n\nTo use the '%s' builder, add the 'cluster' stanza to the 'build' section of your configuration. For information, see https://skaffold.dev/docs/pipeline-stages/builders/", misc.ArtifactType(a), misc.FormatArtifact(a), misc.ArtifactType(a)),
					Location: cfg.YAMLInfos.Locate(&cfg.Build.Artifacts[i].ArtifactType),
				})
			}
		}
	case bc.GoogleCloudBuild != nil:
		for i, a := range bc.Artifacts {
			at := misc.ArtifactType(a)
			if at != misc.Kaniko && at != misc.Docker && at != misc.Jib && at != misc.Buildpack && at != misc.Ko {
				cfgErrs = append(cfgErrs, ErrorWithLocation{
					Error:    fmt.Errorf("found a '%s' artifact, which is incompatible with the 'gcb' builder:\n\n%s\n\nTo use the '%s' builder, remove the 'googleCloudBuild' stanza from the 'build' section of your configuration. For information, see https://skaffold.dev/docs/pipeline-stages/builders/", misc.ArtifactType(a), misc.FormatArtifact(a), misc.ArtifactType(a)),
					Location: cfg.YAMLInfos.Locate(&cfg.Build.Artifacts[i].ArtifactType),
				})
			}
		}
	case bc.Cluster != nil:
		for i, a := range bc.Artifacts {
			if misc.ArtifactType(a) != misc.Kaniko && misc.ArtifactType(a) != misc.Custom {
				cfgErrs = append(cfgErrs, ErrorWithLocation{
					Error:    fmt.Errorf("found a '%s' artifact, which is incompatible with the 'cluster' builder:\n\n%s\n\nTo use the '%s' builder, remove the 'cluster' stanza from the 'build' section of your configuration. For information, see https://skaffold.dev/docs/pipeline-stages/builders/", misc.ArtifactType(a), misc.FormatArtifact(a), misc.ArtifactType(a)),
					Location: cfg.YAMLInfos.Locate(&cfg.Build.Artifacts[i].ArtifactType),
				})
			}
		}
	}
	return cfgErrs
}

// validateGCBConfig checks if GCB config is valid.
func validateGCBConfig(cfg *parser.SkaffoldConfigEntry, bc latest.BuildConfig) (cfgErrs []ErrorWithLocation) {
	if bc.GoogleCloudBuild != nil && bc.GoogleCloudBuild.WorkerPool != "" {
		if !gcbWorkerPoolPattern.MatchString(bc.GoogleCloudBuild.WorkerPool) {
			cfgErrs = append(cfgErrs, ErrorWithLocation{
				Error:    fmt.Errorf("invalid value for worker pool. Must match pattern projects/{project}/locations/{location}/workerPools/{worker_pool}"),
				Location: cfg.YAMLInfos.Locate(&cfg.Build.GoogleCloudBuild.WorkerPool),
			})
		}
	}
	return cfgErrs
}

// validateLogPrefix checks that logs are configured with a valid prefix.
func validateLogPrefix(cfg *parser.SkaffoldConfigEntry, lc latest.LogsConfig) []ErrorWithLocation {
	validPrefixes := []string{"", "auto", "container", "podAndContainer", "none"}

	if !stringslice.Contains(validPrefixes, lc.Prefix) {
		return []ErrorWithLocation{
			{
				Error:    fmt.Errorf("invalid log prefix '%s'. Valid values are 'auto', 'container', 'podAndContainer' or 'none'", lc.Prefix),
				Location: cfg.YAMLInfos.Locate(&cfg.Deploy.Logs),
			},
		}
	}

	return nil
}

// validateVerifyTests
// - makes sure that each test name is unique
// - makes sure that each container name is unique
func validateVerifyTests(runCtx *runcontext.RunContext) []error {
	var errs []error
	seenTestName := map[string]bool{}
	seenContainerName := map[string]bool{}
	tcs := []*latest.VerifyTestCase{}
	for _, pipeline := range runCtx.GetPipelines() {
		tcs = append(tcs, pipeline.Verify...)
	}
	for _, tc := range tcs {
		if _, ok := seenTestName[tc.Name]; ok {
			errs = append(errs, fmt.Errorf("found duplicate test name '%s' in 'verify' test cases. 'verify' test case names must be unique", tc.Name))
		}
		if _, ok := seenContainerName[tc.Container.Name]; ok {
			errs = append(errs, fmt.Errorf("found duplicate container name '%s' in 'verify' test cases. 'verify' container names must be unique", tc.Container.Name))
		}
		seenTestName[tc.Name] = true
		seenContainerName[tc.Container.Name] = true
	}
	return errs
}

// validateCustomTest
// - makes sure that command is not empty
// - makes sure that dependencies.ignore is only used in conjunction with dependencies.paths
func validateCustomTest(cfg *parser.SkaffoldConfigEntry, tcs []*latest.TestCase) (cfgErrs []ErrorWithLocation) {
	for i, tc := range tcs {
		for j, ct := range tc.CustomTests {
			if ct.Command == "" {
				cfgErrs = append(cfgErrs, ErrorWithLocation{
					Error:    fmt.Errorf("custom test command must not be empty;"),
					Location: cfg.YAMLInfos.Locate(&cfg.Test[i].CustomTests[j]),
				})
				return
			}

			if ct.Dependencies == nil {
				continue
			}
			if ct.Dependencies.Command != "" && ct.Dependencies.Paths != nil {
				cfgErrs = append(cfgErrs, ErrorWithLocation{
					Error:    fmt.Errorf("dependencies can use either command or paths, but not both"),
					Location: cfg.YAMLInfos.Locate(&cfg.Test[i].CustomTests[j]),
				})
			}
			if ct.Dependencies.Paths == nil && ct.Dependencies.Ignore != nil {
				cfgErrs = append(cfgErrs, ErrorWithLocation{
					Error:    fmt.Errorf("customTest has invalid dependencies; dependencies.ignore can only be used in conjunction with dependencies.paths"),
					Location: cfg.YAMLInfos.Locate(&cfg.Test[i].CustomTests[j]),
				})
			}
		}
	}
	return
}

func wrapWithContext(config *parser.SkaffoldConfigEntry, errs ...ErrorWithLocation) []ErrorWithLocation {
	var id string
	if config.Metadata.Name != "" {
		id = fmt.Sprintf("in module %q", config.Metadata.Name)
	} else {
		id = fmt.Sprintf("in unnamed config[%d]", config.SourceIndex)
	}

	for i := range errs {
		if errs[i].Location == nil || errs[i].Location.StartLine == -1 {
			errs[i].Error = errors.Wrapf(errs[i].Error, "source: %s, %s", config.SourceFile, id)
			continue
		}
		errs[i].Error = errors.Wrapf(errs[i].Error, "source: %s, %s on line %d column %d",
			config.SourceFile, id, errs[i].Location.StartLine, errs[i].Location.StartColumn)
	}
	return errs
}

// validateKubectlManifests
// - validates that kubectl manifest files specified in the skaffold config exist
func validateKubectlManifests(configs parser.SkaffoldConfigSet) (errs []ErrorWithLocation) {
	for _, c := range configs {
		if c.IsRemote {
			continue
		}
		if len(c.Render.RawK8s) == 1 && c.Render.RawK8s[0] == constants.DefaultKubectlManifests[0] {
			log.Entry(context.TODO()).Debug("skipping validating `kubectl` deployer manifests since only the default manifest list is defined")
			continue
		}

		// validate that manifest files referenced in config exist
		for _, pattern := range c.Render.RawK8s {
			if util.IsURL(pattern) {
				continue
			}
			// filepaths are all absolute from config parsing step via tags.MakeFilePathsAbsolute
			expanded, err := filepath.Glob(pattern)
			if err != nil {
				errs = append(errs, ErrorWithLocation{
					Error: err,
				})
			}
			if len(expanded) == 0 {
				// TODO(aaron-prindle) currently this references the whole manifest list and not the specific entry
				// this is related to the fact that string pointers do not work with the current setup, need to get the closest struct
				// TODO(aaron-prindle) parse the manifest node to extract exact correct line # for the value here (currently it is the parent obj)
				msg := fmt.Sprintf("Manifest file %q referenced in skaffold config could not be found", pattern)
				errMsg := wrapWithContext(c, ErrorWithLocation{
					Error:    fmt.Errorf(msg),
					Location: c.YAMLInfos.Locate(&c.Render.RawK8s),
				})
				errs = append(errs, ErrorWithLocation{
					Error: sErrors.NewError(errMsg[0].Error,
						&proto.ActionableErr{
							Message: errMsg[0].Error.Error(),
							ErrCode: proto.StatusCode_CONFIG_MISSING_MANIFEST_FILE_ERR,
							Suggestions: []*proto.Suggestion{
								{
									SuggestionCode: proto.SuggestionCode_CONFIG_FIX_MISSING_MANIFEST_FILE,
									Action:         fmt.Sprintf("Verify that file %q referenced in config %q exists and the path and naming are correct", pattern, c.SourceFile),
								},
							},
						}),
					Location: errMsg[0].Location,
				})
			}
		}
	}
	return errs
}

func validateLocationSetForCloudRun(rCtx *runcontext.RunContext) []error {
	if !requiresCloudRun(rCtx) {
		// if the current command doesn't require connecting to Cloud Run, a location isn't needed.
		return nil
	}
	runDeployer := false
	hasLocation := false
	if rCtx.Opts.CloudRunLocation != "" {
		hasLocation = true
	}
	if rCtx.Opts.CloudRunProject != "" {
		runDeployer = true
	} else {
		for _, deployer := range rCtx.Pipelines.Deployers() {
			if deployer.CloudRunDeploy != nil {
				runDeployer = true
				if deployer.CloudRunDeploy.Region != "" {
					hasLocation = true
				}
			}
		}
	}
	if runDeployer && !hasLocation {
		return []error{sErrors.NewError(fmt.Errorf("location must be specified with Cloud Run Deployer"),
			&proto.ActionableErr{
				Message: "Cloud Run Location is not specified",
				ErrCode: proto.StatusCode_INIT_CLOUD_RUN_LOCATION_ERROR,
				Suggestions: []*proto.Suggestion{
					{
						SuggestionCode: proto.SuggestionCode_SPECIFY_CLOUD_RUN_LOCATION,
						Action: "Specify a Cloud Run location via the deploy.cloudrun.region field in skaffold.yaml " +
							"or the --cloud-run-location flag",
					},
				},
			}),
		}
	}
	return nil
}

// requiresCloudRun returns true if the current command needs to connect to a Cloud Run regional endpoint.
func requiresCloudRun(rCtx *runcontext.RunContext) bool {
	runCommands := map[string]bool{
		"run":    true,
		"deploy": true,
		"debug":  true,
		"dev":    true,
		"delete": true,
		"apply":  true,
	}
	_, ok := runCommands[rCtx.Opts.Command]
	return ok
}
