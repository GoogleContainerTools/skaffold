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

package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringslice"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/warnings"
)

func (b *Builder) SupportedPlatforms() platform.Matcher {
	return platform.All
}

func (b *Builder) Build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string, matcher platform.Matcher) (string, error) {
	var pls []v1.Platform
	if len(matcher.Platforms) > 0 {
		for _, plat := range matcher.Platforms {
			pls = append(pls, util.ConvertToV1Platform(plat))
		}
	} else {
		pls = append(pls, v1.Platform{})
	}
	a = b.adjustCache(ctx, a, tag)
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"BuildType":   "docker",
		"Context":     instrumentation.PII(a.Workspace),
		"Destination": instrumentation.PII(tag),
	})

	// Fail fast if the Dockerfile can't be found.
	dockerfile, err := docker.NormalizeDockerfilePath(a.Workspace, a.DockerArtifact.DockerfilePath)
	if err != nil {
		return "", dockerfileNotFound(fmt.Errorf("normalizing dockerfile path: %w", err), a.ImageName)
	}
	if _, err := os.Stat(dockerfile); os.IsNotExist(err) {
		return "", dockerfileNotFound(err, a.ImageName)
	}

	for _, pl := range pls {
		if err := b.pullCacheFromImages(ctx, out, a.ArtifactType.DockerArtifact, pl); err != nil {
			return "", cacheFromPullErr(err, a.ImageName)
		}
	}
	opts := docker.BuildOptions{Tag: tag, Mode: b.cfg.Mode(), ExtraBuildArgs: docker.ResolveDependencyImages(a.Dependencies, b.artifacts, true)}

	var imageID string

	// ignore useCLI boolean if buildkit is enabled since buildkit is only implemented for docker CLI at the moment in skaffold.
	// we might consider a different approach in the future.
	// use CLI for cross-platform builds
	if b.useCLI || (b.useBuildKit != nil && *b.useBuildKit) || len(a.DockerArtifact.CliFlags) > 0 || matcher.IsCrossPlatform() {
		imageID, err = b.dockerCLIBuild(ctx, output.GetUnderlyingWriter(out), a.ImageName, a.Workspace, dockerfile, a.ArtifactType.DockerArtifact, opts, pls)
	} else {
		imageID, err = b.localDocker.Build(ctx, out, a.Workspace, a.ImageName, a.ArtifactType.DockerArtifact, opts)
	}

	if err != nil {
		return "", newBuildError(err, b.cfg)
	}

	if !b.useCLI && b.pushImages && !b.buildx {
		// TODO (tejaldesai) Remove https://github.com/GoogleContainerTools/skaffold/blob/main/pkg/skaffold/errors/err_map.go#L56
		// and instead define a pushErr() method here.
		return b.localDocker.Push(ctx, out, tag)
	}

	return imageID, nil
}

func (b *Builder) dockerCLIBuild(ctx context.Context, out io.Writer, name string, workspace string, dockerfilePath string, a *latest.DockerArtifact, opts docker.BuildOptions, pls []v1.Platform) (string, error) {
	args := []string{"build", workspace, "--file", dockerfilePath, "-t", opts.Tag}
	imageInfoEnv, err := docker.EnvTags(opts.Tag)
	if err != nil {
		return "", fmt.Errorf("couldn't parse image tag: %w", err)
	}
	ba, err := docker.EvalBuildArgsWithEnv(b.cfg.Mode(), workspace, a.DockerfilePath, a.BuildArgs, opts.ExtraBuildArgs, imageInfoEnv)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate build args: %w", err)
	}
	cliArgs, err := docker.ToCLIBuildArgs(a, ba, imageInfoEnv)
	if err != nil {
		return "", fmt.Errorf("getting docker build args: %w", err)
	}
	args = append(args, cliArgs...)

	if b.cfg.Prune() {
		args = append(args, "--force-rm")
	}

	var platforms []string
	for _, pl := range pls {
		if pl.String() != "" {
			platforms = append(platforms, pl.String())
		}
	}
	if len(platforms) > 0 {
		args = append(args, "--platform", strings.Join(platforms, ","))
	}

	if b.useBuildKit != nil && *b.useBuildKit {
		if !b.pushImages {
			load := true
			if b.buildx {
				// if docker daemon is not used, do not try to load the image (a buildx warning will be logged)
				_, err := b.localDocker.ServerVersion(ctx)
				load = err == nil
			}
			if load {
				args = append(args, "--load")
			}
		} else if b.buildx {
			// with buildx, push the image directly to the registry (not using the docker daemon)
			args = append(args, "--push")
		}
	}

	if b.buildx {
		args = append(args, "--builder", config.GetBuildXBuilder(b.cfg.GlobalConfig()))
	}

	// temporary file for buildx metadata containing the image digest:
	var metadata string
	if b.buildx {
		metadata, err = getBuildxMetadataFile()
		if err != nil {
			return "", fmt.Errorf("unable to create temp file: %w", err)
		}
		defer os.Remove(metadata)
		args = append(args, "--metadata-file", metadata)
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = append(util.OSEnviron(), b.localDocker.ExtraEnv()...)
	if b.useBuildKit != nil {
		if *b.useBuildKit {
			cmd.Env = append(cmd.Env, "DOCKER_BUILDKIT=1")
		} else {
			cmd.Env = append(cmd.Env, "DOCKER_BUILDKIT=0")
		}
	} else if len(platforms) > 0 { // cross-platform builds require buildkit
		log.Entry(ctx).Debugf("setting DOCKER_BUILDKIT=1 for docker build for artifact %q since it targets platform %q", name, platforms[0])
		cmd.Env = append(cmd.Env, "DOCKER_BUILDKIT=1")
	}
	if len(platforms) > 1 && b.buildx {
		// avoid "unknown/unknown" architecture/OS caused by buildx default image attestation
		log.Entry(ctx).Warnf("setting BUILDX_NO_DEFAULT_ATTESTATIONS=1 for docker buildx for artifact %q since it targets platform %q to avoid unknown/unknown platform issue", name, platforms[0])
		cmd.Env = append(cmd.Env, "BUILDX_NO_DEFAULT_ATTESTATIONS=1")
	}
	cmd.Stdout = out

	var errBuffer bytes.Buffer
	stderr := io.MultiWriter(out, &errBuffer)
	cmd.Stderr = stderr

	if b.buildx {
		log.Entry(ctx).Infof("Running buildx command: %s", cmd.Args)
	}

	if err := util.RunCmd(ctx, cmd); err != nil {
		if !b.buildx {
			err = tryExecFormatErr(fmt.Errorf("running build: %w", err), errBuffer)
		} else {
			err = tryExecFormatErrBuildX(fmt.Errorf("running build: %w", err), errBuffer)
		}
		return "", err
	}

	if !b.buildx {
		return b.localDocker.ImageID(ctx, opts.Tag)
	} else {
		return parseBuildxMetadataFile(ctx, metadata)
	}
}

func (b *Builder) pullCacheFromImages(ctx context.Context, out io.Writer, a *latest.DockerArtifact, pl v1.Platform) error {
	// when using buildx, avoid pulling as the builder not necessarily uses the local docker daemon
	if len(a.CacheFrom) == 0 || b.buildx {
		return nil
	}

	for _, image := range a.CacheFrom {
		imageID, err := b.localDocker.ImageID(ctx, image)
		if err != nil {
			return fmt.Errorf("getting imageID for %q: %w", image, err)
		}
		if imageID != "" {
			// already pulled
			continue
		}

		if err := b.localDocker.Pull(ctx, out, image, pl); err != nil {
			warnings.Printf("cacheFrom image %q couldn't be pulled for platform %q\n", image, pl)
		}
	}

	return nil
}

// adjustCache returns an artifact where any cache references from the artifactImage is changed to the tagged built image name instead.
// Under buildx, templated cache refs will be evaluated (with image rewriting using the default cache repo and tag)
// if no cacheTo was specified in the skaffold yaml, it will add a tagged destination using the same cache source reference
func (b *Builder) adjustCache(ctx context.Context, a *latest.Artifact, artifactTag string) *latest.Artifact {
	if os.Getenv("SKAFFOLD_DISABLE_DOCKER_CACHE_ADJUSTMENT") != "" {
		// allow this behaviour to be disabled
		return a
	}
	if !stringslice.Contains(a.DockerArtifact.CacheFrom, a.ImageName) && !b.buildx {
		return a
	}

	cf := make([]string, 0, len(a.DockerArtifact.CacheFrom))

	for _, image := range a.DockerArtifact.CacheFrom {
		cacheRef := artifactTag // full reference to be used (backward compatibility)
		if b.buildx {
			// change cache reference to to the tagged image name (built or given, including registry)
			cacheRef = b.computeCacheRefTag(ctx, artifactTag, image)
			log.Entry(ctx).Debugf("Adjusting cache source image ref: %s to %s", image, cacheRef)
		}
		cf = append(cf, cacheRef)
	}

	// Create a new copy of CacheTo to modify destinations
	ct := make([]string, max(len(a.DockerArtifact.CacheTo), 1))

	// only process cache destination if cache flags are set (to avoid failures on cache push)
	cacheFlags, _ := config.GetCacheFlags(b.cfg.GlobalConfig())
	if len(cacheFlags) > 0 {
		if len(a.DockerArtifact.CacheTo) > 0 {
			copy(ct, a.DockerArtifact.CacheTo)
		} else if len(a.DockerArtifact.CacheFrom) > 0 {
			log.Entry(ctx).Infof("Using first cache source as destination: %s", a.DockerArtifact.CacheFrom[0])
			ct[0] = a.DockerArtifact.CacheFrom[0]
		}
		for i, image := range ct {
			if b.buildx && b.pushImages {
				cacheRef := b.computeCacheRefTag(ctx, artifactTag, image)
				log.Entry(ctx).Debugf("Adjusting cache destination image ref: %s to %s", image, cacheRef)
				// append a new cache flag with the ref destination:
				cacheFlags = append(cacheFlags, fmt.Sprintf("ref=%s", cacheRef))
				// format the flags (comma separated) and it to the new cache-to array
				ct[i] = strings.Join(cacheFlags, ",")
			}
		}
	} else {
		log.Entry(ctx).Infof("No cache flags set, skipping cache destination adjustment")
		ct = nil
	}
	copy := *a
	copy.DockerArtifact.CacheFrom = cf
	copy.DockerArtifact.CacheTo = ct
	return &copy
}

func (b *Builder) computeCacheRefTag(ctx context.Context, artifactTag string, cacheRef string) string {
	multiLevel, err := config.GetMultiLevelRepo(b.cfg.GlobalConfig())
	if err != nil {
		log.Entry(ctx).Errorf("Getting multi-level repo support: %v", err)
	}
	cacheRepo, err := config.GetCacheRepo(b.cfg.GlobalConfig())
	if err != nil {
		log.Entry(ctx).Errorf("Getting cache-repo %q: %v", cacheRepo, err)
	}
	// build image env and expand cacheTag template:
	imageInfoEnv, err := docker.EnvTags(artifactTag)
	if err != nil {
		log.Entry(ctx).Errorf("Couldn't build env tags: %v", err)
	}
	cacheTag, _ := config.GetCacheTag(b.cfg.GlobalConfig())
	imageInfoEnv["CACHE_TAG"], err = util.ExpandEnvTemplate(cacheTag, imageInfoEnv)
	if err != nil {
		log.Entry(ctx).Errorf("Couldn't expand default cache-tag: %v", err)
	}
	if b.buildx {
		// compute the full cache reference (including registry, preserving tag)
		log.Entry(ctx).Tracef("Expanding cache ref env template: %s", cacheRef)
		cacheRef, err = util.ExpandEnvTemplate(cacheRef, imageInfoEnv)
		if err != nil {
			log.Entry(ctx).Errorf("Couldn't expand cache image tag: %v", err)
		}
		log.Entry(ctx).Tracef("Parsing expanded cache ref: %s", cacheRef)
		imgRef, err := docker.ParseReference(cacheRef)
		if err != nil {
			log.Entry(ctx).Errorf("Couldn't parse image tag %s: %v", cacheRef, err)
		} else {
			// determine the cache tag (use explicit tag, expanded cache-tag or fallback to image tag)
			tag := ""
			switch {
			case imgRef.Tag != "":
				tag = imgRef.Tag
			case cacheTag == "":
				tag = imageInfoEnv["IMAGE_TAG"]
			default:
				tag = imageInfoEnv["CACHE_TAG"]
			}
			if tag == "" {
				log.Entry(ctx).Errorf("Invalid empty computed cache-tag")
			}
			log.Entry(ctx).Tracef("Rewrite cache image base name: %s and tag: %s", imgRef.BaseName, tag)
			cacheRef = fmt.Sprintf("%s:%s", imgRef.BaseName, tag)
			// sustitute the cache repository (registry):
			ref, err := docker.SubstituteDefaultRepoIntoImage(cacheRepo, multiLevel, cacheRef)
			if err != nil {
				log.Entry(ctx).Errorf("Applying cache default repo failed for '%s': %v", cacheRef, err)
			}
			cacheRef = ref
		}
	}
	log.Entry(ctx).Tracef("Computed cache ref: %s", cacheRef)
	return cacheRef
}

// osCreateTemp allows for replacing metadata for testing purposes
var osCreateTemp = os.CreateTemp

func getBuildxMetadataFile() (string, error) {
	metadata, err := osCreateTemp("", "metadata*.json")
	if err != nil {
		return "", err
	}
	metadata.Close()
	return metadata.Name(), nil
}

func parseBuildxMetadataFile(ctx context.Context, filename string) (string, error) {
	var metadata map[string]interface{}
	data, err := os.ReadFile(filename)
	if err == nil {
		err = json.Unmarshal(data, &metadata)
	}
	if err == nil {
		// avoid panic: interface conversion: interface {} is nil, not string (if keys don't exists)
		var digest string
		if value := metadata["containerimage.config.digest"]; value != nil {
			// image loaded to local docker daemon
			digest = value.(string)
		} else if value := metadata["containerimage.digest"]; value != nil {
			// image pushed to registry
			digest = value.(string)
		}
		var name string
		if value := metadata["image.name"]; value != nil {
			name = value.(string)
		}
		if digest != "" {
			log.Entry(ctx).Debugf("Image digest found in buildx metadata: %s for %s", digest, name)
			return digest, nil
		}
	}
	log.Entry(ctx).Warnf("No digest found in buildx metadata: %v", err)
	// if image is not pushed, it could not contain the digest log for debugging:
	log.Entry(ctx).Debugf("Full buildx metadata: %s", data)
	return "", err
}
