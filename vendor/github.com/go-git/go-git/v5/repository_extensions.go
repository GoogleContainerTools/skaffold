package git

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/config"
	cfgformat "github.com/go-git/go-git/v5/plumbing/format/config"
	"github.com/go-git/go-git/v5/storage"
)

var (
	// ErrUnsupportedExtensionRepositoryFormatVersion represents when an
	// extension being used is not compatible with the repository's
	// core.repositoryFormatVersion.
	ErrUnsupportedExtensionRepositoryFormatVersion = errors.New("core.repositoryformatversion does not support extension")

	// ErrUnsupportedRepositoryFormatVersion represents when an repository
	// is using a format version that is not supported.
	ErrUnsupportedRepositoryFormatVersion = errors.New("core.repositoryformatversion not supported")

	// ErrUnknownExtension represents when a repository has an extension
	// which is unknown or unsupported by go-git.
	ErrUnknownExtension = errors.New("unknown extension")

	// builtinExtensions defines the Git extensions that are supported by
	// the core go-git implementation.
	//
	// Some extensions are storage-specific, those are defined by the Storers
	// themselves by implementing the ExtensionChecker interface.
	builtinExtensions = map[string]struct{}{
		// noop does not change git’s behavior at all.
		// It is useful only for testing format-1 compatibility.
		//
		// This extension is respected regardless of the
		// core.repositoryFormatVersion setting.
		"noop": {},

		// noop-v1 does not change git’s behavior at all.
		// It is useful only for testing format-1 compatibility.
		"noop-v1": {},
	}

	// Some Git extensions were supported upstream before the introduction
	// of repositoryformatversion. These are the only extensions that can be
	// enabled while core.repositoryformatversion is unset or set to 0.
	extensionsValidForV0 = map[string]struct{}{
		"noop":            {},
		"partialClone":    {},
		"preciousObjects": {},
		"worktreeConfig":  {},
	}
)

type extension struct {
	name  string
	value string
}

func extensions(cfg *config.Config) []extension {
	if cfg == nil || cfg.Raw == nil {
		return nil
	}

	if !cfg.Raw.HasSection("extensions") {
		return nil
	}

	section := cfg.Raw.Section("extensions")
	out := make([]extension, 0, len(section.Options))
	for _, opt := range section.Options {
		out = append(out, extension{name: strings.ToLower(opt.Key), value: strings.ToLower(opt.Value)})
	}

	return out
}

func verifyExtensions(st storage.Storer, cfg *config.Config) error {
	needed := extensions(cfg)

	switch cfg.Core.RepositoryFormatVersion {
	case "", cfgformat.Version_0, cfgformat.Version_1:
	default:
		return fmt.Errorf("%w: %q",
			ErrUnsupportedRepositoryFormatVersion,
			cfg.Core.RepositoryFormatVersion)
	}

	if len(needed) > 0 {
		if cfg.Core.RepositoryFormatVersion == cfgformat.Version_0 ||
			cfg.Core.RepositoryFormatVersion == "" {
			var unsupported []string
			for _, ext := range needed {
				if _, ok := extensionsValidForV0[ext.name]; !ok {
					unsupported = append(unsupported, ext.name)
				}
			}
			if len(unsupported) > 0 {
				return fmt.Errorf("%w: %s",
					ErrUnsupportedExtensionRepositoryFormatVersion,
					strings.Join(unsupported, ", "))
			}
		}

		var missing []string
		for _, ext := range needed {
			if _, ok := builtinExtensions[ext.name]; ok {
				continue
			}

			missing = append(missing, ext.name)
		}

		if len(missing) > 0 {
			return fmt.Errorf("%w: %s", ErrUnknownExtension, strings.Join(missing, ", "))
		}
	}

	return nil
}
