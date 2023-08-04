package yamlpatch

import (
	"fmt"
	"strconv"
	"strings"
)

// Container is the interface for performing operations on Nodes
type Container interface {
	Get(key string) (*Node, error)
	Set(key string, val *Node) error
	Add(key string, val *Node) error
	Remove(key string) error
}

type nodeMap map[interface{}]*Node

func (n *nodeMap) Set(key string, val *Node) error {
	(*n)[key] = val
	return nil
}

func (n *nodeMap) Add(key string, val *Node) error {
	(*n)[key] = val
	return nil
}

func (n *nodeMap) Get(key string) (*Node, error) {
	return (*n)[key], nil
}

func (n *nodeMap) Remove(key string) error {
	_, ok := (*n)[key]
	if !ok {
		return fmt.Errorf("Unable to remove nonexistent key: %s", key)
	}

	delete(*n, key)
	return nil
}

type nodeSlice []*Node

func (n *nodeSlice) Set(index string, val *Node) error {
	i, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	sz := len(*n)
	if i+1 > sz {
		sz = i + 1
	}

	ary := make([]*Node, sz)

	cur := *n

	copy(ary, cur)

	if i >= len(ary) {
		return fmt.Errorf("Unable to access invalid index: %d", i)
	}

	ary[i] = val

	*n = ary
	return nil
}

func (n *nodeSlice) Add(index string, val *Node) error {
	if index == "-" {
		*n = append(*n, val)
		return nil
	}

	i, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	ary := make([]*Node, len(*n)+1)

	cur := *n

	copy(ary[0:i], cur[0:i])
	ary[i] = val
	copy(ary[i+1:], cur[i:])

	*n = ary
	return nil
}

func (n *nodeSlice) Get(index string) (*Node, error) {
	i, err := strconv.Atoi(index)
	if err != nil {
		return nil, err
	}

	if i >= 0 && i <= len(*n)-1 {
		return (*n)[i], nil
	}

	return nil, fmt.Errorf("Unable to access invalid index: %d", i)
}

func (n *nodeSlice) Remove(index string) error {
	i, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	cur := *n

	if i >= len(cur) {
		return fmt.Errorf("Unable to remove invalid index: %d", i)
	}

	ary := make([]*Node, len(cur)-1)

	copy(ary[0:i], cur[0:i])
	copy(ary[i:], cur[i+1:])

	*n = ary
	return nil

}

func findContainer(c Container, path *OpPath) (Container, string, error) {
	parts, key, err := path.Decompose()
	if err != nil {
		return nil, "", err
	}

	foundContainer := c

	for _, part := range parts {
		node, err := foundContainer.Get(decodePatchKey(part))
		if err != nil {
			return nil, "", err
		}

		if node == nil {
			return nil, "", fmt.Errorf("path does not exist: %s", path)
		}

		foundContainer = node.Container()
	}

	return foundContainer, decodePatchKey(key), nil
}

// From http://tools.ietf.org/html/rfc6901#section-4 :
//
// Evaluation of each reference token begins by decoding any escaped
// character sequence.  This is performed by first transforming any
// occurrence of the sequence '~1' to '/', and then transforming any
// occurrence of the sequence '~0' to '~'.

var (
	rfc6901Decoder = strings.NewReplacer("~1", "/", "~0", "~")
)

func decodePatchKey(k string) string {
	return rfc6901Decoder.Replace(k)
}
