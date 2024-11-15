package client

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/opencontainers/go-digest"

	"github.com/buildpacks/pack/pkg/logging"
)

const (
	dockerHostEnvVar            = "DOCKER_HOST"
	dockerConfigEnvVar          = "DOCKER_CONFIG"
	defaultDockerRootConfigDir  = ".docker"
	defaultDockerConfigFileName = "config.json"

	dockerContextDirName      = "contexts"
	dockerContextMetaDirName  = "meta"
	dockerContextMetaFileName = "meta.json"
	dockerContextEndpoint     = "docker"
	defaultDockerContext      = "default"
)

type configFile struct {
	CurrentContext string `json:"currentContext,omitempty"`
}

type endpoint struct {
	Host string `json:",omitempty"`
}

/*
	 Example Docker context file
	 {
	  "Name": "desktop-linux",
	  "dockerConfigMetadata": {
	    "Description": "Docker Desktop"
	  },
	  "Endpoints": {
	    "docker": {
	      "Host": "unix:///Users/jbustamante/.docker/run/docker.sock",
	      "SkipTLSVerify": false
	    }
	  }
	}
*/
type dockerConfigMetadata struct {
	Name      string              `json:",omitempty"`
	Endpoints map[string]endpoint `json:"endpoints,omitempty"`
}

func ProcessDockerContext(logger logging.Logger) error {
	dockerHost := os.Getenv(dockerHostEnvVar)
	if dockerHost == "" {
		dockerConfigDir, err := configDir()
		if err != nil {
			return err
		}

		logger.Debugf("looking for docker configuration file at: %s", dockerConfigDir)
		configuration, err := readConfigFile(dockerConfigDir)
		if err != nil {
			return errors.Wrapf(err, "reading configuration file at '%s'", dockerConfigDir)
		}

		if skip(configuration) {
			logger.Debug("docker context is default or empty, skipping it")
			return nil
		}

		configMetaData, err := readConfigMetadata(dockerConfigDir, configuration.CurrentContext)
		if err != nil {
			return errors.Wrapf(err, "reading metadata for current context '%s' at '%s'", configuration.CurrentContext, dockerConfigDir)
		}

		if dockerEndpoint, ok := configMetaData.Endpoints[dockerContextEndpoint]; ok {
			os.Setenv(dockerHostEnvVar, dockerEndpoint.Host)
			logger.Debugf("using docker context '%s' with endpoint = '%s'", configuration.CurrentContext, dockerEndpoint.Host)
		} else {
			logger.Warnf("docker endpoint doesn't exist for context '%s'", configuration.CurrentContext)
		}
	} else {
		logger.Debugf("'%s=%s' environment variable is being used", dockerHostEnvVar, dockerHost)
	}
	return nil
}

func configDir() (string, error) {
	dir := os.Getenv(dockerConfigEnvVar)
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", errors.Wrap(err, "determining user home directory")
		}
		dir = filepath.Join(home, defaultDockerRootConfigDir)
	}
	return dir, nil
}

func readConfigFile(configDir string) (*configFile, error) {
	filename := filepath.Join(configDir, defaultDockerConfigFileName)
	config := &configFile{}
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return &configFile{}, nil
		}
		return &configFile{}, err
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(config); err != nil && !errors.Is(err, io.EOF) {
		return &configFile{}, err
	}
	return config, nil
}

func readConfigMetadata(configDir string, context string) (dockerConfigMetadata, error) {
	dockerContextDir := filepath.Join(configDir, dockerContextDirName)
	metaFileName := filepath.Join(dockerContextDir, dockerContextMetaDirName, digest.FromString(context).Encoded(), dockerContextMetaFileName)
	bytes, err := os.ReadFile(metaFileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return dockerConfigMetadata{}, fmt.Errorf("docker context '%s' not found", context)
		}
		return dockerConfigMetadata{}, err
	}
	var meta dockerConfigMetadata
	if err := json.Unmarshal(bytes, &meta); err != nil {
		return dockerConfigMetadata{}, fmt.Errorf("parsing %s: %v", metaFileName, err)
	}
	if meta.Name != context {
		return dockerConfigMetadata{}, fmt.Errorf("context '%s' doesn't match metadata name '%s' at '%s'", context, meta.Name, metaFileName)
	}

	return meta, nil
}

func skip(configuration *configFile) bool {
	return configuration == nil || configuration.CurrentContext == defaultDockerContext || configuration.CurrentContext == ""
}
