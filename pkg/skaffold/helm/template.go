package helm

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// templateFields turns a release containing fields with templates in them into
// a release which has it's templates rendered.
func templateFields(tpl util.Templater, release latest.HelmRelease) (latest.HelmRelease, error) {
	releaseName, err := tpl.RenderNonEmpty(release.Name, release.Overrides.Values)
	if err != nil {
		return release, fmt.Errorf("unable to template release name: %w", err)
	}
	release.Name = releaseName

	chartVersion, err := tpl.Render(release.Version, release.Overrides.Values)
	if err != nil {
		return release, fmt.Errorf("unable to template chart version: %w", err)
	}
	release.Version = chartVersion

	repo, err := tpl.Render(release.Repo, release.Overrides.Values)
	if err != nil {
		return release, fmt.Errorf("unable to template repo: %w", err)
	}
	release.Repo = repo

	namespace, err := tpl.Render(release.Namespace, release.Overrides.Values)
	if err != nil {
		return release, fmt.Errorf("unable to template namespace: %w", err)
	}
	release.Namespace = namespace

	chartPath, err := tpl.Render(release.ChartPath, release.Overrides.Values)
	if err != nil {
		return release, fmt.Errorf("unable to template chart path: %w", err)
	}
	release.ChartPath = chartPath

	if release.Packaged != nil {
		packaged_version, err := tpl.Render(release.Packaged.Version, release.Overrides.Values)
		if err != nil {
			return release, fmt.Errorf("unable to template packaged.version: %w", err)
		}
		release.Packaged.Version = packaged_version

		packagedAppVersion, err := tpl.Render(release.Packaged.AppVersion, release.Overrides.Values)
		if err != nil {
			return release, fmt.Errorf("unable to template packaged.appVersion: %w", err)
		}
		release.Packaged.AppVersion = packagedAppVersion
	}

	return release, nil
}

// Template all provided releases to have their fields filled
func TemplateReleases(tpl util.Templater, releases []latest.HelmRelease) ([]latest.HelmRelease, error) {
	rendered_releases := []latest.HelmRelease{}
	for _, release := range releases {
		rendered_release, err := templateFields(tpl, release)
		if err != nil {
			return nil, fmt.Errorf("expanding %q: %w", release.Name, err)
		}
		rendered_releases = append(rendered_releases, rendered_release)

	}

	return rendered_releases, nil
}
