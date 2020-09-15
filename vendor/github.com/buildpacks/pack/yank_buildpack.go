package pack

import (
	"net/url"
	"runtime"

	"github.com/buildpacks/pack/internal/registry"
)

type YankBuildpackOptions struct {
	ID      string
	Version string
	Type    string
	URL     string
	Yank    bool
}

func (c *Client) YankBuildpack(opts YankBuildpackOptions) error {
	namespace, name, err := registry.ParseNamespaceName(opts.ID)
	if err != nil {
		return err
	}
	issueURL, err := registry.GetIssueURL(opts.URL)
	if err != nil {
		return err
	}

	buildpack := registry.Buildpack{
		Namespace: namespace,
		Name:      name,
		Version:   opts.Version,
		Yanked:    opts.Yank,
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
}
