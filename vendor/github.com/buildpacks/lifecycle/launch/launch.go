package launch

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/buildpacks/lifecycle/api"
)

// Process represents a process to launch at runtime.
type Process struct {
	Type             string       `toml:"type" json:"type"`
	Command          RawCommand   `toml:"command" json:"command"`
	Args             []string     `toml:"args" json:"args"`
	Direct           bool         `toml:"direct" json:"direct"`
	Default          bool         `toml:"default,omitempty" json:"default,omitempty"`
	BuildpackID      string       `toml:"buildpack-id" json:"buildpackID"`
	WorkingDirectory string       `toml:"working-dir,omitempty" json:"working-dir,omitempty"`
	PlatformAPI      *api.Version `toml:"-" json:"-"`
}

func (p Process) NoDefault() Process {
	p.Default = false
	return p
}

func (p Process) WithPlatformAPI(platformAPI *api.Version) Process {
	// set on the process itself
	p.PlatformAPI = platformAPI
	// set on the command as well, this is needed when we serialize the command
	p.Command.PlatformAPI = platformAPI

	// for platform versions < 0.10
	// we only support a single command
	// push any extra entries into the args so they aren't lost
	if p.PlatformAPI.LessThan("0.10") {
		p.Args = append(p.Command.Entries[1:], p.Args[0:]...)
		p.Command.Entries = []string{p.Command.Entries[0]}
	}
	return p
}

type RawCommand struct {
	Entries     []string
	PlatformAPI *api.Version
}

func NewRawCommand(entries []string) RawCommand {
	return RawCommand{Entries: entries}
}

func (c RawCommand) WithPlatformAPI(api *api.Version) RawCommand {
	c.PlatformAPI = api
	return c
}

func (c RawCommand) MarshalTOML() ([]byte, error) {
	if c.PlatformAPI == nil {
		return nil, fmt.Errorf("missing PlatformAPI while encoding RawCommand")
	}

	if c.PlatformAPI.AtLeast("0.10") {
		buffer := &strings.Builder{}
		// turn array into toml array
		buffer.WriteString("[")
		for i, entry := range c.Entries {
			if i != 0 {
				buffer.WriteString(", ")
			}
			escaped, err := json.Marshal(entry) // properly escape special characters in single string
			if err != nil {
				return nil, err
			}
			buffer.WriteString(string(escaped))
		}
		buffer.WriteString("]")
		return []byte(buffer.String()), nil
	}

	return json.Marshal(c.Entries[0]) // properly escape special characters in single string
}

func (c RawCommand) MarshalJSON() ([]byte, error) {
	if c.PlatformAPI == nil {
		return nil, fmt.Errorf("missing PlatformAPI while encoding RawCommand")
	}

	if c.PlatformAPI.AtLeast("0.10") {
		return json.Marshal(c.Entries)
	}

	return json.Marshal(c.Entries[0])
}

// UnmarshalTOML implements toml.Unmarshaler and is needed because we read metadata.toml
// this method will attempt to parse the command in either string or array format
func (c *RawCommand) UnmarshalTOML(data interface{}) error {
	var entries []string
	// the raw value is either "the-command" or ["the-command", "arg1", "arg2"]
	// the latter is exposed as []interface{} by toml library and needs conversion
	switch v := data.(type) {
	case string:
		entries = []string{v}
	case []interface{}:
		s := make([]string, len(v))
		for i, el := range v {
			s[i] = fmt.Sprint(el)
		}
		entries = s
	default:
		return fmt.Errorf("unknown command type %T with data %v", data, data)
	}

	*c = NewRawCommand(entries)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler and is provided to help library consumers who need to read the build metadata label
// this method will attempt to parse the command in either string or array format
func (c *RawCommand) UnmarshalJSON(data []byte) error {
	var entries []string
	strData := string(data)
	// first try to decode array
	err := json.NewDecoder(strings.NewReader(strData)).Decode(&entries)
	if err != nil {
		// then try string
		var s string
		err = json.NewDecoder(strings.NewReader(strData)).Decode(&s)
		if err != nil {
			return err
		}
		entries = []string{s}
	}
	*c = NewRawCommand(entries)
	return nil
}

// ProcessPath returns the absolute path to the symlink for a given process type
func ProcessPath(pType string) string {
	return filepath.Join(ProcessDir, pType+exe)
}

type Metadata struct {
	Processes  []Process   `toml:"processes" json:"processes"`
	Buildpacks []Buildpack `toml:"buildpacks" json:"buildpacks"`
}

// Matches is used by goMock to compare two Metadata objects in tests
// when matching expected calls to methods containing Metadata objects
func (m Metadata) Matches(x interface{}) bool {
	metadatax, ok := x.(Metadata)
	if !ok {
		return false
	}

	// don't compare Processes directly, we will compare those individually next
	if s := cmp.Diff(metadatax, m, cmpopts.IgnoreFields(Metadata{}, "Processes")); s != "" {
		return false
	}

	// we need to ignore the PlatformAPI field because it isn't always set where these are used
	// and trying to compare it will cause a panic
	for i, p := range m.Processes {
		if s := cmp.Diff(metadatax.Processes[i], p,
			cmpopts.IgnoreFields(Process{}, "PlatformAPI"),
			cmpopts.IgnoreFields(RawCommand{}, "PlatformAPI")); s != "" {
			return false
		}
	}

	return true
}

func (m Metadata) String() string {
	return fmt.Sprintf("%+v %+v", m.Processes, m.Buildpacks)
}

func (m Metadata) FindProcessType(pType string) (Process, bool) {
	for _, p := range m.Processes {
		if p.Type == pType {
			return p, true
		}
	}
	return Process{}, false
}

type Buildpack struct {
	API string `toml:"api"`
	ID  string `toml:"id"`
}

func EscapeID(id string) string {
	return strings.ReplaceAll(id, "/", "_")
}

func GetMetadataFilePath(layersDir string) string {
	return path.Join(layersDir, "config", "metadata.toml")
}
