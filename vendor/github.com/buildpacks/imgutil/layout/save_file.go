package layout

import "errors"

func (i *Image) SaveFile() (string, error) {
	// TODO issue https://github.com/buildpacks/imgutil/issues/170
	return "", errors.New("not yet implemented")
}
