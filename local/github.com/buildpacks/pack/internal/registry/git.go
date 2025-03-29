package registry

import (
	"bytes"
	"text/template"

	"github.com/pkg/errors"
)

// GitCommit commits a Buildpack to a registry Cache.
func GitCommit(b Buildpack, username string, registryCache Cache) error {
	if err := registryCache.Initialize(); err != nil {
		return err
	}

	commitTemplate, err := template.New("buildpack").Parse(GitCommitTemplate)
	if err != nil {
		return err
	}

	var commit bytes.Buffer
	if err := commitTemplate.Execute(&commit, b); err != nil {
		return errors.Wrap(err, "creating template")
	}

	if err := registryCache.Commit(b, username, commit.String()); err != nil {
		return err
	}

	return nil
}
