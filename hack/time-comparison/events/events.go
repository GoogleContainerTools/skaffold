package events

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	v1 "github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/golang/protobuf/jsonpb"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

var (
	EventsFileAbsPath string
)

// Get returns a list of entries
func Get(contents []byte) ([]v1.LogEntry, error) {
	entries := strings.Split(string(contents), "\n")
	var logEntries []v1.LogEntry
	unmarshaller := jsonpb.Unmarshaler{}
	for _, entry := range entries {
		if entry == "" {
			continue
		}
		var logEntry v1.LogEntry
		buf := bytes.NewBuffer([]byte(entry))
		if err := unmarshaller.Unmarshal(buf, &logEntry); err != nil {
			return nil, errors.Wrap(err, "unmarshalling")
		}
		logEntries = append(logEntries, logEntry)
	}
	return logEntries, nil
}

func GetFromFile(fp string) ([]v1.LogEntry, error) {
	contents, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, errors.Wrapf(err, "reading %s", fp)
	}
	return Get(contents)
}

func Cleanup() {
	defer os.Remove(EventsFileAbsPath)
	EventsFileAbsPath = ""
}

func File() (string, error) {
	if EventsFileAbsPath != "" {
		return EventsFileAbsPath, nil
	}
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("homedir: %w", err)
	}
	f, err := ioutil.TempFile(home, "events")
	if err != nil {
		return "", fmt.Errorf("temp file: %w", err)
	}
	defer f.Close()
	EventsFileAbsPath = f.Name()
	return f.Name(), nil
}
