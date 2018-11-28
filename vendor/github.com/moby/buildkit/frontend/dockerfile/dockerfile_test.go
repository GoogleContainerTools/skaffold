package dockerfile

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/continuity/fs/fstest"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/moby/buildkit/frontend/dockerfile/dockerfile2llb"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/util/testutil"
	"github.com/moby/buildkit/util/testutil/httpserver"
	"github.com/moby/buildkit/util/testutil/integration"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var allTests = []integration.Test{
	testNoSnapshotLeak,
	testCmdShell,
	testGlobalArg,
	testDockerfileDirs,
	testDockerfileInvalidCommand,
	testDockerfileADDFromURL,
	testDockerfileAddArchive,
	testDockerfileScratchConfig,
	testExportedHistory,
	testExposeExpansion,
	testUser,
	testDockerignore,
	testDockerignoreInvalid,
	testDockerfileFromGit,
	testCopyChown,
	testCopyWildcards,
	testCopyOverrideFiles,
	testMultiStageImplicitFrom,
	testCopyVarSubstitution,
	testMultiStageCaseInsensitive,
	testLabels,
	testCacheImportExport,
	testReproducibleIDs,
	testImportExportReproducibleIDs,
	testNoCache,
	testDockerfileFromHTTP,
	testBuiltinArgs,
	testPullScratch,
	testSymlinkDestination,
	testHTTPDockerfile,
	testNoSnapshotLeak,
	testCopySymlinks,
	testContextChangeDirToFile,
	testPlatformArgsImplicit,
	testPlatformArgsExplicit,
	testExportMultiPlatform,
	testQuotedMetaArgs,
	testIgnoreEntrypoint,
	testCopyThroughSymlinkContext,
	testCopyThroughSymlinkMultiStage,
	testCopyChownCreateDest,
	testEmptyDestDir,
	testSymlinkedDockerfile,
}

var opts []integration.TestOpt

type frontend interface {
	Solve(context.Context, *client.Client, client.SolveOpt, chan *client.SolveStatus) (*client.SolveResponse, error)
	DFCmdArgs(string, string) (string, string)
	RequiresBuildctl(t *testing.T)
}

func init() {
	frontends := map[string]interface{}{}

	opts = []integration.TestOpt{
		integration.WithMirroredImages(integration.OfficialImages("busybox:latest")),
		integration.WithMirroredImages(map[string]string{
			"docker/dockerfile-copy:v0.1.9": "docker.io/" + dockerfile2llb.DefaultCopyImage,
		}),
		integration.WithMatrix("frontend", frontends),
	}

	if os.Getenv("FRONTEND_BUILTIN_ONLY") == "1" {
		frontends["builtin"] = &builtinFrontend{}
	} else if os.Getenv("FRONTEND_CLIENT_ONLY") == "1" {
		frontends["client"] = &clientFrontend{}
	} else if gw := os.Getenv("FRONTEND_GATEWAY_ONLY"); gw != "" {
		name := "buildkit_test/" + identity.NewID() + ":latest"
		opts = append(opts, integration.WithMirroredImages(map[string]string{
			name: gw,
		}))
		frontends["gateway"] = &gatewayFrontend{gw: name}
	} else {
		frontends["builtin"] = &builtinFrontend{}
		frontends["client"] = &clientFrontend{}
	}
}

func TestIntegration(t *testing.T) {
	integration.Run(t, allTests, opts...)
}

func testEmptyDestDir(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM busybox
ENV empty=""
COPY testfile $empty
RUN [ "$(cat testfile)" == "contents0" ]
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("testfile", []byte("contents0"), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)
}

func testSymlinkedDockerfile(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch
ENV foo bar
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile.web", dockerfile, 0600),
		fstest.Symlink("Dockerfile.web", "Dockerfile"),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)
}

func testCopyChownCreateDest(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM busybox
RUN adduser -D user
COPY --chown=user:user . /dest
RUN [ "$(stat -c "%U %G" /dest)" == "user user" ]
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)
}

func testCopyThroughSymlinkContext(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch
COPY link/foo .
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.Symlink("sub", "link"),
		fstest.CreateDir("sub", 0700),
		fstest.CreateFile("sub/foo", []byte(`contents`), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "foo"))
	require.NoError(t, err)
	require.Equal(t, "contents", string(dt))
}

func testCopyThroughSymlinkMultiStage(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM busybox AS build
RUN mkdir -p /out/sub && ln -s /out/sub /sub && ln -s out/sub /sub2 && echo -n "data" > /sub/foo
FROM scratch
COPY --from=build /sub/foo .
COPY --from=build /sub2/foo bar
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "foo"))
	require.NoError(t, err)
	require.Equal(t, "data", string(dt))
}

func testIgnoreEntrypoint(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM busybox
ENTRYPOINT ["/nosuchcmd"]
RUN ["ls"]
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)
}

func testQuotedMetaArgs(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
ARG a1="box"
ARG a2="$a1-foo"
FROM busy$a1 AS build
ARG a2
ARG a3="bar-$a2"
RUN echo -n $a3 > /out
FROM scratch
COPY --from=build /out .
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "out"))
	require.NoError(t, err)
	require.Equal(t, "bar-box-foo", string(dt))
}

func testExportMultiPlatform(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch
ARG TARGETARCH
ARG TARGETPLATFORM
LABEL target=$TARGETPLATFORM
COPY arch-$TARGETARCH whoami
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("arch-arm", []byte(`i am arm`), 0600),
		fstest.CreateFile("arch-amd64", []byte(`i am amd64`), 0600),
		fstest.CreateFile("arch-s390x", []byte(`i am s390x`), 0600),
		fstest.CreateFile("arch-ppc64le", []byte(`i am ppc64le`), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
		FrontendAttrs: map[string]string{
			"platform": "windows/amd64,linux/arm,linux/s390x",
		},
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "windows_amd64/whoami"))
	require.NoError(t, err)
	require.Equal(t, "i am amd64", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "linux_arm_v7/whoami"))
	require.NoError(t, err)
	require.Equal(t, "i am arm", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "linux_s390x/whoami"))
	require.NoError(t, err)
	require.Equal(t, "i am s390x", string(dt))

	// repeat with oci exporter

	destDir, err = ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	out := filepath.Join(destDir, "out.tar")
	outW, err := os.Create(out)
	require.NoError(t, err)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
		FrontendAttrs: map[string]string{
			"platform": "windows/amd64,linux/arm/v6,linux/ppc64le",
		},
		Exporter:       client.ExporterOCI,
		ExporterOutput: outW,
	}, nil)
	require.NoError(t, err)

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "out.tar"))
	require.NoError(t, err)

	m, err := testutil.ReadTarToMap(dt, false)
	require.NoError(t, err)

	var idx ocispec.Index
	err = json.Unmarshal(m["index.json"].Data, &idx)
	require.NoError(t, err)

	mlistHex := idx.Manifests[0].Digest.Hex()

	idx = ocispec.Index{}
	err = json.Unmarshal(m["blobs/sha256/"+mlistHex].Data, &idx)
	require.NoError(t, err)

	require.Equal(t, 3, len(idx.Manifests))

	for i, exp := range []struct {
		p    string
		os   string
		arch string
		dt   string
	}{
		{p: "windows/amd64", os: "windows", arch: "amd64", dt: "i am amd64"},
		{p: "linux/arm/v6", os: "linux", arch: "arm", dt: "i am arm"},
		{p: "linux/ppc64le", os: "linux", arch: "ppc64le", dt: "i am ppc64le"},
	} {
		t.Run(exp.p, func(t *testing.T) {
			require.Equal(t, exp.p, platforms.Format(*idx.Manifests[i].Platform))

			var mfst ocispec.Manifest
			err = json.Unmarshal(m["blobs/sha256/"+idx.Manifests[i].Digest.Hex()].Data, &mfst)
			require.NoError(t, err)

			require.Equal(t, 1, len(mfst.Layers))

			m2, err := testutil.ReadTarToMap(m["blobs/sha256/"+mfst.Layers[0].Digest.Hex()].Data, true)
			require.NoError(t, err)
			require.Equal(t, exp.dt, string(m2["whoami"].Data))

			var img ocispec.Image
			err = json.Unmarshal(m["blobs/sha256/"+mfst.Config.Digest.Hex()].Data, &img)
			require.NoError(t, err)

			require.Equal(t, exp.os, img.OS)
			require.Equal(t, exp.arch, img.Architecture)
			v, ok := img.Config.Labels["target"]
			require.True(t, ok)
			require.Equal(t, exp.p, v)
		})
	}
}

// tonistiigi/fsutil#46
func testContextChangeDirToFile(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch
COPY foo /
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateDir("foo", 0700),
		fstest.CreateFile("foo/bar", []byte(`contents`), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dir, err = tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("foo", []byte(`contents2`), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "foo"))
	require.NoError(t, err)
	require.Equal(t, "contents2", string(dt))
}

func testNoSnapshotLeak(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch
COPY foo /
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("foo", []byte(`contents`), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	du, err := c.DiskUsage(context.TODO())
	require.NoError(t, err)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	du2, err := c.DiskUsage(context.TODO())
	require.NoError(t, err)

	require.Equal(t, len(du), len(du2))
}

func testCopySymlinks(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch
COPY foo /
COPY sub/l* alllinks/
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("bar", []byte(`bar-contents`), 0600),
		fstest.Symlink("bar", "foo"),
		fstest.CreateDir("sub", 0700),
		fstest.CreateFile("sub/lfile", []byte(`baz-contents`), 0600),
		fstest.Symlink("subfile", "sub/l0"),
		fstest.CreateFile("sub/subfile", []byte(`subfile-contents`), 0600),
		fstest.Symlink("second", "sub/l1"),
		fstest.Symlink("baz", "sub/second"),
		fstest.CreateFile("sub/baz", []byte(`baz-contents`), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)
}

func testHTTPDockerfile(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM busybox
RUN echo -n "foo-contents" > /foo
FROM scratch
COPY --from=0 /foo /foo
`)

	srcDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(srcDir)

	err = ioutil.WriteFile(filepath.Join(srcDir, "Dockerfile"), dockerfile, 0600)
	require.NoError(t, err)

	resp := httpserver.Response{
		Etag:    identity.NewID(),
		Content: dockerfile,
	}

	server := httpserver.NewTestServer(map[string]httpserver.Response{
		"/df": resp,
	})
	defer server.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		FrontendAttrs: map[string]string{
			"context":  server.URL + "/df",
			"filename": "mydockerfile", // this is bogus, any name should work
		},
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "foo"))
	require.NoError(t, err)
	require.Equal(t, "foo-contents", string(dt))

}

func testCmdShell(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	var cdAddress string
	if cd, ok := sb.(interface {
		ContainerdAddress() string
	}); !ok {
		t.Skip("requires local image store")
	} else {
		cdAddress = cd.ContainerdAddress()
	}

	dockerfile := []byte(`
FROM scratch
CMD ["test"]
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	target := "docker.io/moby/cmdoverridetest:latest"
	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter: client.ExporterImage,
		ExporterAttrs: map[string]string{
			"name": target,
		},
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dockerfile = []byte(`
FROM docker.io/moby/cmdoverridetest:latest
SHELL ["ls"]
ENTRYPOINT my entrypoint
`)

	dir, err = tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	target = "docker.io/moby/cmdoverridetest2:latest"
	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter: client.ExporterImage,
		ExporterAttrs: map[string]string{
			"name": target,
		},
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	ctr, err := newContainerd(cdAddress)
	require.NoError(t, err)
	defer ctr.Close()

	ctx := namespaces.WithNamespace(context.Background(), "buildkit")

	img, err := ctr.ImageService().Get(ctx, target)
	require.NoError(t, err)

	desc, err := img.Config(ctx, ctr.ContentStore(), platforms.Default())
	require.NoError(t, err)

	dt, err := content.ReadBlob(ctx, ctr.ContentStore(), desc)
	require.NoError(t, err)

	var ociimg ocispec.Image
	err = json.Unmarshal(dt, &ociimg)
	require.NoError(t, err)

	require.Equal(t, []string(nil), ociimg.Config.Cmd)
	require.Equal(t, []string{"ls", "my entrypoint"}, ociimg.Config.Entrypoint)
}

func testPullScratch(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	var cdAddress string
	if cd, ok := sb.(interface {
		ContainerdAddress() string
	}); !ok {
		t.Skip("requires local image store")
	} else {
		cdAddress = cd.ContainerdAddress()
	}

	dockerfile := []byte(`
FROM scratch
LABEL foo=bar
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	target := "docker.io/moby/testpullscratch:latest"
	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter: client.ExporterImage,
		ExporterAttrs: map[string]string{
			"name": target,
		},
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dockerfile = []byte(`
FROM docker.io/moby/testpullscratch:latest
LABEL bar=baz
COPY foo .
`)

	dir, err = tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("foo", []byte("foo-contents"), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	target = "docker.io/moby/testpullscratch2:latest"
	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter: client.ExporterImage,
		ExporterAttrs: map[string]string{
			"name": target,
		},
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	ctr, err := newContainerd(cdAddress)
	require.NoError(t, err)
	defer ctr.Close()

	ctx := namespaces.WithNamespace(context.Background(), "buildkit")

	img, err := ctr.ImageService().Get(ctx, target)
	require.NoError(t, err)

	desc, err := img.Config(ctx, ctr.ContentStore(), platforms.Default())
	require.NoError(t, err)

	dt, err := content.ReadBlob(ctx, ctr.ContentStore(), desc)
	require.NoError(t, err)

	var ociimg ocispec.Image
	err = json.Unmarshal(dt, &ociimg)
	require.NoError(t, err)

	require.Equal(t, "layers", ociimg.RootFS.Type)
	require.Equal(t, 1, len(ociimg.RootFS.DiffIDs))
	v, ok := ociimg.Config.Labels["foo"]
	require.True(t, ok)
	require.Equal(t, "bar", v)
	v, ok = ociimg.Config.Labels["bar"]
	require.True(t, ok)
	require.Equal(t, "baz", v)

	echo := llb.Image("busybox").
		Run(llb.Shlex(`sh -c "echo -n foo0 > /empty/foo"`)).
		AddMount("/empty", llb.Image("docker.io/moby/testpullscratch:latest"))

	def, err := echo.Marshal()
	require.NoError(t, err)

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = c.Solve(context.TODO(), def, client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "foo"))
	require.NoError(t, err)
	require.Equal(t, "foo0", string(dt))
}

func testGlobalArg(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
ARG tag=nosuchtag
FROM busybox:${tag}
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		FrontendAttrs: map[string]string{
			"build-arg:tag": "latest",
		},
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)
}

func testDockerfileDirs(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)
	f.RequiresBuildctl(t)

	dockerfile := []byte(`
	FROM busybox
	COPY foo /foo2
	COPY foo /
	RUN echo -n bar > foo3
	RUN test -f foo
	RUN cmp -s foo foo2
	RUN cmp -s foo foo3
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("foo", []byte("bar"), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	args, trace := f.DFCmdArgs(dir, dir)
	defer os.RemoveAll(trace)

	cmd := sb.Cmd(args)
	require.NoError(t, cmd.Run())

	_, err = os.Stat(trace)
	require.NoError(t, err)

	// relative urls
	args, trace = f.DFCmdArgs(".", ".")
	defer os.RemoveAll(trace)

	cmd = sb.Cmd(args)
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	_, err = os.Stat(trace)
	require.NoError(t, err)

	// different context and dockerfile directories
	dir1, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir1)

	dir2, err := tmpdir(
		fstest.CreateFile("foo", []byte("bar"), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir2)

	args, trace = f.DFCmdArgs(dir2, dir1)
	defer os.RemoveAll(trace)

	cmd = sb.Cmd(args)
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	_, err = os.Stat(trace)
	require.NoError(t, err)

	// TODO: test trace file output, cache hits, logs etc.
	// TODO: output metadata about original dockerfile command in trace
}

func testDockerfileInvalidCommand(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)
	f.RequiresBuildctl(t)
	dockerfile := []byte(`
	FROM busybox
	RUN invalidcmd
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	args, trace := f.DFCmdArgs(dir, dir)
	defer os.RemoveAll(trace)

	cmd := sb.Cmd(args)
	stdout := new(bytes.Buffer)
	cmd.Stderr = stdout
	err = cmd.Run()
	require.Error(t, err)
	require.Contains(t, stdout.String(), "/bin/sh -c invalidcmd")
	require.Contains(t, stdout.String(), "executor failed running")
}

func testDockerfileADDFromURL(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)
	f.RequiresBuildctl(t)

	modTime := time.Now().Add(-24 * time.Hour) // avoid falso positive with current time

	resp := httpserver.Response{
		Etag:    identity.NewID(),
		Content: []byte("content1"),
	}

	resp2 := httpserver.Response{
		Etag:         identity.NewID(),
		LastModified: &modTime,
		Content:      []byte("content2"),
	}

	server := httpserver.NewTestServer(map[string]httpserver.Response{
		"/foo": resp,
		"/":    resp2,
	})
	defer server.Close()

	dockerfile := []byte(fmt.Sprintf(`
FROM scratch
ADD %s /dest/
`, server.URL+"/foo"))

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	args, trace := f.DFCmdArgs(dir, dir)
	defer os.RemoveAll(trace)

	destDir, err := tmpdir()
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	cmd := sb.Cmd(args + fmt.Sprintf(" --exporter=local --exporter-opt output=%s", destDir))
	err = cmd.Run()
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "dest/foo"))
	require.NoError(t, err)
	require.Equal(t, []byte("content1"), dt)

	// test the default properties
	dockerfile = []byte(fmt.Sprintf(`
FROM scratch
ADD %s /dest/
`, server.URL+"/"))

	dir, err = tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	args, trace = f.DFCmdArgs(dir, dir)
	defer os.RemoveAll(trace)

	destDir, err = tmpdir()
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	cmd = sb.Cmd(args + fmt.Sprintf(" --exporter=local --exporter-opt output=%s", destDir))
	err = cmd.Run()
	require.NoError(t, err)

	destFile := filepath.Join(destDir, "dest/__unnamed__")
	dt, err = ioutil.ReadFile(destFile)
	require.NoError(t, err)
	require.Equal(t, []byte("content2"), dt)

	fi, err := os.Stat(destFile)
	require.NoError(t, err)
	require.Equal(t, modTime.Format(http.TimeFormat), fi.ModTime().Format(http.TimeFormat))
}

func testDockerfileAddArchive(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)
	f.RequiresBuildctl(t)

	buf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buf)
	expectedContent := []byte("content0")
	err := tw.WriteHeader(&tar.Header{
		Name:     "foo",
		Typeflag: tar.TypeReg,
		Size:     int64(len(expectedContent)),
		Mode:     0644,
	})
	require.NoError(t, err)
	_, err = tw.Write(expectedContent)
	require.NoError(t, err)
	err = tw.Close()
	require.NoError(t, err)

	dockerfile := []byte(`
FROM scratch
ADD t.tar /
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("t.tar", buf.Bytes(), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	args, trace := f.DFCmdArgs(dir, dir)
	defer os.RemoveAll(trace)

	destDir, err := tmpdir()
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	cmd := sb.Cmd(args + fmt.Sprintf(" --exporter=local --exporter-opt output=%s", destDir))
	require.NoError(t, cmd.Run())

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "foo"))
	require.NoError(t, err)
	require.Equal(t, expectedContent, dt)

	// add gzip tar
	buf2 := bytes.NewBuffer(nil)
	gz := gzip.NewWriter(buf2)
	_, err = gz.Write(buf.Bytes())
	require.NoError(t, err)
	err = gz.Close()
	require.NoError(t, err)

	dockerfile = []byte(`
FROM scratch
ADD t.tar.gz /
`)

	dir, err = tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("t.tar.gz", buf2.Bytes(), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	args, trace = f.DFCmdArgs(dir, dir)
	defer os.RemoveAll(trace)

	destDir, err = tmpdir()
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	cmd = sb.Cmd(args + fmt.Sprintf(" --exporter=local --exporter-opt output=%s", destDir))
	require.NoError(t, cmd.Run())

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "foo"))
	require.NoError(t, err)
	require.Equal(t, expectedContent, dt)

	// COPY doesn't extract
	dockerfile = []byte(`
FROM scratch
COPY t.tar.gz /
`)

	dir, err = tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("t.tar.gz", buf2.Bytes(), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	args, trace = f.DFCmdArgs(dir, dir)
	defer os.RemoveAll(trace)

	destDir, err = tmpdir()
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	cmd = sb.Cmd(args + fmt.Sprintf(" --exporter=local --exporter-opt output=%s", destDir))
	require.NoError(t, cmd.Run())

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "t.tar.gz"))
	require.NoError(t, err)
	require.Equal(t, buf2.Bytes(), dt)

	// ADD from URL doesn't extract
	resp := httpserver.Response{
		Etag:    identity.NewID(),
		Content: buf2.Bytes(),
	}

	server := httpserver.NewTestServer(map[string]httpserver.Response{
		"/t.tar.gz": resp,
	})
	defer server.Close()

	dockerfile = []byte(fmt.Sprintf(`
FROM scratch
ADD %s /
`, server.URL+"/t.tar.gz"))

	dir, err = tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	args, trace = f.DFCmdArgs(dir, dir)
	defer os.RemoveAll(trace)

	destDir, err = tmpdir()
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	cmd = sb.Cmd(args + fmt.Sprintf(" --exporter=local --exporter-opt output=%s", destDir))
	require.NoError(t, cmd.Run())

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "t.tar.gz"))
	require.NoError(t, err)
	require.Equal(t, buf2.Bytes(), dt)

	// https://github.com/moby/buildkit/issues/386
	dockerfile = []byte(fmt.Sprintf(`
FROM scratch
ADD %s /newname.tar.gz
`, server.URL+"/t.tar.gz"))

	dir, err = tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	args, trace = f.DFCmdArgs(dir, dir)
	defer os.RemoveAll(trace)

	destDir, err = tmpdir()
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	cmd = sb.Cmd(args + fmt.Sprintf(" --exporter=local --exporter-opt output=%s", destDir))
	require.NoError(t, cmd.Run())

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "newname.tar.gz"))
	require.NoError(t, err)
	require.Equal(t, buf2.Bytes(), dt)
}

func testSymlinkDestination(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)
	f.RequiresBuildctl(t)

	buf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buf)
	expectedContent := []byte("content0")
	err := tw.WriteHeader(&tar.Header{
		Name:     "symlink",
		Typeflag: tar.TypeSymlink,
		Linkname: "../tmp/symlink-target",
		Mode:     0755,
	})
	require.NoError(t, err)
	err = tw.Close()
	require.NoError(t, err)

	dockerfile := []byte(`
FROM scratch
ADD t.tar /
COPY foo /symlink/
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("foo", expectedContent, 0600),
		fstest.CreateFile("t.tar", buf.Bytes(), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	args, trace := f.DFCmdArgs(dir, dir)
	defer os.RemoveAll(trace)

	destDir, err := tmpdir()
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	cmd := sb.Cmd(args + fmt.Sprintf(" --exporter=local --exporter-opt output=%s", destDir))
	require.NoError(t, cmd.Run())

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "tmp/symlink-target/foo"))
	require.NoError(t, err)
	require.Equal(t, expectedContent, dt)
}

func testDockerfileScratchConfig(t *testing.T, sb integration.Sandbox) {
	var cdAddress string
	if cd, ok := sb.(interface {
		ContainerdAddress() string
	}); !ok {
		t.Skip("only for containerd worker")
	} else {
		cdAddress = cd.ContainerdAddress()
	}

	f := getFrontend(t, sb)
	f.RequiresBuildctl(t)
	dockerfile := []byte(`
FROM scratch
ENV foo=bar
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	args, trace := f.DFCmdArgs(dir, dir)
	defer os.RemoveAll(trace)

	target := "example.com/moby/dockerfilescratch:test"
	cmd := sb.Cmd(args + " --exporter=image --exporter-opt=name=" + target)
	err = cmd.Run()
	require.NoError(t, err)

	client, err := newContainerd(cdAddress)
	require.NoError(t, err)
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), "buildkit")

	img, err := client.ImageService().Get(ctx, target)
	require.NoError(t, err)

	desc, err := img.Config(ctx, client.ContentStore(), platforms.Default())
	require.NoError(t, err)

	dt, err := content.ReadBlob(ctx, client.ContentStore(), desc)
	require.NoError(t, err)

	var ociimg ocispec.Image
	err = json.Unmarshal(dt, &ociimg)
	require.NoError(t, err)

	require.NotEqual(t, "", ociimg.OS)
	require.NotEqual(t, "", ociimg.Architecture)
	require.NotEqual(t, "", ociimg.Config.WorkingDir)
	require.Equal(t, "layers", ociimg.RootFS.Type)
	require.Equal(t, 0, len(ociimg.RootFS.DiffIDs))

	require.Equal(t, 1, len(ociimg.History))
	require.Contains(t, ociimg.History[0].CreatedBy, "ENV foo=bar")
	require.Equal(t, true, ociimg.History[0].EmptyLayer)

	require.Contains(t, ociimg.Config.Env, "foo=bar")
	require.Condition(t, func() bool {
		for _, env := range ociimg.Config.Env {
			if strings.HasPrefix(env, "PATH=") {
				return true
			}
		}
		return false
	})
}

func testExposeExpansion(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch
ARG PORTS="3000 4000/udp"
EXPOSE $PORTS
EXPOSE 5000
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	target := "example.com/moby/dockerfileexpansion:test"
	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter: client.ExporterImage,
		ExporterAttrs: map[string]string{
			"name": target,
		},
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	var cdAddress string
	if cd, ok := sb.(interface {
		ContainerdAddress() string
	}); !ok {
		return
	} else {
		cdAddress = cd.ContainerdAddress()
	}

	client, err := newContainerd(cdAddress)
	require.NoError(t, err)
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), "buildkit")

	img, err := client.ImageService().Get(ctx, target)
	require.NoError(t, err)

	desc, err := img.Config(ctx, client.ContentStore(), platforms.Default())
	require.NoError(t, err)

	dt, err := content.ReadBlob(ctx, client.ContentStore(), desc)
	require.NoError(t, err)

	var ociimg ocispec.Image
	err = json.Unmarshal(dt, &ociimg)
	require.NoError(t, err)

	require.Equal(t, 3, len(ociimg.Config.ExposedPorts))

	var ports []string
	for p := range ociimg.Config.ExposedPorts {
		ports = append(ports, p)
	}

	sort.Strings(ports)

	require.Equal(t, "3000/tcp", ports[0])
	require.Equal(t, "4000/udp", ports[1])
	require.Equal(t, "5000/tcp", ports[2])
}

func testDockerignore(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch
COPY . .
`)

	dockerignore := []byte(`
ba*
Dockerfile
!bay
.dockerignore
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("foo", []byte(`foo-contents`), 0600),
		fstest.CreateFile("bar", []byte(`bar-contents`), 0600),
		fstest.CreateFile("baz", []byte(`baz-contents`), 0600),
		fstest.CreateFile("bay", []byte(`bay-contents`), 0600),
		fstest.CreateFile(".dockerignore", dockerignore, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "foo"))
	require.NoError(t, err)
	require.Equal(t, "foo-contents", string(dt))

	_, err = os.Stat(filepath.Join(destDir, ".dockerignore"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(destDir, "Dockerfile"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(destDir, "bar"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(destDir, "baz"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "bay"))
	require.NoError(t, err)
	require.Equal(t, "bay-contents", string(dt))
}

func testDockerignoreInvalid(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch
COPY . .
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile(".dockerignore", []byte("!\n"), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	c, err := client.New(ctx, sb.Address())
	require.NoError(t, err)
	defer c.Close()

	_, err = f.Solve(ctx, c, client.SolveOpt{
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	// err is either the expected error due to invalid dockerignore or error from the timeout
	require.Error(t, err)
	select {
	case <-ctx.Done():
		t.Fatal("timed out")
	default:
	}
}

func testExportedHistory(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)
	f.RequiresBuildctl(t)

	// using multi-stage to test that history is scoped to one stage
	dockerfile := []byte(`
FROM busybox AS base
ENV foo=bar
COPY foo /foo2
FROM busybox
COPY --from=base foo2 foo3
WORKDIR /
RUN echo bar > foo4
RUN ["ls"]
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("foo", []byte("contents0"), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	args, trace := f.DFCmdArgs(dir, dir)
	defer os.RemoveAll(trace)

	target := "example.com/moby/dockerfilescratch:test"
	cmd := sb.Cmd(args + " --exporter=image --exporter-opt=name=" + target)
	require.NoError(t, cmd.Run())

	// TODO: expose this test to OCI worker

	var cdAddress string
	if cd, ok := sb.(interface {
		ContainerdAddress() string
	}); !ok {
		t.Skip("only for containerd worker")
	} else {
		cdAddress = cd.ContainerdAddress()
	}

	client, err := newContainerd(cdAddress)
	require.NoError(t, err)
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), "buildkit")

	img, err := client.ImageService().Get(ctx, target)
	require.NoError(t, err)

	desc, err := img.Config(ctx, client.ContentStore(), platforms.Default())
	require.NoError(t, err)

	dt, err := content.ReadBlob(ctx, client.ContentStore(), desc)
	require.NoError(t, err)

	var ociimg ocispec.Image
	err = json.Unmarshal(dt, &ociimg)
	require.NoError(t, err)

	require.Equal(t, "layers", ociimg.RootFS.Type)
	// this depends on busybox. should be ok after freezing images
	require.Equal(t, 3, len(ociimg.RootFS.DiffIDs))

	require.Equal(t, 6, len(ociimg.History))
	require.Contains(t, ociimg.History[2].CreatedBy, "COPY foo2 foo3")
	require.Equal(t, false, ociimg.History[2].EmptyLayer)
	require.Contains(t, ociimg.History[3].CreatedBy, "WORKDIR /")
	require.Equal(t, true, ociimg.History[3].EmptyLayer)
	require.Contains(t, ociimg.History[4].CreatedBy, "echo bar > foo4")
	require.Equal(t, false, ociimg.History[4].EmptyLayer)
	require.Contains(t, ociimg.History[5].CreatedBy, "RUN ls")
	require.Equal(t, true, ociimg.History[5].EmptyLayer)
}

func testUser(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM busybox AS base
RUN mkdir -m 0777 /out
RUN id -un > /out/rootuser

# Make sure our defaults work
RUN [ "$(id -u):$(id -g)/$(id -un):$(id -gn)" = '0:0/root:root' ]

# TODO decide if "args.user = strconv.Itoa(syscall.Getuid())" is acceptable behavior for changeUser in sysvinit instead of "return nil" when "USER" isn't specified (so that we get the proper group list even if that is the empty list, even in the default case of not supplying an explicit USER to run as, which implies USER 0)
USER root
RUN [ "$(id -G):$(id -Gn)" = '0 10:root wheel' ]

# Setup dockerio user and group
RUN echo 'dockerio:x:1001:1001::/bin:/bin/false' >> /etc/passwd && \
	echo 'dockerio:x:1001:' >> /etc/group

# Make sure we can switch to our user and all the information is exactly as we expect it to be
USER dockerio
RUN [ "$(id -u):$(id -g)/$(id -un):$(id -gn)/$(id -G):$(id -Gn)" = '1001:1001/dockerio:dockerio/1001:dockerio' ]

# Switch back to root and double check that worked exactly as we might expect it to
USER root
RUN [ "$(id -u):$(id -g)/$(id -un):$(id -gn)/$(id -G):$(id -Gn)" = '0:0/root:root/0 10:root wheel' ] && \
        # Add a "supplementary" group for our dockerio user
	echo 'supplementary:x:1002:dockerio' >> /etc/group

# ... and then go verify that we get it like we expect
USER dockerio
RUN [ "$(id -u):$(id -g)/$(id -un):$(id -gn)/$(id -G):$(id -Gn)" = '1001:1001/dockerio:dockerio/1001 1002:dockerio supplementary' ]
USER 1001
RUN [ "$(id -u):$(id -g)/$(id -un):$(id -gn)/$(id -G):$(id -Gn)" = '1001:1001/dockerio:dockerio/1001 1002:dockerio supplementary' ]

# super test the new "user:group" syntax
USER dockerio:dockerio
RUN [ "$(id -u):$(id -g)/$(id -un):$(id -gn)/$(id -G):$(id -Gn)" = '1001:1001/dockerio:dockerio/1001:dockerio' ]
USER 1001:dockerio
RUN [ "$(id -u):$(id -g)/$(id -un):$(id -gn)/$(id -G):$(id -Gn)" = '1001:1001/dockerio:dockerio/1001:dockerio' ]
USER dockerio:1001
RUN [ "$(id -u):$(id -g)/$(id -un):$(id -gn)/$(id -G):$(id -Gn)" = '1001:1001/dockerio:dockerio/1001:dockerio' ]
USER 1001:1001
RUN [ "$(id -u):$(id -g)/$(id -un):$(id -gn)/$(id -G):$(id -Gn)" = '1001:1001/dockerio:dockerio/1001:dockerio' ]
USER dockerio:supplementary
RUN [ "$(id -u):$(id -g)/$(id -un):$(id -gn)/$(id -G):$(id -Gn)" = '1001:1002/dockerio:supplementary/1002:supplementary' ]
USER dockerio:1002
RUN [ "$(id -u):$(id -g)/$(id -un):$(id -gn)/$(id -G):$(id -Gn)" = '1001:1002/dockerio:supplementary/1002:supplementary' ]
USER 1001:supplementary
RUN [ "$(id -u):$(id -g)/$(id -un):$(id -gn)/$(id -G):$(id -Gn)" = '1001:1002/dockerio:supplementary/1002:supplementary' ]
USER 1001:1002
RUN [ "$(id -u):$(id -g)/$(id -un):$(id -gn)/$(id -G):$(id -Gn)" = '1001:1002/dockerio:supplementary/1002:supplementary' ]

# make sure unknown uid/gid still works properly
USER 1042:1043
RUN [ "$(id -u):$(id -g)/$(id -un):$(id -gn)/$(id -G):$(id -Gn)" = '1042:1043/1042:1043/1043:1043' ]
USER daemon
RUN id -un > /out/daemonuser
FROM scratch
COPY --from=base /out /
USER nobody
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "rootuser"))
	require.NoError(t, err)
	require.Equal(t, "root\n", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "daemonuser"))
	require.NoError(t, err)
	require.Equal(t, "daemon\n", string(dt))

	// test user in exported
	target := "example.com/moby/dockerfileuser:test"
	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter: client.ExporterImage,
		ExporterAttrs: map[string]string{
			"name": target,
		},
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	var cdAddress string
	if cd, ok := sb.(interface {
		ContainerdAddress() string
	}); !ok {
		return
	} else {
		cdAddress = cd.ContainerdAddress()
	}

	client, err := newContainerd(cdAddress)
	require.NoError(t, err)
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), "buildkit")

	img, err := client.ImageService().Get(ctx, target)
	require.NoError(t, err)

	desc, err := img.Config(ctx, client.ContentStore(), platforms.Default())
	require.NoError(t, err)

	dt, err = content.ReadBlob(ctx, client.ContentStore(), desc)
	require.NoError(t, err)

	var ociimg ocispec.Image
	err = json.Unmarshal(dt, &ociimg)
	require.NoError(t, err)

	require.Equal(t, "nobody", ociimg.Config.User)
}

func testCopyChown(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM busybox AS base
RUN mkdir -m 0777 /out
COPY --chown=daemon foo /
COPY --chown=1000:nogroup bar /baz
RUN stat -c "%U %G" /foo  > /out/fooowner
RUN stat -c "%u %G" /baz/sub  > /out/subowner
FROM scratch
COPY --from=base /out /
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("foo", []byte(`foo-contents`), 0600),
		fstest.CreateDir("bar", 0700),
		fstest.CreateFile("bar/sub", nil, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "fooowner"))
	require.NoError(t, err)
	require.Equal(t, "daemon daemon\n", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "subowner"))
	require.NoError(t, err)
	require.Equal(t, "1000 nogroup\n", string(dt))
}

func testCopyOverrideFiles(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch AS base
COPY sub sub
COPY sub sub
COPY files/foo.go dest/foo.go
COPY files/foo.go dest/foo.go
COPY files dest
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateDir("sub", 0700),
		fstest.CreateDir("sub/dir1", 0700),
		fstest.CreateDir("sub/dir1/dir2", 0700),
		fstest.CreateFile("sub/dir1/dir2/foo", []byte(`foo-contents`), 0600),
		fstest.CreateDir("files", 0700),
		fstest.CreateFile("files/foo.go", []byte(`foo.go-contents`), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "sub/dir1/dir2/foo"))
	require.NoError(t, err)
	require.Equal(t, "foo-contents", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "dest/foo.go"))
	require.NoError(t, err)
	require.Equal(t, "foo.go-contents", string(dt))
}

func testCopyVarSubstitution(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch AS base
ENV FOO bar
COPY $FOO baz
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("bar", []byte(`bar-contents`), 0600),
	)

	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "baz"))
	require.NoError(t, err)
	require.Equal(t, "bar-contents", string(dt))
}

func testCopyWildcards(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch AS base
COPY *.go /gofiles/
COPY f*.go foo2.go
COPY sub/* /subdest/
COPY sub/*/dir2/foo /subdest2/
COPY sub/*/dir2/foo /subdest3/bar
COPY . all/
COPY sub/dir1/ subdest4
COPY sub/dir1/. subdest5
COPY sub/dir1 subdest6
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("foo.go", []byte(`foo-contents`), 0600),
		fstest.CreateFile("bar.go", []byte(`bar-contents`), 0600),
		fstest.CreateDir("sub", 0700),
		fstest.CreateDir("sub/dir1", 0700),
		fstest.CreateDir("sub/dir1/dir2", 0700),
		fstest.CreateFile("sub/dir1/dir2/foo", []byte(`foo-contents`), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "gofiles/foo.go"))
	require.NoError(t, err)
	require.Equal(t, "foo-contents", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "gofiles/bar.go"))
	require.NoError(t, err)
	require.Equal(t, "bar-contents", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "foo2.go"))
	require.NoError(t, err)
	require.Equal(t, "foo-contents", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "subdest/dir1/dir2/foo"))
	require.NoError(t, err)
	require.Equal(t, "foo-contents", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "subdest2/foo"))
	require.NoError(t, err)
	require.Equal(t, "foo-contents", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "subdest3/bar"))
	require.NoError(t, err)
	require.Equal(t, "foo-contents", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "all/foo.go"))
	require.NoError(t, err)
	require.Equal(t, "foo-contents", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "subdest4/dir2/foo"))
	require.NoError(t, err)
	require.Equal(t, "foo-contents", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "subdest5/dir2/foo"))
	require.NoError(t, err)
	require.Equal(t, "foo-contents", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "subdest6/dir2/foo"))
	require.NoError(t, err)
	require.Equal(t, "foo-contents", string(dt))
}

func testDockerfileFromGit(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	gitDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(gitDir)

	dockerfile := `
FROM busybox AS build
RUN echo -n fromgit > foo	
FROM scratch
COPY --from=build foo bar
`

	err = ioutil.WriteFile(filepath.Join(gitDir, "Dockerfile"), []byte(dockerfile), 0600)
	require.NoError(t, err)

	err = runShell(gitDir,
		"git init",
		"git config --local user.email test",
		"git config --local user.name test",
		"git add Dockerfile",
		"git commit -m initial",
		"git branch first",
	)
	require.NoError(t, err)

	dockerfile += `
COPY --from=build foo bar2
`

	err = ioutil.WriteFile(filepath.Join(gitDir, "Dockerfile"), []byte(dockerfile), 0600)
	require.NoError(t, err)

	err = runShell(gitDir,
		"git add Dockerfile",
		"git commit -m second",
		"git update-server-info",
	)
	require.NoError(t, err)

	server := httptest.NewServer(http.FileServer(http.Dir(filepath.Join(gitDir))))
	defer server.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		FrontendAttrs: map[string]string{
			"context": server.URL + "/.git#first",
		},
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "bar"))
	require.NoError(t, err)
	require.Equal(t, "fromgit", string(dt))

	_, err = os.Stat(filepath.Join(destDir, "bar2"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	// second request from master branch contains both files
	destDir, err = ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		FrontendAttrs: map[string]string{
			"context": server.URL + "/.git",
		},
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
	}, nil)
	require.NoError(t, err)

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "bar"))
	require.NoError(t, err)
	require.Equal(t, "fromgit", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "bar2"))
	require.NoError(t, err)
	require.Equal(t, "fromgit", string(dt))
}

func testDockerfileFromHTTP(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	buf := bytes.NewBuffer(nil)
	w := tar.NewWriter(buf)

	writeFile := func(fn, dt string) {
		err := w.WriteHeader(&tar.Header{
			Name:     fn,
			Mode:     0600,
			Size:     int64(len(dt)),
			Typeflag: tar.TypeReg,
		})
		require.NoError(t, err)
		_, err = w.Write([]byte(dt))
		require.NoError(t, err)
	}

	writeFile("mydockerfile", `FROM scratch
COPY foo bar
`)

	writeFile("foo", "foo-contents")

	require.NoError(t, w.Flush())

	resp := httpserver.Response{
		Etag:    identity.NewID(),
		Content: buf.Bytes(),
	}

	server := httpserver.NewTestServer(map[string]httpserver.Response{
		"/myurl": resp,
	})
	defer server.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		FrontendAttrs: map[string]string{
			"context":  server.URL + "/myurl",
			"filename": "mydockerfile",
		},
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "bar"))
	require.NoError(t, err)
	require.Equal(t, "foo-contents", string(dt))
}

func testMultiStageImplicitFrom(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch
COPY --from=busybox /etc/passwd test
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "test"))
	require.NoError(t, err)
	require.Contains(t, string(dt), "root")

	// testing masked image will load actual stage

	dockerfile = []byte(`
FROM busybox AS golang
RUN mkdir /usr/bin && echo -n foo > /usr/bin/go

FROM scratch
COPY --from=golang /usr/bin/go go
`)

	dir, err = tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	destDir, err = ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "go"))
	require.NoError(t, err)
	require.Contains(t, string(dt), "foo")
}

func testMultiStageCaseInsensitive(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch AS STAge0
COPY foo bar
FROM scratch AS staGE1
COPY --from=staGE0 bar baz
FROM scratch
COPY --from=stage1 baz bax
`)
	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("foo", []byte("foo-contents"), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
		FrontendAttrs: map[string]string{
			"target": "Stage1",
		},
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "baz"))
	require.NoError(t, err)
	require.Contains(t, string(dt), "foo-contents")
}

func testLabels(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM scratch
LABEL foo=bar
`)
	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	target := "example.com/moby/dockerfilelabels:test"
	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		FrontendAttrs: map[string]string{
			"label:bar": "baz",
		},
		Exporter: client.ExporterImage,
		ExporterAttrs: map[string]string{
			"name": target,
		},
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	var cdAddress string
	if cd, ok := sb.(interface {
		ContainerdAddress() string
	}); !ok {
		t.Skip("only for containerd worker")
	} else {
		cdAddress = cd.ContainerdAddress()
	}

	client, err := newContainerd(cdAddress)
	require.NoError(t, err)
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), "buildkit")

	img, err := client.ImageService().Get(ctx, target)
	require.NoError(t, err)

	desc, err := img.Config(ctx, client.ContentStore(), platforms.Default())
	require.NoError(t, err)

	dt, err := content.ReadBlob(ctx, client.ContentStore(), desc)
	require.NoError(t, err)

	var ociimg ocispec.Image
	err = json.Unmarshal(dt, &ociimg)
	require.NoError(t, err)

	v, ok := ociimg.Config.Labels["foo"]
	require.True(t, ok)
	require.Equal(t, "bar", v)

	v, ok = ociimg.Config.Labels["bar"]
	require.True(t, ok)
	require.Equal(t, "baz", v)
}

func testCacheImportExport(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	registry, err := sb.NewRegistry()
	if errors.Cause(err) == integration.ErrorRequirements {
		t.Skip(err.Error())
	}
	require.NoError(t, err)

	dockerfile := []byte(`
FROM busybox AS base
COPY foo const
#RUN echo -n foobar > const
RUN cat /dev/urandom | head -c 100 | sha256sum > unique
FROM scratch
COPY --from=base const /
COPY --from=base unique /
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("foo", []byte("foobar"), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	target := registry + "/buildkit/testexportdf:latest"

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		ExportCache:       target,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "const"))
	require.NoError(t, err)
	require.Equal(t, "foobar", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "unique"))
	require.NoError(t, err)

	err = c.Prune(context.TODO(), nil, client.PruneAll)
	require.NoError(t, err)

	checkAllRemoved(t, c, sb)

	destDir, err = ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = f.Solve(context.TODO(), c, client.SolveOpt{
		FrontendAttrs: map[string]string{
			"cache-from": target,
		},
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}, nil)
	require.NoError(t, err)

	dt2, err := ioutil.ReadFile(filepath.Join(destDir, "const"))
	require.NoError(t, err)
	require.Equal(t, "foobar", string(dt2))

	dt2, err = ioutil.ReadFile(filepath.Join(destDir, "unique"))
	require.NoError(t, err)
	require.Equal(t, string(dt), string(dt2))

	destDir, err = ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)
}

func testReproducibleIDs(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM busybox
ENV foo=bar
COPY foo /
RUN echo bar > bar
`)
	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("foo", []byte("foo-contents"), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	target := "example.com/moby/dockerfileids:test"
	opt := client.SolveOpt{
		FrontendAttrs: map[string]string{},
		Exporter:      client.ExporterImage,
		ExporterAttrs: map[string]string{
			"name": target,
		},
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}

	_, err = f.Solve(context.TODO(), c, opt, nil)
	require.NoError(t, err)

	target2 := "example.com/moby/dockerfileids2:test"
	opt.ExporterAttrs["name"] = target2

	_, err = f.Solve(context.TODO(), c, opt, nil)
	require.NoError(t, err)

	var cdAddress string
	if cd, ok := sb.(interface {
		ContainerdAddress() string
	}); !ok {
		t.Skip("only for containerd worker")
	} else {
		cdAddress = cd.ContainerdAddress()
	}

	client, err := newContainerd(cdAddress)
	require.NoError(t, err)
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), "buildkit")

	img, err := client.ImageService().Get(ctx, target)
	require.NoError(t, err)
	img2, err := client.ImageService().Get(ctx, target2)
	require.NoError(t, err)

	require.Equal(t, img.Target, img2.Target)
}

func testImportExportReproducibleIDs(t *testing.T, sb integration.Sandbox) {
	var cdAddress string
	if cd, ok := sb.(interface {
		ContainerdAddress() string
	}); !ok {
		t.Skip("only for containerd worker")
	} else {
		cdAddress = cd.ContainerdAddress()
	}

	f := getFrontend(t, sb)

	registry, err := sb.NewRegistry()
	if errors.Cause(err) == integration.ErrorRequirements {
		t.Skip(err.Error())
	}
	require.NoError(t, err)

	dockerfile := []byte(`
FROM busybox
ENV foo=bar
COPY foo /
RUN echo bar > bar
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("foo", []byte("foobar"), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	target := "example.com/moby/dockerfileexpids:test"
	cacheTarget := registry + "/test/dockerfileexpids:cache"
	opt := client.SolveOpt{
		FrontendAttrs: map[string]string{},
		Exporter:      client.ExporterImage,
		ExportCache:   cacheTarget,
		ExporterAttrs: map[string]string{
			"name": target,
		},
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}

	ctd, err := newContainerd(cdAddress)
	require.NoError(t, err)
	defer ctd.Close()

	ctx := namespaces.WithNamespace(context.Background(), "buildkit")

	_, err = f.Solve(context.TODO(), c, opt, nil)
	require.NoError(t, err)

	img, err := ctd.ImageService().Get(ctx, target)
	require.NoError(t, err)

	err = ctd.ImageService().Delete(ctx, target)
	require.NoError(t, err)

	err = c.Prune(context.TODO(), nil, client.PruneAll)
	require.NoError(t, err)

	checkAllRemoved(t, c, sb)

	target2 := "example.com/moby/dockerfileexpids2:test"

	opt.ExporterAttrs["name"] = target2
	opt.FrontendAttrs["cache-from"] = cacheTarget

	_, err = f.Solve(context.TODO(), c, opt, nil)
	require.NoError(t, err)

	img2, err := ctd.ImageService().Get(ctx, target2)
	require.NoError(t, err)

	require.Equal(t, img.Target, img2.Target)
}

func testNoCache(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM busybox AS s0
RUN cat /dev/urandom | head -c 100 | sha256sum | tee unique
FROM busybox AS s1
RUN cat /dev/urandom | head -c 100 | sha256sum | tee unique2
FROM scratch
COPY --from=s0 unique /
COPY --from=s1 unique2 /
`)
	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	opt := client.SolveOpt{
		FrontendAttrs:     map[string]string{},
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}

	_, err = f.Solve(context.TODO(), c, opt, nil)
	require.NoError(t, err)

	destDir2, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	opt.FrontendAttrs["no-cache"] = ""
	opt.ExporterOutputDir = destDir2

	_, err = f.Solve(context.TODO(), c, opt, nil)
	require.NoError(t, err)

	unique1Dir1, err := ioutil.ReadFile(filepath.Join(destDir, "unique"))
	require.NoError(t, err)

	unique1Dir2, err := ioutil.ReadFile(filepath.Join(destDir2, "unique"))
	require.NoError(t, err)

	unique2Dir1, err := ioutil.ReadFile(filepath.Join(destDir, "unique2"))
	require.NoError(t, err)

	unique2Dir2, err := ioutil.ReadFile(filepath.Join(destDir2, "unique2"))
	require.NoError(t, err)

	require.NotEqual(t, string(unique1Dir1), string(unique1Dir2))
	require.NotEqual(t, string(unique2Dir1), string(unique2Dir2))

	destDir3, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	opt.FrontendAttrs["no-cache"] = "s1"
	opt.ExporterOutputDir = destDir3

	_, err = f.Solve(context.TODO(), c, opt, nil)
	require.NoError(t, err)

	unique1Dir3, err := ioutil.ReadFile(filepath.Join(destDir3, "unique"))
	require.NoError(t, err)

	unique2Dir3, err := ioutil.ReadFile(filepath.Join(destDir3, "unique2"))
	require.NoError(t, err)

	require.Equal(t, string(unique1Dir2), string(unique1Dir3))
	require.NotEqual(t, string(unique2Dir1), string(unique2Dir3))
}

func testPlatformArgsImplicit(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(fmt.Sprintf(`
FROM scratch AS build-%s
COPY foo bar
FROM build-${TARGETOS}
COPY foo2 bar2
`, runtime.GOOS))

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
		fstest.CreateFile("foo", []byte("d0"), 0600),
		fstest.CreateFile("foo2", []byte("d1"), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	opt := client.SolveOpt{
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}

	_, err = f.Solve(context.TODO(), c, opt, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "bar"))
	require.NoError(t, err)
	require.Equal(t, "d0", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "bar2"))
	require.NoError(t, err)
	require.Equal(t, "d1", string(dt))
}

func testPlatformArgsExplicit(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM --platform=$BUILDPLATFORM busybox AS build
ARG TARGETPLATFORM
ARG TARGETOS
RUN mkdir /out && echo -n $TARGETPLATFORM > /out/platform && echo -n $TARGETOS > /out/os
FROM scratch
COPY --from=build out .
`)

	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	opt := client.SolveOpt{
		Exporter: client.ExporterLocal,
		FrontendAttrs: map[string]string{
			"platform":           "darwin/ppc64le",
			"build-arg:TARGETOS": "freebsd",
		},
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}

	_, err = f.Solve(context.TODO(), c, opt, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "platform"))
	require.NoError(t, err)
	require.Equal(t, "darwin/ppc64le", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "os"))
	require.NoError(t, err)
	require.Equal(t, "freebsd", string(dt))
}

func testBuiltinArgs(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM busybox AS build
ARG FOO
ARG BAR
ARG BAZ=bazcontent
RUN echo -n $HTTP_PROXY::$NO_PROXY::$FOO::$BAR::$BAZ > /out
FROM scratch
COPY --from=build /out /

`)
	dir, err := tmpdir(
		fstest.CreateFile("Dockerfile", dockerfile, 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := client.New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	opt := client.SolveOpt{
		FrontendAttrs: map[string]string{
			"build-arg:FOO":        "foocontents",
			"build-arg:http_proxy": "hpvalue",
			"build-arg:NO_PROXY":   "npvalue",
		},
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}

	_, err = f.Solve(context.TODO(), c, opt, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "out"))
	require.NoError(t, err)
	require.Equal(t, "hpvalue::npvalue::foocontents::::bazcontent", string(dt))

	// repeat with changed default args should match the old cache
	destDir, err = ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	opt = client.SolveOpt{
		FrontendAttrs: map[string]string{
			"build-arg:FOO":        "foocontents",
			"build-arg:http_proxy": "hpvalue2",
		},
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}

	_, err = f.Solve(context.TODO(), c, opt, nil)
	require.NoError(t, err)

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "out"))
	require.NoError(t, err)
	require.Equal(t, "hpvalue::npvalue::foocontents::::bazcontent", string(dt))

	// changing actual value invalidates cache
	destDir, err = ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	opt = client.SolveOpt{
		FrontendAttrs: map[string]string{
			"build-arg:FOO":        "foocontents2",
			"build-arg:http_proxy": "hpvalue2",
		},
		Exporter:          client.ExporterLocal,
		ExporterOutputDir: destDir,
		LocalDirs: map[string]string{
			builder.LocalNameDockerfile: dir,
			builder.LocalNameContext:    dir,
		},
	}

	_, err = f.Solve(context.TODO(), c, opt, nil)
	require.NoError(t, err)

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "out"))
	require.NoError(t, err)
	require.Equal(t, "hpvalue2::::foocontents2::::bazcontent", string(dt))
}

func tmpdir(appliers ...fstest.Applier) (string, error) {
	tmpdir, err := ioutil.TempDir("", "buildkit-dockerfile")
	if err != nil {
		return "", err
	}
	if err := fstest.Apply(appliers...).Apply(tmpdir); err != nil {
		return "", err
	}
	return tmpdir, nil
}

func runShell(dir string, cmds ...string) error {
	for _, args := range cmds {
		cmd := exec.Command("sh", "-c", args)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			return errors.Wrapf(err, "error running %v", args)
		}
	}
	return nil
}

func checkAllRemoved(t *testing.T, c *client.Client, sb integration.Sandbox) {
	retries := 0
	for {
		require.True(t, 20 > retries)
		retries++
		du, err := c.DiskUsage(context.TODO())
		require.NoError(t, err)
		if len(du) > 0 {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}
}

func newContainerd(cdAddress string) (*containerd.Client, error) {
	return containerd.New(cdAddress, containerd.WithTimeout(60*time.Second))
}

func dfCmdArgs(ctx, dockerfile, args string) (string, string) {
	traceFile := filepath.Join(os.TempDir(), "trace"+identity.NewID())
	return fmt.Sprintf("build --progress=plain %s --local context=%s --local dockerfile=%s --trace=%s", args, ctx, dockerfile, traceFile), traceFile
}

type builtinFrontend struct{}

var _ frontend = &builtinFrontend{}

func (f *builtinFrontend) Solve(ctx context.Context, c *client.Client, opt client.SolveOpt, statusChan chan *client.SolveStatus) (*client.SolveResponse, error) {
	opt.Frontend = "dockerfile.v0"
	return c.Solve(ctx, nil, opt, statusChan)
}

func (f *builtinFrontend) DFCmdArgs(ctx, dockerfile string) (string, string) {
	return dfCmdArgs(ctx, dockerfile, "--frontend dockerfile.v0")
}

func (f *builtinFrontend) RequiresBuildctl(t *testing.T) {}

type clientFrontend struct{}

var _ frontend = &clientFrontend{}

func (f *clientFrontend) Solve(ctx context.Context, c *client.Client, opt client.SolveOpt, statusChan chan *client.SolveStatus) (*client.SolveResponse, error) {
	return c.Build(ctx, opt, "", builder.Build, statusChan)
}

func (f *clientFrontend) DFCmdArgs(ctx, dockerfile string) (string, string) {
	return "", ""
}
func (f *clientFrontend) RequiresBuildctl(t *testing.T) {
	t.Skip()
}

type gatewayFrontend struct {
	gw string
}

var _ frontend = &gatewayFrontend{}

func (f *gatewayFrontend) Solve(ctx context.Context, c *client.Client, opt client.SolveOpt, statusChan chan *client.SolveStatus) (*client.SolveResponse, error) {
	opt.Frontend = "gateway.v0"
	if opt.FrontendAttrs == nil {
		opt.FrontendAttrs = make(map[string]string)
	}
	opt.FrontendAttrs["source"] = f.gw
	return c.Solve(ctx, nil, opt, statusChan)
}

func (f *gatewayFrontend) DFCmdArgs(ctx, dockerfile string) (string, string) {
	return dfCmdArgs(ctx, dockerfile, "--frontend gateway.v0 --frontend-opt=source="+f.gw)
}

func (f *gatewayFrontend) RequiresBuildctl(t *testing.T) {}

func getFrontend(t *testing.T, sb integration.Sandbox) frontend {
	v := sb.Value("frontend")
	require.NotNil(t, v)
	fn, ok := v.(frontend)
	require.True(t, ok)
	return fn
}
