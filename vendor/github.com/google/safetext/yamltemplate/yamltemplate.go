/*
 *
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package yamltemplate is a drop-in-replacement for using text/template to produce YAML, that adds automatic detection for YAML injection
package yamltemplate

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"text/template"
	"text/template/parse"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"github.com/google/safetext/common"
)

// ErrInvalidYAMLTemplate indicates the requested template is not valid YAML.
var ErrInvalidYAMLTemplate error = errors.New("Invalid YAML Template")

// ErrYAMLInjection indicates the inputs resulted in YAML injection.
var ErrYAMLInjection error = errors.New("YAML Injection Detected")

// ExecError is the custom error type returned when Execute has an
// error evaluating its template. (If a write error occurs, the actual
// error is returned; it will not be of type ExecError.)
type ExecError = template.ExecError

// FuncMap is the type of the map defining the mapping from names to functions.
// Each function must have either a single return value, or two return values of
// which the second has type error. In that case, if the second (error)
// return value evaluates to non-nil during execution, execution terminates and
// Execute returns that error.
//
// Errors returned by Execute wrap the underlying error; call errors.As to
// uncover them.
//
// When template execution invokes a function with an argument list, that list
// must be assignable to the function's parameter types. Functions meant to
// apply to arguments of arbitrary type can use parameters of type interface{} or
// of type reflect.Value. Similarly, functions meant to return a result of arbitrary
// type can return interface{} or reflect.Value.
type FuncMap = template.FuncMap

// Template is the representation of a parsed template. The *parse.Tree
// field is exported only for use by html/template and should be treated
// as unexported by all other clients.
type Template struct {
	unsafeTemplate *template.Template
}

// New allocates a new, undefined template with the given name.
func New(name string) *Template {
	return &Template{unsafeTemplate: template.New(name).Funcs(common.FuncMap)}
}

func mapOrArray(in interface{}) bool {
	return in != nil && (reflect.TypeOf(in).Kind() == reflect.Map || reflect.TypeOf(in).Kind() == reflect.Slice || reflect.TypeOf(in).Kind() == reflect.Array)
}

func allKeysMatch(base interface{}, a interface{}, b interface{}) bool {
	if base == nil {
		return a == nil && b == nil
	}

	switch reflect.TypeOf(base).Kind() {
	case reflect.Ptr:
		if reflect.TypeOf(a).Kind() != reflect.Ptr || reflect.TypeOf(b).Kind() != reflect.Ptr {
			return false
		}

		if !allKeysMatch(reflect.ValueOf(base).Elem().Interface(), reflect.ValueOf(a).Elem().Interface(), reflect.ValueOf(b).Elem().Interface()) {
			return true
		}
	case reflect.Map:
		if reflect.TypeOf(a).Kind() != reflect.Map || reflect.TypeOf(b).Kind() != reflect.Map {
			return false
		}

		if reflect.ValueOf(a).Len() != reflect.ValueOf(base).Len() || reflect.ValueOf(b).Len() != reflect.ValueOf(base).Len() {
			return false
		}

		basei := reflect.ValueOf(base).MapRange()
		for basei.Next() {
			av := reflect.ValueOf(a).MapIndex(basei.Key())
			bv := reflect.ValueOf(b).MapIndex(basei.Key())
			if !av.IsValid() || !bv.IsValid() ||
				!allKeysMatch(basei.Value().Interface(), av.Interface(), bv.Interface()) {
				return false
			}
		}
	case reflect.Slice, reflect.Array:
		if reflect.TypeOf(a).Kind() != reflect.Slice && reflect.TypeOf(a).Kind() != reflect.Array &&
			reflect.TypeOf(b).Kind() != reflect.Slice && reflect.TypeOf(b).Kind() != reflect.Array {
			return false
		}

		if reflect.ValueOf(a).Len() != reflect.ValueOf(base).Len() || reflect.ValueOf(b).Len() != reflect.ValueOf(base).Len() {
			return false
		}

		for i := 0; i < reflect.ValueOf(base).Len(); i++ {
			if !allKeysMatch(reflect.ValueOf(base).Index(i).Interface(), reflect.ValueOf(a).Index(i).Interface(), reflect.ValueOf(b).Index(i).Interface()) {
				return false
			}
		}
	case reflect.Struct:
		n := reflect.ValueOf(base).NumField()
		for i := 0; i < n; i++ {
			baseit := reflect.TypeOf(base).Field(i)
			ait := reflect.TypeOf(a).Field(i)
			bit := reflect.TypeOf(b).Field(i)

			if baseit.Name != ait.Name || baseit.Name != bit.Name {
				return false
			}

			// Only compare public members (private members cannot be overwritten by text/template)
			decodedName, _ := utf8.DecodeRuneInString(baseit.Name)
			if unicode.IsUpper(decodedName) {
				basei := reflect.ValueOf(base).Field(i)
				ai := reflect.ValueOf(a).Field(i)
				bi := reflect.ValueOf(b).Field(i)

				if !allKeysMatch(basei.Interface(), ai.Interface(), bi.Interface()) {
					return false
				}
			}
		}
	case reflect.String:
		// Baseline type of string was chosen arbitrarily,  so just check that there isn't a new a map or slice/array injected, which would change the structure of the YAML
		if mapOrArray(a) || mapOrArray(b) {
			return false
		}
	default:
		if reflect.TypeOf(a) != reflect.TypeOf(base) || reflect.TypeOf(b) != reflect.TypeOf(base) {
			return false
		}
	}

	return true
}

func unmarshalYaml(data []byte) ([]interface{}, error) {
	r := make([]interface{}, 0)

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var t interface{}
		err := decoder.Decode(&t)

		if err == io.EOF {
			break
		} else if err == nil {
			r = append(r, t)
		} else {
			return nil, err
		}
	}

	return r, nil
}

// Mutation algorithm
func mutateString(s string) string {
	// Longest possible output string is 2x the original
	out := make([]rune, len(s)*2)

	i := 0
	for _, r := range s {
		out[i] = r
		i++

		// Don't repeat quoting-related characters so as to not allow YAML context change in the mutation result
		if r != '\\' && r != '\'' && r != '"' {
			out[i] = r
			i++
		}
	}

	return string(out[:i])
}

// Execute applies a parsed template to the specified data object,
// and writes the output to wr.
// If an error occurs executing the template or writing its output,
// execution stops, but partial results may already have been written to
// the output writer.
// A template may be executed safely in parallel, although if parallel
// executions share a Writer the output may be interleaved.
//
// If data is a reflect.Value, the template applies to the concrete
// value that the reflect.Value holds, as in fmt.Print.
func (t *Template) Execute(wr io.Writer, data interface{}) (err error) {
	if data == nil {
		return common.ExecuteWithCallback(t.unsafeTemplate, common.EchoString, wr, data)
	}

	// An attacker may be able to cause type confusion or nil dereference panic during allKeysMatch
	defer func() {
		if r := recover(); r != nil {
			err = ErrYAMLInjection
		}
	}()

	// Calculate requested result first
	var requestedResult bytes.Buffer

	if err := common.ExecuteWithCallback(t.unsafeTemplate, common.EchoString, &requestedResult, data); err != nil {
		return err
	}

	walked, err := t.unsafeTemplate.Clone()
	if err != nil {
		return err
	}
	walked.Tree = walked.Tree.Copy()

	common.WalkApplyFuncToNonDeclaractiveActions(walked, walked.Tree.Root)

	// Get baseline
	var baselineResult bytes.Buffer
	if err = common.ExecuteWithCallback(walked, common.BaselineString, &baselineResult, data); err != nil {
		return err
	}

	parsedBaselineResult, err := unmarshalYaml(baselineResult.Bytes())
	if err != nil {
		return ErrInvalidYAMLTemplate
	}

	// If baseline was valid, request must also be valid YAML for no injection to have occurred
	parsedRequestedResult, err := unmarshalYaml(requestedResult.Bytes())
	if err != nil {
		return ErrYAMLInjection
	}

	// Mutate the input
	var mutatedResult bytes.Buffer
	if err = common.ExecuteWithCallback(walked, mutateString, &mutatedResult, data); err != nil {
		return err
	}

	parsedMutatedResult, err := unmarshalYaml(mutatedResult.Bytes())
	if err != nil {
		return ErrYAMLInjection
	}

	// Compare results
	if !allKeysMatch(parsedBaselineResult, parsedRequestedResult, parsedMutatedResult) {
		return ErrYAMLInjection
	}

	requestedResult.WriteTo(wr)
	return nil
}

// Name returns the name of the template.
func (t *Template) Name() string {
	return t.unsafeTemplate.Name()
}

// New allocates a new, undefined template associated with the given one and with the same
// delimiters. The association, which is transitive, allows one template to
// invoke another with a {{template}} action.
//
// Because associated templates share underlying data, template construction
// cannot be done safely in parallel. Once the templates are constructed, they
// can be executed in parallel.
func (t *Template) New(name string) *Template {
	return &Template{unsafeTemplate: t.unsafeTemplate.New(name).Funcs(common.FuncMap)}
}

// Clone returns a duplicate of the template, including all associated
// templates. The actual representation is not copied, but the name space of
// associated templates is, so further calls to Parse in the copy will add
// templates to the copy but not to the original. Clone can be used to prepare
// common templates and use them with variant definitions for other templates
// by adding the variants after the clone is made.
func (t *Template) Clone() (*Template, error) {
	nt, err := t.unsafeTemplate.Clone()
	return &Template{unsafeTemplate: nt}, err
}

// AddParseTree associates the argument parse tree with the template t, giving
// it the specified name. If the template has not been defined, this tree becomes
// its definition. If it has been defined and already has that name, the existing
// definition is replaced; otherwise a new template is created, defined, and returned.
func (t *Template) AddParseTree(name string, tree *parse.Tree) (*Template, error) {
	nt, err := t.unsafeTemplate.AddParseTree(name, tree)

	if nt != t.unsafeTemplate {
		return &Template{unsafeTemplate: nt}, err
	}
	return t, err
}

// Option sets options for the template. Options are described by
// strings, either a simple string or "key=value". There can be at
// most one equals sign in an option string. If the option string
// is unrecognized or otherwise invalid, Option panics.
//
// Known options:
//
// missingkey: Control the behavior during execution if a map is
// indexed with a key that is not present in the map.
//
//	"missingkey=default" or "missingkey=invalid"
//		The default behavior: Do nothing and continue execution.
//		If printed, the result of the index operation is the string
//		"<no value>".
//	"missingkey=zero"
//		The operation returns the zero value for the map type's element.
//	"missingkey=error"
//		Execution stops immediately with an error.
func (t *Template) Option(opt ...string) *Template {
	for _, s := range opt {
		t.unsafeTemplate.Option(s)
	}
	return t
}

// Templates returns a slice of defined templates associated with t.
func (t *Template) Templates() []*Template {
	s := t.unsafeTemplate.Templates()

	var ns []*Template
	for _, nt := range s {
		ns = append(ns, &Template{unsafeTemplate: nt})
	}

	return ns
}

// ExecuteTemplate applies the template associated with t that has the given name
// to the specified data object and writes the output to wr.
// If an error occurs executing the template or writing its output,
// execution stops, but partial results may already have been written to
// the output writer.
// A template may be executed safely in parallel, although if parallel
// executions share a Writer the output may be interleaved.
func (t *Template) ExecuteTemplate(wr io.Writer, name string, data interface{}) error {
	tmpl := t.Lookup(name)
	if tmpl == nil {
		return fmt.Errorf("template: no template %q associated with template %q", name, t.Name())
	}
	return tmpl.Execute(wr, data)
}

// Delims sets the action delimiters to the specified strings, to be used in
// subsequent calls to Parse, ParseFiles, or ParseGlob. Nested template
// definitions will inherit the settings. An empty delimiter stands for the
// corresponding default: {{ or }}.
// The return value is the template, so calls can be chained.
func (t *Template) Delims(left, right string) *Template {
	t.unsafeTemplate.Delims(left, right)
	return t
}

// DefinedTemplates returns a string listing the defined templates,
// prefixed by the string "; defined templates are: ". If there are none,
// it returns the empty string. For generating an error message here
// and in html/template.
func (t *Template) DefinedTemplates() string {
	return t.unsafeTemplate.DefinedTemplates()
}

// Funcs adds the elements of the argument map to the template's function map.
// It must be called before the template is parsed.
// It panics if a value in the map is not a function with appropriate return
// type or if the name cannot be used syntactically as a function in a template.
// It is legal to overwrite elements of the map. The return value is the template,
// so calls can be chained.
func (t *Template) Funcs(funcMap FuncMap) *Template {
	t.unsafeTemplate.Funcs(funcMap)
	return t
}

// Lookup returns the template with the given name that is associated with t.
// It returns nil if there is no such template or the template has no definition.
func (t *Template) Lookup(name string) *Template {
	nt := t.unsafeTemplate.Lookup(name)

	if nt == nil {
		return nil
	}

	if nt != t.unsafeTemplate {
		return &Template{unsafeTemplate: nt}
	}

	return t
}

// Parse parses text as a template body for t.
// Named template definitions ({{define ...}} or {{block ...}} statements) in text
// define additional templates associated with t and are removed from the
// definition of t itself.
//
// Templates can be redefined in successive calls to Parse.
// A template definition with a body containing only white space and comments
// is considered empty and will not replace an existing template's body.
// This allows using Parse to add new named template definitions without
// overwriting the main template body.
func (t *Template) Parse(text string) (*Template, error) {
	nt, err := t.unsafeTemplate.Parse(text)

	if nt != t.unsafeTemplate {
		return &Template{unsafeTemplate: nt}, err
	}

	return t, err
}

// Must is a helper that wraps a call to a function returning (*Template, error)
// and panics if the error is non-nil. It is intended for use in variable
// initializations such as
//
//	var t = template.Must(template.New("name").Parse("text"))
func Must(t *Template, err error) *Template {
	if err != nil {
		panic(err)
	}
	return t
}

func readFileOS(file string) (name string, b []byte, err error) {
	name = filepath.Base(file)
	b, err = os.ReadFile(file)
	return
}

func readFileFS(fsys fs.FS) func(string) (string, []byte, error) {
	return func(file string) (name string, b []byte, err error) {
		name = path.Base(file)
		b, err = fs.ReadFile(fsys, file)
		return
	}
}

func parseFiles(t *Template, readFile func(string) (string, []byte, error), filenames ...string) (*Template, error) {
	if len(filenames) == 0 {
		// Not really a problem, but be consistent.
		return nil, fmt.Errorf("template: no files named in call to ParseFiles")
	}
	for _, filename := range filenames {
		name, b, err := readFile(filename)
		if err != nil {
			return nil, err
		}
		s := string(b)
		// First template becomes return value if not already defined,
		// and we use that one for subsequent New calls to associate
		// all the templates together. Also, if this file has the same name
		// as t, this file becomes the contents of t, so
		//  t, err := New(name).Funcs(xxx).ParseFiles(name)
		// works. Otherwise we create a new template associated with t.
		var tmpl *Template
		if t == nil {
			t = New(name)
		}
		if name == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(name)
		}
		_, err = tmpl.Parse(s)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

// parseGlob is the implementation of the function and method ParseGlob.
func parseGlob(t *Template, pattern string) (*Template, error) {
	filenames, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(filenames) == 0 {
		return nil, fmt.Errorf("template: pattern matches no files: %#q", pattern)
	}
	return parseFiles(t, readFileOS, filenames...)
}

func parseFS(t *Template, fsys fs.FS, patterns []string) (*Template, error) {
	var filenames []string
	for _, pattern := range patterns {
		list, err := fs.Glob(fsys, pattern)
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			return nil, fmt.Errorf("template: pattern matches no files: %#q", pattern)
		}
		filenames = append(filenames, list...)
	}
	return parseFiles(t, readFileFS(fsys), filenames...)
}

// ParseFiles creates a new Template and parses the template definitions from
// the named files. The returned template's name will have the base name and
// parsed contents of the first file. There must be at least one file.
// If an error occurs, parsing stops and the returned *Template is nil.
//
// When parsing multiple files with the same name in different directories,
// the last one mentioned will be the one that results.
// For instance, ParseFiles("a/foo", "b/foo") stores "b/foo" as the template
// named "foo", while "a/foo" is unavailable.
func ParseFiles(filenames ...string) (*Template, error) {
	return parseFiles(nil, readFileOS, filenames...)
}

// ParseFiles parses the named files and associates the resulting templates with
// t. If an error occurs, parsing stops and the returned template is nil;
// otherwise it is t. There must be at least one file.
// Since the templates created by ParseFiles are named by the base
// names of the argument files, t should usually have the name of one
// of the (base) names of the files. If it does not, depending on t's
// contents before calling ParseFiles, t.Execute may fail. In that
// case use t.ExecuteTemplate to execute a valid template.
//
// When parsing multiple files with the same name in different directories,
// the last one mentioned will be the one that results.
func (t *Template) ParseFiles(filenames ...string) (*Template, error) {
	// Ensure template is inited
	t.Option()

	return parseFiles(t, readFileOS, filenames...)
}

// ParseGlob creates a new Template and parses the template definitions from
// the files identified by the pattern. The files are matched according to the
// semantics of filepath.Match, and the pattern must match at least one file.
// The returned template will have the (base) name and (parsed) contents of the
// first file matched by the pattern. ParseGlob is equivalent to calling
// ParseFiles with the list of files matched by the pattern.
//
// When parsing multiple files with the same name in different directories,
// the last one mentioned will be the one that results.
func ParseGlob(pattern string) (*Template, error) {
	return parseGlob(nil, pattern)
}

// ParseGlob parses the template definitions in the files identified by the
// pattern and associates the resulting templates with t. The files are matched
// according to the semantics of filepath.Match, and the pattern must match at
// least one file. ParseGlob is equivalent to calling t.ParseFiles with the
// list of files matched by the pattern.
//
// When parsing multiple files with the same name in different directories,
// the last one mentioned will be the one that results.
func (t *Template) ParseGlob(pattern string) (*Template, error) {
	// Ensure template is inited
	t.Option()

	return parseGlob(t, pattern)
}

// ParseFS is like ParseFiles or ParseGlob but reads from the file system fsys
// instead of the host operating system's file system.
// It accepts a list of glob patterns.
// (Note that most file names serve as glob patterns matching only themselves.)
func ParseFS(fsys fs.FS, patterns ...string) (*Template, error) {
	return parseFS(nil, fsys, patterns)
}

// ParseFS is like ParseFiles or ParseGlob but reads from the file system fsys
// instead of the host operating system's file system.
// It accepts a list of glob patterns.
// (Note that most file names serve as glob patterns matching only themselves.)
func (t *Template) ParseFS(fsys fs.FS, patterns ...string) (*Template, error) {
	// Ensure template is inited
	t.Option()

	return parseFS(t, fsys, patterns)
}

// HTMLEscape writes to w the escaped HTML equivalent of the plain text data b.
func HTMLEscape(w io.Writer, b []byte) {
	template.HTMLEscape(w, b)
}

// HTMLEscapeString returns the escaped HTML equivalent of the plain text data s.
func HTMLEscapeString(s string) string {
	return template.HTMLEscapeString(s)
}

// HTMLEscaper returns the escaped HTML equivalent of the textual
// representation of its arguments.
func HTMLEscaper(args ...interface{}) string {
	return template.HTMLEscaper(args)
}

// IsTrue reports whether the value is 'true', in the sense of not the zero of its type,
// and whether the value has a meaningful truth value. This is the definition of
// truth used by if and other such actions.
func IsTrue(val interface{}) (truth, ok bool) {
	return template.IsTrue(val)
}

// JSEscape writes to w the escaped JavaScript equivalent of the plain text data b.
func JSEscape(w io.Writer, b []byte) {
	template.JSEscape(w, b)
}

// JSEscapeString returns the escaped JavaScript equivalent of the plain text data s.
func JSEscapeString(s string) string {
	return template.JSEscapeString(s)
}

// JSEscaper returns the escaped JavaScript equivalent of the textual
// representation of its arguments.
func JSEscaper(args ...interface{}) string {
	return template.JSEscaper(args)
}

// URLQueryEscaper returns the escaped value of the textual representation of
// its arguments in a form suitable for embedding in a URL query.
func URLQueryEscaper(args ...interface{}) string {
	return template.URLQueryEscaper(args)
}
