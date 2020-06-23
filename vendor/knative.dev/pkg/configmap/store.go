/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package configmap

import (
	"reflect"
	"sync/atomic"

	corev1 "k8s.io/api/core/v1"
)

// ExampleKey signifies a given example configuration in a ConfigMap.
const ExampleKey = "_example"

// Logger is the interface that UntypedStore expects its logger to conform to.
// UntypedStore will log when updates succeed or fail.
type Logger interface {
	Infof(string, ...interface{})
	Fatalf(string, ...interface{})
	Errorf(string, ...interface{})
}

// Constructors is a map for specifying config names to
// their function constructors
//
// The values of this map must be functions with the definition
//
// func(*k8s.io/api/core/v1.ConfigMap) (... , error)
//
// These functions can return any type along with an error
type Constructors map[string]interface{}

// An UntypedStore is a responsible for storing and
// constructing configs from Kubernetes ConfigMaps
//
// WatchConfigs should be used with a configmap.Watcher
// in order for this store to remain up to date
type UntypedStore struct {
	name   string
	logger Logger

	storages     map[string]*atomic.Value
	constructors map[string]reflect.Value

	onAfterStore []func(name string, value interface{})
}

// NewUntypedStore creates an UntypedStore with given name,
// Logger and Constructors
//
// The Logger must not be nil
//
// The values in the Constructors map must be functions with
// the definition
//
// func(*k8s.io/api/core/v1.ConfigMap) (... , error)
//
// These functions can return any type along with an error.
// If the function definition differs then NewUntypedStore
// will panic.
//
// onAfterStore is a variadic list of callbacks to run
// after the ConfigMap has been transformed (via the appropriate Constructor)
// and stored. These callbacks run sequentially (in the argument order) in a
// separate go-routine and are of type func(name string, value interface{})
// where name is the config-map name and value is the object that has been
// constructed from the config-map and stored.
func NewUntypedStore(
	name string,
	logger Logger,
	constructors Constructors,
	onAfterStore ...func(name string, value interface{})) *UntypedStore {

	store := &UntypedStore{
		name:         name,
		logger:       logger,
		storages:     make(map[string]*atomic.Value),
		constructors: make(map[string]reflect.Value),
		onAfterStore: onAfterStore,
	}

	for configName, constructor := range constructors {
		store.registerConfig(configName, constructor)
	}

	return store
}

func (s *UntypedStore) registerConfig(name string, constructor interface{}) {
	if err := ValidateConstructor(constructor); err != nil {
		panic(err)
	}

	s.storages[name] = &atomic.Value{}
	s.constructors[name] = reflect.ValueOf(constructor)
}

// WatchConfigs uses the provided configmap.Watcher
// to setup watches for the config names provided in the
// Constructors map
func (s *UntypedStore) WatchConfigs(w Watcher) {
	for configMapName := range s.constructors {
		w.Watch(configMapName, s.OnConfigChanged)
	}
}

// UntypedLoad will return the constructed value for a given
// ConfigMap name
func (s *UntypedStore) UntypedLoad(name string) interface{} {
	storage := s.storages[name]
	return storage.Load()
}

// OnConfigChanged will invoke the mapped constructor against
// a Kubernetes ConfigMap. If successful it will be stored.
// If construction fails during the first appearance the store
// will log a fatal error. If construction fails while updating
// the store will log an error message.
func (s *UntypedStore) OnConfigChanged(c *corev1.ConfigMap) {
	name := c.ObjectMeta.Name

	storage := s.storages[name]
	constructor := s.constructors[name]

	inputs := []reflect.Value{
		reflect.ValueOf(c),
	}

	outputs := constructor.Call(inputs)
	result := outputs[0].Interface()
	errVal := outputs[1]

	if !errVal.IsNil() {
		err := errVal.Interface()
		if storage.Load() != nil {
			s.logger.Errorf("Error updating %s config %q: %q", s.name, name, err)
		} else {
			s.logger.Fatalf("Error initializing %s config %q: %q", s.name, name, err)
		}
		return
	}

	s.logger.Infof("%s config %q config was added or updated: %#v", s.name, name, result)
	storage.Store(result)

	go func() {
		for _, f := range s.onAfterStore {
			f(name, result)
		}
	}()
}
