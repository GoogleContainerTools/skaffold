// +build dfrunmount

package dockerfile

import (
	"context"
	"os"
	"testing"

	"github.com/containerd/continuity/fs/fstest"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/moby/buildkit/util/testutil/integration"
	"github.com/stretchr/testify/require"
)

var mountTests = []integration.Test{
	testMountContext,
}

func init() {
	allTests = append(allTests, mountTests...)
}

func testMountContext(t *testing.T, sb integration.Sandbox) {
	f := getFrontend(t, sb)

	dockerfile := []byte(`
FROM busybox
RUN --mount=target=/context [ "$(cat /context/testfile)" == "contents0" ]
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
