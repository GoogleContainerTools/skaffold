package build

import "strings"

type mountPaths struct {
	volume    string
	separator string
	workspace string
}

func mountPathsForOS(os, workspace string) mountPaths {
	if workspace == "" {
		workspace = "workspace"
	}
	if os == "windows" {
		return mountPaths{
			volume:    `c:`,
			separator: `\`,
			workspace: workspace,
		}
	}
	return mountPaths{
		volume:    "",
		separator: "/",
		workspace: workspace,
	}
}

func (m mountPaths) join(parts ...string) string {
	return strings.Join(parts, m.separator)
}

func (m mountPaths) cnbDir() string {
	return m.join(m.volume, "cnb")
}

func (m mountPaths) layersDir() string {
	return m.join(m.volume, "layers")
}

func (m mountPaths) stackPath() string {
	return m.join(m.layersDir(), "stack.toml")
}

func (m mountPaths) runPath() string {
	return m.join(m.layersDir(), "run.toml")
}

func (m mountPaths) projectPath() string {
	return m.join(m.layersDir(), "project-metadata.toml")
}

func (m mountPaths) reportPath() string {
	return m.join(m.layersDir(), "report.toml")
}

func (m mountPaths) appDirName() string {
	return m.workspace
}

func (m mountPaths) appDir() string {
	return m.join(m.volume, m.appDirName())
}

func (m mountPaths) cacheDir() string {
	return m.join(m.volume, "cache")
}

func (m mountPaths) kanikoCacheDir() string {
	return m.join(m.volume, "kaniko")
}

func (m mountPaths) launchCacheDir() string {
	return m.join(m.volume, "launch-cache")
}

func (m mountPaths) sbomDir() string {
	return m.join(m.volume, "layers", "sbom")
}
