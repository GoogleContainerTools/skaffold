package podman

import (
	"context"
	"fmt"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/letsencrypt/boulder/errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (b *Builder) Build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string, matcher platform.Matcher) (string, error) {

	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"BuildType":   "podman",
		"Context":     instrumentation.PII(a.Workspace),
		"Destination": instrumentation.PII(tag),
	})

	// Fail fast if the Dockerfile can't be found.
	buildfile, err := getCleanBuildFilePath(a.Workspace, a.PodmanArtifact.ContainerfilePath)
	if err != nil {
		return "", nil
	}
	cmd := exec.CommandContext(ctx, "podman")
	cmd.Args = append(cmd.Args, "build")
	cmd.Args = append(cmd.Args, "-t", tag)
	cmd.Args = append(cmd.Args, "-f", buildfile)
	for _, pl := range matcher.Platforms {
		cmd.Args = append(cmd.Args, "--platform", fmt.Sprintf("%s/%s/%s", pl.OS, pl.Architecture, pl.Variant))
	}
	err = util.RunCmd(ctx, cmd)
	if err != nil {
		return "", err
	}
	if b.pushImages {
		if err := pushImage(ctx, tag); err != nil {
			return "", err
		}
		return crane.Digest(tag)
	}

	//return getImageID(ctx, tag)

	return b.loadToDocker(ctx, out, tag)
	// "podman images tag -q --no-trunc" to get imageID
	// "podman save image-archive xxx.tar" to save image to tar
	// may not use imageID as tag for remote podman server or windows/mac local machines as

	//if _, err := os.Stat(dockerfile); os.IsNotExist(err) {
	//	return "", dockerfileNotFound(err, a.ImageName)
	//
	//
	//if err := b.pullCacheFromImages(ctx, out, a.ArtifactType.DockerArtifact, pl); err != nil {
	//	return "", cacheFromPullErr(err, a.ImageName)
	//}
	//opts := docker.BuildOptions{Tag: tag, Mode: b.cfg.Mode(), ExtraBuildArgs: docker.ResolveDependencyImages(a.Dependencies, b.artifacts, true)}

	//if b.pushImages {
	//	// TODO (tejaldesai) Remove https://github.com/GoogleContainerTools/skaffold/blob/main/pkg/skaffold/errors/err_map.go#L56
	//	// and instead define a pushErr() method here.
	//	return b.localDocker.Push(ctx, out, tag)
	//}
	//

}

func (b *Builder) loadToDocker(ctx context.Context, out io.Writer, tag string) (string, error) {
	cmd := exec.CommandContext(ctx, "podman")
	cmd.Args = append(cmd.Args, "save", tag)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	fmt.Println(cmd.Args)
	err = cmd.Start()
	if err != nil {
		return "", err
	}
	if _, err = b.localDocker.RawClient().ImageLoad(ctx, r, false); err != nil {
		return "", err
	}
	return getImageID(ctx, tag)
}

func pushImage(ctx context.Context, tag string) error {

	cmd := exec.CommandContext(ctx, "podman")
	cmd.Args = append(cmd.Args, "push", tag)
	if err := util.RunCmd(ctx, cmd); err != nil {
		fmt.Println("failed to push")
		return err
	}
	return nil
}

func getImageID(ctx context.Context, ref string) (string, error) {
	cmd := exec.CommandContext(ctx, "podman")
	cmd.Args = append(cmd.Args, "images", ref, "-q", "--no-trunc")
	out, err := util.RunCmdOut(ctx, cmd)
	return strings.TrimSpace(string(out)), err
}

func getCleanBuildFilePath(context string, buildFilePath string) (string, error) {
	if context == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		context = wd
	}
	if filepath.IsAbs(buildFilePath) && util.IsFile(buildFilePath) {
		return buildFilePath, nil
	}
	if buildFilePath == "" {
		path := filepath.Join(context, "Containerfile")
		if util.IsFile(path) {
			return path, nil
		}
		path = filepath.Join(context, "Dockerfile")
		if util.IsFile(path) {
			return path, nil
		}
	}
	path := filepath.Join(context, buildFilePath)
	if util.IsFile(path) {
		return path, nil
	}
	return "", errors.NotFound
}

func (b *Builder) SupportedPlatforms() platform.Matcher {
	return platform.All
}
