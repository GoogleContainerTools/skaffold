package yit

import "gopkg.in/yaml.v3"

func (next Iterator) AnyMatch(p Predicate) bool {
	iterator := next.Filter(p)
	_, ok := iterator()
	return ok
}

func (next Iterator) AllMatch(p Predicate) bool {
	result := true
	for node, ok := next(); ok && result; node, ok = next() {
		result = result && p(node)
	}

	return result
}

func (next Iterator) ToArray() (result []*yaml.Node) {
	for node, ok := next(); ok; node, ok = next() {
		result = append(result, node)
	}
	return
}
