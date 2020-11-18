package lifecycle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"
)

func WriteTOML(path string, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(data)
}

func ReadGroup(path string) (BuildpackGroup, error) {
	var group BuildpackGroup
	_, err := toml.DecodeFile(path, &group)
	return group, err
}

func ReadOrder(path string) (BuildpackOrder, error) {
	var order struct {
		Order BuildpackOrder `toml:"order"`
	}
	_, err := toml.DecodeFile(path, &order)
	return order.Order, err
}

func TruncateSha(sha string) string {
	rawSha := strings.TrimPrefix(sha, "sha256:")
	if len(sha) > 12 {
		return rawSha[0:12]
	}
	return rawSha
}

func DecodeLabel(image imgutil.Image, label string, v interface{}) error {
	if !image.Found() {
		return nil
	}
	contents, err := image.Label(label)
	if err != nil {
		return errors.Wrapf(err, "retrieving label '%s' for image '%s'", label, image.Name())
	}
	if contents == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(contents), v); err != nil {
		return errors.Wrapf(err, "failed to unmarshal context of label '%s'", label)
	}
	return nil
}

func syncLabels(sourceImg imgutil.Image, destImage imgutil.Image, test func(string) bool) error {
	if err := removeLabels(destImage, test); err != nil {
		return err
	}
	return copyLabels(sourceImg, destImage, test)
}

func removeLabels(image imgutil.Image, test func(string) bool) error {
	labels, err := image.Labels()
	if err != nil {
		return err
	}

	for label := range labels {
		if test(label) {
			if err := image.RemoveLabel(label); err != nil {
				return errors.Wrapf(err, "failed to remove label '%s'", label)
			}
		}
	}
	return nil
}

func copyLabels(fromImage imgutil.Image, destImage imgutil.Image, test func(string) bool) error {
	fromLabels, err := fromImage.Labels()
	if err != nil {
		return err
	}

	for label, labelValue := range fromLabels {
		if test(label) {
			if err := destImage.SetLabel(label, labelValue); err != nil {
				return err
			}
		}
	}
	return nil
}
