package fsutil

import (
	"os"
	"strings"
)

type OSInfo struct {
	Name    string
	Version string
}

type Detector interface {
	HasSystemdFile() bool
	ReadSystemdFile() (string, error)
	GetInfo(osReleaseContents string) OSInfo
}

type Detect struct {
}

func (d *Detect) HasSystemdFile() bool {
	finfo, err := os.Stat("/etc/os-release")
	if err != nil {
		return false
	}
	return !finfo.IsDir() && finfo.Size() > 0
}

func (d *Detect) ReadSystemdFile() (string, error) {
	bs, err := os.ReadFile("/etc/os-release")
	return string(bs), err
}

func (d *Detect) GetInfo(osReleaseContents string) OSInfo {
	ret := OSInfo{}
	lines := strings.Split(osReleaseContents, "\n")
	for _, line := range lines {
		// os-release is described as a CSV file with "=" as the separator char, but it's also a key-value pairs file.
		parts := strings.Split(line, "=")
		if len(parts) > 2 {
			continue // this shouldn't happen but what's an error, really?
		}
		toTrim := "\" " // we'll strip these chars (runes) from the string. What's a parser, really?
		if parts[0] == "ID" {
			ret.Name = strings.Trim(parts[1], toTrim)
		} else if parts[0] == "VERSION_ID" {
			ret.Version = strings.Trim(parts[1], toTrim)
		}
		if len(ret.Name) > 0 && len(ret.Version) > 0 {
			break
		}
	}
	return ret
}
