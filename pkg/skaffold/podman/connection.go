package podman

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"

	"github.com/containers/common/pkg/config"
)

// GetConnection queries the podman config for the default destination
func GetConnection(connName string) (config.Destination, error) {
	var useDefault bool
	if connName == "" {
		useDefault = true
	}

	cfg, err := config.ReadCustomConfig()
	if err != nil {
		return config.Destination{}, fmt.Errorf("podman reading custom config: %w")
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

// OverrideDockerHost overrides the DOCKER_HOST Environment variable.
// If the connection is via ssh and with a key, the function will add it to the ssh-agent
func OverrideDockerHost(conn config.Destination) error {
	os.Setenv("DOCKER_HOST", conn.URI)
	uri, err := url.Parse(conn.URI)
	if err != nil {
		return err
	}
	if uri.Scheme == "ssh" && conn.Identity != "" {
		return addSSHKeyToAgent(conn.Identity)
	}
	return nil
}

// TODO: This works only on unix systems
func addSSHKeyToAgent(keyPath string) error {
	err := exec.Command("ssh-agent").Run()
	if err != nil {
		return err
	}
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return errors.New("SSH_AUTH_SOCK environment variable is empty")
	}

	err = exec.Command("ssh-add", keyPath).Run()
	if err != nil {
		return fmt.Errorf("adding podman ssh key to ssh agent: %w", err)
	}
	return nil
}
