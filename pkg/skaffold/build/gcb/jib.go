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

package gcb

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	cloudbuild "google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func (b *Builder) jibBuildSpec(artifact *latest.Artifact, tag string) (cloudbuild.Build, error) {
	t, err := jib.DeterminePluginType(artifact.Workspace, artifact.JibArtifact)
	if err != nil {
		return cloudbuild.Build{}, err
	}

	switch t {
	case jib.JibMaven:
		return cloudbuild.Build{
			Steps: []*cloudbuild.BuildStep{{
				Name:       b.MavenImage,
				Entrypoint: "sh",
				Args:       fixHome("mvn", jib.GenerateMavenBuildArgs("build", tag, artifact.JibArtifact, b.skipTests, b.insecureRegistries)),
			}},
		}, nil
	case jib.JibGradle:
		return cloudbuild.Build{
			Steps: []*cloudbuild.BuildStep{{
				Name:       b.GradleImage,
				Entrypoint: "sh",
				Args:       fixHome("gradle", jib.GenerateGradleBuildArgs("jib", tag, artifact.JibArtifact, b.skipTests, b.insecureRegistries)),
			}},
		}, nil
	default:
		return cloudbuild.Build{}, errors.New("skaffold can't determine Jib artifact type for Google Cloud Build")
	}
}

func fixHome(command string, args []string) []string {
	return []string{"-c", command + " -Duser.home=$$HOME " + strings.Join(args, " ")}
}

func jibAddWorkspaceToDependencies(workspace string, dependencies []string) ([]string, error) {
	dependencyMap := make(map[string]bool)
	for _, d := range dependencies {
		dependencyMap[d] = true
	}

	err := filepath.Walk(workspace,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if info.Name() == "target" {
					if util.IsFile(filepath.Join(filepath.Dir(path), "pom.xml")) {
						return filepath.SkipDir
					}
				} else if info.Name() == "build" {
					if util.IsFile(filepath.Join(filepath.Dir(path), "build.gradle")) {
						return filepath.SkipDir
					}
				}
			}
			if _, ok := dependencyMap[path]; !ok {
				dependencies = append(dependencies, path)
			}
			return nil
		})
	return dependencies, err
}
