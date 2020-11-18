package registry

import (
	"bytes"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
)

type GithubIssue struct {
	Title string
	Body  string
}

func CreateGithubIssue(b Buildpack) (GithubIssue, error) {
	titleTemplate, err := template.New("buildpack").Parse(GithubIssueTitleTemplate)
	if err != nil {
		return GithubIssue{}, err
	}

	bodyTemplate, err := template.New("buildpack").Parse(GithubIssueBodyTemplate)
	if err != nil {
		return GithubIssue{}, err
	}

	var title bytes.Buffer
	err = titleTemplate.Execute(&title, b)
	if err != nil {
		return GithubIssue{}, err
	}

	var body bytes.Buffer
	err = bodyTemplate.Execute(&body, b)
	if err != nil {
		return GithubIssue{}, err
	}

	return GithubIssue{
		title.String(),
		body.String(),
	}, nil
}

func CreateBrowserCmd(browserURL, os string) (*exec.Cmd, error) {
	_, err := url.ParseRequestURI(browserURL)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid URL %s", style.Symbol(browserURL))
	}

	switch os {
	case "linux":
		return exec.Command("xdg-open", browserURL), nil
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", browserURL), nil
	case "darwin":
		return exec.Command("open", browserURL), nil
	default:
		return nil, fmt.Errorf("unsupported platform %s", style.Symbol(os))
	}
}

func GetIssueURL(githubURL string) (*url.URL, error) {
	if githubURL == "" {
		return nil, errors.New("missing github URL")
	}
	return url.Parse(fmt.Sprintf("%s/issues/new", strings.TrimSuffix(githubURL, "/")))
}
