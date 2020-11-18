package build

import "strings"

type mountPaths struct {
	volume    string
	separator string
}

func mountPathsForOS(os string) mountPaths {
	if os == "windows" {
		return mountPaths{
			volume:    `c:`,
			separator: `\`,
		}
	}
	return mountPaths{
		volume:    "",
		separator: "/",
	}
}

func (m mountPaths) join(parts ...string) string {
	return strings.Join(parts, m.separator)
}

func (m mountPaths) layersDir() string {
	return m.join(m.volume, "layers")
}

func (m mountPaths) stackPath() string {
	return m.join(m.layersDir(), "stack.toml")
}

func (m mountPaths) appDirName() string {
	return "workspace"
}

func (m mountPaths) appDir() string {
	return m.join(m.volume, m.appDirName())
}

func (m mountPaths) cacheDir() string {
	return m.join(m.volume, "cache")
}

func (m mountPaths) launchCacheDir() string {
	return m.join(m.volume, "launch-cache")
}

func (m mountPaths) platformDir() string {
	return m.join(m.volume, "platform")
}
