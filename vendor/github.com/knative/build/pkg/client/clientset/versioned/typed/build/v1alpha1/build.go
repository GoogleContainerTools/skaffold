/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package v1alpha1

import (
	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	scheme "github.com/knative/build/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// BuildsGetter has a method to return a BuildInterface.
// A group's client should implement this interface.
type BuildsGetter interface {
	Builds(namespace string) BuildInterface
}

// BuildInterface has methods to work with Build resources.
type BuildInterface interface {
	Create(*v1alpha1.Build) (*v1alpha1.Build, error)
	Update(*v1alpha1.Build) (*v1alpha1.Build, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.Build, error)
	List(opts v1.ListOptions) (*v1alpha1.BuildList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Build, err error)
	BuildExpansion
}

// builds implements BuildInterface
type builds struct {
	client rest.Interface
	ns     string
}

// newBuilds returns a Builds
func newBuilds(c *BuildV1alpha1Client, namespace string) *builds {
	return &builds{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the build, and returns the corresponding build object, and an error if there is any.
func (c *builds) Get(name string, options v1.GetOptions) (result *v1alpha1.Build, err error) {
	result = &v1alpha1.Build{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("builds").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Builds that match those selectors.
func (c *builds) List(opts v1.ListOptions) (result *v1alpha1.BuildList, err error) {
	result = &v1alpha1.BuildList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("builds").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested builds.
func (c *builds) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("builds").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a build and creates it.  Returns the server's representation of the build, and an error, if there is any.
func (c *builds) Create(build *v1alpha1.Build) (result *v1alpha1.Build, err error) {
	result = &v1alpha1.Build{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("builds").
		Body(build).
		Do().
		Into(result)
	return
}

// Update takes the representation of a build and updates it. Returns the server's representation of the build, and an error, if there is any.
func (c *builds) Update(build *v1alpha1.Build) (result *v1alpha1.Build, err error) {
	result = &v1alpha1.Build{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("builds").
		Name(build.Name).
		Body(build).
		Do().
		Into(result)
	return
}

// Delete takes name of the build and deletes it. Returns an error if one occurs.
func (c *builds) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("builds").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *builds) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("builds").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched build.
func (c *builds) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Build, err error) {
	result = &v1alpha1.Build{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("builds").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
