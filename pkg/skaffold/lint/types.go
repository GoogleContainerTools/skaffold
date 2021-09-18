/*
Copyright 2021 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliecf.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lint

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/moby/buildkit/frontend/dockerfile/command"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

type Rule struct {
	RuleID      RuleID
	RuleType    RuleType
	Explanation string
	Result      Result
	Severity    string // TODO(aaron-prindle) make this an enum and plumb all throughout
	// TODO(aaron-prindle) split this out or refactor so that not all match types bundled here
	// Filter {}interface OR Filter interface?
	// Value ?
	LintString       string
	RegExp           string
	YamlField        string
	YamlFieldLinter  string
	YamlValue        string
	YamlFilter       yaml.Filter
	DockerCommand    string
	DockerCopyDest   string
	DockerCopySource string
	// TODO(aaron-prindle) generalize the LintConditions struct
	LintConditions []func(string) bool
}

type Result struct {
	RuleID      RuleID
	AbsFilePath string
	RelFilePath string
	Line        int
	Column      int
}

// type RegExpRule struct {
// 	RegExp string
// 	Rule
// }

type ConfigFile struct {
	AbsPath string
	RelPath string
	Text    string
}

type RuleType int

const (
	StringEqualsLintRule RuleType = iota
	RegExpLintLintRule
	YamlFieldLintRule
	DockerfileCommandLintRule
	BuildGraphConditionLintRule
	MetricsConditionLintRule
)

func (a RuleType) String() string {
	return [...]string{"StringEqualsLintRule", "RegExpLintLintRule", "YamlFieldLintRule", "DockerfileCommandLintRule", "BuildGraphConditionLintRule", "MetricsConditionLintRule"}[a]
}

type RuleID int

const (
	DOCKERFILE_COPY_DOT_OVER_100_FILES RuleID = iota

	SKAFFOLD_YAML_REPO_IS_HARD_CODED
	SKAFFOLD_YAML_API_VERSION_OUT_OF_DATE
	SKAFFOLD_YAML_SUGGEST_INFER_STANZA

	K8S_YAML_MANAGED_BY_LABEL_IS_IN_USE
)

func (a RuleID) String() string {
	// DFC:DockerfileCommand Lint Rule, REG: RegExp Lint Rule, YF:Yaml Field Lint Rule
	return [...]string{"DFC000001", "REG000001", "YF000001", "YF000002", "YF000003"}[a]
	// TODO(aaron-prindle) fix, hacky af
	// return fmt.Sprintf("ID%06d", a+1)
}

var RuleIDToLintRuleMap = map[RuleID]Rule{
	DOCKERFILE_COPY_DOT_OVER_100_FILES: {
		RuleType:         DockerfileCommandLintRule,
		DockerCommand:    command.Copy,
		DockerCopySource: ".",
		RuleID:           DOCKERFILE_COPY_DOT_OVER_100_FILES,
		// TODO(aaron-prindle) figure out how to best do conditions...
		// can do them here which is better or can hardcode them in the linters with the specific IDs
		Explanation: "Found 'COPY . <DEST>', for a source directory that has > 100 files.  This has the potential to dramatically slow 'skaffold dev' down by " +
			"having skaffold watch all of the files in the copied directory for changes. " +
			"If you notice skaffold rebuilding images unnecessarily when non-image-critical files are " +
			"modified, consider changing this to `COPY $REQUIRED_SOURCE_FILE <DEST>` for each required source file instead of " +
			"using 'COPY . <DEST>'",
		LintConditions: []func(string) bool{func(sourcePath string) bool {
			files := 0
			err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					logrus.Errorf("%s lint condition encountered error: %v", DOCKERFILE_COPY_DOT_OVER_100_FILES, err)
					return err
				}
				files++
				return nil
			})
			if err != nil {
				logrus.Errorf("%s lint condition encountered error: %v", DOCKERFILE_COPY_DOT_OVER_100_FILES, err)
				return false
			}
			return files > 100
		}},
	},
	K8S_YAML_MANAGED_BY_LABEL_IS_IN_USE: {
		YamlFilter: yaml.Lookup("metadata", "labels"),
		YamlValue:  "app.kubernetes.io/managed-by",
		RuleID:     K8S_YAML_MANAGED_BY_LABEL_IS_IN_USE,
		RuleType:   YamlFieldLintRule,
		Explanation: "Found usage of label 'app.kubernetes.io/managed-by'.  skaffold overwrites the 'app.kubernetes.io/managed-by' field to 'app.kubernetes.io/managed-by: skaffold'. " +
			"Remove this label or use the --dont-apply-managed-by-label flag to not have skaffold modify this label",
	},
	SKAFFOLD_YAML_API_VERSION_OUT_OF_DATE: {
		// TODO(aaron-prindle) check to see how kyaml supports regexp and how to best plumb that through
		YamlFilter: yaml.Get("apiVersion"),
		YamlValue:  "skaffold/v2beta21",
		// YamlValue:          version.Get().ConfigVersion,
		RuleID:   SKAFFOLD_YAML_API_VERSION_OUT_OF_DATE,
		RuleType: YamlFieldLintRule,
		Explanation: fmt.Sprintf("Found 'apiVersion' field with value that is not the latest skaffold apiVersion. Modify the apiVersion to the latest supported version: `apiVersion: %s` "+
			"or run the 'skaffold fix' command to have skaffold upgrade this for you", version.Get().ConfigVersion),
	},
	SKAFFOLD_YAML_REPO_IS_HARD_CODED: {
		// TODO(aaron-prindle) make a better recommendation regexp
		RegExp:   "gcr.io/|docker.io/|amazonaws.com/",
		RuleID:   SKAFFOLD_YAML_REPO_IS_HARD_CODED,
		RuleType: RegExpLintLintRule,
		Explanation: "Found image registry prefix on an image skaffold manages directly (eg: in a skaffold.yaml).  This is not recommended as it reduces the re-usability " +
			"of skaffold project as it disallows configuration of the image registries (it is hardcoded). " +
			"The image registry prefix should be removed and an image registry should be added programatically via skaffold, for example with the --default-repo flag",
	},
	SKAFFOLD_YAML_SUGGEST_INFER_STANZA: {
		// TODO(aaron-prindle) check to see how kyaml supports regexp and how to best plumb that through
		YamlFilter: yaml.Get("build"),
		YamlValue:  ".*",
		// YamlValue:          version.Get().ConfigVersion,

		// ideas on how this could be implemented
		// LintConditions
		//  - if no current 'infer' stanza
		//
		//  - if docker build
		//    (if Dockerfile found?)
		//  - if *.txt or *.html artifacts found in as artifacts in a docker graph
		//    - highlight build stanza and say to add
		/*
			   sync:
			     # Sync files with matching suffixes directly into container via rsync:
			     - '*.txt'
				 - '*.html'
		*/
		// TODO(aaron-prindle) verify rsync^^^^^ is correct term to use
		RuleID:   SKAFFOLD_YAML_SUGGEST_INFER_STANZA,
		RuleType: YamlFieldLintRule,
		Explanation: "Found files in docker build container image that should be synced (via rsync) vs watched for rebuilding image. " +
			"It is recommended to put the following stanza in the `build` section of the flagged skaffold.yaml:\n" + `    sync:
      # Sync files with matching suffixes directly into container via skaffold's rsync implementation for faster dev loop iteration
      - '*.txt'
      - '*.html'
	  `,
	},
}

var ignoreRules = map[RuleID]bool{}

type Linter interface {
	Lint(ConfigFile, *[]Rule) (*[]Result, error)
}

type DockerfileRulesList struct {
	DockerfileRules []Result `json:"dockerfileRules"`
}

type SkaffoldYamlRuleList struct {
	SkaffoldYamlRules []Result `json:"skaffoldYamls"`
}

type K8sYamlRuleList struct {
	K8sYamlRules []Result `json:"k8sYamls"`
}

type AllRuleLists struct {
	SkaffoldYamlRuleList SkaffoldYamlRuleList `json:"skaffoldYamlRuleList"`
	DockerfileRuleList   DockerfileRulesList  `json:"dockerfileRuleList"`
	K8sYamlRuleList      K8sYamlRuleList      `json:"k8sYamlRuleList"`
}

type RuleList struct {
	LinterResultList []Result `json:"recommendationList"`
}
