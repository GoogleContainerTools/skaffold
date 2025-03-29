package config

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/buildpacks/pack/internal/style"
)

func FormatUndecodedKeys(undecodedKeys []toml.Key) string {
	unusedKeys := map[string]interface{}{}
	for _, key := range undecodedKeys {
		keyName := key.String()

		parent := strings.Split(keyName, ".")[0]

		if _, ok := unusedKeys[parent]; !ok {
			unusedKeys[keyName] = nil
		}
	}

	var errorKeys []string
	for errorKey := range unusedKeys {
		errorKeys = append(errorKeys, style.Symbol(errorKey))
	}

	pluralizedElement := "element"
	if len(errorKeys) > 1 {
		pluralizedElement += "s"
	}
	elements := strings.Join(errorKeys, ", ")

	return fmt.Sprintf("unknown configuration %s %s", pluralizedElement, elements)
}
