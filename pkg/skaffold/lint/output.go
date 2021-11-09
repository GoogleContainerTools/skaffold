/*
Copyright 2021 The Skaffold Authors

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

package lint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"text/template"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/format"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

const (
	PlainTextOutput string = "plain-text"
	JSONOutput      string = "json"
)

func OutputFormatter(out io.Writer, opt string) format.Formatter {
	if opt == PlainTextOutput {
		return plainTextFormatter{out: out}
	}
	return format.JSONFormatter{Out: out}
}

type plainTextFormatter struct {
	out io.Writer
}
type plainTextOutput struct {
	RelFilePath    string
	LineNumber     int
	ColumnNumber   int
	RuleID         string
	Explanation    string
	RuleType       string
	FlaggedText    string
	ColPointerLine string
}

func genColPointerLine(colIdx int) string {
	s := ""
	for i := 1; i < colIdx; i++ {
		s += " "
	}
	s += "^"
	return s
}

func generatePlainTextOutput(res *Result) (string, error) {
	text, err := ioutil.ReadFile(res.AbsFilePath)
	if err != nil {
		return "", err
	}
	out := plainTextOutput{
		RelFilePath:    res.RelFilePath,
		LineNumber:     res.Line,
		ColumnNumber:   res.Column,
		RuleID:         res.Rule.RuleID.String(),
		Explanation:    res.Explanation,
		RuleType:       res.Rule.RuleType.String(),
		FlaggedText:    strings.Split(string(text), "\n")[res.Line-1],
		ColPointerLine: genColPointerLine(res.Column),
	}
	// TODO(aaron-prindle) - support different template for multiline matches -> (no ColPointerLine, show multi line match)
	// if flagged text contains \n character, don't show colpointerline
	tmpl, err := template.New("plainTextOutput").Parse("{{.RelFilePath}}:{{.LineNumber}}:{{.ColumnNumber}}: {{.RuleID}}: {{.RuleType}}: {{.Explanation}}\n{{.FlaggedText}}\n{{.ColPointerLine}}")
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	err = tmpl.Execute(&b, out)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func (p plainTextFormatter) Write(data interface{}) error {
	l := data.([]Result)
	for _, rec := range l {
		out, err := generatePlainTextOutput(&rec)
		if err != nil {
			return err
		}
		fmt.Fprintln(p.out, out)
	}
	return nil
}

type plainTextErrorOutput struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

// TODO(aaron-prindle) verify json errors make sense for plainText formatter
func (p plainTextFormatter) WriteErr(err error) error {
	var sErr sErrors.Error
	var plainTextErr plainTextErrorOutput
	if errors.As(err, &sErr) {
		plainTextErr = plainTextErrorOutput{ErrorCode: sErr.StatusCode().String(), ErrorMessage: sErr.Error()}
	} else {
		plainTextErr = plainTextErrorOutput{ErrorCode: proto.StatusCode_INSPECT_UNKNOWN_ERR.String(), ErrorMessage: err.Error()}
	}
	return json.NewEncoder(p.out).Encode(plainTextErr)
}
