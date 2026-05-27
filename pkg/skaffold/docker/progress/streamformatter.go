/*
Copyright 2024 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package progress

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/jsonstream"
)

const streamNewline = "\r\n"

func appendNewline(source []byte) []byte {
	return append(source, '\r', '\n')
}

type jsonProgressFormatter struct{}

func (sf *jsonProgressFormatter) format(prog Progress) []byte {
	if sf == nil {
		return nil
	}
	if prog.Message != "" {
		return sf.formatStatus(prog.ID, prog.Message)
	}
	jsonProgress := jsonstream.Progress{
		Current:    prog.Current,
		Total:      prog.Total,
		HideCounts: prog.HideCounts,
		Units:      prog.Units,
	}
	return sf.formatProgress(prog.ID, prog.Action, &jsonProgress, prog.Aux)
}

func (sf *jsonProgressFormatter) emptyMessage() []byte {
	return []byte(`{}` + streamNewline)
}

// formatStatus formats the id and status.
func (sf *jsonProgressFormatter) formatStatus(id, status string) []byte {
	b, err := json.Marshal(&jsonstream.Message{
		ID:     id,
		Status: status,
	})
	if err != nil {
		// should never happen with the given struct.
		return nil
	}
	return appendNewline(b)
}

// formatProgress formats the progress information for a specified action.
func (sf *jsonProgressFormatter) formatProgress(id, action string, progress *jsonstream.Progress, aux any) []byte {
	var auxJSON *json.RawMessage
	if aux != nil {
		b, err := json.Marshal(aux)
		if err != nil {
			return nil
		}
		auxJSON = (*json.RawMessage)(&b)
	}
	if progress == nil {
		progress = &jsonstream.Progress{}
	}
	b, err := json.Marshal(&jsonstream.Message{
		Status:   action,
		Progress: progress,
		ID:       id,
		Aux:      auxJSON,
	})
	if err != nil {
		return nil
	}
	return appendNewline(b)
}

type rawProgressFormatter struct{}

func (sf *rawProgressFormatter) format(prog Progress) []byte {
	if sf == nil {
		return nil
	}
	if prog.Message != "" {
		return []byte(prog.Message + streamNewline)
	}

	return sf.formatProgress(prog.Action, &jsonstream.Progress{
		Current:    prog.Current,
		Total:      prog.Total,
		HideCounts: prog.HideCounts,
		Units:      prog.Units,
	})
}

func (sf *rawProgressFormatter) emptyMessage() []byte {
	return []byte(streamNewline)
}

func rawProgressString(p *jsonstream.Progress) string {
	if p == nil || (p.Current <= 0 && p.Total <= 0) {
		return ""
	}
	if p.Total <= 0 {
		switch p.Units {
		case "":
			return fmt.Sprintf("%8v", units.HumanSize(float64(p.Current)))
		default:
			return fmt.Sprintf("%d %s", p.Current, p.Units)
		}
	}

	percentage := min(int(float64(p.Current)/float64(p.Total)*100)/2, 50)

	numSpaces := max(50-percentage, 0)
	pbBox := fmt.Sprintf("[%s>%s] ", strings.Repeat("=", percentage), strings.Repeat(" ", numSpaces))

	var numbersBox string
	switch {
	case p.HideCounts:
	case p.Units == "": // no units, use bytes
		current := units.HumanSize(float64(p.Current))
		total := units.HumanSize(float64(p.Total))

		numbersBox = fmt.Sprintf("%8v/%v", current, total)

		if p.Current > p.Total {
			// remove total display if the reported current is wonky.
			numbersBox = fmt.Sprintf("%8v", current)
		}
	default:
		numbersBox = fmt.Sprintf("%d/%d %s", p.Current, p.Total, p.Units)

		if p.Current > p.Total {
			// remove total display if the reported current is wonky.
			numbersBox = fmt.Sprintf("%d %s", p.Current, p.Units)
		}
	}

	var timeLeftBox string
	if p.Current > 0 && p.Start > 0 && percentage < 50 {
		fromStart := time.Since(time.Unix(p.Start, 0))
		perEntry := fromStart / time.Duration(p.Current)
		left := time.Duration(p.Total-p.Current) * perEntry
		timeLeftBox = " " + left.Round(time.Second).String()
	}
	return pbBox + numbersBox + timeLeftBox
}

func (sf *rawProgressFormatter) formatProgress(action string, progress *jsonstream.Progress) []byte {
	endl := "\r"
	out := rawProgressString(progress)
	if out == "" {
		endl += "\n"
	}
	return []byte(action + " " + out + endl)
}

// NewProgressOutput returns a progress.Output object that can be passed to
// progress.NewProgressReader.
func NewProgressOutput(out io.Writer) Output {
	return &progressOutput{sf: &rawProgressFormatter{}, out: out, newLines: true}
}

// NewJSONProgressOutput returns a progress.Output that formats output
// using JSON objects
func NewJSONProgressOutput(out io.Writer, newLines bool) Output {
	return &progressOutput{sf: &jsonProgressFormatter{}, out: out, newLines: newLines}
}

type formatProgress interface {
	// format formats progress information from a ProgressReader.
	format(prog Progress) []byte

	// emptyMessage returns the output of an empty message if the formatter
	// is configured to emit a trailing newline.
	emptyMessage() []byte
}

type progressOutput struct {
	sf  formatProgress
	out io.Writer

	newLines bool
	mu       sync.Mutex
}

// WriteProgress formats progress information from a ProgressReader.
func (out *progressOutput) WriteProgress(prog Progress) error {
	formatted := out.sf.format(prog)
	out.mu.Lock()
	defer out.mu.Unlock()
	_, err := out.out.Write(formatted)
	if err != nil {
		return err
	}

	if out.newLines && prog.LastUpdate {
		_, err = out.out.Write(out.sf.emptyMessage())
		return err
	}

	return nil
}
