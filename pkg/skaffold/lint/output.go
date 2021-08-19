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
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

type Formatter interface {
	Write(interface{}) error
	WriteErr(error) error
}

func OutputFormatter(out io.Writer, opt string) Formatter {
	switch opt {
	case "plain-text":
		return recommendFormatter{out: out}
	case "num-only":
		return numOnlyLintFormatter{out: out}
	}
	return jsonFormatter{out: out}
}

type jsonFormatter struct {
	out io.Writer
}

func (j jsonFormatter) Write(data interface{}) error {
	return json.NewEncoder(j.out).Encode(data)
}

type jsonErrorOutput struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

func (j jsonFormatter) WriteErr(err error) error {
	var sErr sErrors.Error
	var jsonErr jsonErrorOutput
	if errors.As(err, &sErr) {
		jsonErr = jsonErrorOutput{ErrorCode: sErr.StatusCode().String(), ErrorMessage: sErr.Error()}
	} else {
		jsonErr = jsonErrorOutput{ErrorCode: proto.StatusCode_INSPECT_UNKNOWN_ERR.String(), ErrorMessage: err.Error()}
	}
	return json.NewEncoder(j.out).Encode(jsonErr)
}

type recommendFormatter struct {
	out io.Writer
}

func convert1DFileIndexTo2D(input string, idx int) (int, int) {
	line := 1
	col := 0
	for i := 0; i < idx; i++ {
		col++
		if input[i] == '\n' {
			line++
			col = 0
		}
	}
	return line, col
}

type golangCIStyleOutput struct {
	RelFilePath    string
	LineNumber     int
	ColumnNumber   int
	LintRuleId     string
	Explanation    string
	LintRuleType   string
	FlaggedText    string
	ColPointerLine string
}

func genColPointerLine(colIdx int) string {
	s := ""
	for i := 0; i < colIdx; i++ {
		s += " "
	}
	s += "^"
	return s
}

func generateGolangCIStyleOutput(mr *MatchResult) string {
	// TODO(aaron-prindle) fix hack, re-reading file to not store text in match result
	text, err := ioutil.ReadFile(mr.AbsFilePath)
	if err != nil {
		return ""
	}

	// TODO(aaron-prindle) avoid multiple string(text) castes?

	var line, col int
	if mr.Line != nil {
		// use line and column numbers already set, use those
		line = *mr.Line
		col = *mr.Column - 1 // TODO(aaron-prindle) index seems to be one off what I would expect?
		// col = *mr.Column // TODO(aaron-prindle) currently this is only used for json field values and these are maybe off atm?
		// TODO(aaron-prindle) stub, FIXX!!
	} else {
		line, col = convert1DFileIndexTo2D(string(text), mr.TextStartIndex)
	}
	flaggedText := strings.Split(string(text), "\n")[line-1]

	gcisout := golangCIStyleOutput{
		RelFilePath:    mr.RelFilePath,
		LineNumber:     line,
		ColumnNumber:   col,
		LintRuleId:     mr.LintRuleId.String(),
		Explanation:    mr.Explanation,
		LintRuleType:   mr.LintRuleType.String(),
		FlaggedText:    flaggedText, // TODO(aaron-prindle) should this be the whole line?
		ColPointerLine: genColPointerLine(col),
	}
	// TODO(aaron-prindle) - different template for multiline matches -> (no ColPointerLine, show multi line match)
	// if flagged text contains \n character, don't show colpointerline

	// TODO(aaron-prindle) - different template based on LintRuleType
	tmpl, err := template.New("golangCIStyleOutput").Parse("{{.RelFilePath}}:{{.LineNumber}}:{{.ColumnNumber}}: {{.LintRuleId}}: {{.Explanation}}: ({{.LintRuleType}})\n{{.FlaggedText}}\n{{.ColPointerLine}}")
	if err != nil {
		panic(err)
	}
	var b bytes.Buffer
	err = tmpl.Execute(&b, gcisout)
	if err != nil {
		panic(err)
	}

	return b.String()
}

func (j recommendFormatter) Write(data interface{}) error {
	l := data.(LintRuleList)
	for _, rec := range l.LinterResultList {
		fmt.Fprintln(j.out, generateGolangCIStyleOutput(&rec))
	}
	return nil
}

// type recommendErrorOutput struct {
// 	ErrorCode    string `recommend:"errorCode"`
// 	ErrorMessage string `recommend:"errorMessage"`
// }

func (j recommendFormatter) WriteErr(err error) error {
	// var sErr sErrors.Error
	// var recommendErr recommendErrorOutput
	// if errors.As(err, &sErr) {
	// 	recommendErr = recommendErrorOutput{ErrorCode: sErr.StatusCode().String(), ErrorMessage: sErr.Error()}
	// } else {
	// 	recommendErr = recommendErrorOutput{ErrorCode: proto.StatusCode_INSPECT_UNKNOWN_ERR.String(), ErrorMessage: err.Error()}
	// }
	// return json.NewEncoder(j.out).Encode(recommendErr)

	// TODO(aaron-prindle) Correctly plumb WriteErr method
	return fmt.Errorf("WriteErr not fully implemented for plain-text formatter, this is a stub message")
}

// =======================

type numOnlyLintFormatter struct {
	out io.Writer
}

func generateNumOnlyLintOutput(mrs *[]MatchResult) string {
	return fmt.Sprintf("%d configuration recommendations found for your application.  Run 'skaffold recommend' for the detailed list of these recommendations", len(*mrs))
}

// TODO(aaron-prindle) fix this to have a context timeout and to NOT use DockerfileLintRules (use LintRules instead)
func (j numOnlyLintFormatter) Write(data interface{}) error {
	l := data.(AllLintRuleList)
	df := l.DockerfileLintRuleList
	fmt.Fprintln(j.out, generateNumOnlyLintOutput(&df.DockerfileLintRules))
	return nil
}

// type numOnlyLintErrorOutput struct {
// 	ErrorCode    string `recommend:"errorCode"`
// 	ErrorMessage string `recommend:"errorMessage"`
// }

func (j numOnlyLintFormatter) WriteErr(err error) error {
	return fmt.Errorf("WriteErr not fully implemented for plain-text formatter, this is a stub message")
}
