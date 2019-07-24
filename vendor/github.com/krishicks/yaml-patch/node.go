package yamlpatch

import "reflect"

// Node holds a YAML document that has not yet been processed into a NodeMap or
// NodeSlice
type Node struct {
	raw       *interface{}
	container Container
}

// NewNode returns a new Node. It expects a pointer to an interface{}
func NewNode(raw *interface{}) *Node {
	return &Node{
		raw: raw,
	}
}

// MarshalYAML implements yaml.Marshaler, and returns the correct interface{}
// to be marshaled
func (n *Node) MarshalYAML() (interface{}, error) {
	if n.container != nil {
		return n.container, nil
	}

	return *n.raw, nil
}

// UnmarshalYAML implements yaml.Unmarshaler
func (n *Node) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var data interface{}

	err := unmarshal(&data)
	if err != nil {
		return err
	}

	n.raw = &data
	return nil
}

// Empty returns whether the raw value is nil
func (n *Node) Empty() bool {
	return *n.raw == nil
}

// Container returns the node as a Container
func (n *Node) Container() Container {
	if n.container != nil {
		return n.container
	}

	switch rt := (*n.raw).(type) {
	case []interface{}:
		c := make(nodeSlice, len(rt))
		n.container = &c

		for i := range rt {
			c[i] = NewNode(&rt[i])
		}
	case map[interface{}]interface{}:
		c := make(nodeMap, len(rt))
		n.container = &c

		for k := range rt {
			v := rt[k]
			c[k] = NewNode(&v)
		}
	}

	return n.container
}

// Equal compares the values of the raw interfaces that the YAML was
// unmarshaled into
func (n *Node) Equal(other *Node) bool {
	return reflect.DeepEqual(*n.raw, *other.raw)
}

// Value returns the raw value of the node
func (n *Node) Value() interface{} {
	return *n.raw
}
