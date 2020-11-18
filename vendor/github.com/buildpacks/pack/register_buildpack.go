package pack

import (
	"context"
	"errors"
	"net/url"
	"runtime"
	"strings"

	"github.com/buildpacks/pack/config"

	"github.com/buildpacks/pack/internal/buildpackage"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/registry"
)

// RegisterBuildpackOptions is a configuration struct that controls the
// behavior of the RegisterBuildpack function.
type RegisterBuildpackOptions struct {
	ImageName string
	Type      string
	URL       string
	Name      string
}

// RegisterBuildpack updates the Buildpack Registry with to include a new buildpack specified in
// the opts argument
func (c *Client) RegisterBuildpack(ctx context.Context, opts RegisterBuildpackOptions) error {
	appImage, err := c.imageFetcher.Fetch(ctx, opts.ImageName, false, config.PullAlways)
	if err != nil {
		return err
	}

	var buildpackInfo dist.BuildpackInfo
	if _, err := dist.GetLabel(appImage, buildpackage.MetadataLabel, &buildpackInfo); err != nil {
		return err
	}

	namespace, name, err := parseID(buildpackInfo.ID)
	if err != nil {
		return err
	}

	id, err := appImage.Identifier()
	if err != nil {
		return err
	}

	buildpack := registry.Buildpack{
		Namespace: namespace,
		Name:      name,
		Version:   buildpackInfo.Version,
		Address:   id.String(),
		Yanked:    false,
	}

	if opts.Type == "github" {
		issueURL, err := registry.GetIssueURL(opts.URL)
		if err != nil {
			return err
		}

		issue, err := registry.CreateGithubIssue(buildpack)
		if err != nil {
			return err
		}

		params := url.Values{}
		params.Add("title", issue.Title)
		params.Add("body", issue.Body)
		issueURL.RawQuery = params.Encode()

		c.logger.Debugf("Open URL in browser: %s", issueURL)
		cmd, err := registry.CreateBrowserCmd(issueURL.String(), runtime.GOOS)
		if err != nil {
			return err
		}

		return cmd.Start()
	} else if opts.Type == "git" {
		registryCache, err := c.getRegistry(c.logger, opts.Name)
		if err != nil {
			return err
		}

		username, err := parseUsernameFromURL(opts.URL)
		if err != nil {
			return err
		}

		if err := registry.GitCommit(buildpack, username, registryCache); err != nil {
			return err
		}
	}

	return nil
}

func parseUsernameFromURL(url string) (string, error) {
	parts := strings.Split(url, "/")
	if len(parts) < 3 {
		return "", errors.New("invalid url: cannot parse username from url")
	}
	if parts[3] == "" {
		return "", errors.New("invalid url: username is empty")
	}

	return parts[3], nil
}

func parseID(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) < 2 {
		return "", "", errors.New("invalid id: does not contain a namespace")
	} else if len(parts) > 2 {
		return "", "", errors.New("invalid id: contains unexpected characters")
	}

	return parts[0], parts[1], nil
}
