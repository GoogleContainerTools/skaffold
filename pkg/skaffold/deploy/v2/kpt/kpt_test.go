/*
Copyright 2021 The Skaffold Authors

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

package kpt

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	v2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	kptfilePkgInit = `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
`
	kptfileLiveInit = `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
inventory:
  namespace: default
  inventoryID: 11111
`

	badkptfileLiveInit = `apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: skaffold
inventory:
  namespace: default
  inventoryID: 
  - bad-type
`
	manifests = `apiVersion: v1
kind: Pod
metadata:
   namespace: test-kptv2
spec:
   containers:
   - image: gcr.io/project/image1
   name: image1
`
)

func TestKptfileInitIfNot(t *testing.T) {
	tests := []struct {
		description string
		commands    util.Command
		// preExistKptfile creates the file content before `KptfileInitIfNot` run.
		preExistKptfile string
		// fakeKptfile overrides the Kptfile content (whether it pre-exists or
		// created by some `KptfileInitIfNot` processes).
		fakeKptfile string
		shouldErr   bool
		expectedErr string
	}{
		{
			description: "Kptfile not exist",
			commands: testutil.
				CmdRun("kpt pkg init .").
				AndRun("kpt live init ."),
			fakeKptfile: kptfilePkgInit,
			shouldErr:   false,
		},
		{
			description: "Kptfile pkg init failed",
			commands: testutil.
				CmdRunErr("kpt pkg init .", fmt.Errorf("fake err")),
			shouldErr:   true,
			expectedErr: "fake err",
		},
		{
			description: "Kptfile live init failed",
			commands: testutil.
				CmdRun("kpt pkg init .").
				AndRunErr("kpt live init .", fmt.Errorf("fake err")),
			fakeKptfile: kptfilePkgInit,
			shouldErr:   true,
			expectedErr: "fake err",
		},
		{
			description: "Kptfile parse err",
			commands: testutil.
				CmdRun("kpt pkg init .").
				AndRun("kpt live init ."),
			fakeKptfile: badkptfileLiveInit,
			shouldErr:   true,
			expectedErr: "unable to parse Kptfile",
		},
		{
			description:     "Kptfile exist, no Inventory info",
			commands:        testutil.CmdRun("kpt live init ."),
			preExistKptfile: kptfilePkgInit,
			fakeKptfile:     kptfilePkgInit,
			shouldErr:       false,
		},
		{
			description:     "Kptfile exist with Inventory info",
			preExistKptfile: kptfileLiveInit,
			fakeKptfile:     kptfileLiveInit,
			shouldErr:       false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)

			tmpDir := t.NewTempDir()
			t.Override(&openFile, func(f string) (*os.File, error) {
				tmpDir.Write(f, test.fakeKptfile)
				return os.OpenFile(filepath.Join(tmpDir.Root(), f), os.O_RDONLY, 0)
			})
			if test.preExistKptfile != "" {
				tmpDir.Write("Kptfile", test.preExistKptfile)
			}
			tmpDir.Chdir()

			k := NewDeployer(&kptConfig{}, nil, deploy.NoopComponentProvider, &latestV2.KptV2Deploy{Dir: "."},
				config.SkaffoldOptions{})
			err := kptfileInitIfNot(context.Background(), ioutil.Discard, k)
			if !test.shouldErr {
				t.CheckNoError(err)
			} else {
				t.CheckErrorContains(test.expectedErr, err)
			}
		})
	}
}

func TestDeploy(t *testing.T) {
	tests := []struct {
		description string
		builds      []graph.Artifact
		kpt         latestV2.KptV2Deploy
		commands    util.Command
	}{
		{
			description: "deploy succeeds",
			kpt:         latestV2.KptV2Deploy{Dir: "."},
			commands: testutil.
				CmdRunOut("kpt fn source .", manifests).
				AndRun("kpt live apply ."),
		},
		{
			description: "deploy succeeds",
			kpt:         latestV2.KptV2Deploy{Dir: "."},
			commands: testutil.
				CmdRunOut("kpt fn source .", manifests).
				AndRun("kpt live apply ."),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
			kptInitFunc = func(context.Context, io.Writer, *Deployer) error { return nil }

			k := NewDeployer(&kptConfig{}, nil, deploy.NoopComponentProvider, &test.kpt, config.SkaffoldOptions{})
			ns, err := k.Deploy(context.Background(), ioutil.Discard, test.builds)
			t.CheckNoError(err)
			t.CheckDeepEqual(ns, []string{"test-kptv2"})
		})
	}
}

type kptConfig struct {
	v2.RunContext // Embedded to provide the default values.
	workingDir    string
	config        string
}

func (c *kptConfig) WorkingDir() string                                    { return c.workingDir }
func (c *kptConfig) GetKubeContext() string                                { return "" }
func (c *kptConfig) GetKubeNamespace() string                              { return "" }
func (c *kptConfig) GetKubeConfig() string                                 { return c.config }
func (c *kptConfig) PortForwardResources() []*latestV2.PortForwardResource { return nil }
