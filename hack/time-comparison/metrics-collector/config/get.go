package config

import (
	"fmt"
	"io/ioutil"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

func Get(file string) (Applications, error) {
	contents, err := ioutil.ReadFile(file)
	if err != nil {
		return Applications{}, fmt.Errorf("reading %s: %w", file, err)
	}
	var apps Applications
	if err := yaml.Unmarshal(contents, &apps); err != nil {
		return Applications{}, fmt.Errorf("unmarshalling: %w", err)
	}
	return apps, nil
}
