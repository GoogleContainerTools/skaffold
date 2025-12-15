package volumespec

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/moby/moby/api/types/mount"
)

const endOfSpec = rune(0)

// Parse parses a volume spec without any knowledge of the target platform
func Parse(spec string) (VolumeConfig, error) {
	volume := VolumeConfig{}

	switch len(spec) {
	case 0:
		return volume, errors.New("invalid empty volume spec")
	case 1, 2:
		volume.Target = spec
		volume.Type = string(mount.TypeVolume)
		return volume, nil
	}

	var buffer []rune
	for _, char := range spec + string(endOfSpec) {
		switch {
		case isWindowsDrive(buffer, char):
			buffer = append(buffer, char)
		case char == ':' || char == endOfSpec:
			if err := populateFieldFromBuffer(char, buffer, &volume); err != nil {
				populateType(&volume)
				return volume, fmt.Errorf("invalid spec: %s: %w", spec, err)
			}
			buffer = []rune{}
		default:
			buffer = append(buffer, char)
		}
	}

	populateType(&volume)
	return volume, nil
}

func isWindowsDrive(buffer []rune, char rune) bool {
	return char == ':' && len(buffer) == 1 && unicode.IsLetter(buffer[0])
}

func populateFieldFromBuffer(char rune, buffer []rune, volume *VolumeConfig) error {
	strBuffer := string(buffer)
	switch {
	case len(buffer) == 0:
		return errors.New("empty section between colons")
	// Anonymous volume
	case volume.Source == "" && char == endOfSpec:
		volume.Target = strBuffer
		return nil
	case volume.Source == "":
		volume.Source = strBuffer
		return nil
	case volume.Target == "":
		volume.Target = strBuffer
		return nil
	case char == ':':
		return errors.New("too many colons")
	}
	for _, option := range strings.Split(strBuffer, ",") {
		switch option {
		case "ro":
			volume.ReadOnly = true
		case "rw":
			volume.ReadOnly = false
		case "nocopy":
			volume.Volume = &VolumeOpts{NoCopy: true}
		default:
			if isBindOption(option) {
				volume.Bind = &BindOpts{Propagation: option}
			}
			// ignore unknown options
		}
	}
	return nil
}

func isBindOption(option string) bool {
	for _, propagation := range mount.Propagations {
		if mount.Propagation(option) == propagation {
			return true
		}
	}
	return false
}

func populateType(volume *VolumeConfig) {
	switch {
	// Anonymous volume
	case volume.Source == "":
		volume.Type = string(mount.TypeVolume)
	case isFilePath(volume.Source):
		volume.Type = string(mount.TypeBind)
	default:
		volume.Type = string(mount.TypeVolume)
	}
}

func isFilePath(source string) bool {
	switch source[0] {
	case '.', '/', '~':
		return true
	}
	if len([]rune(source)) == 1 {
		return false
	}

	// windows named pipes
	if strings.HasPrefix(source, `\\`) {
		return true
	}

	first, nextIndex := utf8.DecodeRuneInString(source)
	return isWindowsDrive([]rune{first}, rune(source[nextIndex]))
}
