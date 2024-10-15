// Package files contains schema and helper methods for working with lifecycle configuration files.
package files

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/internal/encoding"
	"github.com/buildpacks/lifecycle/log"
)

// Handler is the default handler used to read and write lifecycle configuration files.
var Handler = &TOMLHandler{}

// TOMLHandler reads and writes lifecycle configuration files in TOML format.
type TOMLHandler struct{}

// NewHandler returns a new file handler.
func NewHandler() *TOMLHandler {
	return &TOMLHandler{}
}

// ReadAnalyzed reads the provided analyzed.toml file.
// It logs a warning and returns empty analyzed metadata if the file does not exist.
func (h *TOMLHandler) ReadAnalyzed(path string, logger log.Logger) (Analyzed, error) {
	var analyzed Analyzed
	if _, err := toml.DecodeFile(path, &analyzed); err != nil {
		if os.IsNotExist(err) {
			logger.Warnf("No analyzed metadata found at path %q", path)
			return Analyzed{}, nil
		}
		return Analyzed{}, fmt.Errorf("failed to read analyzed file: %w", err)
	}
	return analyzed, nil
}

// WriteAnalyzed writes the provided analyzed metadata at the provided path.
func (h *TOMLHandler) WriteAnalyzed(path string, analyzedMD *Analyzed, logger log.Logger) error {
	logger.Debugf("Run image info in analyzed metadata is: ")
	logger.Debugf(encoding.ToJSONMaybe(analyzedMD.RunImage))
	if err := encoding.WriteTOML(path, analyzedMD); err != nil {
		return fmt.Errorf("failed to write analyzed file: %w", err)
	}
	return nil
}

// ReadGroup reads the provided group.toml file.
func (h *TOMLHandler) ReadGroup(path string) (group buildpack.Group, err error) {
	if _, err = toml.DecodeFile(path, &group); err != nil {
		return buildpack.Group{}, fmt.Errorf("failed to read group file: %w", err)
	}
	for e := range group.GroupExtensions {
		group.GroupExtensions[e].Extension = true
		group.GroupExtensions[e].Optional = true
	}
	return group, nil
}

// WriteGroup writes the provided group information at the provided path.
func (h *TOMLHandler) WriteGroup(path string, group *buildpack.Group) error {
	for e := range group.GroupExtensions {
		group.GroupExtensions[e].Extension = false
		group.GroupExtensions[e].Optional = false // avoid printing redundant information (extensions are always optional)
	}
	if err := encoding.WriteTOML(path, group); err != nil {
		return fmt.Errorf("failed to write group file: %w", err)
	}
	return nil
}

// ReadBuildMetadata reads the provided metadata.toml file,
// and sets the provided Platform API version on the returned struct so that the data can be re-encoded properly later.
func (h *TOMLHandler) ReadBuildMetadata(path string, platformAPI *api.Version) (*BuildMetadata, error) {
	buildMD := BuildMetadata{}
	_, err := toml.DecodeFile(path, &buildMD)
	if err != nil {
		return nil, fmt.Errorf("failed to read build metadata file: %w", err)
	}
	buildMD.PlatformAPI = platformAPI
	for i, process := range buildMD.Processes {
		buildMD.Processes[i] = process.WithPlatformAPI(platformAPI)
	}
	return &buildMD, nil
}

// WriteBuildMetadata writes the provided build metadata at the provided path.
func (h *TOMLHandler) WriteBuildMetadata(path string, buildMD *BuildMetadata) error {
	if err := encoding.WriteTOML(path, buildMD); err != nil {
		return fmt.Errorf("failed to write build metadata file: %w", err)
	}
	return nil
}

// ReadOrder reads the provided order.toml file.
func (h *TOMLHandler) ReadOrder(path string) (buildpack.Order, buildpack.Order, error) {
	orderBp, orderExt, err := readOrder(path)
	if err != nil {
		return buildpack.Order{}, buildpack.Order{}, err
	}
	return orderBp, orderExt, nil
}

func readOrder(path string) (buildpack.Order, buildpack.Order, error) {
	var order struct {
		Order           buildpack.Order `toml:"order"`
		OrderExtensions buildpack.Order `toml:"order-extensions"`
	}
	_, err := toml.DecodeFile(path, &order)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read order file: %w", err)
	}
	for g, group := range order.OrderExtensions {
		for e := range group.Group {
			group.Group[e].Extension = true
			group.Group[e].Optional = true
		}
		order.OrderExtensions[g] = group
	}
	return order.Order, order.OrderExtensions, nil
}

// ReadPlan reads the provided plan.toml file.
func (h *TOMLHandler) ReadPlan(path string) (Plan, error) {
	var plan Plan
	if _, err := toml.DecodeFile(path, &plan); err != nil {
		return Plan{}, fmt.Errorf("failed to read plan file: %w", err)
	}
	return plan, nil
}

// WritePlan writes the provided plan information at the provided path.
func (h *TOMLHandler) WritePlan(path string, plan *Plan) error {
	if err := encoding.WriteTOML(path, plan); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}
	return nil
}

// ReadProjectMetadata reads the provided project_metadata.toml file.
// It logs a warning and returns empty project metadata if the file does not exist.
func (h *TOMLHandler) ReadProjectMetadata(path string, logger log.Logger) (ProjectMetadata, error) {
	var projectMD ProjectMetadata
	if _, err := toml.DecodeFile(path, &projectMD); err != nil {
		if os.IsNotExist(err) {
			logger.Debugf("No project metadata found at path %q, project metadata will not be exported", path)
			return ProjectMetadata{}, nil
		}
		return ProjectMetadata{}, fmt.Errorf("failed to read project metadata file: %w", err)
	}
	return projectMD, nil
}

// WriteReport writes the provided report information at the provided path.
func (h *TOMLHandler) WriteReport(path string, report *Report) error {
	if err := encoding.WriteTOML(path, report); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}
	return nil
}

// WriteRebaseReport writes the provided report information at the provided path.
func (h *TOMLHandler) WriteRebaseReport(path string, report *RebaseReport) error {
	if err := encoding.WriteTOML(path, report); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}
	return nil
}

// ReadRun reads the provided run.toml file.
func (h *TOMLHandler) ReadRun(path string, logger log.Logger) (Run, error) {
	var runMD Run
	if _, err := toml.DecodeFile(path, &runMD); err != nil {
		if os.IsNotExist(err) {
			logger.Infof("No run metadata found at path %q", path)
			return Run{}, nil
		}
		return Run{}, fmt.Errorf("failed to read run file: %w", err)
	}
	return runMD, nil
}

// ReadStack reads the provided stack.toml file.
func (h *TOMLHandler) ReadStack(path string, logger log.Logger) (Stack, error) {
	var stackMD Stack
	if _, err := toml.DecodeFile(path, &stackMD); err != nil {
		if os.IsNotExist(err) {
			logger.Infof("No stack metadata found at path %q", path)
			return Stack{}, nil
		}
		return Stack{}, fmt.Errorf("failed to read stack file: %w", err)
	}
	return stackMD, nil
}
