package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

func (b *Builder) buildPlease(ctx context.Context, out io.Writer, workspace string, a *latest.PleaseArtifact, tag string) (string, error) {
	fmt.Printf("running please builder with the tag:%s\n", tag)
	args := []string{"run"}
	args = append(args, a.BuildArgs...)
	args = append(args, a.BuildTarget+"_save")

	cmd := exec.CommandContext(ctx, "please", args...)
	cmd.Dir = workspace
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(cmd); err != nil {
		return "", errors.Wrap(err, "running command")
	}

	cmd = exec.CommandContext(ctx, "please", "query", "output", a.BuildTarget+"_tar")
	cmd.Dir = workspace
	buf, err := util.RunCmdOut(cmd)
	if err != nil {
		return "", errors.Wrap(err, "can't determine image tar file for "+a.BuildTarget)
	}

	tarPath := strings.Trim(strings.Trim(string(buf), "\n"), " ")

	if b.pushImages {
		return pushImage(tarPath, tag)
	}

	return b.loadPleaseImage(ctx, out, tarPath, a, tag)
}

func (b *Builder) loadPleaseImage(ctx context.Context, out io.Writer, tarPath string, a *latest.PleaseArtifact, tag string) (string, error) {
	imageTar, err := os.Open(tarPath)
	if err != nil {
		return "", errors.Wrap(err, "opening image tarball")
	}
	defer imageTar.Close()

	bazelTag := buildImageTag(a.BuildTarget)
	imageID, err := b.localDocker.Load(ctx, out, imageTar, bazelTag)
	if err != nil {
		return "", errors.Wrap(err, "loading image into docker daemon")
	}

	if err := b.localDocker.Tag(ctx, imageID, tag); err != nil {
		return "", errors.Wrap(err, "tagging the image")
	}

	return imageID, nil
}
