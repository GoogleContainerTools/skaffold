package testhelpers

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/google/go-containerregistry/pkg/registry"
)

type DockerRegistry struct {
	Host            string
	Port            string
	Name            string
	server          *httptest.Server
	DockerDirectory string
	username        string
	password        string
	regHandler      http.Handler
	authnHandler    http.Handler
	imagePrivileges map[string]ImagePrivileges // map from an imageName to its permissions
}

type RegistryOption func(registry *DockerRegistry)

// WithSharedHandler allows two instances to share the same data by re-using the registry handler.
// Use an authenticated registry to write to a read-only unauthenticated registry.
func WithSharedHandler(handler http.Handler) RegistryOption {
	return func(registry *DockerRegistry) {
		registry.regHandler = handler
	}
}

// WithImagePrivileges enables the execution of read/write access validations based on the image name
func WithImagePrivileges() RegistryOption {
	var permissions = make(map[string]ImagePrivileges)
	return func(registry *DockerRegistry) {
		registry.imagePrivileges = permissions
	}
}

// WithAuth adds credentials to registry. Omitting will make the registry read-only
func WithAuth(dockerConfigDir string) RegistryOption {
	return func(r *DockerRegistry) {
		r.username = RandString(10)
		r.password = RandString(10)
		r.DockerDirectory = dockerConfigDir
	}
}

func NewDockerRegistry(ops ...RegistryOption) *DockerRegistry {
	dockerRegistry := &DockerRegistry{
		Name: "test-registry-" + RandString(10),
	}

	for _, op := range ops {
		op(dockerRegistry)
	}

	return dockerRegistry
}

// BasicAuth wraps a handler, allowing requests with matching username and password headers, otherwise rejecting with a 401
func BasicAuth(handler http.Handler, username, password, realm string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != username || pass != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			w.WriteHeader(401)
			_, _ = w.Write([]byte("Unauthorized.\n"))
			return
		}
		handler.ServeHTTP(w, r)
	})
}

// ReadOnly wraps a handler, allowing only GET and HEAD requests, otherwise rejecting with a 405
func ReadOnly(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if !isReadRequest(request) {
			response.WriteHeader(405)
			_, _ = response.Write([]byte("Method Not Allowed.\n"))
			return
		}

		handler.ServeHTTP(response, request)
	})
}

func delegator(basicAuthHandler http.Handler, regHandler http.Handler, permissions map[string]ImagePrivileges) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var (
			image ImagePrivileges
			ok    bool
		)
		if image, ok = permissions[extractImageName(request.URL.Path)]; !ok {
			basicAuthHandler.ServeHTTP(response, request)
			return
		}
		if isReadRequest(request) {
			if !image.readable {
				response.WriteHeader(401)
				_, _ = response.Write([]byte("Unauthorized.\n"))
				return
			}
		} else { // assume write request
			if !image.writable {
				response.WriteHeader(401)
				_, _ = response.Write([]byte("Unauthorized.\n"))
				return
			}
		}
		regHandler.ServeHTTP(response, request)
	})
}

// Start creates a docker registry following these rules:
//   - Shared handler will be used, otherwise a new one will be created
//   - By default the shared handler will be wrapped with a read only handler
//   - In case credentials are configured, the shared handler will be wrapped with a basic authentication handler and
//     if any image privileges were set, then the custom handler will be used to wrap the auth handler.
func (r *DockerRegistry) Start(t *testing.T) {
	t.Helper()

	r.Host = DockerHostname(t)

	// create registry handler, if not re-using a shared one
	if r.regHandler == nil {
		// change to os.Stderr for verbose output
		logger := registry.Logger(log.New(io.Discard, "registry ", log.Lshortfile))
		r.regHandler = registry.New(logger)
	}

	// wrap registry handler with authentication handler, defaulting to read-only
	r.authnHandler = ReadOnly(r.regHandler)
	if r.username != "" {
		if r.imagePrivileges != nil {
			// wrap registry handler with basic auth
			basicAuthHandler := BasicAuth(r.regHandler, r.username, r.password, "registry")
			r.authnHandler = delegator(basicAuthHandler, r.regHandler, r.imagePrivileges)
		} else {
			// wrap registry handler basic auth
			r.authnHandler = BasicAuth(r.regHandler, r.username, r.password, "registry")
		}
	}

	// listen on specific interface with random port, relying on authorization to prevent untrusted writes
	listenIP := "127.0.0.1"
	if r.Host != "localhost" {
		listenIP = r.Host
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(listenIP, "0"))
	AssertNil(t, err)

	r.server = &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: r.authnHandler}, //nolint
	}

	r.server.Start()

	tcpAddr := r.server.Listener.Addr().(*net.TCPAddr)

	r.Port = strconv.Itoa(tcpAddr.Port)
	t.Logf("run registry on %s:%s", r.Host, r.Port)

	if r.username != "" {
		// Write Docker config and configure auth headers
		writeDockerConfig(t, r.DockerDirectory, r.Host, r.Port, r.EncodedAuth())
	}
}

func (r *DockerRegistry) Stop(t *testing.T) {
	t.Helper()
	t.Log("stop registry")

	r.server.Close()
}

func (r *DockerRegistry) RepoName(name string) string {
	return r.Host + ":" + r.Port + "/" + name
}

func (r *DockerRegistry) EncodedLabeledAuth() string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`{"username":"%s","password":"%s"}`, r.username, r.password)))
}

// setImagePrivileges sets the given image name with the provided ImagePrivileges if WithImagePrivileges was called during registry creation.
// For example SetPrivilegesToImage("my-image", NewImagePrivileges(Readable, Writable)) will save "my-image" as a
// readable and writable image into the registry.
func (r *DockerRegistry) setImagePrivileges(imageName string, privilege ImagePrivileges) {
	if r.imagePrivilegesEnabled() {
		r.imagePrivileges[imageName] = privilege
	}
}

// SetReadOnly set the given image name to be readable when the ImagePrivileges feature was enabled
// Returns RepoName(imageName)
func (r *DockerRegistry) SetReadOnly(imageName string) string {
	r.setImagePrivileges(imageName, NewImagePrivileges(Readable))
	return r.RepoName(imageName)
}

// SetWriteOnly set the given image name to be writable when the ImagePrivileges feature was enabled
// Returns RepoName(imageName)
func (r *DockerRegistry) SetWriteOnly(imageName string) string {
	r.setImagePrivileges(imageName, NewImagePrivileges(Writable))
	return r.RepoName(imageName)
}

// SetReadWrite set the given image name to be readable and writable when the ImagePrivileges feature was enabled
// Returns RepoName(imageName)
func (r *DockerRegistry) SetReadWrite(imageName string) string {
	r.setImagePrivileges(imageName, NewImagePrivileges(Readable, Writable))
	return r.RepoName(imageName)
}

// SetInaccessible set the given image name to do not have any access when the ImagePrivileges feature was enabled
// Returns RepoName(imageName)
func (r *DockerRegistry) SetInaccessible(imageName string) string {
	r.setImagePrivileges(imageName, NewImagePrivileges())
	return r.RepoName(imageName)
}

func (r *DockerRegistry) imagePrivilegesEnabled() bool {
	return r.imagePrivileges != nil
}

func isReadRequest(req *http.Request) bool {
	return req.Method == "GET" || req.Method == "HEAD"
}

// DockerHostname discovers the appropriate registry hostname.
// For test to run where "localhost" is not the daemon host, a `insecure-registries` entry of `<host net>/<mask>` with a range that contains the host's non-loopback IP.
// For Docker Desktop, this can be set here: https://docs.docker.com/docker-for-mac/#docker-engine
// Otherwise, its set in the daemon.json: https://docs.docker.com/engine/reference/commandline/dockerd/#daemon-configuration-file
// If the entry is not found, the fallback is "localhost"
func DockerHostname(t *testing.T) string {
	dockerCli := DockerCli(t)

	// query daemon for insecure-registry network ranges
	var insecureRegistryNets []*net.IPNet
	daemonInfo, err := dockerCli.Info(context.TODO())
	if err != nil {
		t.Fatalf("unable to fetch client.DockerInfo: %s", err)
	}
	for _, ipnet := range daemonInfo.RegistryConfig.InsecureRegistryCIDRs {
		insecureRegistryNets = append(insecureRegistryNets, (*net.IPNet)(ipnet))
	}

	// search for non-loopback interface IPs contained by a insecure-registry range
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("unable to fetch interfaces: %s", err)
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			t.Fatalf("unable to fetch interface address: %s", err)
		}

		for _, addr := range addrs {
			var interfaceIP net.IP
			switch interfaceAddr := addr.(type) {
			case *net.IPAddr:
				interfaceIP = interfaceAddr.IP
			case *net.IPNet:
				interfaceIP = interfaceAddr.IP
			}

			// ignore blanks and loopbacks
			if interfaceIP == nil || interfaceIP.IsLoopback() {
				continue
			}

			// return first matching interface IP
			for _, regNet := range insecureRegistryNets {
				if regNet.Contains(interfaceIP) {
					return interfaceIP.String()
				}
			}
		}
	}

	// Fallback to localhost, only works for Linux using --network=host
	return "localhost"
}

func (r *DockerRegistry) EncodedAuth() string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", r.username, r.password)))
}

func writeDockerConfig(t *testing.T, configDir, host, port, auth string) {
	AssertNil(t, os.WriteFile(
		filepath.Join(configDir, "config.json"),
		[]byte(fmt.Sprintf(`{
			  "auths": {
			    "%s:%s": {
			      "auth": "%s"
			    }
			  }
			}
			`, host, port, auth)),
		0600,
	))
}
