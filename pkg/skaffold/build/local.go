/*
Copyright 2018 Google LLC

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

package build

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/constants"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// LocalBuilder uses the host docker daemon to build and tag the image
type LocalBuilder struct {
	*config.BuildConfig

	api          docker.DockerAPIClient
	localCluster bool
	kubeContext  string
}

// NewLocalBuilder returns an new instance of a LocalBuilder
func NewLocalBuilder(cfg *config.BuildConfig, kubeContext string) (*LocalBuilder, error) {
	api, err := docker.NewDockerAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "getting docker client")
	}

	l := &LocalBuilder{
		BuildConfig: cfg,

		kubeContext:  kubeContext,
		api:          api,
		localCluster: kubeContext == constants.DefaultMinikubeContext || kubeContext == constants.DefaultDockerForDesktopContext,
	}

	if cfg.LocalBuild.SkipPush == nil {
		logrus.Debugf("skipPush value not present. defaulting to cluster default %t (minikube=true, d4d=true, gke=false)", l.localCluster)
		cfg.LocalBuild.SkipPush = &l.localCluster
	}

	return l, nil
}

func (l *LocalBuilder) runBuildForArtifact(ctx context.Context, out io.Writer, artifact *config.Artifact) (string, error) {
	if artifact.DockerArtifact != nil {
		return l.buildDocker(ctx, out, artifact)
	}
	if artifact.BazelArtifact != nil {
		return l.buildBazel(ctx, out, artifact)
	}
	artifact.DockerArtifact = config.DefaultDockerArtifact
	return l.buildDocker(ctx, out, artifact)
}

// Build runs a docker build on the host and tags the resulting image with
// its checksum. It streams build progress to the writer argument.
func (l *LocalBuilder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*config.Artifact) (*BuildResult, error) {
	if l.localCluster {
		if _, err := fmt.Fprintf(out, "Found [%s] context, using local docker daemon.\n", l.kubeContext); err != nil {
			return nil, errors.Wrap(err, "writing status")
		}
	}
	defer l.api.Close()
	res := &BuildResult{
		Builds: []Build{},
	}
	for _, artifact := range artifacts {
		initialTag, err := l.runBuildForArtifact(ctx, out, artifact)
		if err != nil {
			return nil, errors.Wrap(err, "running build for artifact")
		}

		digest, err := docker.Digest(ctx, l.api, initialTag)
		if err != nil {
			return nil, errors.Wrapf(err, "build and tag: %s", initialTag)
		}
		if digest == "" {
			return nil, fmt.Errorf("digest not found")
		}
		tag, err := tagger.GenerateFullyQualifiedImageName(".", &tag.TagOptions{
			ImageName: artifact.ImageName,
			Digest:    digest,
		})
		if err != nil {
			return nil, errors.Wrap(err, "generating tag")
		}
		if err := l.api.ImageTag(ctx, initialTag, tag); err != nil {
			return nil, errors.Wrap(err, "tagging image")
		}
		if _, err := io.WriteString(out, fmt.Sprintf("Successfully tagged %s\n", tag)); err != nil {
			return nil, errors.Wrap(err, "writing tag status")
		}
		if !*l.LocalBuild.SkipPush {
			if err := docker.RunPush(ctx, l.api, tag, out); err != nil {
				return nil, errors.Wrap(err, "running push")
			}
		}

		res.Builds = append(res.Builds, Build{
			ImageName: artifact.ImageName,
			Tag:       tag,
			Artifact:  artifact,
		})
	}

	return res, nil
}

func (l *LocalBuilder) buildBazel(ctx context.Context, out io.Writer, a *config.Artifact) (string, error) {
	cmd := exec.Command("bazel", "build", a.BazelArtifact.BuildTarget)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return "", errors.Wrap(err, "running command")
	}
	//TODO(r2d4): strip off leading //:, bad
	tarPath := strings.TrimPrefix(a.BazelArtifact.BuildTarget, "//:")
	//TODO(r2d4): strip off trailing .tar, even worse
	imageTag := strings.TrimSuffix(tarPath, ".tar")
	imageTar, err := os.Open(filepath.Join("bazel-bin", tarPath))
	if err != nil {
		return "", errors.Wrap(err, "opening image tarball")
	}
	defer imageTar.Close()
	resp, err := l.api.ImageLoad(ctx, imageTar, false)
	if err != nil {
		return "", errors.Wrap(err, "loading image into docker daemon")
	}
	defer resp.Body.Close()
	respStr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "reading from image load response")
	}
	out.Write(respStr)

	return fmt.Sprintf("bazel:%s", imageTag), nil
}

func (l *LocalBuilder) buildDocker(ctx context.Context, out io.Writer, a *config.Artifact) (string, error) {
	if a.DockerArtifact.DockerfilePath == "" {
		a.DockerArtifact.DockerfilePath = constants.DefaultDockerfilePath
	}
	initialTag := util.RandomID()
	err := docker.RunBuild(ctx, l.api, &docker.BuildOptions{
		ImageName:   initialTag,
		Dockerfile:  a.DockerArtifact.DockerfilePath,
		ContextDir:  a.Workspace,
		ProgressBuf: out,
		BuildBuf:    out,
		BuildArgs:   a.DockerArtifact.BuildArgs,
	})
	if err != nil {
		return "", errors.Wrap(err, "running build")
	}
	return fmt.Sprintf("%s:latest", initialTag), nil
}
