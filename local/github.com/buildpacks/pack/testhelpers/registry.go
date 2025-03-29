package testhelpers

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockerregistry "github.com/docker/docker/api/types/registry"
	"github.com/docker/go-connections/nat"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"golang.org/x/crypto/bcrypt"

	"github.com/buildpacks/pack/pkg/archive"
)

var registryContainerNames = map[string]string{
	"linux":   "library/registry:2",
	"windows": "micahyoung/registry:latest",
}

type TestRegistryConfig struct {
	runRegistryName       string
	registryContainerName string
	RunRegistryHost       string
	RunRegistryPort       string
	DockerConfigDir       string
	username              string
	password              string
}

func RegistryHost(host, port string) string {
	return fmt.Sprintf("%s:%s", host, port)
}

func CreateRegistryFixture(t *testing.T, tmpDir, fixturePath string) string {
	t.Helper()
	// copy fixture to temp dir
	registryFixtureCopy := filepath.Join(tmpDir, "registryCopy")

	RecursiveCopyNow(t, fixturePath, registryFixtureCopy)

	// git init that dir
	repository, err := git.PlainInit(registryFixtureCopy, false)
	AssertNil(t, err)

	// git add . that dir
	worktree, err := repository.Worktree()
	AssertNil(t, err)

	_, err = worktree.Add(".")
	AssertNil(t, err)

	// git commit that dir
	commit, err := worktree.Commit("first", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "John Doe",
			Email: "john@doe.org",
			When:  time.Now(),
		},
	})
	AssertNil(t, err)

	_, err = repository.CommitObject(commit)
	AssertNil(t, err)

	return registryFixtureCopy
}

func RunRegistry(t *testing.T) *TestRegistryConfig {
	t.Log("run registry")
	t.Helper()

	runRegistryName := "test-registry-" + RandString(10)
	username := RandString(10)
	password := RandString(10)

	runRegistryHost, runRegistryPort, registryCtnrName := startRegistry(t, runRegistryName, username, password)
	dockerConfigDir := setupDockerConfigWithAuth(t, username, password, runRegistryHost, runRegistryPort)

	registryConfig := &TestRegistryConfig{
		runRegistryName:       runRegistryName,
		registryContainerName: registryCtnrName,
		RunRegistryHost:       runRegistryHost,
		RunRegistryPort:       runRegistryPort,
		DockerConfigDir:       dockerConfigDir,
		username:              username,
		password:              password,
	}

	waitForRegistryToBeAvailable(t, registryConfig)

	return registryConfig
}

func waitForRegistryToBeAvailable(t *testing.T, registryConfig *TestRegistryConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		_, err := registryConfig.RegistryCatalog()
		if err == nil {
			break
		}

		ctxErr := ctx.Err()
		if ctxErr != nil {
			t.Fatal("registry not ready:", ctxErr.Error(), ":", err.Error())
		}

		time.Sleep(500 * time.Microsecond)
	}
}

func (rc *TestRegistryConfig) AuthConfig() dockerregistry.AuthConfig {
	return dockerregistry.AuthConfig{
		Username:      rc.username,
		Password:      rc.password,
		ServerAddress: RegistryHost(rc.RunRegistryHost, rc.RunRegistryPort),
	}
}

func (rc *TestRegistryConfig) Login(t *testing.T, username string, password string) {
	Eventually(t, func() bool {
		_, err := dockerCli(t).RegistryLogin(context.Background(), dockerregistry.AuthConfig{
			Username:      username,
			Password:      password,
			ServerAddress: RegistryHost(rc.RunRegistryHost, rc.RunRegistryPort),
		})
		return err == nil
	}, 100*time.Millisecond, 10*time.Second)
}

func startRegistry(t *testing.T, runRegistryName, username, password string) (string, string, string) {
	ctx := context.Background()

	daemonInfo, err := dockerCli(t).Info(ctx)
	AssertNil(t, err)

	registryContainerName := registryContainerNames[daemonInfo.OSType]
	AssertNil(t, PullImageWithAuth(dockerCli(t), registryContainerName, ""))

	htpasswdTar := generateHtpasswd(t, username, password)
	defer htpasswdTar.Close()

	ctr, err := dockerCli(t).ContainerCreate(ctx, &dockercontainer.Config{
		Image:  registryContainerName,
		Labels: map[string]string{"author": "pack"},
		Env: []string{
			"REGISTRY_AUTH=htpasswd",
			"REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm",
			"REGISTRY_AUTH_HTPASSWD_PATH=/registry_test_htpasswd",
		},
	}, &dockercontainer.HostConfig{
		AutoRemove: true,
		PortBindings: nat.PortMap{
			"5000/tcp": []nat.PortBinding{{HostPort: "0"}},
		},
	}, nil, nil, runRegistryName)
	AssertNil(t, err)
	err = dockerCli(t).CopyToContainer(ctx, ctr.ID, "/", htpasswdTar, dockercontainer.CopyToContainerOptions{})
	AssertNil(t, err)

	err = dockerCli(t).ContainerStart(ctx, ctr.ID, dockercontainer.StartOptions{})
	AssertNil(t, err)

	runRegistryPort, err := waitForPortBinding(t, ctr.ID, "5000/tcp", 30*time.Second)
	AssertNil(t, err)

	runRegistryHost := DockerHostname(t)
	return runRegistryHost, runRegistryPort, registryContainerName
}

func waitForPortBinding(t *testing.T, containerID, portSpec string, duration time.Duration) (binding string, err error) {
	t.Helper()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			inspect, err := dockerCli(t).ContainerInspect(context.TODO(), containerID)
			if err != nil {
				return "", err
			}

			portBindings := inspect.NetworkSettings.Ports[nat.Port(portSpec)]
			if len(portBindings) > 0 {
				return portBindings[0].HostPort, nil
			}
		case <-timer.C:
			t.Fatalf("timeout waiting for port binding: %v", duration)
		}
	}
}

func DockerHostname(t *testing.T) string {
	dockerCli := dockerCli(t)

	daemonHost := dockerCli.DaemonHost()
	u, err := url.Parse(daemonHost)
	if err != nil {
		t.Fatalf("unable to parse URI client.DaemonHost: %s", err)
	}

	switch u.Scheme {
	// DOCKER_HOST is usually remote so always use its hostname/IP
	// Note: requires "insecure-registries" CIDR entry on Daemon config
	case "tcp":
		return u.Hostname()

	// if DOCKER_HOST is non-tcp, we assume that we are
	// talking to the daemon over a local pipe.
	default:
		daemonInfo, err := dockerCli.Info(context.TODO())
		if err != nil {
			t.Fatalf("unable to fetch client.DockerInfo: %s", err)
		}

		if daemonInfo.OSType == "windows" {
			// try to lookup the host IP by helper domain name (https://docs.docker.com/docker-for-windows/networking/#use-cases-and-workarounds)
			// Note: pack appears to not support /etc/hosts-based insecure-registries
			addrs, err := net.LookupHost("host.docker.internal")
			if err != nil {
				t.Fatalf("unknown address response: %+v %s", addrs, err)
			}
			if len(addrs) != 1 {
				t.Fatalf("ambiguous address response: %v", addrs)
			}
			return addrs[0]
		}

		// Linux can use --network=host so always use "localhost"
		return "localhost"
	}
}

func generateHtpasswd(t *testing.T, username string, password string) io.ReadCloser {
	// https://docs.docker.com/registry/deploying/#restricting-access
	// HTPASSWD format: https://github.com/foomo/htpasswd/blob/e3a90e78da9cff06a83a78861847aa9092cbebdd/hashing.go#L23
	passwordBytes, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	reader := archive.CreateSingleFileTarReader("/registry_test_htpasswd", username+":"+string(passwordBytes))
	return reader
}

func setupDockerConfigWithAuth(t *testing.T, username string, password string, runRegistryHost string, runRegistryPort string) string {
	dockerConfigDir, err := os.MkdirTemp("", "pack.test.docker.config.dir")
	AssertNil(t, err)

	AssertNil(t, os.WriteFile(filepath.Join(dockerConfigDir, "config.json"), []byte(fmt.Sprintf(`{
			  "auths": {
			    "%s": {
			      "auth": "%s"
			    }
			  }
			}
			`, RegistryHost(runRegistryHost, runRegistryPort), encodedUserPass(username, password))), 0666))
	return dockerConfigDir
}

func encodedUserPass(username string, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
}

func (rc *TestRegistryConfig) RmRegistry(t *testing.T) {
	rc.StopRegistry(t)

	t.Log("remove registry")
	t.Helper()

	id := ImageID(t, rc.registryContainerName)
	DockerRmi(dockerCli(t), id)
}

func (rc *TestRegistryConfig) StopRegistry(t *testing.T) {
	t.Log("stop registry")
	t.Helper()
	dockerCli(t).ContainerKill(context.Background(), rc.runRegistryName, "SIGKILL")

	err := os.RemoveAll(rc.DockerConfigDir)
	AssertNil(t, err)
}

func (rc *TestRegistryConfig) RepoName(name string) string {
	return RegistryHost(rc.RunRegistryHost, rc.RunRegistryPort) + "/" + name
}

func (rc *TestRegistryConfig) RegistryAuth() string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`{"username":"%s","password":"%s"}`, rc.username, rc.password)))
}

func (rc *TestRegistryConfig) RegistryCatalog() (string, error) {
	return HTTPGetE(fmt.Sprintf("http://%s/v2/_catalog", RegistryHost(rc.RunRegistryHost, rc.RunRegistryPort)), map[string]string{
		"Authorization": "Basic " + encodedUserPass(rc.username, rc.password),
	})
}
