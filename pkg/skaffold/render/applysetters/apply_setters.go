/*
Copyright 2023 The Skaffold Authors

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

package applysetters

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
)

const SetterCommentIdentifier = "# from-param: "

var _ kio.Filter = &ApplySetters{}

// ApplySetters applies the setter values to the resource fields which are tagged
// by the setter reference comments
type ApplySetters struct {
	// Setters holds the user provided values for all the setters
	Setters []Setter

	// Results are the results of applying setter values
	Results []*Result

	// filePath file path of resource
	filePath string
}

type Setter struct {
	// Name is the name of the setter
	Name string

	// Value is the input value for setter
	Value string
}

// Result holds result of search and replace operation
type Result struct {
	// FilePath is the file path of the matching field
	FilePath string

	// FieldPath is field path of the matching field
	FieldPath string

	// Value of the matching field
	Value string
}

// Filter implements Set as a yaml.Filter
func (as *ApplySetters) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	for i := range nodes {
		filePath, _, err := kioutil.GetFileAnnotations(nodes[i])
		if err != nil {
			return nodes, err
		}
		as.filePath = filePath
		err = accept(as, nodes[i])
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}
	return nodes, nil
}

// Apply Transform manifestList with Filter
func (as *ApplySetters) Apply(ctx context.Context, ml manifest.ManifestList) (manifest.ManifestList, error) {
	reader := ml.Reader()
	byteReader := kio.ByteReader{Reader: reader, OmitReaderAnnotations: true}
	nodes, err := byteReader.Read()
	var updated manifest.ManifestList
	if err != nil {
		return updated, err
	}
	nodes, err = as.Filter(nodes)
	if err != nil {
		return updated, err
	}
	for i := range nodes {
		updated.Append([]byte(nodes[i].MustString()))
	}
	return updated, nil
}

// ApplyPath Transform a yaml file with Filter
func (as *ApplySetters) ApplyPath(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	nodes, err := kio.FromBytes(content)
	if err != nil {
		return err
	}
	nodes, err = as.Filter(nodes)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	err = (&kio.ByteWriter{Writer: &b}).Write(nodes)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b.Bytes(), 0600)
}

/*
visitMapping takes input mapping node, and performs following steps
checks if the key node of the input mapping node has line comment with SetterCommentIdentifier
checks if the value node is of sequence node type
if yes to both, resolves the setter value for the setter name in the line comment
replaces the existing sequence node with the new values provided by user

e.g. for input of Mapping node

environments: # from-param: ${env}
- dev
- stage

For input ApplySetters [name: env, value: "[stage, prod]"], qthe yaml node is transformed to

environments: # from-param: ${env}
- stage
- prod
*/
func (as *ApplySetters) visitMapping(object *yaml.RNode, path string) error {
	return object.VisitFields(func(node *yaml.MapNode) error {
		if node == nil || node.Key.IsNil() || node.Value.IsNil() {
			// don't do IsNilOrEmpty check as empty sequences are allowed
			return nil
		}

		// the aim of this method is to apply-setter for sequence nodes
		if node.Value.YNode().Kind != yaml.SequenceNode {
			// return if it is not a sequence node
			return nil
		}

		lineComment := node.Key.YNode().LineComment
		if node.Value.YNode().Style == yaml.FlowStyle {
			// if node is FlowStyle e.g. env: [foo, bar] # from-param: ${env}
			// the setter comment will be on value node
			lineComment = node.Value.YNode().LineComment
		}

		setterPattern := extractSetterPattern(lineComment)
		if setterPattern == "" {
			// the node is not tagged with setter pattern
			return nil
		}

		if !shouldSet(setterPattern, as.Setters) {
			// this means there is no intent from user to modify this setter tagged resources
			return nil
		}

		// since this setter pattern is found on sequence node, make sure that it is
		// not interpolation of setters, it should be simple setter e.g. ${environments}
		if !validArraySetterPattern(setterPattern) {
			return errors.Errorf("invalid setter pattern for array node: %q", setterPattern)
		}

		// get the setter value for the setter name in the comment
		sv := setterValue(as.Setters, setterPattern)

		// add the key to the field path
		fieldPath := strings.TrimPrefix(fmt.Sprintf("%s.%s", path, node.Key.YNode().Value), ".")

		if sv == "" {
			node.Value.YNode().Content = []*yaml.Node{}
			// empty sequence must be FlowStyle e.g. env: [] # from-param: ${env}
			node.Value.YNode().Style = yaml.FlowStyle
			// setter pattern comment must be on value node
			node.Value.YNode().LineComment = lineComment
			node.Key.YNode().LineComment = ""
			as.Results = append(as.Results, &Result{
				FilePath:  as.filePath,
				FieldPath: fieldPath,
				Value:     sv,
			})
			return nil
		}

		// parse the setter value as yaml node
		rn, err := yaml.Parse(sv)
		if err != nil {
			return errors.Errorf("input to array setter must be an array of values, but found %q", sv)
		}

		// the setter value must parse as sequence node
		if rn.YNode().Kind != yaml.SequenceNode {
			return errors.Errorf("input to array setter must be an array of values, but found %q", sv)
		}

		node.Value.YNode().Content = rn.YNode().Content
		node.Key.YNode().LineComment = lineComment
		// non-empty sequences should be standardized to FoldedStyle
		// env: # from-param: ${env}
		//  - foo
		//  - bar
		node.Value.YNode().Style = yaml.FoldedStyle

		as.Results = append(as.Results, &Result{
			FilePath:  as.filePath,
			FieldPath: fieldPath,
			Value:     sv,
		})
		return nil
	})
}

/*
visitScalar accepts the input scalar node and performs following steps,
checks if the line comment of input scalar node has prefix SetterCommentIdentifier
resolves the setter values for the setter name in the comment
replaces the existing value of the scalar node with the new value

e.g.for input of scalar node 'nginx:1.7.1 # from-param: ${image}:${tag}' in the yaml node

apiVersion: v1
...

	image: nginx:1.7.1 # from-param: ${image}:${tag}

and for input ApplySetters [[name: image, value: ubuntu], [name: tag, value: 1.8.0]]
The yaml node is transformed to

apiVersion: v1
...

	image: ubuntu:1.8.0 # from-param: ${image}:${tag}
*/
func (as *ApplySetters) visitScalar(object *yaml.RNode, path string) error {
	if object.IsNil() {
		return nil
	}

	if object.YNode().Kind != yaml.ScalarNode {
		// return if it is not a scalar node
		return nil
	}

	// perform a direct set of the field if it matches
	setterPattern := extractSetterPattern(object.YNode().LineComment)
	if setterPattern == "" {
		// the node is not tagged with setter pattern
		return nil
	}

	curPattern := setterPattern
	if !shouldSet(setterPattern, as.Setters) {
		// this means there is no intent from user to modify this setter tagged resources
		return nil
	}

	// replace the setter names in comment pattern with provided values
	for _, setter := range as.Setters {
		setterPattern = strings.ReplaceAll(
			setterPattern,
			fmt.Sprintf("${%s}", setter.Name),
			fmt.Sprintf("%v", setter.Value),
		)
	}

	// replace the remaining setter names in comment pattern with values derived from current
	// field value, these values are not provided by user
	currentSetterValues := currentSetterValues(curPattern, object.YNode().Value)
	for setterName, setterValue := range currentSetterValues {
		setterPattern = strings.ReplaceAll(
			setterPattern,
			fmt.Sprintf("${%s}", setterName),
			fmt.Sprintf("%v", setterValue),
		)
	}

	// check if there are unresolved setters and throw error
	urs := unresolvedSetters(setterPattern)
	if len(urs) > 0 {
		return errors.Errorf("values for setters %v must be provided", urs)
	}

	object.YNode().Value = setterPattern
	if setterPattern == "" {
		object.YNode().Style = yaml.DoubleQuotedStyle
	}
	object.YNode().Tag = yaml.NodeTagEmpty
	as.Results = append(as.Results, &Result{
		FilePath:  as.filePath,
		FieldPath: strings.TrimPrefix(path, "."),
		Value:     object.YNode().Value,
	})
	return nil
}

// shouldSet takes the setter pattern comment and setter values map and returns true
// iff at least one of the setter names in the pattern match with the setter names
// in input setterValues map
func shouldSet(pattern string, setters []Setter) bool {
	for _, s := range setters {
		if strings.Contains(pattern, fmt.Sprintf("${%s}", s.Name)) {
			return true
		}
	}
	return false
}

// currentSetterValues takes pattern and value and returns setter names to values
// derived using pattern matching
// e.g. pattern = my-app-layer.${stage}.${domain}.${tld}, value = my-app-layer.dev.example.com
// returns {"stage":"dev", "domain":"example", "tld":"com"}
func currentSetterValues(pattern, value string) map[string]string {
	res := make(map[string]string)
	// get all setter names enclosed in ${}
	// e.g. value: my-app-layer.dev.example.com
	// pattern: my-app-layer.${stage}.${domain}.${tld}
	// urs: [${stage}, ${domain}, ${tld}]
	urs := unresolvedSetters(pattern)
	// and escape pattern
	pattern = regexp.QuoteMeta(pattern)
	// escaped pattern: my-app-layer\.\$\{stage\}\.\$\{domain\}\.\$\{tld\}

	for _, setterName := range urs {
		// escape setter name
		// we need to escape the setterName as well to replace it in the escaped pattern string later
		setterName = regexp.QuoteMeta(setterName)
		pattern = strings.ReplaceAll(
			pattern,
			setterName,
			`(?P<x>.*)`) // x is just a place holder, it could be any alphanumeric string
	}
	// pattern: my-app-layer\.(?P<x>.*)\.(?P<x>.*)\.(?P<x>.*)
	r, err := regexp.Compile(pattern)
	if err != nil {
		// just return empty map if values can't be derived from pattern
		return res
	}
	setterValues := r.FindStringSubmatch(value)
	if len(setterValues) == 0 {
		return res
	}
	// setterValues: [ "my-app-layer.dev.example.com", "dev", "example", "com"]
	setterValues = setterValues[1:]
	// setterValues: [ "dev", "example", "com"]
	if len(urs) != len(setterValues) {
		// just return empty map if values can't be derived
		return res
	}
	for i := range setterValues {
		if setterValues[i] == "" {
			// if any of the value is unresolved return empty map
			// and expect users to provide all values
			return make(map[string]string)
		}
		res[clean(urs[i])] = setterValues[i]
	}
	return res
}

// setterValue returns the value for the setter
func setterValue(setters []Setter, setterName string) string {
	for _, setter := range setters {
		if setter.Name == clean(setterName) {
			return setter.Value
		}
	}
	return ""
}

// extractSetterPattern extracts the setter pattern from the line comment of the
// yaml RNode. If the the line comment doesn't contain SetterCommentIdentifier
// prefix, then it returns empty string
func extractSetterPattern(lineComment string) string {
	if !strings.HasPrefix(lineComment, SetterCommentIdentifier) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(lineComment, SetterCommentIdentifier))
}

// validArraySetterPattern returns true if the array setter pattern is valid
// pattern must not interpolation of setters, it should be simple setter e.g. ${environments}
func validArraySetterPattern(pattern string) bool {
	return len(unresolvedSetters(pattern)) == 1 &&
		strings.HasPrefix(pattern, "${") &&
		strings.HasSuffix(pattern, "}")
}

// unresolvedSetters returns the list of values enclosed in ${} present within given
// pattern e.g. pattern = foo-${image}:${tag}-bar return ["${image}", "${tag}"]
func unresolvedSetters(pattern string) []string {
	re := regexp.MustCompile(`\$\{([^}]*)\}`)
	return re.FindAllString(pattern, -1)
}

// clean extracts value enclosed in ${}
func clean(input string) string {
	input = strings.TrimSpace(input)
	return strings.TrimSuffix(strings.TrimPrefix(input, "${"), "}")
}

// Decode decodes the input yaml node into Set struct
func Decode(rn *yaml.RNode, fcd *ApplySetters) {
	for k, v := range rn.GetDataMap() {
		fcd.Setters = append(fcd.Setters, Setter{Name: k, Value: v})
	}
}
