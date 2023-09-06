package extend

type Dockerfile struct {
	ExtensionID string
	Path        string `toml:"path"`
	Args        []Arg
}

type Arg struct {
	Name  string `toml:"name"`
	Value string `toml:"value"`
}
