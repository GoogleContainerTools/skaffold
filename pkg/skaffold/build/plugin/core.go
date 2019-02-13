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

package plugin

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugin/builders/docker"
	hashiplugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
)

// SkaffoldCorePluginExecutionMap maps the core plugin name to the execution function
var SkaffoldCorePluginExecutionMap = map[string]func() error{
	"docker": docker.Execute,
}

// ShouldExecuteCorePlugin returns true if env variables for plugins are set properly
// and the plugin passed in is a core plugin
func ShouldExecuteCorePlugin() bool {
	if os.Getenv(constants.SkaffoldPluginKey) != constants.SkaffoldPluginValue {
		return false
	}
	plugin := os.Getenv(constants.SkaffoldPluginName)
	_, ok := SkaffoldCorePluginExecutionMap[plugin]
	return ok
}

var cancelError error

// Execute executes a plugin, assumes ShouldExecuteCorePlugin has already been called
func Execute() error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	plugin := os.Getenv(constants.SkaffoldPluginName)

	errCh := make(chan error, 1)

	go func() {
		errCh <- SkaffoldCorePluginExecutionMap[plugin]()
	}()

	go func() {
		<-sigs
		errCh <- cancelError
	}()

	var err error
	err = <-errCh

	if err == cancelError {
		hashiplugin.CleanupClients()
	}
	if err != nil {
		return errors.Wrap(err, "executing plugin")
	}

	return nil
}
