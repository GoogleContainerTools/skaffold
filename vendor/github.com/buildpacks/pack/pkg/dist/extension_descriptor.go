package dist

import (
	"strings"

	"github.com/buildpacks/lifecycle/api"
)

type ExtensionDescriptor struct {
	WithAPI  *api.Version `toml:"api"`
	WithInfo ModuleInfo   `toml:"extension"`
}

func (e *ExtensionDescriptor) EnsureStackSupport(_ string, _ []string, _ bool) error {
	return nil
}

func (e *ExtensionDescriptor) EnsureTargetSupport(_, _, _, _ string) error {
	return nil
}

func (e *ExtensionDescriptor) EscapedID() string {
	return strings.ReplaceAll(e.Info().ID, "/", "_")
}

func (e *ExtensionDescriptor) Kind() string {
	return "extension"
}

func (e *ExtensionDescriptor) API() *api.Version {
	return e.WithAPI
}

func (e *ExtensionDescriptor) Info() ModuleInfo {
	return e.WithInfo
}

func (e *ExtensionDescriptor) Order() Order {
	return nil
}

func (e *ExtensionDescriptor) Stacks() []Stack {
	return nil
}

func (e *ExtensionDescriptor) Targets() []Target {
	return nil
}
