package phase

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/platform"

	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/env"
	"github.com/buildpacks/lifecycle/internal/fsutil"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/log"
	"github.com/buildpacks/lifecycle/platform/files"
)

type Generator struct {
	AppDir         string
	BuildConfigDir string
	GeneratedDir   string // e.g., <layers>/generated
	PlatformAPI    *api.Version
	PlatformDir    string
	AnalyzedMD     files.Analyzed
	DirStore       DirStore
	Executor       buildpack.GenerateExecutor
	Extensions     []buildpack.GroupElement
	Logger         log.Logger
	Out, Err       io.Writer
	Plan           files.Plan
	RunMetadata    files.Run
}

// NewGenerator constructs a new Generator by initializing services and reading the provided analyzed, group, plan, and run files.
func (f *HermeticFactory) NewGenerator(inputs platform.LifecycleInputs, stdout, stderr io.Writer, logger log.Logger) (*Generator, error) {
	generator := &Generator{
		AppDir:         inputs.AppDir,
		BuildConfigDir: inputs.BuildConfigDir,
		GeneratedDir:   inputs.GeneratedDir,
		PlatformAPI:    inputs.PlatformAPI,
		PlatformDir:    inputs.PlatformDir,
		DirStore:       f.dirStore,
		Executor:       &buildpack.DefaultGenerateExecutor{},
		Logger:         logger,
		Out:            stdout,
		Err:            stderr,
	}
	var err error
	if generator.Extensions, err = f.getExtensions(inputs.GroupPath, logger); err != nil {
		return nil, err
	}
	if generator.Plan, err = f.configHandler.ReadPlan(inputs.PlanPath); err != nil {
		return nil, err
	}
	if generator.RunMetadata, err = f.configHandler.ReadRun(inputs.RunPath, logger); err != nil {
		return nil, err
	}
	if generator.AnalyzedMD, err = f.configHandler.ReadAnalyzed(inputs.AnalyzedPath, logger); err != nil {
		return nil, err
	}
	return generator, nil
}

type GenerateResult struct {
	AnalyzedMD files.Analyzed
	Plan       files.Plan
}

func (g *Generator) Generate() (GenerateResult, error) {
	defer log.NewMeasurement("Generator", g.Logger)()
	inputs := g.getGenerateInputs()

	var err error
	if g.PlatformAPI.LessThan("0.13") {
		extensionOutputParentDir, err := os.MkdirTemp("", "cnb-extensions-generated.")
		if err != nil {
			return GenerateResult{}, err
		}
		defer func() {
			err = os.RemoveAll(extensionOutputParentDir)
		}()
		inputs.OutputDir = extensionOutputParentDir
	} else {
		inputs.OutputDir = g.GeneratedDir
	}

	var dockerfiles []buildpack.DockerfileInfo
	filteredPlan := g.Plan
	for _, ext := range g.Extensions {
		g.Logger.Debugf("Running generate for extension %s", ext)

		g.Logger.Debug("Looking up extension")
		descriptor, err := g.DirStore.LookupExt(ext.ID, ext.Version)
		if err != nil {
			return GenerateResult{}, err
		}

		g.Logger.Debug("Finding plan")
		inputs.Plan = filteredPlan.Find(buildpack.KindExtension, ext.ID)

		g.Logger.Debug("Invoking command")
		result, err := g.Executor.Generate(*descriptor, inputs, g.Logger)
		if err != nil {
			return GenerateResult{}, err
		}

		// aggregate build results
		dockerfiles = append(dockerfiles, result.Dockerfiles...)
		filteredPlan = filteredPlan.Filter(result.MetRequires)

		g.Logger.Debugf("Finished running generate for extension %s", ext)
	}

	g.Logger.Debug("Checking run image")
	finalAnalyzedMD := g.AnalyzedMD
	generatedRunImageRef, extend := g.runImageFrom(dockerfiles)
	if generatedRunImageRef != "" && g.isNew(generatedRunImageRef) {
		if !g.RunMetadata.Contains(generatedRunImageRef) {
			g.Logger.Warnf("new runtime base image '%s' not found in run metadata", generatedRunImageRef)
		}
		g.Logger.Debugf("Updating analyzed metadata with new run image '%s'", generatedRunImageRef)
		finalAnalyzedMD.RunImage = &files.RunImage{ // target data is cleared
			Extend:    extend,
			Reference: generatedRunImageRef,
			Image:     generatedRunImageRef,
		}
	}
	if extend {
		if finalAnalyzedMD.RunImage != nil { // sanity check to prevent panic
			g.Logger.Debug("Updating analyzed metadata to indicate run image extension")
			finalAnalyzedMD.RunImage.Extend = true
		}
	}

	g.Logger.Debug("Copying Dockerfiles")
	if err := g.copyDockerfiles(dockerfiles); err != nil {
		return GenerateResult{}, err
	}

	return GenerateResult{
		AnalyzedMD: finalAnalyzedMD,
		Plan:       filteredPlan,
	}, err
}

func (g *Generator) getGenerateInputs() buildpack.GenerateInputs {
	return buildpack.GenerateInputs{
		AppDir:         g.AppDir,
		BuildConfigDir: g.BuildConfigDir,
		PlatformDir:    g.PlatformDir,
		Env:            env.NewBuildEnv(os.Environ()),
		TargetEnv:      platform.EnvVarsFor(&fsutil.Detect{}, g.AnalyzedMD.RunImageTarget(), g.Logger),
		Out:            g.Out,
		Err:            g.Err,
	}
}

func (g *Generator) copyDockerfiles(dockerfiles []buildpack.DockerfileInfo) error {
	for _, dockerfile := range dockerfiles {
		ignoreDockerfile := dockerfile.Kind == buildpack.DockerfileKindRun && dockerfile.Ignore

		if g.PlatformAPI.AtLeast("0.13") {
			if ignoreDockerfile {
				if err := fsutil.RenameWithWindowsFallback(dockerfile.Path, dockerfile.Path+".ignore"); err != nil {
					return fmt.Errorf("failed to rename Dockerfile at %s: %w", dockerfile.Path, err)
				}
			}

			continue
		}

		targetDir := filepath.Join(g.GeneratedDir, dockerfile.Kind, launch.EscapeID(dockerfile.ExtensionID))
		targetPath := filepath.Join(targetDir, "Dockerfile")
		if ignoreDockerfile {
			targetPath += ".ignore"
		}

		if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
			return err
		}
		g.Logger.Debugf("Copying %s to %s", dockerfile.Path, targetPath)
		if err := fsutil.Copy(dockerfile.Path, targetPath); err != nil {
			return fmt.Errorf("failed to copy Dockerfile at %s: %w", dockerfile.Path, err)
		}
		// check for extend-config.toml and if found, copy
		extendConfigPath := filepath.Join(filepath.Dir(dockerfile.Path), "extend-config.toml")
		if err := fsutil.Copy(extendConfigPath, filepath.Join(targetDir, "extend-config.toml")); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to copy extend config at %s: %w", extendConfigPath, err)
			}
		}
	}
	return nil
}

func (g *Generator) runImageFrom(dockerfiles []buildpack.DockerfileInfo) (newBase string, extend bool) {
	var ignoreNext bool
	for i := len(dockerfiles) - 1; i >= 0; i-- {
		// There may be extensions that contribute only a build.Dockerfile;
		// work backward through extensions until we find a run.Dockerfile.
		if dockerfiles[i].Kind != buildpack.DockerfileKindRun {
			continue
		}
		if ignoreNext {
			// If a run.Dockerfile following this one (in the build, not in the loop) switches the run image,
			// we can ignore this run.Dockerfile as it has no effect.
			// We set Ignore to true so that when the Dockerfiles are copied to the "generated" directory,
			// we'll add the suffix `.ignore` so that the extender won't try to apply them.
			dockerfiles[i].Ignore = true
			continue
		}
		if dockerfiles[i].Extend {
			extend = true
		}
		if dockerfiles[i].WithBase != "" {
			newBase = dockerfiles[i].WithBase
			g.Logger.Debugf("Found a run.Dockerfile from extension '%s' setting run image to '%s' ", dockerfiles[i].ExtensionID, newBase)
			ignoreNext = true
		}
	}
	return newBase, extend
}

func (g *Generator) isNew(ref string) bool {
	if g.PlatformAPI.AtLeast("0.12") {
		return ref != g.AnalyzedMD.RunImageImage() // don't use `name.ParseMaybe` as this will strip the digest, and we want to use exactly what the extension author wrote
	}
	return ref != ""
}
