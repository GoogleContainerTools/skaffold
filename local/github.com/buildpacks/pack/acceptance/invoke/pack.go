//go:build acceptance

package invoke

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/Masterminds/semver"

	acceptanceOS "github.com/buildpacks/pack/acceptance/os"
	h "github.com/buildpacks/pack/testhelpers"
)

type PackInvoker struct {
	testObject      *testing.T
	assert          h.AssertionManager
	path            string
	home            string
	dockerConfigDir string
	fixtureManager  PackFixtureManager
	verbose         bool
}

type packPathsProvider interface {
	Path() string
	FixturePaths() []string
}

func NewPackInvoker(
	testObject *testing.T,
	assert h.AssertionManager,
	packAssets packPathsProvider,
	dockerConfigDir string,
) *PackInvoker {

	testObject.Helper()

	home, err := os.MkdirTemp("", "buildpack.pack.home.")
	if err != nil {
		testObject.Fatalf("couldn't create home folder for pack: %s", err)
	}

	return &PackInvoker{
		testObject:      testObject,
		assert:          assert,
		path:            packAssets.Path(),
		home:            home,
		dockerConfigDir: dockerConfigDir,
		verbose:         true,
		fixtureManager: PackFixtureManager{
			testObject: testObject,
			assert:     assert,
			locations:  packAssets.FixturePaths(),
		},
	}
}

func (i *PackInvoker) Cleanup() {
	if i == nil {
		return
	}
	i.testObject.Helper()

	err := os.RemoveAll(i.home)
	i.assert.Nil(err)
}

func (i *PackInvoker) cmd(name string, args ...string) *exec.Cmd {
	i.testObject.Helper()

	cmdArgs := append([]string{name}, args...)
	cmdArgs = append(cmdArgs, "--no-color")
	if i.verbose {
		cmdArgs = append(cmdArgs, "--verbose")
	}

	cmd := i.baseCmd(cmdArgs...)

	cmd.Env = append(os.Environ(), "DOCKER_CONFIG="+i.dockerConfigDir)
	if i.home != "" {
		cmd.Env = append(cmd.Env, "PACK_HOME="+i.home)
	}

	return cmd
}

func (i *PackInvoker) baseCmd(parts ...string) *exec.Cmd {
	return exec.Command(i.path, parts...)
}

func (i *PackInvoker) Run(name string, args ...string) (string, error) {
	i.testObject.Helper()

	output, err := i.cmd(name, args...).CombinedOutput()

	return string(output), err
}

func (i *PackInvoker) SetVerbose(verbose bool) {
	i.verbose = verbose
}

func (i *PackInvoker) RunSuccessfully(name string, args ...string) string {
	i.testObject.Helper()

	output, err := i.Run(name, args...)
	i.assert.NilWithMessage(err, output)

	return output
}

func (i *PackInvoker) JustRunSuccessfully(name string, args ...string) {
	i.testObject.Helper()

	_ = i.RunSuccessfully(name, args...)
}

func (i *PackInvoker) StartWithWriter(combinedOutput *bytes.Buffer, name string, args ...string) *InterruptCmd {
	cmd := i.cmd(name, args...)
	cmd.Stderr = combinedOutput
	cmd.Stdout = combinedOutput

	err := cmd.Start()
	i.assert.Nil(err)

	return &InterruptCmd{
		testObject:     i.testObject,
		assert:         i.assert,
		cmd:            cmd,
		combinedOutput: combinedOutput,
	}
}

func (i *PackInvoker) Home() string {
	return i.home
}

type InterruptCmd struct {
	testObject     *testing.T
	assert         h.AssertionManager
	cmd            *exec.Cmd
	combinedOutput *bytes.Buffer
	outputMux      sync.Mutex
}

func (c *InterruptCmd) TerminateAtStep(pattern string) {
	c.testObject.Helper()

	for {
		c.outputMux.Lock()
		if strings.Contains(c.combinedOutput.String(), pattern) {
			err := c.cmd.Process.Signal(acceptanceOS.InterruptSignal)
			c.assert.Nil(err)
			h.AssertNil(c.testObject, err)
			return
		}
		c.outputMux.Unlock()
	}
}

func (c *InterruptCmd) Wait() error {
	return c.cmd.Wait()
}

func (i *PackInvoker) Version() string {
	i.testObject.Helper()
	return strings.TrimSpace(i.RunSuccessfully("version"))
}

func (i *PackInvoker) SanitizedVersion() string {
	i.testObject.Helper()
	// Sanitizing any git commit sha and build number from the version output
	re := regexp.MustCompile(`\d+\.\d+\.\d+`)
	return re.FindString(strings.TrimSpace(i.RunSuccessfully("version")))
}

func (i *PackInvoker) EnableExperimental() {
	i.testObject.Helper()

	i.JustRunSuccessfully("config", "experimental", "true")
}

// Supports returns whether or not the executor's pack binary supports a
// given command string. The command string can take one of four forms:
//   - "<command>" (e.g. "create-builder")
//   - "<flag>" (e.g. "--verbose")
//   - "<command> <flag>" (e.g. "build --network")
//   - "<command>... <flag>" (e.g. "config trusted-builder --network")
//
// Any other form may return false.
func (i *PackInvoker) Supports(command string) bool {
	i.testObject.Helper()

	parts := strings.Split(command, " ")

	var cmdParts = []string{"help"}
	var search string

	if len(parts) > 1 {
		last := len(parts) - 1
		cmdParts = append(cmdParts, parts[:last]...)
		search = parts[last]
	} else {
		cmdParts = append(cmdParts, command)
		search = command
	}

	re := regexp.MustCompile(fmt.Sprint(`\b%s\b`, search))
	output, err := i.baseCmd(cmdParts...).CombinedOutput()
	i.assert.Nil(err)

	// FIXME: this doesn't appear to be working as expected,
	// as tests against "build --creation-time" and "build --cache" are returning unsupported
	// even on the latest version of pack.
	return re.MatchString(string(output)) && !strings.Contains(string(output), "Unknown help topic")
}

type Feature int

const (
	CreationTime = iota
	Cache
	BuildImageExtensions
	RunImageExtensions
	StackValidation
	ForceRebase
	BuildpackFlatten
	MetaBuildpackFolder
	PlatformRetries
	FlattenBuilderCreationV2
	FixesRunImageMetadata
	ManifestCommands
	PlatformOption
	MultiPlatformBuildersAndBuildPackages
	StackWarning
)

var featureTests = map[Feature]func(i *PackInvoker) bool{
	CreationTime: func(i *PackInvoker) bool {
		return i.Supports("build --creation-time")
	},
	Cache: func(i *PackInvoker) bool {
		return i.Supports("build --cache")
	},
	BuildImageExtensions: func(i *PackInvoker) bool {
		return i.laterThan("v0.27.0")
	},
	RunImageExtensions: func(i *PackInvoker) bool {
		return i.laterThan("v0.29.0")
	},
	StackValidation: func(i *PackInvoker) bool {
		return !i.atLeast("v0.30.0")
	},
	ForceRebase: func(i *PackInvoker) bool {
		return i.atLeast("v0.30.0")
	},
	BuildpackFlatten: func(i *PackInvoker) bool {
		return i.atLeast("v0.30.0")
	},
	MetaBuildpackFolder: func(i *PackInvoker) bool {
		return i.atLeast("v0.30.0")
	},
	PlatformRetries: func(i *PackInvoker) bool {
		return i.atLeast("v0.32.1")
	},
	FlattenBuilderCreationV2: func(i *PackInvoker) bool {
		return i.atLeast("v0.33.1")
	},
	FixesRunImageMetadata: func(i *PackInvoker) bool {
		return i.atLeast("v0.34.0")
	},
	ManifestCommands: func(i *PackInvoker) bool {
		return i.atLeast("v0.34.0")
	},
	PlatformOption: func(i *PackInvoker) bool {
		return i.atLeast("v0.34.0")
	},
	MultiPlatformBuildersAndBuildPackages: func(i *PackInvoker) bool {
		return i.atLeast("v0.34.0")
	},
	StackWarning: func(i *PackInvoker) bool {
		return i.atLeast("v0.37.0")
	},
}

func (i *PackInvoker) SupportsFeature(f Feature) bool {
	return featureTests[f](i)
}

func (i *PackInvoker) semanticVersion() *semver.Version {
	version := i.Version()
	semanticVersion, err := semver.NewVersion(strings.TrimPrefix(strings.Split(version, " ")[0], "v"))
	i.assert.Nil(err)

	return semanticVersion
}

// laterThan returns true if pack version is older than the provided version
func (i *PackInvoker) laterThan(version string) bool {
	providedVersion := semver.MustParse(version)
	ver := i.semanticVersion()
	return ver.Compare(providedVersion) > 0 || ver.Equal(semver.MustParse("0.0.0"))
}

// atLeast returns true if pack version is the same or older than the provided version
func (i *PackInvoker) atLeast(version string) bool {
	minimalVersion := semver.MustParse(version)
	ver := i.semanticVersion()
	return ver.Equal(minimalVersion) || ver.GreaterThan(minimalVersion) || ver.Equal(semver.MustParse("0.0.0"))
}

func (i *PackInvoker) ConfigFileContents() string {
	i.testObject.Helper()

	contents, err := os.ReadFile(filepath.Join(i.home, "config.toml"))
	i.assert.Nil(err)

	return string(contents)
}

func (i *PackInvoker) FixtureManager() PackFixtureManager {
	return i.fixtureManager
}
