//go:build acceptance

package assertions

import (
	"fmt"
	"testing"
	"time"

	"github.com/buildpacks/pack/acceptance/managers"
	h "github.com/buildpacks/pack/testhelpers"
)

type ImageAssertionManager struct {
	testObject   *testing.T
	assert       h.AssertionManager
	imageManager managers.ImageManager
	registry     *h.TestRegistryConfig
}

func NewImageAssertionManager(t *testing.T, imageManager managers.ImageManager, registry *h.TestRegistryConfig) ImageAssertionManager {
	return ImageAssertionManager{
		testObject:   t,
		assert:       h.NewAssertionManager(t),
		imageManager: imageManager,
		registry:     registry,
	}
}

func (a ImageAssertionManager) ExistsLocally(name string) {
	a.testObject.Helper()
	_, err := a.imageManager.InspectLocal(name)
	a.assert.Nil(err)
}

func (a ImageAssertionManager) NotExistsLocally(name string) {
	a.testObject.Helper()
	_, err := a.imageManager.InspectLocal(name)
	a.assert.ErrorContains(err, "No such image")
}

func (a ImageAssertionManager) HasBaseImage(image, base string) {
	a.testObject.Helper()
	imageInspect, err := a.imageManager.InspectLocal(image)
	a.assert.Nil(err)
	baseInspect, err := a.imageManager.InspectLocal(base)
	a.assert.Nil(err)
	for i, layer := range baseInspect.RootFS.Layers {
		a.assert.Equal(imageInspect.RootFS.Layers[i], layer)
	}
}

func (a ImageAssertionManager) HasCreateTime(image string, expectedTime time.Time) {
	a.testObject.Helper()
	inspect, err := a.imageManager.InspectLocal(image)
	a.assert.Nil(err)
	actualTime, err := time.Parse("2006-01-02T15:04:05Z", inspect.Created)
	a.assert.Nil(err)
	a.assert.TrueWithMessage(actualTime.Sub(expectedTime) < 5*time.Second && expectedTime.Sub(actualTime) < 5*time.Second, fmt.Sprintf("expected image create time %s to match expected time %s", actualTime, expectedTime))
}

func (a ImageAssertionManager) HasLabelContaining(image, label, data string) {
	a.testObject.Helper()
	inspect, err := a.imageManager.InspectLocal(image)
	a.assert.Nil(err)
	label, ok := inspect.Config.Labels[label]
	a.assert.TrueWithMessage(ok, fmt.Sprintf("expected label %s to exist", label))
	a.assert.Contains(label, data)
}

func (a ImageAssertionManager) HasLabelNotContaining(image, label, data string) {
	a.testObject.Helper()
	inspect, err := a.imageManager.InspectLocal(image)
	a.assert.Nil(err)
	label, ok := inspect.Config.Labels[label]
	a.assert.TrueWithMessage(ok, fmt.Sprintf("expected label %s to exist", label))
	a.assert.NotContains(label, data)
}

func (a ImageAssertionManager) HasLengthLayers(image string, length int) {
	a.testObject.Helper()
	inspect, err := a.imageManager.InspectLocal(image)
	a.assert.Nil(err)
	a.assert.TrueWithMessage(len(inspect.RootFS.Layers) == length, fmt.Sprintf("expected image to have %d layers, found %d", length, len(inspect.RootFS.Layers)))
}

func (a ImageAssertionManager) RunsWithOutput(image string, expectedOutputs ...string) {
	a.testObject.Helper()
	containerName := "test-" + h.RandString(10)
	container := a.imageManager.ExposePortOnImage(image, containerName)
	defer container.Cleanup()

	output := container.WaitForResponse(managers.DefaultDuration)
	a.assert.ContainsAll(output, expectedOutputs...)
}

func (a ImageAssertionManager) RunsWithLogs(image string, expectedOutputs ...string) {
	a.testObject.Helper()
	container := a.imageManager.CreateContainer(image)
	defer container.Cleanup()

	output := container.RunWithOutput()
	a.assert.ContainsAll(output, expectedOutputs...)
}

func (a ImageAssertionManager) CanBePulledFromRegistry(name string) {
	a.testObject.Helper()
	a.imageManager.PullImage(name, a.registry.RegistryAuth())
	a.ExistsLocally(name)
}

func (a ImageAssertionManager) ExistsInRegistryCatalog(name string) {
	a.testObject.Helper()
	contents, err := a.registry.RegistryCatalog()
	a.assert.Nil(err)
	a.assert.ContainsWithMessage(contents, name, fmt.Sprintf("Expected to see image %s in %%s", name))
}

func (a ImageAssertionManager) NotExistsInRegistry(name string) {
	a.testObject.Helper()
	contents, err := a.registry.RegistryCatalog()
	a.assert.Nil(err)
	a.assert.NotContainWithMessage(
		contents,
		name,
		"Didn't expect to see image %s in the registry",
	)
}

func (a ImageAssertionManager) DoesNotHaveDuplicateLayers(name string) {
	a.testObject.Helper()

	out, err := a.imageManager.InspectLocal(name)
	a.assert.Nil(err)

	layerSet := map[string]interface{}{}
	for _, layer := range out.RootFS.Layers {
		_, ok := layerSet[layer]
		if ok {
			a.testObject.Fatalf("duplicate layer found in builder %s", layer)
		}
		layerSet[layer] = true
	}
}
