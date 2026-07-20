package opts

import (
	"errors"
	"fmt"
	"strings"

	"github.com/moby/moby/api/types/mount"
)

// validateMountOptions performs client-side validation of mount options. Similar
// validation happens on the daemon side, but this validation allows us to
// produce user-friendly errors matching command-line options.
func validateMountOptions(m *mount.Mount) error {
	if err := validateExclusiveOptions(m); err != nil {
		return err
	}

	if m.BindOptions != nil {
		if m.BindOptions.ReadOnlyNonRecursive && !m.ReadOnly {
			return errors.New("option 'bind-recursive=writable' requires 'readonly' to be specified in conjunction")
		}
		if m.BindOptions.ReadOnlyForceRecursive {
			if !m.ReadOnly {
				return errors.New("option 'bind-recursive=readonly' requires 'readonly' to be specified in conjunction")
			}
			if m.BindOptions.Propagation != mount.PropagationRPrivate {
				// FIXME(thaJeztah): this is missing daemon-side validation
				//
				//	docker run --rm --mount type=bind,src=/var/run,target=/foo,bind-recursive=readonly,readonly alpine
				//	# no error
				return errors.New("option 'bind-recursive=readonly' requires 'bind-propagation=rprivate' to be specified in conjunction")
			}
		}
	}

	return nil
}

// validateExclusiveOptions checks if the given mount config only contains
// options for the given mount-type.
//
// This is the client-side equivalent of [mounts.validateExclusiveOptions] in
// the daemon, but with error-messages matching client-side flags / options.
//
// [mounts.validateExclusiveOptions]: https://github.com/moby/moby/blob/v2.0.0-beta.6/daemon/volume/mounts/validate.go#L31-L50
func validateExclusiveOptions(m *mount.Mount) error {
	if m.Type == "" {
		return errors.New("type is required")
	}

	if m.Type != mount.TypeBind && m.BindOptions != nil {
		return fmt.Errorf("cannot mix 'bind-*' options with mount type '%s'", m.Type)
	}
	if m.Type != mount.TypeVolume && m.VolumeOptions != nil {
		return fmt.Errorf("cannot mix 'volume-*' options with mount type '%s'", m.Type)
	}
	if m.Type != mount.TypeImage && m.ImageOptions != nil {
		return fmt.Errorf("cannot mix 'image-*' options with mount type '%s'", m.Type)
	}
	if m.Type != mount.TypeTmpfs && m.TmpfsOptions != nil {
		return fmt.Errorf("cannot mix 'tmpfs-*' options with mount type '%s'", m.Type)
	}
	if m.Type != mount.TypeCluster && m.ClusterOptions != nil {
		return fmt.Errorf("cannot mix 'cluster-*' options with mount type '%s'", m.Type)
	}
	return nil
}

// parseBoolValue returns the boolean value represented by the string. It returns
// true if no value is set.
//
// It is similar to [strconv.ParseBool], but only accepts 1, true, 0, false.
// Any other value returns an error.
func parseBoolValue(key string, val string, hasValue bool) (bool, error) {
	if !hasValue {
		return true, nil
	}
	switch val {
	case "1", "true":
		return true, nil
	case "0", "false":
		return false, nil
	default:
		return false, fmt.Errorf(`invalid value for '%s': invalid boolean value (%q): must be one of "true", "1", "false", or "0" (default "true")`, key, val)
	}
}

func ensureVolumeOptions(m *mount.Mount) *mount.VolumeOptions {
	if m.VolumeOptions == nil {
		m.VolumeOptions = &mount.VolumeOptions{}
	}
	return m.VolumeOptions
}

func ensureVolumeDriver(m *mount.Mount) *mount.Driver {
	ensureVolumeOptions(m)
	if m.VolumeOptions.DriverConfig == nil {
		m.VolumeOptions.DriverConfig = &mount.Driver{}
	}
	return m.VolumeOptions.DriverConfig
}

func ensureImageOptions(m *mount.Mount) *mount.ImageOptions {
	if m.ImageOptions == nil {
		m.ImageOptions = &mount.ImageOptions{}
	}
	return m.ImageOptions
}

func ensureBindOptions(m *mount.Mount) *mount.BindOptions {
	if m.BindOptions == nil {
		m.BindOptions = &mount.BindOptions{}
	}
	return m.BindOptions
}

func ensureTmpfsOptions(m *mount.Mount) *mount.TmpfsOptions {
	if m.TmpfsOptions == nil {
		m.TmpfsOptions = &mount.TmpfsOptions{}
	}
	return m.TmpfsOptions
}

func setValueOnMap(target map[string]string, keyValue string) map[string]string {
	k, v, _ := strings.Cut(keyValue, "=")
	if k == "" {
		return target
	}
	if target == nil {
		target = map[string]string{}
	}
	target[k] = v
	return target
}
