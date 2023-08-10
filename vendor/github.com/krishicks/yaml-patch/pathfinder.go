package yamlpatch

import (
	"fmt"
	"strings"
)

// PathFinder can be used to find RFC6902-standard paths given non-standard
// (key=value) pointer syntax
type PathFinder struct {
	root Container
}

// NewPathFinder takes an interface that represents a YAML document and returns
// a new PathFinder
func NewPathFinder(container Container) *PathFinder {
	return &PathFinder{
		root: container,
	}
}

// Find expands the given path into all matching paths, returning the canonical
// versions of those matching paths
func (p *PathFinder) Find(path string) []string {
	parts := strings.Split(path, "/")

	if parts[1] == "" {
		return []string{"/"}
	}

	routes := map[string]Container{
		"": p.root,
	}

	for _, part := range parts[1:] {
		routes = find(decodePatchKey(part), routes)
	}

	var paths []string
	for k := range routes {
		paths = append(paths, k)
	}

	return paths
}

func find(part string, routes map[string]Container) map[string]Container {
	matches := map[string]Container{}

	for prefix, container := range routes {
		if part == "-" {
			for k := range routes {
				matches[fmt.Sprintf("%s/-", k)] = routes[k]
			}
			return matches
		}

		if kv := strings.Split(part, "="); len(kv) == 2 {
			if newMatches := findAll(prefix, kv[0], kv[1], container); len(newMatches) > 0 {
				matches = newMatches
			}
			continue
		}

		if node, err := container.Get(part); err == nil {
			path := fmt.Sprintf("%s/%s", prefix, part)
			if node == nil {
				matches[path] = container
			} else {
				matches[path] = node.Container()
			}
		}
	}

	return matches
}

func findAll(prefix, findKey, findValue string, container Container) map[string]Container {
	if container == nil {
		return nil
	}

	if v, err := container.Get(findKey); err == nil && v != nil {
		if vs, ok := v.Value().(string); ok && vs == findValue {
			return map[string]Container{
				prefix: container,
			}
		}
	}

	matches := map[string]Container{}

	switch it := container.(type) {
	case *nodeMap:
		for k, v := range *it {
			for route, match := range findAll(fmt.Sprintf("%s/%s", prefix, k), findKey, findValue, v.Container()) {
				matches[route] = match
			}
		}
	case *nodeSlice:
		for i, v := range *it {
			for route, match := range findAll(fmt.Sprintf("%s/%d", prefix, i), findKey, findValue, v.Container()) {
				matches[route] = match
			}
		}
	}

	return matches
}
