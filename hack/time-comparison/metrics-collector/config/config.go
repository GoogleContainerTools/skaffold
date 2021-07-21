package config

// Applications is a list of test applications we are going to sample
type Applications struct {
	Apps []Application `yaml:"applications" yamltags:"required"`
}

// Application represents a single test application
type Application struct {
	Name    string            `yaml:"name" yamltags:"required"`
	Context string            `yaml:"context" yamltags:"required"`
	Dev     Dev               `yaml:"dev" yamltags:"required"`
	Labels  map[string]string `yaml:"labels" yamltags:"required"`
}

// Dev describes necessary info for running `skaffold dev` on a test application
type Dev struct {
	Command     string `yaml:"command" yamltags:"required"`
	UndoCommand string `yaml:"undoCommand,omitempty" yamltags:"required"`
}
