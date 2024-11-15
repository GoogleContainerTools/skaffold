// Package extend contains extensions specific helpers
package extend

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/buildpacks/lifecycle/log"
)

const (
	// BuildContextDir defines the context for build-image extensions
	BuildContextDir = "context.build"
	// RunContextDir defines the context for run-image extensions
	RunContextDir = "context.run"
	// SharedContextDir defines a shared context for build- and run-image extensions
	SharedContextDir = "context"
)

// ContextInfo captures info about the context used for build- and run-image extensions
type ContextInfo struct {
	ExtensionID string
	Path        string
}

// FindContexts returns paths to Build Context folders produced by an extension. If Shared Context is provided with Run or Build Contexts the error will be thrown.
func FindContexts(extensionID string, extOutputDir string, logger log.Logger) ([]ContextInfo, error) {
	var contexts []ContextInfo
	var sharedIsProvided bool

	sharedContextDir := filepath.Join(extOutputDir, SharedContextDir)
	if s, err := os.Stat(sharedContextDir); err == nil && s.IsDir() {
		logger.Debugf("found shared extension context %q", sharedContextDir)
		sharedIsProvided = true
		contexts = append(contexts, ContextInfo{
			ExtensionID: extensionID,
			Path:        sharedContextDir,
		})
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	for _, dir := range []string{BuildContextDir, RunContextDir} {
		contextDir := filepath.Join(extOutputDir, dir)

		if s, err := os.Stat(contextDir); err == nil && s.IsDir() {
			if sharedIsProvided {
				return nil, fmt.Errorf("image-specific context dir is provided together with a shared context")
			}

			logger.Debugf("found extension context %q", contextDir)
			contexts = append(contexts, ContextInfo{
				ExtensionID: extensionID,
				Path:        contextDir,
			})
		} else if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	return contexts, nil
}
