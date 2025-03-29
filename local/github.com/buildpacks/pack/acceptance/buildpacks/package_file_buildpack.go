//go:build acceptance

package buildpacks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buildpacks/pack/acceptance/invoke"
	h "github.com/buildpacks/pack/testhelpers"
)

type PackageFile struct {
	testObject           *testing.T
	pack                 *invoke.PackInvoker
	destination          string
	sourceConfigLocation string
	buildpacks           []TestBuildModule
}

func (p *PackageFile) SetBuildpacks(buildpacks []TestBuildModule) {
	p.buildpacks = buildpacks
}

func (p *PackageFile) SetPublish() {}

func NewPackageFile(
	t *testing.T,
	pack *invoke.PackInvoker,
	destination, configLocation string,
	modifiers ...PackageModifier,
) PackageFile {

	p := PackageFile{
		testObject:           t,
		pack:                 pack,
		destination:          destination,
		sourceConfigLocation: configLocation,
	}
	for _, mod := range modifiers {
		mod(&p)
	}

	return p
}

func (p PackageFile) Prepare(sourceDir, _ string) error {
	p.testObject.Helper()
	p.testObject.Log("creating package file from:", sourceDir)

	tmpDir, err := os.MkdirTemp("", "package-buildpacks")
	if err != nil {
		return fmt.Errorf("creating temp dir for package buildpacks: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, buildpack := range p.buildpacks {
		err = buildpack.Prepare(sourceDir, tmpDir)
		if err != nil {
			return fmt.Errorf("preparing buildpack %s: %w", buildpack, err)
		}
	}

	configLocation := filepath.Join(tmpDir, "package.toml")
	h.CopyFile(p.testObject, p.sourceConfigLocation, configLocation)

	packArgs := []string{
		p.destination,
		"--no-color",
		"-c", configLocation,
		"--format", "file",
	}

	output := p.pack.RunSuccessfully("buildpack", append([]string{"package"}, packArgs...)...)
	if !strings.Contains(output, fmt.Sprintf("Successfully created package '%s'", p.destination)) {
		return errors.New("failed to create package")
	}

	return nil
}
