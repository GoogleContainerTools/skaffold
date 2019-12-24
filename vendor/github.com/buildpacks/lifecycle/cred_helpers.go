package lifecycle

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"github.com/google/go-containerregistry/pkg/name"
)

func SetupCredHelpers(dockerPath string, refs ...string) error {
	configPath := filepath.Join(dockerPath, "config.json")

	config := map[string]interface{}{}
	if f, err := os.Open(configPath); err == nil {
		err := json.NewDecoder(f).Decode(&config)
		if f.Close(); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if _, ok := config["credHelpers"]; !ok {
		config["credHelpers"] = make(map[string]interface{})
	}

	credHelpers := make(map[string]string)
	for _, refStr := range refs {
		ref, err := name.ParseReference(refStr, name.WeakValidation)
		if err != nil {
			return err
		}

		registry := ref.Context().RegistryStr()
		for _, ch := range []struct {
			domain string
			helper string
		}{
			{"([.]|^)gcr[.]io$", "gcr"},
			{"[.]amazonaws[.]", "ecr-login"},
			{"([.]|^)azurecr[.]io$", "acr"},
		} {
			match, err := regexp.MatchString("(?i)"+ch.domain, registry)
			if err != nil || !match {
				continue
			}
			credHelpers[registry] = ch.helper
		}
	}

	if len(credHelpers) == 0 {
		return nil
	}

	ch, ok := config["credHelpers"].(map[string]interface{})
	if !ok {
		return errors.New("failed to parse docker config 'credHelpers'")
	}

	for k, v := range credHelpers {
		if _, ok := ch[k]; !ok {
			ch[k] = v
		}
	}

	if err := os.MkdirAll(dockerPath, 0777); err != nil {
		return err
	}

	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(config)
}
