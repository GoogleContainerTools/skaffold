package yamlpatch

import (
	"errors"
	"fmt"
	"strings"
)

// Op is a type alias
type Op string

// Ops
const (
	opAdd     Op = "add"
	opRemove  Op = "remove"
	opReplace Op = "replace"
	opMove    Op = "move"
	opCopy    Op = "copy"
	opTest    Op = "test"
)

// OpPath is an RFC6902 'pointer'
type OpPath string

// Decompose returns the pointer's components:
// "/foo" => [], "foo"
// "/foo/1" => ["foo"], "1"
// "/foo/1/bar" => ["foo", "1"], "bar"
func (p *OpPath) Decompose() ([]string, string, error) {
	path := string(*p)

	if !strings.HasPrefix(path, "/") {
		return nil, "", fmt.Errorf("operation path is missing leading '/': %s", path)
	}

	parts := strings.Split(path, "/")[1:]

	return parts[:len(parts)-1], parts[len(parts)-1], nil
}

// ContainsExtendedSyntax returns whether the OpPath uses the "key=value"
// format, as in "/foo/name=bar", where /foo points at an array that contains
// an object with a key "name" that has a value "bar"
func (p *OpPath) ContainsExtendedSyntax() bool {
	return strings.Contains(string(*p), "=")
}

// String returns the OpPath as a string
func (p *OpPath) String() string {
	return string(*p)
}

// Operation is an RFC6902 'Operation'
// https://tools.ietf.org/html/rfc6902#section-4
type Operation struct {
	Op    Op     `yaml:"op,omitempty"`
	Path  OpPath `yaml:"path,omitempty"`
	From  OpPath `yaml:"from,omitempty"`
	Value *Node  `yaml:"value,omitempty"`
}

// Perform executes the operation on the given container
func (o *Operation) Perform(c Container) error {
	var err error

	switch o.Op {
	case opAdd:
		err = tryAdd(c, o)
	case opRemove:
		err = tryRemove(c, o)
	case opReplace:
		err = tryReplace(c, o)
	case opMove:
		err = tryMove(c, o)
	case opCopy:
		err = tryCopy(c, o)
	case opTest:
		err = tryTest(c, o)
	default:
		err = fmt.Errorf("Unexpected op: %s", o.Op)
	}

	return err
}

func tryAdd(doc Container, op *Operation) error {
	con, key, err := findContainer(doc, &op.Path)
	if err != nil {
		return fmt.Errorf("yamlpatch add operation does not apply: doc is missing path: %s", op.Path)
	}

	return con.Add(key, op.Value)
}

func tryRemove(doc Container, op *Operation) error {
	con, key, err := findContainer(doc, &op.Path)
	if err != nil {
		return fmt.Errorf("yamlpatch remove operation does not apply: doc is missing path: %s", op.Path)
	}

	return con.Remove(key)
}

func tryReplace(doc Container, op *Operation) error {
	con, key, err := findContainer(doc, &op.Path)
	if err != nil {
		return fmt.Errorf("yamlpatch replace operation does not apply: doc is missing path: %s", op.Path)
	}

	val, err := con.Get(key)
	if val == nil || err != nil {
		return fmt.Errorf("yamlpatch replace operation does not apply: doc is missing key: %s", op.Path)
	}

	return con.Set(key, op.Value)
}

func tryMove(doc Container, op *Operation) error {
	con, key, err := findContainer(doc, &op.From)
	if err != nil {
		return fmt.Errorf("yamlpatch move operation does not apply: doc is missing from path: %s", op.From)
	}

	val, err := con.Get(key)
	if err != nil {
		return err
	}

	err = con.Remove(key)
	if err != nil {
		return err
	}

	con, key, err = findContainer(doc, &op.Path)
	if err != nil {
		return fmt.Errorf("yamlpatch move operation does not apply: doc is missing destination path: %s", op.Path)
	}

	return con.Set(key, val)
}

func tryCopy(doc Container, op *Operation) error {
	con, key, err := findContainer(doc, &op.From)
	if err != nil {
		return fmt.Errorf("copy operation does not apply: doc is missing from path: %s", op.From)
	}

	val, err := con.Get(key)
	if err != nil {
		return err
	}

	con, key, err = findContainer(doc, &op.Path)
	if err != nil {
		return fmt.Errorf("copy operation does not apply: doc is missing destination path: %s", op.Path)
	}

	return con.Set(key, val)
}

func tryTest(doc Container, op *Operation) error {
	con, key, err := findContainer(doc, &op.Path)
	if err != nil {
		return fmt.Errorf("test operation does not apply: doc is missing from path: %s", op.From)
	}

	val, err := con.Get(key)
	if err != nil {
		return err
	}

	if op.Value.Empty() && val == nil {
		return nil
	}

	if op.Value.Equal(val) {
		return nil
	}

	return errors.New("test failed")
}
