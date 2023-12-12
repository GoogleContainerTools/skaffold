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

package bazel

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/tarball"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// Build builds an artifact with Bazel.
func (b *Builder) Build(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string, matcher platform.Matcher) (string, error) {
	// TODO: Implement building multi-platform images
	if matcher.IsMultiPlatform() {
		log.Entry(ctx).Warnf("multiple target platforms %q found for artifact %q. Skaffold doesn't yet support multi-platform builds for the bazel builder. Consider specifying a single target platform explicitly. See https://skaffold.dev/docs/pipeline-stages/builders/#cross-platform-build-support", matcher.String(), artifact.ImageName)
	}

	a := artifact.ArtifactType.BazelArtifact

	tarPath, err := b.buildTar(ctx, out, artifact.Workspace, a)
	if err != nil {
		return "", err
	}

	tarballManifest, indexManifest, err := b.tarballOrIndexManifest(ctx, tarPath, tag)
	if err != nil {
		return "", err
	}

	switch {
	case tarballManifest != nil && b.pushImages:
		return docker.Push(tarPath, tag, b.cfg, nil)
	case tarballManifest != nil && !b.pushImages:
		return b.loadImage(ctx, out, tarballManifest, tarPath, tag)
	case indexManifest != nil && b.pushImages:
		// TODO: should push the image index using docker push
		panic("implement me!")
	case indexManifest != nil && !b.pushImages:
		return b.loadImageIndex(ctx, out, indexManifest, tarPath, tag)
	default:
		return "", fmt.Errorf("unexpected state, neither manifest nor image index was found")
	}
}

func (b *Builder) SupportedPlatforms() platform.Matcher { return platform.All }

func (b *Builder) buildTar(ctx context.Context, out io.Writer, workspace string, a *latest.BazelArtifact) (string, error) {
	args := []string{"build"}
	args = append(args, a.BuildArgs...)
	args = append(args, a.BuildTarget)

	if output.IsColorable(out) {
		args = append(args, "--color=yes")
	} else {
		args = append(args, "--color=no")
	}

	// FIXME: is it possible to apply b.skipTests?
	cmd := exec.CommandContext(ctx, "bazel", args...)
	cmd.Dir = workspace
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(ctx, cmd); err != nil {
		return "", fmt.Errorf("running command: %w", err)
	}

	tarPath, err := bazelTarPath(ctx, workspace, a)
	if err != nil {
		return "", fmt.Errorf("getting bazel tar path: %w", err)
	}

	return tarPath, nil
}

func (b *Builder) tarballOrIndexManifest(ctx context.Context, tarPath, tag string) (tarball.Manifest, *v1.IndexManifest, error) {
	manifestFile, err := b.findFileInTar(tarPath, "manifest.json")
	if err != nil && !errors.Is(err, errFileNotFound) {
		return nil, nil, fmt.Errorf("load manifest from tarball failed: %w", err)
	}

	if err == nil {
		manifest, err := b.parseManifest(ctx, manifestFile)
		return manifest, nil, err
	}

	// NOTE: the index.json file needs to be extracted, as the current `layout`
	// package does not allow for passing an io.Reader.
	tmpDirPath, err := b.extractFileFromTar(tarPath, "index.json", tag)
	if err != nil {
		return nil, nil, fmt.Errorf("load image index from tarball failed: %w", err)
	}

	lp, err := layout.ImageIndexFromPath(tmpDirPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load image index from path failed: %w", err)
	}

	manifest, err := lp.IndexManifest()
	if err != nil {
		return nil, nil, fmt.Errorf("load index manifest from image index failed: %w", err)
	}

	return nil, manifest, nil
}

func (b *Builder) parseManifest(ctx context.Context, manifestFile io.ReadCloser) (tarball.Manifest, error) {
	defer manifestFile.Close()

	var manifest tarball.Manifest

	if err := json.NewDecoder(manifestFile).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest file failed: %w", err)
	}

	return manifest, nil
}

func (b *Builder) loadImage(ctx context.Context, out io.Writer, manifest tarball.Manifest, tarPath, tag string) (string, error) {
	imageTar, err := os.Open(tarPath)
	if err != nil {
		return "", fmt.Errorf("opening image tarball: %w", err)
	}

	defer imageTar.Close()

	bazelTag := manifest[0].RepoTags[0]

	return b.localDockerLoadImage(ctx, out, imageTar, bazelTag, tag)
}

const ociImageRefName = "org.opencontainers.image.ref.name"

func (b *Builder) loadImageIndex(ctx context.Context, out io.Writer, manifest *v1.IndexManifest, tarPath, tag string) (string, error) {
	imageTar, err := os.Open(tarPath)
	if err != nil {
		return "", fmt.Errorf("opening image tarball: %w", err)
	}

	defer imageTar.Close()

	bazelTag := manifest.Annotations[ociImageRefName]

	return b.localDockerLoadImage(ctx, out, imageTar, bazelTag, tag)
}

func (b *Builder) localDockerLoadImage(ctx context.Context, out io.Writer, imageTar io.ReadCloser, bazelTag, tag string) (string, error) {
	imageID, err := b.localDocker.Load(ctx, out, imageTar, bazelTag)
	if err != nil {
		return "", fmt.Errorf("loading image into docker daemon: %w", err)
	}

	if err := b.localDocker.Tag(ctx, imageID, tag); err != nil {
		return "", fmt.Errorf("tagging the image: %w", err)
	}

	return imageID, nil
}

func bazelTarPath(ctx context.Context, workspace string, a *latest.BazelArtifact) (string, error) {
	args := []string{
		"cquery",
		a.BuildTarget,
		"--output",
		"starlark",
		// Bazel docker .tar output targets have a single output file, which is
		// the path to the image tar.
		"--starlark:expr",
		"target.files.to_list()[0].path",
	}
	args = append(args, a.BuildArgs...)

	cmd := exec.CommandContext(ctx, "bazel", args...)
	cmd.Dir = workspace

	buf, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		return "", err
	}

	targetPath := strings.TrimSpace(string(buf))

	cmd = exec.CommandContext(ctx, "bazel", "info", "execution_root")
	cmd.Dir = workspace

	buf, err = util.RunCmdOut(ctx, cmd)
	if err != nil {
		return "", err
	}

	execRoot := strings.TrimSpace(string(buf))

	return filepath.Join(execRoot, targetPath), nil
}

// tarFile represents a single file inside a tar.
// Closing it closes the tar itself.
type tarFile struct {
	io.Reader
	io.Closer
}

var errFileNotFound = errors.New("bazel.findFileInTar: file not found in tar")

// extractFileFromTar extracts the specified file in the path, into a temp folder.
// Returns the path of the temp folder, or an error if it occurred.
func (b *Builder) extractFileFromTar(tarPath, filePath, tag string) (string, error) {
	file, err := b.findFileInTar(tarPath, filePath)
	if err != nil {
		return "", err
	}

	defer file.Close()

	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("%s-*", tag))
	if err != nil {
		return "", nil
	}

	extractedFilePath := path.Join(tmpDir, filePath)

	dest, err := os.Create(extractedFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to build temp file to be extracted from tar: %w", err)
	}

	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		return "", fmt.Errorf("failed to extract file from tar: %w", err)
	}

	return tmpDir, nil
}

// Copied from tarball.extractFileFromTar:
// https://github.com/google/go-containerregistry/blob/4fdaa32ee934cd178b6eb41b3096419a52ef426a/pkg/v1/tarball/image.go#L221-L255
func (b *Builder) findFileInTar(tarPath, filePath string) (io.ReadCloser, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open the tar file: %w", err)
	}

	needClose := true
	defer func() {
		if needClose {
			f.Close()
		}
	}()

	tf := tar.NewReader(f)

	for {
		hdr, err := tf.Next()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}

		if hdr.Name == filePath {
			if hdr.Typeflag == tar.TypeSymlink || hdr.Typeflag == tar.TypeLink {
				currentDir := filepath.Dir(filePath)
				return b.findFileInTar(tarPath, path.Join(currentDir, path.Clean(hdr.Linkname)))
			}

			needClose = false

			return tarFile{
				Reader: tf,
				Closer: f,
			}, nil
		}
	}

	return nil, errFileNotFound
}
