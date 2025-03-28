//go:build acceptance

package config

type PackAsset struct {
	path         string
	fixturePaths []string
}

func (a AssetManager) NewPackAsset(kind ComboValue) PackAsset {
	path, fixtures := a.PackPaths(kind)

	return PackAsset{
		path:         path,
		fixturePaths: fixtures,
	}
}

func (p PackAsset) Path() string {
	return p.path
}

func (p PackAsset) FixturePaths() []string {
	return p.fixturePaths
}
