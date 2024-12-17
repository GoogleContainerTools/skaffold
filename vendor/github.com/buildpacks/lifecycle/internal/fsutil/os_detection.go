package fsutil

import (
	"os"
	"strings"
	"sync"

	"github.com/buildpacks/lifecycle/log"
)

type OSInfo struct {
	Name    string
	Version string
}

type Detector interface {
	HasSystemdFile() bool
	ReadSystemdFile() (string, error)
	GetInfo(osReleaseContents string) OSInfo
	StoredInfo() *OSInfo
	InfoOnce(logger log.Logger)
}

// DefaultDetector implements Detector
type DefaultDetector struct {
	once sync.Once
	info *OSInfo
}

// HasSystemdFile returns true if /etc/os-release exists with contents
func (d *DefaultDetector) HasSystemdFile() bool {
	finfo, err := os.Stat("/etc/os-release")
	if err != nil {
		return false
	}
	return !finfo.IsDir() && finfo.Size() > 0
}

// ReadSystemdFile returns the contents of /etc/os-release
func (d *DefaultDetector) ReadSystemdFile() (string, error) {
	bs, err := os.ReadFile("/etc/os-release")
	return string(bs), err
}

// GetInfo returns the OS distribution name and version from the contents of /etc/os-release
func (d *DefaultDetector) GetInfo(osReleaseContents string) OSInfo {
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
	d.info = &ret // store for future use
	return ret
}

// StoredInfo returns any OSInfo found during the last call to GetInfo
func (d *DefaultDetector) StoredInfo() *OSInfo {
	return d.info
}

// InfoOnce logs an info message to the provided logger, but only once in the lifetime of the receiving DefaultDetector.
func (d *DefaultDetector) InfoOnce(logger log.Logger) {
	d.once.Do(func() {
		logger.Info("target distro name/version labels not found, reading /etc/os-release file")
	})
}
