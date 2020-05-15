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

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	misc "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"io"
	"io/ioutil"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	yaml2 "gopkg.in/yaml.v2"
	yaml "gopkg.in/yaml.v3"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/validation"
)

var toVersion string

func NewCmdFix() *cobra.Command {
	return NewCmd("fix").
		WithDescription("Update old configuration to a newer schema version").
		WithExample("Update \"skaffold.yaml\" in the current folder to the latest version", "fix").
		WithExample("Update \"skaffold.yaml\" in the current folder to version \"skaffold/v1\"", "fix --version skaffold/v1").
		WithCommonFlags().
		WithFlags(func(f *pflag.FlagSet) {
			f.BoolVar(&overwrite, "overwrite", false, "Overwrite original config with fixed config")
			f.StringVar(&toVersion, "version", latest.Version, "Target schema version to upgrade to")
		}).
		NoArgs(doFix)
}

func doFix(_ context.Context, out io.Writer) error {
	return fix(out, opts.ConfigurationFile, toVersion, overwrite)
}

func fix(out io.Writer, configFile string, toVersion string, overwrite bool) error {
	cfg, err := schema.ParseConfig(configFile)
	if err != nil {
		return err
	}

	if cfg.GetVersion() == latest.Version {
		color.Default.Fprintln(out, "config is already latest version")
		return nil
	}

	upCfg, err := schema.ParseConfigAndUpgrade(configFile, toVersion)
	if err != nil {
		return err
	}

	if err := validation.Process(upCfg.(*latest.SkaffoldConfig)); err != nil {
		return fmt.Errorf("validating upgraded config: %w", err)
	}

	newCfg, err := marshallPreservingComments(configFile, upCfg)
	if err != nil {
		return fmt.Errorf("marshaling new config: %w", err)
	}

	if overwrite {
		if err := ioutil.WriteFile(configFile, newCfg, 0644); err != nil {
			return fmt.Errorf("writing config file: %w", err)
		}
		color.Default.Fprintf(out, "New config at version %s generated and written to %s\n", cfg.GetVersion(), opts.ConfigurationFile)
	} else {
		out.Write(newCfg)
	}

	return nil
}

func marshallPreservingComments(filename string, cfg util.VersionedConfig) ([]byte, error) {
	fallbackCfg, err := yaml2.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshaling new config: %w", err)
	}
	prev := yaml.Node{}
	buf, _ := misc.ReadConfiguration(filename)
	err = yaml.Unmarshal(buf, &prev)
	// If any error preserve old behavior and return cfg without comments.
	if err != nil {
		return fallbackCfg, nil
	}
	// marshal upgraded config with yaml3.Marshal.
	newNode := yaml.Node{}
	err = yaml.Unmarshal(fallbackCfg, &newNode)
	if err != nil {
		return fallbackCfg, nil
	}
	recursivelyCopyComment(prev.Content[0], newNode.Content[0])
	if newCfg, err := encode(&newNode); err == nil {
		return newCfg, nil
	}

	return fallbackCfg, nil
}

func recursivelyCopyComment(old *yaml.Node, newNode *yaml.Node) {
	newNode.HeadComment = old.HeadComment
	newNode.LineComment = old.LineComment
	newNode.FootComment = old.FootComment
	if old.Content == nil || newNode.Content == nil {
		return
	}
	renamed := false
	j := 0
	for i, c := range old.Content {
		if renamed && c.Value != newNode.Content[j].Value {
			j++
			continue
		}
		renamed = false
		if i > len(newNode.Content) {
			// break since no matching nodes in new cfg.
			// this might happen in case of deletions.
			return
		}
		if c.Value != newNode.Content[j].Value {
			// rename or additions happened.
			renamed = true
		}
		recursivelyCopyComment(c, newNode.Content[j])
		j++
	}
}

func encode(in interface{}) (out []byte, err error) {
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	if err := encoder.Encode(in); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
