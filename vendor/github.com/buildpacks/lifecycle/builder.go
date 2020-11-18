package lifecycle

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/layers"
)

type Builder struct {
	AppDir        string
	LayersDir     string
	PlatformDir   string
	BuildpacksDir string
	PlatformAPI   *api.Version
	Env           BuildEnv
	Group         BuildpackGroup
	Plan          BuildPlan
	Out, Err      io.Writer
}

type BuildEnv interface {
	AddRootDir(baseDir string) error
	AddEnvDir(envDir string) error
	WithPlatform(platformDir string) ([]string, error)
	List() []string
}

type LaunchTOML struct {
	Labels    []Label
	Processes []launch.Process `toml:"processes"`
	Slices    []layers.Slice   `toml:"slices"`
}

type Label struct {
	Key   string `toml:"key"`
	Value string `toml:"value"`
}

type BOMEntry struct {
	Require
	Buildpack Buildpack `toml:"buildpack" json:"buildpack"`
}

type BuildpackPlan struct {
	Entries []Require `toml:"entries"`
}

func (b *Builder) Build() (*BuildMetadata, error) {
	platformDir, err := filepath.Abs(b.PlatformDir)
	if err != nil {
		return nil, err
	}
	layersDir, err := filepath.Abs(b.LayersDir)
	if err != nil {
		return nil, err
	}
	appDir, err := filepath.Abs(b.AppDir)
	if err != nil {
		return nil, err
	}
	planDir, err := ioutil.TempDir("", "plan.")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(planDir)

	procMap := processMap{}
	plan := b.Plan
	var bom []BOMEntry
	var slices []layers.Slice
	var labels []Label

	for _, bp := range b.Group.Group {
		bpInfo, err := bp.Lookup(b.BuildpacksDir)
		if err != nil {
			return nil, err
		}
		bpDirName := launch.EscapeID(bp.ID)
		bpLayersDir := filepath.Join(layersDir, bpDirName)
		bpPlanDir := filepath.Join(planDir, bpDirName)
		if err := os.MkdirAll(bpLayersDir, 0777); err != nil {
			return nil, err
		}

		if err := os.MkdirAll(bpPlanDir, 0777); err != nil {
			return nil, err
		}
		bpPlanPath := filepath.Join(bpPlanDir, "plan.toml")

		foundPlan := plan.find(bp.noAPI())
		if api.MustParse(bp.API).Equal(api.MustParse("0.2")) {
			for i := range foundPlan.Entries {
				foundPlan.Entries[i].convertMetadataToVersion()
			}
		}
		if err := WriteTOML(bpPlanPath, foundPlan); err != nil {
			return nil, err
		}

		cmd := exec.Command(
			filepath.Join(bpInfo.Path, "bin", "build"),
			bpLayersDir,
			platformDir,
			bpPlanPath,
		)
		cmd.Dir = appDir
		cmd.Stdout = b.Out
		cmd.Stderr = b.Err

		if bpInfo.Buildpack.ClearEnv {
			cmd.Env = b.Env.List()
		} else {
			cmd.Env, err = b.Env.WithPlatform(platformDir)
			if err != nil {
				return nil, err
			}
		}
		cmd.Env = append(cmd.Env, EnvBuildpackDir+"="+bpInfo.Path)

		if err := cmd.Run(); err != nil {
			return nil, NewLifecycleError(err, ErrTypeBuildpack)
		}
		if err := setupEnv(b.Env, bpLayersDir); err != nil {
			return nil, err
		}
		var bpPlanOut BuildpackPlan
		if _, err := toml.DecodeFile(bpPlanPath, &bpPlanOut); err != nil {
			return nil, err
		}
		var bpBOM []BOMEntry
		plan, bpBOM = plan.filter(bp, bpPlanOut)
		bom = append(bom, bpBOM...)

		var launch LaunchTOML
		tomlPath := filepath.Join(bpLayersDir, "launch.toml")
		if _, err := toml.DecodeFile(tomlPath, &launch); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		for i := range launch.Processes {
			launch.Processes[i].BuildpackID = bp.ID
		}
		procMap.add(launch.Processes)
		slices = append(slices, launch.Slices...)
		labels = append(labels, launch.Labels...)
	}

	if b.PlatformAPI.Compare(api.MustParse("0.4")) < 0 {
		//plaformApiVersion is less than comparisonVersion
		for i := range bom {
			if err := bom[i].convertMetadataToVersion(); err != nil {
				return nil, err
			}
		}
	} else {
		for i := range bom {
			if err := bom[i].convertVersionToMetadata(); err != nil {
				return nil, err
			}
		}
	}

	return &BuildMetadata{
		BOM:        bom,
		Buildpacks: b.Group.Group,
		Labels:     labels,
		Processes:  procMap.list(),
		Slices:     slices,
	}, nil
}

func (p BuildPlan) find(bp Buildpack) BuildpackPlan {
	var out []Require
	for _, entry := range p.Entries {
		for _, provider := range entry.Providers {
			if provider == bp {
				out = append(out, entry.Requires...)
				break
			}
		}
	}
	return BuildpackPlan{Entries: out}
}

// TODO: ensure at least one claimed entry of each name is provided by the BP
func (p BuildPlan) filter(bp Buildpack, plan BuildpackPlan) (BuildPlan, []BOMEntry) {
	var out []BuildPlanEntry
	for _, entry := range p.Entries {
		if !plan.has(entry) {
			out = append(out, entry)
		}
	}
	var bom []BOMEntry
	for _, entry := range plan.Entries {
		bom = append(bom, BOMEntry{Require: entry, Buildpack: bp.noAPI()})
	}
	return BuildPlan{Entries: out}, bom
}

func (p BuildpackPlan) has(entry BuildPlanEntry) bool {
	for _, buildEntry := range p.Entries {
		for _, req := range entry.Requires {
			if req.Name == buildEntry.Name {
				return true
			}
		}
	}
	return false
}

func (bom *BOMEntry) convertMetadataToVersion() error {
	if version, ok := bom.Metadata["version"]; ok {
		metadataVersion := fmt.Sprintf("%v", version)
		if bom.Version != "" && bom.Version != metadataVersion {
			return errors.New("top level version does not match metadata version")
		}
		bom.Version = metadataVersion
	}
	return nil
}

func (bom *BOMEntry) convertVersionToMetadata() error {
	if bom.Version != "" {
		if bom.Metadata == nil {
			bom.Metadata = make(map[string]interface{})
		}
		if version, ok := bom.Metadata["version"]; ok {
			metadataVersion := fmt.Sprintf("%v", version)
			if metadataVersion != "" && metadataVersion != bom.Version {
				return errors.New("metadata version does not match top level version")
			}
		}
		bom.Metadata["version"] = bom.Version
		bom.Version = ""
	}
	return nil
}

func setupEnv(env BuildEnv, layersDir string) error {
	if err := eachDir(layersDir, func(path string) error {
		if !isBuild(path + ".toml") {
			return nil
		}
		return env.AddRootDir(path)
	}); err != nil {
		return err
	}

	return eachDir(layersDir, func(path string) error {
		if !isBuild(path + ".toml") {
			return nil
		}
		if err := env.AddEnvDir(filepath.Join(path, "env")); err != nil {
			return err
		}
		return env.AddEnvDir(filepath.Join(path, "env.build"))
	})
}

func eachDir(dir string, fn func(path string) error) error {
	files, err := ioutil.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		if err := fn(filepath.Join(dir, f.Name())); err != nil {
			return err
		}
	}
	return nil
}

func isBuild(path string) bool {
	var layerTOML struct {
		Build bool `toml:"build"`
	}
	_, err := toml.DecodeFile(path, &layerTOML)
	return err == nil && layerTOML.Build
}

type processMap map[string]launch.Process

func (m processMap) add(l []launch.Process) {
	for _, proc := range l {
		m[proc.Type] = proc
	}
}

func (m processMap) list() []launch.Process {
	var keys []string
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	procs := []launch.Process{}
	for _, key := range keys {
		procs = append(procs, m[key])
	}
	return procs
}
