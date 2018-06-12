/*
Copyright 2018 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	opts      = &config.SkaffoldOptions{}
	v         string
	filename  string
	overwrite bool
)

var rootCmd = &cobra.Command{
	Use:   "skaffold",
	Short: "A tool that facilitates continuous development for Kubernetes applications.",
}

func NewSkaffoldCommand(out, err io.Writer) *cobra.Command {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := SetUpLogs(err, v); err != nil {
			return err
		}
		rootCmd.SilenceUsage = true
		logrus.Infof("Skaffold %+v", version.Get())
		return nil
	}

	rootCmd.SilenceErrors = true
	rootCmd.AddCommand(NewCmdCompletion(out))
	rootCmd.AddCommand(NewCmdVersion(out))
	rootCmd.AddCommand(NewCmdRun(out))
	rootCmd.AddCommand(NewCmdDev(out))
	rootCmd.AddCommand(NewCmdBuild(out))
	rootCmd.AddCommand(NewCmdDeploy(out))
	rootCmd.AddCommand(NewCmdDelete(out))
	rootCmd.AddCommand(NewCmdFix(out))
	rootCmd.AddCommand(NewCmdDocker(out))

	rootCmd.PersistentFlags().StringVarP(&v, "verbosity", "v", constants.DefaultLogLevel.String(), "Log level (debug, info, warn, error, fatal, panic")
	return rootCmd
}

func AddDevFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&opts.Cleanup, "cleanup", true, "Delete deployments after dev mode is interrupted")
}

func AddRunDevFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&filename, "filename", "f", "skaffold.yaml", "Filename or URL to the pipeline file")
	cmd.Flags().BoolVar(&opts.Notification, "toot", false, "Emit a terminal beep after the deploy is complete")
	cmd.Flags().StringArrayVarP(&opts.Profiles, "profile", "p", nil, "Activate profiles by name")
	cmd.Flags().StringVarP(&opts.Namespace, "namespace", "n", "", "Run Helm deployments in the specified namespace")
}

func AddFixFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&filename, "filename", "f", "skaffold.yaml", "Filename or URL to the pipeline file")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite original config with fixed config")
}

func SetUpLogs(out io.Writer, level string) error {
	logrus.SetOutput(out)
	lvl, err := logrus.ParseLevel(v)
	if err != nil {
		return errors.Wrap(err, "parsing log level")
	}
	logrus.SetLevel(lvl)
	return nil
}

func readConfiguration(filename string) (*config.SkaffoldConfig, error) {
	buf, err := util.ReadConfiguration(filename)
	if err != nil {
		return nil, errors.Wrap(err, "read skaffold config")
	}

	if filename == "docker-compose.yaml" {
		cfg, err := newConfigForCompose(filename, buf)
		if err != nil {
			return nil, errors.Wrap(err, "converting compose config to skaffold config")
		}
		return cfg, nil
	}

	apiVersion := &config.APIVersion{}
	if err := yaml.Unmarshal(buf, apiVersion); err != nil {
		return nil, errors.Wrap(err, "parsing api version")
	}

	if apiVersion.Version != config.LatestVersion {
		return nil, errors.New("Config version out of date: run `skaffold fix`")
	}

	cfg, err := config.GetConfig(buf, true)
	if err != nil {
		return nil, errors.Wrap(err, "parsing skaffold config")
	}

	// we already ensured that the versions match in the previous block,
	// so this type assertion is safe.
	latestConfig := cfg.(*config.SkaffoldConfig)

	err = latestConfig.ApplyProfiles(opts.Profiles)
	if err != nil {
		return nil, errors.Wrap(err, "applying profiles")
	}

	return latestConfig, nil
}

func newConfigForCompose(filename string, buf []byte) (*config.SkaffoldConfig, error) {
	parsedComposeFile, err := loader.ParseYAML(buf)
	if err != nil {
		return nil, err
	}

	configFile := types.ConfigFile{
		Filename: filename,
		Config:   parsedComposeFile,
	}

	configDetails := types.ConfigDetails{
		WorkingDir:  filepath.Dir(filename),
		ConfigFiles: []types.ConfigFile{configFile},
		Environment: environ(),
	}
	composeCfg, err := loader.Load(configDetails)
	if err != nil {
		return nil, err
	}
	return convertToSkaffoldConfig(composeCfg), nil
}

func convertToSkaffoldConfig(composeCfg *types.Config) *config.SkaffoldConfig {
	cfg := &config.SkaffoldConfig{
		Build: v1alpha2.BuildConfig{
			BuildType: v1alpha2.BuildType{
				LocalBuild: &v1alpha2.LocalBuild{},
			},
			Artifacts: []*v1alpha2.Artifact{},
		},
		Deploy: v1alpha2.DeployConfig{
			DeployType: v1alpha2.DeployType{
				ComposeDeploy: &v1alpha2.ComposeDeploy{},
			},
		},
	}
	if err := cfg.SetDefaultValues(); err != nil {
		logrus.Fatal(err)
	}
	for _, s := range composeCfg.Services {
		if s.Build.Context == "" {
			continue
		}
		cfg.Build.Artifacts = append(cfg.Build.Artifacts, &v1alpha2.Artifact{
			ImageName: s.Image,
			ArtifactType: v1alpha2.ArtifactType{
				DockerArtifact: &v1alpha2.DockerArtifact{
					DockerfilePath: s.Build.Dockerfile,
					BuildArgs:      s.Build.Args,
				},
			},
		})
	}
	if err := cfg.SetDefaultValues(); err != nil {
		logrus.Fatal(err)
	}
	return cfg
}

func environ() map[string]string {
	m := map[string]string{}
	for _, kv := range os.Environ() {
		kvSplit := strings.Split(kv, "=")
		m[kvSplit[0]] = kvSplit[1]
	}
	return m
}
