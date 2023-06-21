// Package applysetters /*
// https://github.com/GoogleContainerTools/kpt-functions-catalog/tree/master/functions/go/apply-setters
/*

                                 Apache License
                           Version 2.0, January 2004
                        http://www.apache.org/licenses/

   TERMS AND CONDITIONS FOR USE, REPRODUCTION, AND DISTRIBUTION

   1. Definitions.

      "License" shall mean the terms and conditions for use, reproduction,
      and distribution as defined by Sections 1 through 9 of this document.

      "Licensor" shall mean the copyright owner or entity authorized by
      the copyright owner that is granting the License.

      "Legal Entity" shall mean the union of the acting entity and all
      other entities that control, are controlled by, or are under common
      control with that entity. For the purposes of this definition,
      "control" means (i) the power, direct or indirect, to cause the
      direction or management of such entity, whether by contract or
      otherwise, or (ii) ownership of fifty percent (50%) or more of the
      outstanding shares, or (iii) beneficial ownership of such entity.

      "You" (or "Your") shall mean an individual or Legal Entity
      exercising permissions granted by this License.

      "Source" form shall mean the preferred form for making modifications,
      including but not limited to software source code, documentation
      source, and configuration files.

      "Object" form shall mean any form resulting from mechanical
      transformation or translation of a Source form, including but
      not limited to compiled object code, generated documentation,
      and conversions to other media types.

      "Work" shall mean the work of authorship, whether in Source or
      Object form, made available under the License, as indicated by a
      copyright notice that is included in or attached to the work
      (an example is provided in the Appendix below).

      "Derivative Works" shall mean any work, whether in Source or Object
      form, that is based on (or derived from) the Work and for which the
      editorial revisions, annotations, elaborations, or other modifications
      represent, as a whole, an original work of authorship. For the purposes
      of this License, Derivative Works shall not include works that remain
      separable from, or merely link (or bind by name) to the interfaces of,
      the Work and Derivative Works thereof.

      "Contribution" shall mean any work of authorship, including
      the original version of the Work and any modifications or additions
      to that Work or Derivative Works thereof, that is intentionally
      submitted to Licensor for inclusion in the Work by the copyright owner
      or by an individual or Legal Entity authorized to submit on behalf of
      the copyright owner. For the purposes of this definition, "submitted"
      means any form of electronic, verbal, or written communication sent
      to the Licensor or its representatives, including but not limited to
      communication on electronic mailing lists, source code control systems,
      and issue tracking systems that are managed by, or on behalf of, the
      Licensor for the purpose of discussing and improving the Work, but
      excluding communication that is conspicuously marked or otherwise
      designated in writing by the copyright owner as "Not a Contribution."

      "Contributor" shall mean Licensor and any individual or Legal Entity
      on behalf of whom a Contribution has been received by Licensor and
      subsequently incorporated within the Work.

   2. Grant of Copyright License. Subject to the terms and conditions of
      this License, each Contributor hereby grants to You a perpetual,
      worldwide, non-exclusive, no-charge, royalty-free, irrevocable
      copyright license to reproduce, prepare Derivative Works of,
      publicly display, publicly perform, sublicense, and distribute the
      Work and such Derivative Works in Source or Object form.

   3. Grant of Patent License. Subject to the terms and conditions of
      this License, each Contributor hereby grants to You a perpetual,
      worldwide, non-exclusive, no-charge, royalty-free, irrevocable
      (except as stated in this section) patent license to make, have made,
      use, offer to sell, sell, import, and otherwise transfer the Work,
      where such license applies only to those patent claims licensable
      by such Contributor that are necessarily infringed by their
      Contribution(s) alone or by combination of their Contribution(s)
      with the Work to which such Contribution(s) was submitted. If You
      institute patent litigation against any entity (including a
      cross-claim or counterclaim in a lawsuit) alleging that the Work
      or a Contribution incorporated within the Work constitutes direct
      or contributory patent infringement, then any patent licenses
      granted to You under this License for that Work shall terminate
      as of the date such litigation is filed.

   4. Redistribution. You may reproduce and distribute copies of the
      Work or Derivative Works thereof in any medium, with or without
      modifications, and in Source or Object form, provided that You
      meet the following conditions:

      (a) You must give any other recipients of the Work or
          Derivative Works a copy of this License; and

      (b) You must cause any modified files to carry prominent notices
          stating that You changed the files; and

      (c) You must retain, in the Source form of any Derivative Works
          that You distribute, all copyright, patent, trademark, and
          attribution notices from the Source form of the Work,
          excluding those notices that do not pertain to any part of
          the Derivative Works; and

      (d) If the Work includes a "NOTICE" text file as part of its
          distribution, then any Derivative Works that You distribute must
          include a readable copy of the attribution notices contained
          within such NOTICE file, excluding those notices that do not
          pertain to any part of the Derivative Works, in at least one
          of the following places: within a NOTICE text file distributed
          as part of the Derivative Works; within the Source form or
          documentation, if provided along with the Derivative Works; or,
          within a display generated by the Derivative Works, if and
          wherever such third-party notices normally appear. The contents
          of the NOTICE file are for informational purposes only and
          do not modify the License. You may add Your own attribution
          notices within Derivative Works that You distribute, alongside
          or as an addendum to the NOTICE text from the Work, provided
          that such additional attribution notices cannot be construed
          as modifying the License.

      You may add Your own copyright statement to Your modifications and
      may provide additional or different license terms and conditions
      for use, reproduction, or distribution of Your modifications, or
      for any such Derivative Works as a whole, provided Your use,
      reproduction, and distribution of the Work otherwise complies with
      the conditions stated in this License.

   5. Submission of Contributions. Unless You explicitly state otherwise,
      any Contribution intentionally submitted for inclusion in the Work
      by You to the Licensor shall be under the terms and conditions of
      this License, without any additional terms or conditions.
      Notwithstanding the above, nothing herein shall supersede or modify
      the terms of any separate license agreement you may have executed
      with Licensor regarding such Contributions.

   6. Trademarks. This License does not grant permission to use the trade
      names, trademarks, service marks, or product names of the Licensor,
      except as required for reasonable and customary use in describing the
      origin of the Work and reproducing the content of the NOTICE file.

   7. Disclaimer of Warranty. Unless required by applicable law or
      agreed to in writing, Licensor provides the Work (and each
      Contributor provides its Contributions) on an "AS IS" BASIS,
      WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
      implied, including, without limitation, any warranties or conditions
      of TITLE, NON-INFRINGEMENT, MERCHANTABILITY, or FITNESS FOR A
      PARTICULAR PURPOSE. You are solely responsible for determining the
      appropriateness of using or redistributing the Work and assume any
      risks associated with Your exercise of permissions under this License.

   8. Limitation of Liability. In no event and under no legal theory,
      whether in tort (including negligence), contract, or otherwise,
      unless required by applicable law (such as deliberate and grossly
      negligent acts) or agreed to in writing, shall any Contributor be
      liable to You for damages, including any direct, indirect, special,
      incidental, or consequential damages of any character arising as a
      result of this License or out of the use or inability to use the
      Work (including but not limited to damages for loss of goodwill,
      work stoppage, computer failure or malfunction, or any and all
      other commercial damages or losses), even if such Contributor
      has been advised of the possibility of such damages.

   9. Accepting Warranty or Additional Liability. While redistributing
      the Work or Derivative Works thereof, You may choose to offer,
      and charge a fee for, acceptance of support, warranty, indemnity,
      or other liability obligations and/or rights consistent with this
      License. However, in accepting such obligations, You may act only
      on Your own behalf and on Your sole responsibility, not on behalf
      of any other Contributor, and only if You agree to indemnify,
      defend, and hold each Contributor harmless for any liability
      incurred by, or claims asserted against, such Contributor by reason
      of your accepting any such warranty or additional liability.

   END OF TERMS AND CONDITIONS

   APPENDIX: How to apply the Apache License to your work.

      To apply the Apache License to your work, attach the following
      boilerplate notice, with the fields enclosed by brackets "[]"
      replaced with your own identifying information. (Don't include
      the brackets!)  The text should be enclosed in the appropriate
      comment syntax for the file format. We also recommend that a
      file or class name and description of purpose be included on the
      same "printed page" as the copyright notice for easier
      identification within third-party archives.

   Copyright [yyyy] [name of copyright owner]

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
	"context"
	"fmt"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"regexp"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const SetterCommentIdentifier = "# kpt-set: "

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

func (as *ApplySetters) Apply(ctx context.Context, ml manifest.ManifestList) (manifest.ManifestList, error) {
	reader := ml.Reader()
	byteReader := kio.ByteReader{Reader: reader, OmitReaderAnnotations: true}
	nodes, err := byteReader.Read()
	var updated manifest.ManifestList
	if err != nil {
		return updated, err
	}
	nodes, err = as.Filter(nodes)
	for i := range nodes {
		updated.Append([]byte(nodes[i].MustString()))
	}
	return updated, nil
}

/*
visitMapping takes input mapping node, and performs following steps
checks if the key node of the input mapping node has line comment with SetterCommentIdentifier
checks if the value node is of sequence node type
if yes to both, resolves the setter value for the setter name in the line comment
replaces the existing sequence node with the new values provided by user

e.g. for input of Mapping node

environments: # kpt-set: ${env}
- dev
- stage

For input ApplySetters [name: env, value: "[stage, prod]"], qthe yaml node is transformed to

environments: # kpt-set: ${env}
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
			// if node is FlowStyle e.g. env: [foo, bar] # kpt-set: ${env}
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
			// empty sequence must be FlowStyle e.g. env: [] # kpt-set: ${env}
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
		// env: # kpt-set: ${env}
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

e.g.for input of scalar node 'nginx:1.7.1 # kpt-set: ${image}:${tag}' in the yaml node

apiVersion: v1
...

	image: nginx:1.7.1 # kpt-set: ${image}:${tag}

and for input ApplySetters [[name: image, value: ubuntu], [name: tag, value: 1.8.0]]
The yaml node is transformed to

apiVersion: v1
...

	image: ubuntu:1.8.0 # kpt-set: ${image}:${tag}
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
