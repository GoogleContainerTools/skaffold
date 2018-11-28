package partial

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1"
)

type testUIC struct {
	UncompressedImageCore
	configFile []byte
}

func (t testUIC) RawConfigFile() ([]byte, error) {
	return t.configFile, nil
}

type testCIC struct {
	CompressedImageCore
	configFile []byte
}

func (t testCIC) LayerByDigest(h v1.Hash) (CompressedLayer, error) {
	return nil, fmt.Errorf("no layer by diff ID %v", h)
}

func (t testCIC) RawConfigFile() ([]byte, error) {
	return t.configFile, nil
}

func TestConfigLayer(t *testing.T) {
	cases := []v1.Image{
		&compressedImageExtender{
			CompressedImageCore: testCIC{
				configFile: []byte("{}"),
			},
		},
		&uncompressedImageExtender{
			UncompressedImageCore: testUIC{
				configFile: []byte("{}"),
			},
		},
	}

	for _, image := range cases {
		hash, err := image.ConfigName()
		if err != nil {
			t.Fatalf("Error getting config name: %v", err)
		}

		if _, err := image.LayerByDigest(hash); err == nil {
			t.Error("LayerByDigest(config hash) returned nil error, wanted error")
		}

		layer, err := ConfigLayer(image)
		if err != nil {
			t.Errorf("ConfigLayer: %v", err)
		}
		lr, err := layer.Uncompressed()
		if err != nil {
			t.Fatalf("Error getting uncompressed layer: %v", err)
		}

		cfgLayerBytes, err := ioutil.ReadAll(lr)
		if err != nil {
			t.Fatalf("Error reading config layer bytes: %v", err)
		}

		cfgFile, err := image.RawConfigFile()
		if err != nil {
			t.Fatalf("Error getting raw config file: %v", err)
		}

		if string(cfgFile) != string(cfgLayerBytes) {
			t.Errorf("Config file layer doesn't match raw config file")
		}
	}
}
