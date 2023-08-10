package env

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
)

// Env is used to modify and return environment variables
type Env struct {
	// RootDirMap maps directories in a posix root filesystem to a slice of environment variables that
	RootDirMap map[string][]string
	Vars       *Vars
}

// AddRootDir modifies the environment given a root dir. If the root dir contains a directory that matches a key in
// the Env RooDirMap, the absolute path to the keyed directory will be prepended to all the associated environment variables
// using the OS path list separator as a delimiter.
func (p *Env) AddRootDir(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	for dir, vars := range p.RootDirMap {
		childDir := filepath.Join(absDir, dir)
		if _, err := os.Stat(childDir); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		for _, key := range vars {
			p.Vars.Set(key, childDir+prefix(p.Vars.Get(key), os.PathListSeparator))
		}
	}
	return nil
}

func (p *Env) isRootEnv(name string) bool {
	for _, m := range p.RootDirMap {
		for _, k := range m {
			if k == name {
				return true
			}
		}
	}
	return false
}

type ActionType string

const (
	ActionTypePrepend     ActionType = "prepend"
	ActionTypeAppend      ActionType = "append"
	ActionTypeOverride    ActionType = "override"
	ActionTypeDefault     ActionType = "default"
	ActionTypePrependPath ActionType = ""
)

// DefaultActionType returns the default action to perform for an unsuffixed env file as specified for the given
// buildpack API
func DefaultActionType(bpAPI *api.Version) ActionType {
	if bpAPI != nil && bpAPI.LessThan("0.5") {
		return ActionTypePrependPath
	}
	return ActionTypeOverride
}

// AddEnvDir modified the Env given a directory containing env files. For each file in the envDir, if the file has
// a period delimited suffix, the action matching the given suffix will be performed. If the file has no suffix,
// the default action will be performed. If the suffix does not match a known type, AddEnvDir will ignore the file.
func (p *Env) AddEnvDir(envDir string, defaultAction ActionType) error {
	if err := eachEnvFile(envDir, func(k, v string) error {
		parts := strings.SplitN(k, ".", 2)
		name := parts[0]
		var action ActionType
		if len(parts) > 1 {
			action = ActionType(parts[1])
		} else {
			action = defaultAction
		}
		switch action {
		case ActionTypePrepend:
			p.Vars.Set(name, v+prefix(p.Vars.Get(name), delim(envDir, name)...))
		case ActionTypeAppend:
			p.Vars.Set(name, suffix(p.Vars.Get(name), delim(envDir, name)...)+v)
		case ActionTypeOverride:
			p.Vars.Set(name, v)
		case ActionTypeDefault:
			if p.Vars.Get(name) != "" {
				return nil
			}
			p.Vars.Set(name, v)
		case ActionTypePrependPath:
			p.Vars.Set(name, v+prefix(p.Vars.Get(name), delim(envDir, name, os.PathListSeparator)...))
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "apply env files from dir '%s'", envDir)
	}
	return nil
}

// Set sets the environment variable with the given name to the given value.
func (p *Env) Set(name, v string) {
	p.Vars.Set(name, v)
}

// WithPlatform returns the environment after applying modifications from the given platform dir.
// For each file in the platformDir, if the name of the file does not match an environment variable name in the
// RootDirMap, the given variable will be set to the contents of the file. If the name does match an environment
// variable name in the RootDirMap, the contents of the file will be prepended to the environment variable value
// using the OS path list separator as a delimiter.
func (p *Env) WithPlatform(platformDir string) (out []string, err error) {
	vars := NewVars(p.Vars.vals, p.Vars.ignoreCase)

	if err := eachEnvFile(filepath.Join(platformDir, "env"), func(k, v string) error {
		if p.isRootEnv(k) {
			vars.Set(k, v+prefix(vars.Get(k), os.PathListSeparator))
			return nil
		}
		vars.Set(k, v)
		return nil
	}); err != nil {
		return nil, err
	}
	return vars.List(), nil
}

func prefix(s string, prefix ...byte) string {
	if s == "" {
		return ""
	}
	return string(prefix) + s
}

func suffix(s string, suffix ...byte) string {
	if s == "" {
		return ""
	}
	return s + string(suffix)
}

func delim(dir, name string, def ...byte) []byte {
	value, err := ioutil.ReadFile(filepath.Join(dir, name+".delim"))
	if err != nil {
		return def
	}
	return value
}

func eachEnvFile(dir string, fn func(k, v string) error) error {
	files, err := ioutil.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			lnFile, err := os.Stat(filepath.Join(dir, f.Name()))
			if err != nil {
				return err
			}
			if lnFile.IsDir() {
				continue
			}
		}
		value, err := ioutil.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return err
		}
		if err := fn(f.Name(), string(value)); err != nil {
			return err
		}
	}
	return nil
}

// List returns the environment
func (p *Env) List() []string {
	return p.Vars.List()
}

// Get returns the value for the given key
func (p *Env) Get(k string) string {
	return p.Vars.Get(k)
}
