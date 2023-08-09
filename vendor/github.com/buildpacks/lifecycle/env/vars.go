package env

import (
	"strings"
)

type Vars struct {
	vals       map[string]string
	ignoreCase bool
}

func varsFromEnv(env []string, ignoreCase bool, removeKey func(string) bool) *Vars {
	vars := NewVars(nil, ignoreCase)
	for _, kv := range env {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if removeKey(parts[0]) {
			continue
		}
		vars.Set(parts[0], parts[1])
	}
	return vars
}

func NewVars(vars map[string]string, ignoreCase bool) *Vars {
	s := &Vars{
		vals:       map[string]string{},
		ignoreCase: ignoreCase,
	}
	for k, v := range vars {
		s.Set(k, v)
	}
	return s
}

func (s *Vars) Get(key string) string {
	return s.vals[s.key(key)]
}

func (s *Vars) Set(key, value string) {
	s.vals[s.key(key)] = value
}

func (s *Vars) key(k string) string {
	if s.ignoreCase {
		return strings.ToUpper(k)
	}
	return k
}

func (s *Vars) List() []string {
	var result []string
	for k, v := range s.vals {
		result = append(result, k+"="+v)
	}
	return result
}
