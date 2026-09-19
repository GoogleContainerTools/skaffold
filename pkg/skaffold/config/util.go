/*
Copyright 2019 The Skaffold Authors

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

package config

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dario.cat/mergo"
	"github.com/mitchellh/go-homedir"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	api_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/cluster"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	kubeclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	kubectx "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	timeutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/time"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

const (
	defaultConfigDir  = ".skaffold"
	defaultConfigFile = "config"
)

var (
	// config-file content
	configFile     *GlobalConfig
	configFileErr  error
	configFileOnce sync.Once

	// config for a kubeContext
	config     *ContextConfig
	configErr  error
	configOnce sync.Once

	ReadConfigFile             = readConfigFileCached
	GetConfigForCurrentKubectx = getConfigForCurrentKubectx
	DiscoverLocalRegistry      = discoverLocalRegistry

	current = time.Now

	// update global config with the time the survey was last taken
	updateLastTaken = "skaffold config set --survey --global last-taken %s"
	updateUserTaken = "skaffold config set --survey --global --id %s taken true"
	// update global config with the time the survey was last prompted
	updateLastPrompted = "skaffold config set --survey --global last-prompted %s"
)

// readConfigFileCached reads the specified file and returns the contents
// parsed into a GlobalConfig struct.
// This function will always return the identical data from the first read.
func readConfigFileCached(filename string) (*GlobalConfig, error) {
	configFileOnce.Do(func() {
		filenameOrDefault, err := ResolveConfigFile(filename)
		if err != nil {
			configFileErr = err
			log.Entry(context.TODO()).Warnf("Could not load global Skaffold defaults. Error resolving config file %q", filenameOrDefault)
			return
		}
		configFile, configFileErr = ReadConfigFileNoCache(filenameOrDefault)
		if configFileErr == nil {
			log.Entry(context.TODO()).Infof("Loaded Skaffold defaults from %q", filenameOrDefault)
		}
	})
	return configFile, configFileErr
}

// ResolveConfigFile determines the default config location, if the configFile argument is empty.
func ResolveConfigFile(configFile string) (string, error) {
	if configFile == "" {
		home, err := homedir.Dir()
		if err != nil {
			return "", fmt.Errorf("retrieving home directory: %w", err)
		}
		configFile = filepath.Join(home, defaultConfigDir, defaultConfigFile)
	}
	return configFile, util.VerifyOrCreateFile(configFile)
}

// ReadConfigFileNoCache reads the given config yaml file and unmarshals the contents.
// Only visible for testing, use ReadConfigFile instead.
func ReadConfigFileNoCache(configFile string) (*GlobalConfig, error) {
	contents, err := os.ReadFile(configFile)
	if err != nil {
		log.Entry(context.TODO()).Warnf("Could not load global Skaffold defaults. Error encounter while reading file %q", configFile)
		return nil, fmt.Errorf("reading global config: %w", err)
	}
	config := GlobalConfig{}
	if err := yaml.Unmarshal(contents, &config); err != nil {
		log.Entry(context.TODO()).Warnf("Could not load global Skaffold defaults. Error encounter while unmarshalling the contents of file %q", configFile)
		return nil, fmt.Errorf("unmarshalling global skaffold config: %w", err)
	}
	return &config, nil
}

// GetConfigForCurrentKubectx returns the specific config to be modified based on the kubeContext.
// Either returns the config corresponding to the provided or current context,
// or the global config.
func getConfigForCurrentKubectx(configFile string) (*ContextConfig, error) {
	configOnce.Do(func() {
		cfg, err := ReadConfigFile(configFile)
		if err != nil {
			configErr = err
			return
		}
		kubeconfig, err := kubectx.CurrentConfig()
		if err != nil {
			configErr = err
			return
		}
		config, configErr = getConfigForKubeContextWithGlobalDefaults(cfg, kubeconfig.CurrentContext)
	})

	return config, configErr
}

func getConfigForKubeContextWithGlobalDefaults(cfg *GlobalConfig, kubeContext string) (*ContextConfig, error) {
	if kubeContext == "" {
		if cfg.Global == nil {
			return &ContextConfig{}, nil
		}
		return cfg.Global, nil
	}

	var mergedConfig ContextConfig
	for _, contextCfg := range cfg.ContextConfigs {
		if util.RegexEqual(contextCfg.Kubecontext, kubeContext) {
			log.Entry(context.TODO()).Debugf("found config for context %q", kubeContext)
			mergedConfig = *contextCfg
		}
	}
	// in case there was no config for this kubeContext in cfg.ContextConfigs
	mergedConfig.Kubecontext = kubeContext

	if cfg.Global != nil {
		// if values are unset for the current context, retrieve
		// the global config and use its values as a fallback.
		if err := mergo.Merge(&mergedConfig, cfg.Global, mergo.WithAppendSlice); err != nil {
			return nil, fmt.Errorf("merging context-specific and global config: %w", err)
		}
	}
	return &mergedConfig, nil
}

func GetDefaultRepo(configFile string, cliValue *string) (string, error) {
	// CLI flag takes precedence.
	if cliValue != nil {
		return *cliValue, nil
	}
	cfg, err := GetConfigForCurrentKubectx(configFile)
	if err != nil {
		return "", err
	}
	if cfg.DefaultRepo != "" {
		log.Entry(context.TODO()).Infof("Using default-repo=%s from config", cfg.DefaultRepo)
	}
	return cfg.DefaultRepo, nil
}

func GetMultiLevelRepo(configFile string) (*bool, error) {
	cfg, err := GetConfigForCurrentKubectx(configFile)
	if err != nil {
		return nil, err
	}
	if cfg.MultiLevelRepo != nil {
		log.Entry(context.TODO()).Infof("Using multi-level-repo=%t from config", *cfg.MultiLevelRepo)
	}
	return cfg.MultiLevelRepo, nil
}

func GetInsecureRegistries(configFile string) ([]string, error) {
	cfg, err := GetConfigForCurrentKubectx(configFile)
	if err != nil {
		return nil, err
	}
	if len(cfg.InsecureRegistries) > 0 {
		log.Entry(context.TODO()).Infof("Using insecure-registries=%v from config", cfg.InsecureRegistries)
	}
	return cfg.InsecureRegistries, nil
}

func GetDebugHelpersRegistry(configFile string) (string, error) {
	cfg, err := GetConfigForCurrentKubectx(configFile)
	if err != nil {
		return "", err
	}

	if cfg.DebugHelpersRegistry != "" {
		log.Entry(context.TODO()).Infof("Using debug-helpers-registry=%s from config", cfg.DebugHelpersRegistry)
		return cfg.DebugHelpersRegistry, nil
	}
	return constants.DefaultDebugHelpersRegistry, nil
}

func GetCacheTag(configFile string) (string, error) {
	cfg, err := GetConfigForCurrentKubectx(configFile)
	if err != nil {
		log.Entry(context.TODO()).Errorf("Cannot read cache-tag from config: %v", err)
		return "", err
	}
	if cfg.CacheTag != "" {
		log.Entry(context.TODO()).Infof("Using cache-tag=%s from config", cfg.CacheTag)
	}
	return cfg.CacheTag, nil
}

func GetBuildXBuilder(configFile string) string {
	cfg, err := GetConfigForCurrentKubectx(configFile)
	if err != nil {
		log.Entry(context.TODO()).Errorf("Cannot read buildx-builder option from config: %v", err)
	} else if cfg.BuildXBuilder != "" {
		log.Entry(context.TODO()).Infof("Using buildx-builder=%s from config", cfg.BuildXBuilder)
		return cfg.BuildXBuilder
	}
	return ""
}

func GetDetectBuildX(configFile string) bool {
	return GetBuildXBuilder(configFile) != ""
}

type GetClusterOpts struct {
	ConfigFile      string
	DefaultRepo     StringOrUndefined
	MinikubeProfile string
	DetectMinikube  bool
}

func GetCluster(ctx context.Context, opts GetClusterOpts) (Cluster, error) {
	cfg, err := GetConfigForCurrentKubectx(opts.ConfigFile)
	if err != nil {
		return Cluster{}, err
	}

	kubeContext := cfg.Kubecontext
	isKindCluster, isK3dCluster := IsKindCluster(kubeContext), IsK3dCluster(kubeContext)
	isMixedPlatform := IsMixedPlatformCluster(ctx, kubeContext)

	var local bool
	switch {
	case opts.MinikubeProfile != "":
		local = true

	case cfg.LocalCluster != nil:
		log.Entry(context.TODO()).Infof("Using local-cluster=%t from config", *cfg.LocalCluster)
		local = *cfg.LocalCluster

	case kubeContext == constants.DefaultMinikubeContext ||
		kubeContext == constants.DefaultDockerForDesktopContext ||
		kubeContext == constants.DefaultDockerDesktopContext ||
		isKindCluster || isK3dCluster:
		local = true

	case opts.DetectMinikube:
		local = cluster.GetClient().IsMinikube(ctx, kubeContext)

	default:
		local = false
	}
	var defaultRepo = opts.DefaultRepo

	if local && defaultRepo.Value() == nil {
		registry, err := DiscoverLocalRegistry(ctx, kubeContext)
		switch {
		case err != nil:
			log.Entry(context.TODO()).Tracef("failed to discover local registry %v", err)
		case registry != nil:
			log.Entry(context.TODO()).Infof("using default-repo=%s from cluster configmap", *registry)
			return Cluster{
				Local:           local,
				LoadImages:      false,
				PushImages:      true,
				DefaultRepo:     NewStringOrUndefined(registry),
				IsMixedPlatform: isMixedPlatform,
			}, nil
		}
	}

	kindDisableLoad := cfg.KindDisableLoad != nil && *cfg.KindDisableLoad
	k3dDisableLoad := cfg.K3dDisableLoad != nil && *cfg.K3dDisableLoad

	// load images for local kind/k3d cluster unless explicitly disabled
	loadImages := local && ((isKindCluster && !kindDisableLoad) || (isK3dCluster && !k3dDisableLoad))

	// push images for remote cluster or local kind/k3d cluster with image loading disabled
	pushImages := !local || (isKindCluster && kindDisableLoad) || (isK3dCluster && k3dDisableLoad)

	return Cluster{
		Local:           local,
		LoadImages:      loadImages,
		PushImages:      pushImages,
		DefaultRepo:     defaultRepo,
		IsMixedPlatform: isMixedPlatform,
	}, nil
}

func IsMixedPlatformCluster(ctx context.Context, kubeContext string) bool {
	client, err := kubeclient.Client(kubeContext)
	if err != nil {
		return false
	}
	nodes, err := client.CoreV1().Nodes().List(ctx, api_v1.ListOptions{})
	if err != nil || nodes == nil {
		return false
	}
	set := make(map[string]struct{})
	for _, n := range nodes.Items {
		set[fmt.Sprintf("%s/%s", n.Status.NodeInfo.OperatingSystem, n.Status.NodeInfo.Architecture)] = struct{}{}

		if len(set) > 1 {
			return true
		}
	}
	return false
}

// IsKindCluster checks that the given `kubeContext` is talking to `kind`.
func IsKindCluster(kubeContext string) bool {
	switch {
	// With kind version < 0.6.0, the k8s context
	// is `[CLUSTER NAME]@kind`.
	// For eg: `cluster@kind`
	// the default name is `kind@kind`
	case strings.HasSuffix(kubeContext, "@kind"):
		return true

	// With kind version >= 0.6.0, the k8s context
	// is `kind-[CLUSTER NAME]`.
	// For eg: `kind-cluster`
	// the default name is `kind-kind`
	case strings.HasPrefix(kubeContext, "kind-"):
		return true

	default:
		return false
	}
}

// KindClusterName returns the internal kind name of a kubernetes cluster.
func KindClusterName(clusterName string) string {
	switch {
	// With kind version < 0.6.0, the k8s context
	// is `[CLUSTER NAME]@kind`.
	// For eg: `cluster@kind`
	// the default name is `kind@kind`
	case strings.HasSuffix(clusterName, "@kind"):
		return strings.TrimSuffix(clusterName, "@kind")

	// With kind version >= 0.6.0, the k8s context
	// is `kind-[CLUSTER NAME]`.
	// For eg: `kind-cluster`
	// the default name is `kind-kind`
	case strings.HasPrefix(clusterName, "kind-"):
		return strings.TrimPrefix(clusterName, "kind-")
	}

	return clusterName
}

// IsK3dCluster checks that the given `kubeContext` is talking to `k3d`.
func IsK3dCluster(kubeContext string) bool {
	return strings.HasPrefix(kubeContext, "k3d-")
}

// K3dClusterName returns the internal name of a k3d cluster.
func K3dClusterName(clusterName string) string {
	if strings.HasPrefix(clusterName, "k3d-") {
		return strings.TrimPrefix(clusterName, "k3d-")
	}
	return clusterName
}

func discoverLocalRegistry(ctx context.Context, kubeContext string) (*string, error) {
	clientset, err := kubeclient.Client(kubeContext)
	if err != nil {
		return nil, err
	}

	configMap, err := clientset.CoreV1().ConfigMaps("kube-public").Get(ctx, "local-registry-hosting", api_v1.GetOptions{})

	statusErr := &api_errors.StatusError{}
	switch {
	case errors.As(err, &statusErr) && statusErr.Status().Code == http.StatusNotFound:
		return nil, nil
	case err != nil:
		return nil, err
	}

	data, ok := configMap.Data["localRegistryHosting.v1"]
	if !ok {
		return nil, errors.New("invalid local-registry-hosting ConfigMap")
	}

	dst := struct {
		Host *string `yaml:"host"`
	}{}

	if err := yaml.Unmarshal([]byte(data), &dst); err != nil {
		return nil, errors.New("invalid local-registry-hosting ConfigMap")
	}

	return dst.Host, nil
}

func IsUpdateCheckEnabled(configfile string) bool {
	cfg, err := GetConfigForCurrentKubectx(configfile)
	if err != nil {
		return true
	}
	return IsUpdateCheckEnabledForContext(cfg)
}

func IsUpdateCheckEnabledForContext(cfg *ContextConfig) bool {
	return cfg == nil || cfg.UpdateCheck == nil || *cfg.UpdateCheck
}

func ShouldDisplayUpdateMsg(configfile string) bool {
	cfg, err := GetConfigForCurrentKubectx(configfile)
	if err != nil {
		return true
	}
	if cfg == nil || cfg.UpdateCheckConfig == nil {
		return true
	}
	return !timeutil.LessThan(cfg.UpdateCheckConfig.LastPrompted, 24*time.Hour)
}

// UpdateMsgDisplayed updates the `last-prompted` config for `update-config` in
// the skaffold config
func UpdateMsgDisplayed(configFile string) error {
	// Today's date
	today := current().Format(time.RFC3339)

	configFile, err := ResolveConfigFile(configFile)
	if err != nil {
		return err
	}
	fullConfig, err := ReadConfigFile(configFile)
	if err != nil {
		return err
	}
	if !IsUpdateCheckEnabledForContext(fullConfig.Global) {
		return nil
	}
	if fullConfig.Global == nil {
		fullConfig.Global = &ContextConfig{}
	}
	if fullConfig.Global.UpdateCheckConfig == nil {
		fullConfig.Global.UpdateCheckConfig = &UpdateConfig{}
	}
	fullConfig.Global.UpdateCheckConfig.LastPrompted = today
	err = WriteFullConfig(configFile, fullConfig)
	return err
}

func UpdateHaTSSurveyTaken(configFile string) error {
	// Today's date
	today := current().Format(time.RFC3339)
	ai := fmt.Sprintf(updateLastTaken, today)
	aiErr := fmt.Errorf("could not automatically update the survey timestamp - please run `%s`", ai)

	configFile, err := ResolveConfigFile(configFile)
	if err != nil {
		return aiErr
	}
	fullConfig, err := ReadConfigFile(configFile)
	if err != nil {
		return aiErr
	}
	if fullConfig.Global == nil {
		fullConfig.Global = &ContextConfig{}
	}
	if fullConfig.Global.Survey == nil {
		fullConfig.Global.Survey = &SurveyConfig{}
	}
	fullConfig.Global.Survey.LastTaken = today
	err = WriteFullConfig(configFile, fullConfig)
	if err != nil {
		return aiErr
	}
	return err
}

func UpdateGlobalSurveyPrompted(configFile string) error {
	// Today's date
	today := current().Format(time.RFC3339)
	ai := fmt.Sprintf(updateLastPrompted, today)
	aiErr := fmt.Errorf("could not automatically update the survey prompted timestamp - please run `%s`", ai)

	configFile, err := ResolveConfigFile(configFile)
	if err != nil {
		return aiErr
	}
	fullConfig, err := ReadConfigFile(configFile)
	if err != nil {
		return aiErr
	}
	if fullConfig.Global == nil {
		fullConfig.Global = &ContextConfig{}
	}
	if fullConfig.Global.Survey == nil {
		fullConfig.Global.Survey = &SurveyConfig{}
	}
	fullConfig.Global.Survey.LastPrompted = today
	err = WriteFullConfig(configFile, fullConfig)
	if err != nil {
		return aiErr
	}
	return err
}

func UpdateGlobalCollectMetrics(configFile string, collectMetrics bool) error {
	configFile, err := ResolveConfigFile(configFile)
	if err != nil {
		return err
	}
	fullConfig, err := ReadConfigFile(configFile)
	if err != nil {
		return err
	}
	if fullConfig.Global == nil {
		fullConfig.Global = &ContextConfig{}
	}
	fullConfig.Global.CollectMetrics = &collectMetrics
	err = WriteFullConfig(configFile, fullConfig)
	if err != nil {
		return err
	}
	return err
}

func WriteFullConfig(configFile string, cfg *GlobalConfig) error {
	contents, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	configFileOrDefault, err := ResolveConfigFile(configFile)
	if err != nil {
		return err
	}
	if err := os.WriteFile(configFileOrDefault, contents, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

func UpdateUserSurveyTaken(configFile string, id string) error {
	ai := fmt.Sprintf(updateUserTaken, id)
	aiErr := fmt.Errorf("could not automatically update the survey prompted timestamp - please run `%s`", ai)
	configFile, err := ResolveConfigFile(configFile)
	if err != nil {
		return aiErr
	}
	fullConfig, err := ReadConfigFile(configFile)
	if err != nil {
		return aiErr
	}
	if fullConfig.Global == nil {
		fullConfig.Global = &ContextConfig{}
	}
	if fullConfig.Global.Survey == nil {
		fullConfig.Global.Survey = &SurveyConfig{}
	}
	fullConfig.Global.Survey.UserSurveys = updatedUserSurveys(fullConfig.Global.Survey.UserSurveys, id)
	err = WriteFullConfig(configFile, fullConfig)
	if err != nil {
		return aiErr
	}
	return nil
}

func updatedUserSurveys(us []*UserSurvey, id string) []*UserSurvey {
	for _, s := range us {
		if s.ID == id {
			s.Taken = util.Ptr(true)
			return us
		}
	}
	return append(us, &UserSurvey{ID: id, Taken: util.Ptr(true)})
}
