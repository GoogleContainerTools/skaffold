package yit

import "gopkg.in/yaml.v3"

type (
	Iterator  func() (*yaml.Node, bool)
	Predicate func(*yaml.Node) bool
)

func FromNode(node *yaml.Node) Iterator {
	return FromNodes(node)
}

func FromNodes(nodes ...*yaml.Node) Iterator {
	i := 0

	return func() (node *yaml.Node, ok bool) {
		ok = i < len(nodes)
		if !ok {
			return
		}

		node = nodes[i]
		i++

		return
	}
}

func FromIterators(its ...Iterator) Iterator {
	return func() (node *yaml.Node, ok bool) {
		for {
			if len(its) == 0 {
				return
			}

			next := its[0]
			node, ok = next()

			if ok {
				return
			}

			its = its[1:]
		}
	}
}

func (next Iterator) MapKeys() Iterator {
	var content []*yaml.Node

	return func() (node *yaml.Node, ok bool) {
		for {
			if len(content) > 0 {
				node = content[0]
				content = content[2:]
				ok = true
				return
			}

			var parent *yaml.Node
			for parent, ok = next(); ok; parent, ok = next() {
				if parent.Kind == yaml.MappingNode && len(parent.Content) > 0 {
					break
				}
			}

			if !ok {
				return
			}

			content = parent.Content
		}
	}
}

func (next Iterator) MapValues() Iterator {
	var content []*yaml.Node

	return func() (node *yaml.Node, ok bool) {
		for {
			if len(content) > 0 {
				node = content[1]
				content = content[2:]
				ok = true
				return
			}

			var parent *yaml.Node
			for parent, ok = next(); ok; parent, ok = next() {
				if parent.Kind == yaml.MappingNode && len(parent.Content) > 0 {
					break
				}
			}

			if !ok {
				return
			}

			content = parent.Content
		}
	}
}

func (next Iterator) ValuesForMap(keyPredicate, valuePredicate Predicate) Iterator {
	var content []*yaml.Node

	return func() (node *yaml.Node, ok bool) {
		for {
			for len(content) > 0 {
				key := content[0]
				node = content[1]
				content = content[2:]

				if ok = keyPredicate(key) && valuePredicate(node); ok {
					return
				}
			}

			var parent *yaml.Node
			for parent, ok = next(); ok; parent, ok = next() {
				if parent.Kind == yaml.MappingNode && len(parent.Content) > 0 {
					break
				}
			}

			if !ok {
				return
			}

			content = parent.Content
		}
	}
}

func (next Iterator) RecurseNodes() Iterator {
	var stack []*yaml.Node

	return func() (node *yaml.Node, ok bool) {
		if len(stack) > 0 {
			node = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			ok = true
		} else {
			node, ok = next()
			if !ok {
				return
			}
		}

		// iterate backwards so the iteration
		// is predictable (for testing)
		for i := len(node.Content) - 1; i >= 0; i-- {
			stack = append(stack, node.Content[i])
		}

		return
	}
}

func (next Iterator) Filter(p Predicate) Iterator {
	return func() (node *yaml.Node, ok bool) {
		for node, ok = next(); ok; node, ok = next() {
			if p(node) {
				return
			}
		}
		return
	}
}

func (next Iterator) Values() Iterator {
	var content []*yaml.Node

	return func() (node *yaml.Node, ok bool) {
		if len(content) > 0 {
			node = content[0]
			content = content[1:]
			ok = true
			return
		}

		var parent *yaml.Node
		for parent, ok = next(); ok; parent, ok = next() {
			if len(parent.Content) > 0 {
				break
			}
		}

		if !ok {
			return
		}

		content = parent.Content
		node = content[0]
		content = content[1:]

		return
	}
}

func (next Iterator) Iterate(op func(Iterator) Iterator) Iterator {
	return op(next)
}
