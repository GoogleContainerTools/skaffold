package app

import "github.com/buildpacks/pack/logging"

type Image struct {
	RepoName string
	Logger   logging.Logger
}
