package image

import (
	"encoding/json"

	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"
)

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

func SyncLabels(sourceImg imgutil.Image, destImage imgutil.Image, test func(string) bool) error {
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
