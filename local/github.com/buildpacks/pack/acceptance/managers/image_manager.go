//go:build acceptance

package managers

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	h "github.com/buildpacks/pack/testhelpers"
)

var DefaultDuration = 10 * time.Second

type ImageManager struct {
	testObject *testing.T
	assert     h.AssertionManager
	dockerCli  client.CommonAPIClient
}

func NewImageManager(t *testing.T, dockerCli client.CommonAPIClient) ImageManager {
	return ImageManager{
		testObject: t,
		assert:     h.NewAssertionManager(t),
		dockerCli:  dockerCli,
	}
}

func (im ImageManager) CleanupImages(imageNames ...string) {
	im.testObject.Helper()
	err := h.DockerRmi(im.dockerCli, imageNames...)
	if err != nil {
		im.testObject.Logf("%s: Failed to remove image from %s", err, imageNames)
	}
}

func (im ImageManager) InspectLocal(image string) (dockertypes.ImageInspect, error) {
	im.testObject.Helper()
	inspect, _, err := im.dockerCli.ImageInspectWithRaw(context.Background(), image)
	return inspect, err
}

func (im ImageManager) GetImageID(image string) string {
	im.testObject.Helper()
	inspect, err := im.InspectLocal(image)
	im.assert.Nil(err)
	return inspect.ID
}

func (im ImageManager) HostOS() string {
	im.testObject.Helper()
	daemonInfo, err := im.dockerCli.Info(context.Background())
	im.assert.Nil(err)
	return daemonInfo.OSType
}

func (im ImageManager) TagImage(image, ref string) {
	im.testObject.Helper()
	err := im.dockerCli.ImageTag(context.Background(), image, ref)
	im.assert.Nil(err)
}

func (im ImageManager) PullImage(image, registryAuth string) {
	im.testObject.Helper()
	err := h.PullImageWithAuth(im.dockerCli, image, registryAuth)
	im.assert.Nil(err)
}

func (im ImageManager) ExposePortOnImage(image, containerName string) TestContainer {
	im.testObject.Helper()
	ctx := context.Background()

	ctr, err := im.dockerCli.ContainerCreate(ctx, &container.Config{
		Image:        image,
		ExposedPorts: map[nat.Port]struct{}{"8080/tcp": {}},
		Healthcheck:  nil,
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"8080/tcp": []nat.PortBinding{{}},
		},
		AutoRemove: true,
	}, nil, nil, containerName)
	im.assert.Nil(err)

	err = im.dockerCli.ContainerStart(ctx, ctr.ID, container.StartOptions{})
	im.assert.Nil(err)
	return TestContainer{
		testObject: im.testObject,
		dockerCli:  im.dockerCli,
		assert:     im.assert,
		name:       containerName,
		id:         ctr.ID,
	}
}

func (im ImageManager) CreateContainer(name string) TestContainer {
	im.testObject.Helper()
	containerName := "test-" + h.RandString(10)
	ctr, err := im.dockerCli.ContainerCreate(context.Background(), &container.Config{
		Image: name,
	}, nil, nil, nil, containerName)
	im.assert.Nil(err)

	return TestContainer{
		testObject: im.testObject,
		dockerCli:  im.dockerCli,
		assert:     im.assert,
		name:       containerName,
		id:         ctr.ID,
	}
}

type TestContainer struct {
	testObject *testing.T
	dockerCli  client.CommonAPIClient
	assert     h.AssertionManager
	name       string
	id         string
}

func (t TestContainer) RunWithOutput() string {
	t.testObject.Helper()
	var b bytes.Buffer
	err := h.RunContainer(context.Background(), t.dockerCli, t.id, &b, &b)
	t.assert.Nil(err)
	return b.String()
}

func (t TestContainer) Cleanup() {
	t.testObject.Helper()
	t.dockerCli.ContainerKill(context.Background(), t.name, "SIGKILL")
	t.dockerCli.ContainerRemove(context.Background(), t.name, container.RemoveOptions{Force: true})
}

func (t TestContainer) WaitForResponse(duration time.Duration) string {
	t.testObject.Helper()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(duration)
	defer timer.Stop()

	appURI := fmt.Sprintf("http://%s", h.RegistryHost(h.DockerHostname(t.testObject), t.hostPort()))
	for {
		select {
		case <-ticker.C:
			resp, err := h.HTTPGetE(appURI, map[string]string{})
			if err != nil {
				break
			}
			return resp
		case <-timer.C:
			t.testObject.Fatalf("timeout waiting for response: %v", duration)
		}
	}
}

func (t TestContainer) hostPort() string {
	t.testObject.Helper()
	i, err := t.dockerCli.ContainerInspect(context.Background(), t.name)
	t.assert.Nil(err)
	for _, port := range i.NetworkSettings.Ports {
		for _, binding := range port {
			return binding.HostPort
		}
	}

	t.testObject.Fatalf("Failed to fetch host port for %s: no ports exposed", t.name)
	return ""
}
