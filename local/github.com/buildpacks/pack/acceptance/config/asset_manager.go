//go:build acceptance

package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	acceptanceOS "github.com/buildpacks/pack/acceptance/os"
	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/blob"
	h "github.com/buildpacks/pack/testhelpers"
)

const (
	defaultCompilePackVersion = "0.0.0"
)

var (
	currentPackFixturesDir           = filepath.Join("testdata", "pack_fixtures")
	previousPackFixturesOverridesDir = filepath.Join("testdata", "pack_previous_fixtures_overrides")
	lifecycleTgzExp                  = regexp.MustCompile(`lifecycle-v\d+.\d+.\d+(-pre.\d+)?(-rc.\d+)?\+linux.x86-64.tgz`)
)

type AssetManager struct {
	packPath                    string
	packFixturesPath            string
	previousPackPath            string
	previousPackFixturesPaths   []string
	githubAssetFetcher          *GithubAssetFetcher
	lifecyclePath               string
	lifecycleDescriptor         builder.LifecycleDescriptor
	lifecycleImage              string
	previousLifecyclePath       string
	previousLifecycleDescriptor builder.LifecycleDescriptor
	previousLifecycleImage      string
	defaultLifecycleDescriptor  builder.LifecycleDescriptor
	testObject                  *testing.T
}

func ConvergedAssetManager(t *testing.T, assert h.AssertionManager, inputConfig InputConfigurationManager) AssetManager {
	t.Helper()

	var (
		convergedCurrentPackPath             string
		convergedPreviousPackPath            string
		convergedPreviousPackFixturesPaths   []string
		convergedCurrentLifecyclePath        string
		convergedCurrentLifecycleImage       string
		convergedCurrentLifecycleDescriptor  builder.LifecycleDescriptor
		convergedPreviousLifecyclePath       string
		convergedPreviousLifecycleImage      string
		convergedPreviousLifecycleDescriptor builder.LifecycleDescriptor
		convergedDefaultLifecycleDescriptor  builder.LifecycleDescriptor
	)

	githubAssetFetcher, err := NewGithubAssetFetcher(t, inputConfig.githubToken)
	h.AssertNil(t, err)

	assetBuilder := assetManagerBuilder{
		testObject:         t,
		assert:             assert,
		inputConfig:        inputConfig,
		githubAssetFetcher: githubAssetFetcher,
	}

	if inputConfig.combinations.requiresCurrentPack() {
		convergedCurrentPackPath = assetBuilder.ensureCurrentPack()
	}

	if inputConfig.combinations.requiresPreviousPack() {
		convergedPreviousPackPath = assetBuilder.ensurePreviousPack()
		convergedPreviousPackFixturesPath := assetBuilder.ensurePreviousPackFixtures()

		convergedPreviousPackFixturesPaths = []string{previousPackFixturesOverridesDir, convergedPreviousPackFixturesPath}
	}

	if inputConfig.combinations.requiresCurrentLifecycle() {
		convergedCurrentLifecyclePath, convergedCurrentLifecycleImage, convergedCurrentLifecycleDescriptor = assetBuilder.ensureCurrentLifecycle()
	}

	if inputConfig.combinations.requiresPreviousLifecycle() {
		convergedPreviousLifecyclePath, convergedPreviousLifecycleImage, convergedPreviousLifecycleDescriptor = assetBuilder.ensurePreviousLifecycle()
	}

	if inputConfig.combinations.requiresDefaultLifecycle() {
		convergedDefaultLifecycleDescriptor = assetBuilder.defaultLifecycleDescriptor()
	}

	return AssetManager{
		packPath:                    convergedCurrentPackPath,
		packFixturesPath:            currentPackFixturesDir,
		previousPackPath:            convergedPreviousPackPath,
		previousPackFixturesPaths:   convergedPreviousPackFixturesPaths,
		lifecyclePath:               convergedCurrentLifecyclePath,
		lifecycleImage:              convergedCurrentLifecycleImage,
		lifecycleDescriptor:         convergedCurrentLifecycleDescriptor,
		previousLifecyclePath:       convergedPreviousLifecyclePath,
		previousLifecycleImage:      convergedPreviousLifecycleImage,
		previousLifecycleDescriptor: convergedPreviousLifecycleDescriptor,
		defaultLifecycleDescriptor:  convergedDefaultLifecycleDescriptor,
		testObject:                  t,
	}
}

func (a AssetManager) PackPaths(kind ComboValue) (packPath string, packFixturesPaths []string) {
	a.testObject.Helper()

	switch kind {
	case Current:
		packPath = a.packPath
		packFixturesPaths = []string{a.packFixturesPath}
	case Previous:
		packPath = a.previousPackPath
		packFixturesPaths = a.previousPackFixturesPaths
	default:
		a.testObject.Fatalf("pack kind must be current or previous, was %s", kind)
	}

	return packPath, packFixturesPaths
}

func (a AssetManager) LifecyclePath(kind ComboValue) string {
	a.testObject.Helper()

	switch kind {
	case Current:
		return a.lifecyclePath
	case Previous:
		return a.previousLifecyclePath
	case DefaultKind:
		return ""
	}

	a.testObject.Fatalf("lifecycle kind must be previous, current or default was %s", kind)
	return "" // Unreachable
}

func (a AssetManager) LifecycleDescriptor(kind ComboValue) builder.LifecycleDescriptor {
	a.testObject.Helper()

	switch kind {
	case Current:
		return a.lifecycleDescriptor
	case Previous:
		return a.previousLifecycleDescriptor
	case DefaultKind:
		return a.defaultLifecycleDescriptor
	}

	a.testObject.Fatalf("lifecycle kind must be previous, current or default was %s", kind)
	return builder.LifecycleDescriptor{} // Unreachable
}

func (a AssetManager) LifecycleImage(kind ComboValue) string {
	a.testObject.Helper()

	switch kind {
	case Current:
		return a.lifecycleImage
	case Previous:
		return a.previousLifecycleImage
	case DefaultKind:
		return fmt.Sprintf("%s:%s", config.DefaultLifecycleImageRepo, a.defaultLifecycleDescriptor.Info.Version)
	}

	a.testObject.Fatalf("lifecycle kind must be previous, current or default was %s", kind)
	return "" // Unreachable
}

type assetManagerBuilder struct {
	testObject         *testing.T
	assert             h.AssertionManager
	inputConfig        InputConfigurationManager
	githubAssetFetcher *GithubAssetFetcher
}

func (b assetManagerBuilder) ensureCurrentPack() string {
	b.testObject.Helper()

	if b.inputConfig.packPath != "" {
		return b.inputConfig.packPath
	}

	compileWithVersion := b.inputConfig.compilePackWithVersion
	if compileWithVersion == "" {
		compileWithVersion = defaultCompilePackVersion
	}

	return b.buildPack(compileWithVersion)
}

func (b assetManagerBuilder) ensurePreviousPack() string {
	b.testObject.Helper()

	if b.inputConfig.previousPackPath != "" {
		return b.inputConfig.previousPackPath
	}

	b.testObject.Logf(
		"run combinations %+v require %s to be set",
		b.inputConfig.combinations,
		style.Symbol(envPreviousPackPath),
	)

	version, err := b.githubAssetFetcher.FetchReleaseVersion("buildpacks", "pack", 0)
	b.assert.Nil(err)

	assetDir, err := b.githubAssetFetcher.FetchReleaseAsset(
		"buildpacks",
		"pack",
		version,
		acceptanceOS.PackBinaryExp,
		true,
	)
	b.assert.Nil(err)
	assetPath := filepath.Join(assetDir, acceptanceOS.PackBinaryName)

	b.testObject.Logf("using %s for previous pack path", assetPath)

	return assetPath
}

func (b assetManagerBuilder) ensurePreviousPackFixtures() string {
	b.testObject.Helper()

	if b.inputConfig.previousPackFixturesPath != "" {
		return b.inputConfig.previousPackFixturesPath
	}

	b.testObject.Logf(
		"run combinations %+v require %s to be set",
		b.inputConfig.combinations,
		style.Symbol(envPreviousPackFixturesPath),
	)

	version, err := b.githubAssetFetcher.FetchReleaseVersion("buildpacks", "pack", 0)
	b.assert.Nil(err)

	sourceDir, err := b.githubAssetFetcher.FetchReleaseSource("buildpacks", "pack", version)
	b.assert.Nil(err)

	sourceDirFiles, err := os.ReadDir(sourceDir)
	b.assert.Nil(err)
	// GitHub source tarballs have a top-level directory whose name includes the current commit sha.
	innerDir := sourceDirFiles[0].Name()
	fixturesDir := filepath.Join(sourceDir, innerDir, "acceptance", "testdata", "pack_fixtures")

	b.testObject.Logf("using %s for previous pack fixtures path", fixturesDir)

	return fixturesDir
}

func (b assetManagerBuilder) ensureCurrentLifecycle() (string, string, builder.LifecycleDescriptor) {
	b.testObject.Helper()

	lifecyclePath := b.inputConfig.lifecyclePath

	if lifecyclePath == "" {
		b.testObject.Logf(
			"run combinations %+v require %s to be set",
			b.inputConfig.combinations,
			style.Symbol(envLifecyclePath),
		)

		lifecyclePath = b.downloadLifecycleRelative(0)

		b.testObject.Logf("using %s for current lifecycle path", lifecyclePath)
	}

	lifecycle, err := builder.NewLifecycle(blob.NewBlob(lifecyclePath))
	b.assert.Nil(err)

	lifecycleImage := b.inputConfig.lifecycleImage

	if lifecycleImage == "" {
		lifecycleImage = fmt.Sprintf("%s:%s", config.DefaultLifecycleImageRepo, lifecycle.Descriptor().Info.Version)

		b.testObject.Logf("using %s for current lifecycle image", lifecycleImage)
	}

	return lifecyclePath, lifecycleImage, lifecycle.Descriptor()
}

func (b assetManagerBuilder) ensurePreviousLifecycle() (string, string, builder.LifecycleDescriptor) {
	b.testObject.Helper()

	previousLifecyclePath := b.inputConfig.previousLifecyclePath

	if previousLifecyclePath == "" {
		b.testObject.Logf(
			"run combinations %+v require %s to be set",
			b.inputConfig.combinations,
			style.Symbol(envPreviousLifecyclePath),
		)

		previousLifecyclePath = b.downloadLifecycleRelative(-1)

		b.testObject.Logf("using %s for previous lifecycle path", previousLifecyclePath)
	}

	lifecycle, err := builder.NewLifecycle(blob.NewBlob(previousLifecyclePath))
	b.assert.Nil(err)

	previousLifecycleImage := b.inputConfig.previousLifecycleImage

	if previousLifecycleImage == "" {
		previousLifecycleImage = fmt.Sprintf("%s:%s", config.DefaultLifecycleImageRepo, lifecycle.Descriptor().Info.Version)

		b.testObject.Logf("using %s for previous lifecycle image", previousLifecycleImage)
	}

	return previousLifecyclePath, previousLifecycleImage, lifecycle.Descriptor()
}

func (b assetManagerBuilder) downloadLifecycle(version string) string {
	path, err := b.githubAssetFetcher.FetchReleaseAsset(
		"buildpacks",
		"lifecycle",
		version,
		lifecycleTgzExp,
		false,
	)
	b.assert.Nil(err)

	return path
}

func (b assetManagerBuilder) downloadLifecycleRelative(relativeVersion int) string {
	b.testObject.Helper()

	version, err := b.githubAssetFetcher.FetchReleaseVersion("buildpacks", "lifecycle", relativeVersion)
	b.assert.Nil(err)

	return b.downloadLifecycle(version)
}

func (b assetManagerBuilder) buildPack(compileVersion string) string {
	b.testObject.Helper()

	packTmpDir, err := os.MkdirTemp("", "pack.acceptance.binary.")
	b.assert.Nil(err)

	packPath := filepath.Join(packTmpDir, acceptanceOS.PackBinaryName)

	cwd, err := os.Getwd()
	b.assert.Nil(err)

	cmd := exec.Command("go", "build",
		"-ldflags", fmt.Sprintf("-X 'github.com/buildpacks/pack/cmd.Version=%s'", compileVersion),
		"-o", packPath,
		"./cmd/pack",
	)
	if filepath.Base(cwd) == "acceptance" {
		cmd.Dir = filepath.Dir(cwd)
	}

	b.testObject.Logf("building pack: [CWD=%s] %s", cmd.Dir, cmd.Args)
	_, err = cmd.CombinedOutput()
	b.assert.Nil(err)

	return packPath
}

func (b assetManagerBuilder) defaultLifecycleDescriptor() builder.LifecycleDescriptor {
	lifecyclePath := b.downloadLifecycle("v" + builder.DefaultLifecycleVersion)

	b.testObject.Logf("using %s for default lifecycle path", lifecyclePath)

	lifecycle, err := builder.NewLifecycle(blob.NewBlob(lifecyclePath))
	b.assert.Nil(err)

	return lifecycle.Descriptor()
}
