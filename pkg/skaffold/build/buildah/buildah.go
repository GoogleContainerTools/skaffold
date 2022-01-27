package buildah

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/storage/pkg/archive"
)

func (b *Builder) Build(ctx context.Context, out io.Writer, a *latestV1.Artifact, tag string) (string, error) {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"BuildType":   "buildah",
		"Context":     instrumentation.PII(a.Workspace),
		"Destination": instrumentation.PII(tag),
	})

	// Fail fast if the Containerfile can't be found.
	containerfile, err := docker.NormalizeDockerfilePath(a.Workspace, a.BuildahArtifact.ContainerFilePath)
	if err != nil {
		return "", containerfileNotFound(fmt.Errorf("normalizing containerfile path: %w", err), a.ImageName)
	}

	format, err := getFormat(a.BuildahArtifact.Format)
	if err != nil {
		return "", fmt.Errorf("buildah format: %w", err)
	}

	compression, err := getCompression(a.BuildahArtifact.Compression)
	if err != nil {
		return "", fmt.Errorf("buildah compression: %w", err)
	}

	absContextDir, err := filepath.Abs(a.Workspace)
	if err != nil {
		return "", fmt.Errorf("getting absolute path of context for image %v: %w", a.ImageName, err)
	}

	buildOptions := define.BuildOptions{
		ContextDirectory: absContextDir,
		NoCache:          a.BuildahArtifact.NoCache,
		Args:             a.BuildahArtifact.BuildArgs,
		Target:           a.BuildahArtifact.Target,
		AdditionalTags:   []string{tag},
		Err:              out,
		Out:              out,
		ReportWriter:     out,
		Squash:           a.BuildahArtifact.Squash,
		Output:           a.ImageName,
		Compression:      compression,
		OutputFormat:     format,
		// running in rootless mode, so isolate chroot
		// otherwise buildah will try to create devices,
		// which I am not allowed to do in a rootless environment.
		Isolation: buildah.IsolationChroot,
		CommonBuildOpts: &define.CommonBuildOptions{
			Secrets: a.BuildahArtifact.Secrets,
		},
	}

	id, err := b.runtime.Build(ctx, containerfile, buildOptions)
	if err != nil {
		return "", fmt.Errorf("building image %v: %w", a.ImageName, err)
	}
	if b.pushImages {
		digest, err := b.runtime.Push(ctx, id)
		if err != nil {
			return "", fmt.Errorf("pushing image %v: %w", a.ImageName, err)
		}
		return digest, nil
	}
	return id, nil
}

func getCompression(compression string) (archive.Compression, error) {
	switch compression {
	case xzCompression:
		return archive.Xz, nil
	case zstdCompression:
		return archive.Zstd, nil
	case gzipCompression:
		return archive.Gzip, nil
	case bzip2Compression:
		return archive.Bzip2, nil
	case uncompressed:
		return archive.Uncompressed, nil
	case "":
		return archive.Gzip, nil
	default:
		return -1, fmt.Errorf("unknown compression algorithm: %q", compression)
	}
}

func getFormat(format string) (string, error) {
	switch format {
	case define.OCI:
		return define.OCIv1ImageManifest, nil
	case define.DOCKER:
		return define.Dockerv2ImageManifest, nil
	case "":
		return define.OCIv1ImageManifest, nil
	default:
		return "", fmt.Errorf("unrecognized image type %q", format)
	}
}
