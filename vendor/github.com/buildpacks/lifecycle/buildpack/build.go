package buildpack

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/env"
	"github.com/buildpacks/lifecycle/internal/encoding"
	"github.com/buildpacks/lifecycle/internal/fsutil"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/layers"
	"github.com/buildpacks/lifecycle/log"
)

const (
	// EnvBpPlanPath is the absolute path of the filtered build plan, containing relevant Buildpack Plan entries from detection
	EnvBpPlanPath = "CNB_BP_PLAN_PATH"
	// EnvLayersDir is the absolute path of the buildpack layers directory (read-write); a different copy is provided for each buildpack;
	// contents may be saved to either or both of: the final output image or the cache
	EnvLayersDir = "CNB_LAYERS_DIR"
	// Also provided during build: EnvBuildpackDir, EnvPlatformDir (see detect.go)
)

type BuildInputs struct {
	AppDir         string
	BuildConfigDir string
	LayersDir      string
	PlatformDir    string
	Env            BuildEnv
	TargetEnv      []string
	Out, Err       io.Writer
	Plan           Plan
}

type BuildEnv interface {
	AddRootDir(baseDir string) error
	AddEnvDir(envDir string, defaultAction env.ActionType) error
	WithOverrides(platformDir string, buildConfigDir string) ([]string, error)
	List() []string
}

type BuildOutputs struct {
	BOMFiles    []BOMFile
	BuildBOM    []BOMEntry
	Labels      []Label
	LaunchBOM   []BOMEntry
	MetRequires []string
	Processes   []launch.Process
	Slices      []layers.Slice
}

// BuildExecutor executes a single buildpack's `./bin/build` binary,
// providing inputs as defined in the Buildpack Interface Specification,
// and processing outputs for the platform.
//
//go:generate mockgen -package testmock -destination ../phase/testmock/build_executor.go github.com/buildpacks/lifecycle/buildpack BuildExecutor
type BuildExecutor interface {
	Build(d BpDescriptor, inputs BuildInputs, logger log.Logger) (BuildOutputs, error)
}

type DefaultBuildExecutor struct{}

func (e *DefaultBuildExecutor) Build(d BpDescriptor, inputs BuildInputs, logger log.Logger) (BuildOutputs, error) {
	logger.Debug("Creating plan directory")
	planDir, err := os.MkdirTemp("", launch.EscapeID(d.Buildpack.ID)+"-")
	if err != nil {
		return BuildOutputs{}, err
	}
	defer os.RemoveAll(planDir)

	logger.Debug("Preparing paths")
	bpLayersDir, planPath, err := prepareInputPaths(d.Buildpack.ID, inputs.Plan, inputs.LayersDir, planDir)
	if err != nil {
		return BuildOutputs{}, err
	}

	logger.Debug("Running build command")
	if err := runBuildCmd(d, bpLayersDir, planPath, inputs, inputs.Env); err != nil {
		return BuildOutputs{}, err
	}

	logger.Debug("Processing layers")
	createdLayers, err := d.processLayers(bpLayersDir, logger)
	if err != nil {
		return BuildOutputs{}, err
	}

	logger.Debug("Updating environment")
	if err := d.setupEnv(bpLayersDir, createdLayers, inputs.Env); err != nil {
		return BuildOutputs{}, err
	}

	logger.Debug("Reading output files")
	return d.readOutputFilesBp(bpLayersDir, planPath, inputs.Plan, createdLayers, logger)
}

func prepareInputPaths(bpID string, plan Plan, layersDir, parentPlanDir string) (string, string, error) {
	bpDirName := launch.EscapeID(bpID) // FIXME: this logic should eventually move to the platform package

	// Create e.g., <layers>/<buildpack-id> or <output>/<extension-id>
	bpLayersDir := filepath.Join(layersDir, bpDirName)
	if err := os.MkdirAll(bpLayersDir, 0777); err != nil {
		return "", "", err
	}

	// Create Buildpack Plan
	childPlanDir := filepath.Join(parentPlanDir, bpDirName) // FIXME: it's unclear if this child directory is necessary; consider removing
	if err := os.MkdirAll(childPlanDir, 0777); err != nil {
		return "", "", err
	}
	planPath := filepath.Join(childPlanDir, "plan.toml")
	if err := encoding.WriteTOML(planPath, plan); err != nil {
		return "", "", err
	}

	return bpLayersDir, planPath, nil
}

func runBuildCmd(d BpDescriptor, bpLayersDir, planPath string, inputs BuildInputs, buildEnv BuildEnv) error {
	cmd := exec.Command(
		filepath.Join(d.WithRootDir, "bin", "build"),
		bpLayersDir,
		inputs.PlatformDir,
		planPath,
	) // #nosec G204
	cmd.Dir = inputs.AppDir
	cmd.Stdout = inputs.Out
	cmd.Stderr = inputs.Err

	var err error
	if d.Buildpack.ClearEnv {
		cmd.Env, err = buildEnv.WithOverrides("", inputs.BuildConfigDir)
	} else {
		cmd.Env, err = buildEnv.WithOverrides(inputs.PlatformDir, inputs.BuildConfigDir)
	}
	if err != nil {
		return err
	}
	cmd.Env = append(cmd.Env, EnvBuildpackDir+"="+d.WithRootDir)
	if api.MustParse(d.WithAPI).AtLeast("0.8") {
		cmd.Env = append(cmd.Env,
			EnvPlatformDir+"="+inputs.PlatformDir,
			EnvBpPlanPath+"="+planPath,
			EnvLayersDir+"="+bpLayersDir,
		)
	}
	if api.MustParse(d.API()).AtLeast("0.10") {
		cmd.Env = append(cmd.Env, inputs.TargetEnv...)
	}

	if err = cmd.Run(); err != nil {
		return NewError(err, ErrTypeBuildpack)
	}
	return nil
}

func (d BpDescriptor) processLayers(bpLayersDir string, logger log.Logger) (map[string]LayerMetadataFile, error) {
	bpLayers := make(map[string]LayerMetadataFile)
	if err := eachLayer(bpLayersDir, func(layerPath string) error {
		layerFile, err := DecodeLayerMetadataFile(layerPath+".toml", d.WithAPI, logger)
		if err != nil {
			return fmt.Errorf("failed to decode layer metadata file: %w", err)
		}
		if err = renameLayerDirIfNeeded(layerFile, layerPath); err != nil {
			return fmt.Errorf("failed to rename layer directory: %w", err)
		}
		bpLayers[layerPath] = layerFile
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to process buildpack layer: %w", err)
	}
	return bpLayers, nil
}

func eachLayer(bpLayersDir string, fn func(layerPath string) error) error {
	files, err := os.ReadDir(bpLayersDir)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".toml") {
			continue
		}
		path := filepath.Join(bpLayersDir, strings.TrimSuffix(f.Name(), ".toml"))
		if err = fn(path); err != nil {
			return err
		}
	}
	return nil
}

func renameLayerDirIfNeeded(layerMetadataFile LayerMetadataFile, layerDir string) error {
	// rename <layers>/<layer> to <layers>/<layer>.ignore if all the types flags are set to false
	if !layerMetadataFile.Launch && !layerMetadataFile.Cache && !layerMetadataFile.Build {
		if err := fsutil.RenameWithWindowsFallback(layerDir, layerDir+".ignore"); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (d BpDescriptor) setupEnv(bpLayersDir string, createdLayers map[string]LayerMetadataFile, buildEnv BuildEnv) error {
	bpAPI := api.MustParse(d.WithAPI)
	return eachLayer(bpLayersDir, func(layerPath string) error {
		var err error
		layerMetadataFile, ok := createdLayers[layerPath]
		if !ok {
			return fmt.Errorf("failed to find layer metadata for %s", layerPath)
		}
		if !layerMetadataFile.Build {
			return nil
		}
		if err = buildEnv.AddRootDir(layerPath); err != nil {
			return err
		}
		if err = buildEnv.AddEnvDir(filepath.Join(layerPath, "env"), env.DefaultActionType(bpAPI)); err != nil {
			return err
		}
		return buildEnv.AddEnvDir(filepath.Join(layerPath, "env.build"), env.DefaultActionType(bpAPI))
	})
}

func (d BpDescriptor) readOutputFilesBp(bpLayersDir, bpPlanPath string, bpPlanIn Plan, bpLayers map[string]LayerMetadataFile, logger log.Logger) (BuildOutputs, error) {
	br := BuildOutputs{}
	bpFromBpInfo := GroupElement{ID: d.Buildpack.ID, Version: d.Buildpack.Version}

	// setup launch.toml
	var launchTOML LaunchTOML
	launchPath := filepath.Join(bpLayersDir, "launch.toml")

	bomValidator := NewBOMValidator(d.WithAPI, bpLayersDir, logger)

	var err error
	// read build.toml
	var buildTOML BuildTOML
	buildPath := filepath.Join(bpLayersDir, "build.toml")
	if _, err := toml.DecodeFile(buildPath, &buildTOML); err != nil && !os.IsNotExist(err) {
		return BuildOutputs{}, err
	}
	if _, err := bomValidator.ValidateBOM(bpFromBpInfo, buildTOML.BOM); err != nil {
		return BuildOutputs{}, err
	}
	br.BuildBOM, err = bomValidator.ValidateBOM(bpFromBpInfo, buildTOML.BOM)
	if err != nil {
		return BuildOutputs{}, err
	}

	// set MetRequires
	if err := validateUnmet(buildTOML.Unmet, bpPlanIn); err != nil {
		return BuildOutputs{}, err
	}
	br.MetRequires = names(bpPlanIn.filter(buildTOML.Unmet).Entries)

	// set BOM files
	br.BOMFiles, err = d.processSBOMFiles(bpLayersDir, bpFromBpInfo, bpLayers, logger)
	if err != nil {
		return BuildOutputs{}, err
	}

	// read launch.toml, return if not exists
	if err := DecodeLaunchTOML(launchPath, d.WithAPI, &launchTOML); os.IsNotExist(err) {
		return br, nil
	} else if err != nil {
		return BuildOutputs{}, err
	}

	// set BOM
	br.LaunchBOM, err = bomValidator.ValidateBOM(bpFromBpInfo, launchTOML.BOM)
	if err != nil {
		return BuildOutputs{}, err
	}

	if err := validateNoMultipleDefaults(launchTOML.Processes); err != nil {
		return BuildOutputs{}, err
	}

	// set data from launch.toml
	br.Labels = append([]Label{}, launchTOML.Labels...)
	for i := range launchTOML.Processes {
		if api.MustParse(d.WithAPI).LessThan("0.8") {
			if launchTOML.Processes[i].WorkingDirectory != "" {
				logger.Warn(fmt.Sprintf("Warning: process working directory isn't supported in this buildpack api version. Ignoring working directory for process '%s'", launchTOML.Processes[i].Type))
				launchTOML.Processes[i].WorkingDirectory = ""
			}
		}
	}
	br.Processes = append([]launch.Process{}, launchTOML.ToLaunchProcessesForBuildpack(d.Buildpack.ID)...)
	br.Slices = append([]layers.Slice{}, launchTOML.Slices...)

	return br, nil
}

func names(requires []Require) []string {
	var out []string
	for _, req := range requires {
		out = append(out, req.Name)
	}
	return out
}

func validateUnmet(unmet []Unmet, bpPlan Plan) error {
	for _, unmet := range unmet {
		if unmet.Name == "" {
			return errors.New("unmet.name is required")
		}
		found := false
		for _, req := range bpPlan.Entries {
			if unmet.Name == req.Name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unmet.name '%s' must match a requested dependency", unmet.Name)
		}
	}
	return nil
}

func validateNoMultipleDefaults(processes []ProcessEntry) error {
	defaultType := ""
	for _, process := range processes {
		if process.Default && defaultType != "" {
			return fmt.Errorf("multiple default process types aren't allowed")
		}
		if process.Default {
			defaultType = process.Type
		}
	}
	return nil
}
