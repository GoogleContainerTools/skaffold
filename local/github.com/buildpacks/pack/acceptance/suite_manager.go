//go:build acceptance

package acceptance

type SuiteManager struct {
	out          func(format string, args ...interface{})
	results      map[string]interface{}
	cleanUpTasks map[string]func() error
}

func (s *SuiteManager) CleanUp() error {
	for key, cleanUp := range s.cleanUpTasks {
		s.out("Running cleanup task '%s'\n", key)
		if err := cleanUp(); err != nil {
			return err
		}
	}

	return nil
}

func (s *SuiteManager) RegisterCleanUp(key string, cleanUp func() error) {
	if s.cleanUpTasks == nil {
		s.cleanUpTasks = map[string]func() error{}
	}

	s.cleanUpTasks[key] = cleanUp
}

func (s *SuiteManager) RunTaskOnceString(key string, run func() (string, error)) (string, error) {
	v, err := s.runTaskOnce(key, func() (interface{}, error) {
		return run()
	})
	if err != nil {
		return "", err
	}

	return v.(string), nil
}

func (s *SuiteManager) runTaskOnce(key string, run func() (interface{}, error)) (interface{}, error) {
	if s.results == nil {
		s.results = map[string]interface{}{}
	}

	value, found := s.results[key]
	if !found {
		s.out("Running task '%s'\n", key)
		v, err := run()
		if err != nil {
			return nil, err
		}

		s.results[key] = v

		return v, nil
	}

	return value, nil
}
