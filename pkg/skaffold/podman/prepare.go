package podman

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/containers/buildah"
	"github.com/containers/common/pkg/config"
	"github.com/containers/storage/pkg/unshare"
)

func Prepare(ctx context.Context, runCtx *runcontext.RunContext) error {
	buildCfgs := runCtx.BuildConfigs()

	for _, cfg := range buildCfgs {
		if cfg.LocalBuild != nil && cfg.LocalBuild.Podman != nil {
			if util.Runtime() == "linux" {
				reexec()
				return nil
			}
			// not running on linux, so use podman socket
			conn, err := getConnection(cfg.LocalBuild.Podman.Connection)
			if err != nil {
				return err
			}
			err = overrideDockerHost(ctx, conn)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// getConnection queries the podman config for the default destination
func getConnection(connName string) (config.Destination, error) {
	var useDefault bool
	if connName == "" {
		useDefault = true
	}

	cfg, err := config.ReadCustomConfig()
	if err != nil {
		return config.Destination{}, fmt.Errorf("podman reading custom config: %w", err)
	}

	if useDefault {
		defaultConn, exists := cfg.Engine.ServiceDestinations[cfg.Engine.ActiveService]
		if !exists {
			return defaultConn, errors.New("No default connection in podman")
		}
		return defaultConn, nil
	}

	conn, exists := cfg.Engine.ServiceDestinations[connName]
	if !exists {
		return conn, fmt.Errorf("getting connection %v: connection doesnt exist", connName)
	}
	return conn, nil
}

// overrideDockerHost overrides the DOCKER_HOST Environment variable.
// If the connection is via ssh and with a key, the function will add it to the ssh-agent
func overrideDockerHost(ctx context.Context, conn config.Destination) error {
	os.Setenv("DOCKER_HOST", conn.URI)
	uri, err := url.Parse(conn.URI)
	if err != nil {
		return fmt.Errorf("parsing podman connection uri: %w", err)
	}
	if uri.Scheme == "ssh" && conn.Identity != "" {
		return addSSHKeyToAgent(ctx, conn.Identity)
	}
	return nil
}

// TODO: This works only on unix systems
func addSSHKeyToAgent(ctx context.Context, keyPath string) error {
	err := util.DefaultExecCommand.RunCmd(ctx, exec.Command("ssh-agent"))
	if err != nil {
		return err
	}
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return errors.New("SSH_AUTH_SOCK environment variable is empty")
	}

	err = util.DefaultExecCommand.RunCmd(ctx, exec.Command("ssh-add", keyPath))
	if err != nil {
		return fmt.Errorf("adding podman ssh key to ssh agent: %w", err)
	}
	return nil
}

// TODO: reexec only when detecting buildah and running natively on linux
// Debugging is kinda hard
// have to use `buildah unshare` before launching
func reexec() {
	if buildah.InitReexec() {
		return
	}
	unshare.MaybeReexecUsingUserNamespace(false)
}
