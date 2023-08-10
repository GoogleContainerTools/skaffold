package yamlpatch

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

// Patch is an ordered collection of operations.
type Patch []Operation

// DecodePatch decodes the passed YAML document as if it were an RFC 6902 patch
func DecodePatch(bs []byte) (Patch, error) {
	var p Patch

	err := yaml.Unmarshal(bs, &p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Apply returns a YAML document that has been mutated per the patch
func (p Patch) Apply(doc []byte) ([]byte, error) {
	var iface interface{}
	err := yaml.Unmarshal(doc, &iface)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshaling doc: %s\n\n%s", string(doc), err)
	}

	var c Container
	c = NewNode(&iface).Container()

	for _, op := range p {
		pathfinder := NewPathFinder(c)
		if op.Path.ContainsExtendedSyntax() {
			paths := pathfinder.Find(string(op.Path))
			if paths == nil {
				return nil, fmt.Errorf("could not expand pointer: %s", op.Path)
			}

			for _, path := range paths {
				newOp := op
				newOp.Path = OpPath(path)
				err = newOp.Perform(c)
				if err != nil {
					return nil, err
				}
			}
		} else {
			err = op.Perform(c)
			if err != nil {
				return nil, err
			}
		}
	}

	return yaml.Marshal(c)
}
