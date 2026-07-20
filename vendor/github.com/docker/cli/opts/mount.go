package opts

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/go-units"
	mounttypes "github.com/moby/moby/api/types/mount"
)

// MountOpt is a Value type for parsing mounts
type MountOpt struct {
	values []mounttypes.Mount
}

// Set a new mount value
//
//nolint:gocyclo
func (m *MountOpt) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("value is empty")
	}

	csvReader := csv.NewReader(strings.NewReader(value))
	fields, err := csvReader.Read()
	if err != nil {
		return err
	}

	mount := mounttypes.Mount{
		Type: mounttypes.TypeVolume, // default to volume mounts
	}

	for _, field := range fields {
		key, val, hasValue := strings.Cut(field, "=")
		if k := strings.TrimSpace(key); k != key {
			return fmt.Errorf("invalid option '%s' in '%s': option should not have whitespace", k, field)
		}
		if hasValue {
			v := strings.TrimSpace(val)
			if v == "" {
				return fmt.Errorf("invalid value for '%s': value is empty", key)
			}
			if v != val {
				return fmt.Errorf("invalid value for '%s' in '%s': value should not have whitespace", key, field)
			}
		}

		// TODO(thaJeztah): these options should not be case-insensitive.
		key = strings.ToLower(key)

		if !hasValue {
			switch key {
			case "readonly", "ro", "volume-nocopy", "bind-nonrecursive", "bind-create-src":
				// boolean values
			default:
				return fmt.Errorf("invalid field '%s' must be a key=value pair", field)
			}
		}

		switch key {
		case "type":
			mount.Type = mounttypes.Type(strings.ToLower(val))
		case "source", "src":
			mount.Source = val
			if !filepath.IsAbs(val) && strings.HasPrefix(val, ".") {
				if abs, err := filepath.Abs(val); err == nil {
					mount.Source = abs
				}
			}
		case "target", "dst", "destination":
			mount.Target = val
		case "readonly", "ro":
			mount.ReadOnly, err = parseBoolValue(key, val, hasValue)
			if err != nil {
				return err
			}
		case "consistency":
			mount.Consistency = mounttypes.Consistency(strings.ToLower(val))
		case "bind-propagation":
			ensureBindOptions(&mount).Propagation = mounttypes.Propagation(strings.ToLower(val))
		case "bind-nonrecursive":
			return errors.New("bind-nonrecursive is deprecated, use bind-recursive=disabled instead")
		case "bind-recursive":
			switch val {
			case "enabled": // read-only mounts are recursively read-only if Engine >= v25 && kernel >= v5.12, otherwise writable
				// NOP
			case "disabled": // previously "bind-nonrecursive=true"
				ensureBindOptions(&mount).NonRecursive = true
			case "writable": // conforms to the default read-only bind-mount of Docker v24; read-only mounts are recursively mounted but not recursively read-only
				ensureBindOptions(&mount).ReadOnlyNonRecursive = true
			case "readonly": // force recursively read-only, or raise an error
				ensureBindOptions(&mount).ReadOnlyForceRecursive = true
				// TODO: implicitly set propagation and error if the user specifies a propagation in a future refactor/UX polish pass
				// https://github.com/docker/cli/pull/4316#discussion_r1341974730
			default:
				return fmt.Errorf(`invalid value for %s: %s (must be "enabled", "disabled", "writable", or "readonly")`, key, val)
			}
		case "bind-create-src":
			ensureBindOptions(&mount).CreateMountpoint, err = parseBoolValue(key, val, hasValue)
			if err != nil {
				return err
			}
		case "volume-subpath":
			ensureVolumeOptions(&mount).Subpath = val
		case "volume-nocopy":
			ensureVolumeOptions(&mount).NoCopy, err = parseBoolValue(key, val, hasValue)
			if err != nil {
				return err
			}
		case "volume-label":
			volumeOpts := ensureVolumeOptions(&mount)
			volumeOpts.Labels = setValueOnMap(volumeOpts.Labels, val)
		case "volume-driver":
			ensureVolumeDriver(&mount).Name = val
		case "volume-opt":
			volumeDriver := ensureVolumeDriver(&mount)
			volumeDriver.Options = setValueOnMap(volumeDriver.Options, val)
		case "image-subpath":
			ensureImageOptions(&mount).Subpath = val
		case "tmpfs-size":
			sizeBytes, err := units.RAMInBytes(val)
			if err != nil {
				return fmt.Errorf("invalid value for %s: %s", key, val)
			}
			ensureTmpfsOptions(&mount).SizeBytes = sizeBytes
		case "tmpfs-mode":
			ui64, err := strconv.ParseUint(val, 8, 32)
			if err != nil {
				return fmt.Errorf("invalid value for %s: %s", key, val)
			}
			ensureTmpfsOptions(&mount).Mode = os.FileMode(ui64)
		default:
			return fmt.Errorf("unknown option '%s' in '%s'", key, field)
		}
	}

	if err := validateMountOptions(&mount); err != nil {
		return err
	}

	m.values = append(m.values, mount)
	return nil
}

// Type returns the type of this option
func (*MountOpt) Type() string {
	return "mount"
}

// String returns a string repr of this option
func (m *MountOpt) String() string {
	mounts := make([]string, 0, len(m.values))
	for _, mount := range m.values {
		repr := fmt.Sprintf("%s %s %s", mount.Type, mount.Source, mount.Target)
		mounts = append(mounts, repr)
	}
	return strings.Join(mounts, ", ")
}

// Value returns the mounts
func (m *MountOpt) Value() []mounttypes.Mount {
	return m.values
}
