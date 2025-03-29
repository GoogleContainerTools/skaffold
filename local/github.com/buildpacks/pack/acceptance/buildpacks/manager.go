//go:build acceptance

package buildpacks

import (
	"path/filepath"
	"testing"

	"github.com/buildpacks/pack/testhelpers"
)

type BuildModuleManager struct {
	testObject *testing.T
	assert     testhelpers.AssertionManager
	sourceDir  string
}

type BuildModuleManagerModifier func(b *BuildModuleManager)

func NewBuildModuleManager(t *testing.T, assert testhelpers.AssertionManager, modifiers ...BuildModuleManagerModifier) BuildModuleManager {
	m := BuildModuleManager{
		testObject: t,
		assert:     assert,
		sourceDir:  filepath.Join("testdata", "mock_buildpacks"),
	}

	for _, mod := range modifiers {
		mod(&m)
	}

	return m
}

type TestBuildModule interface {
	Prepare(source, destination string) error
}

func (b BuildModuleManager) PrepareBuildModules(destination string, modules ...TestBuildModule) {
	b.testObject.Helper()

	for _, module := range modules {
		err := module.Prepare(b.sourceDir, destination)
		b.assert.Nil(err)
	}
}

type Modifiable interface {
	SetPublish()
	SetBuildpacks([]TestBuildModule)
}
type PackageModifier func(p Modifiable)

func WithRequiredBuildpacks(buildpacks ...TestBuildModule) PackageModifier {
	return func(p Modifiable) {
		p.SetBuildpacks(buildpacks)
	}
}

func WithPublish() PackageModifier {
	return func(p Modifiable) {
		p.SetPublish()
	}
}
