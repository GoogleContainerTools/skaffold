package cmd

import (
	"fmt"
	"sort"
	"sync"
)

var registry struct {
	sync.Mutex
	commands map[string]func()
}

// Register a boulder subcommand to be run when the binary name matches `name`.
func RegisterCommand(name string, f func()) {
	registry.Lock()
	defer registry.Unlock()

	if registry.commands == nil {
		registry.commands = make(map[string]func())
	}

	if registry.commands[name] != nil {
		panic(fmt.Sprintf("command %q was registered twice", name))
	}
	registry.commands[name] = f
}

func LookupCommand(name string) func() {
	registry.Lock()
	defer registry.Unlock()
	return registry.commands[name]
}

func AvailableCommands() []string {
	registry.Lock()
	defer registry.Unlock()
	var avail []string
	for name := range registry.commands {
		avail = append(avail, name)
	}
	sort.Strings(avail)
	return avail
}
