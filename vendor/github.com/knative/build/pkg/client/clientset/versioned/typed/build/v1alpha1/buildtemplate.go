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

// BuildTemplatesGetter has a method to return a BuildTemplateInterface.
// A group's client should implement this interface.
type BuildTemplatesGetter interface {
	BuildTemplates(namespace string) BuildTemplateInterface
}

// BuildTemplateInterface has methods to work with BuildTemplate resources.
type BuildTemplateInterface interface {
	Create(*v1alpha1.BuildTemplate) (*v1alpha1.BuildTemplate, error)
	Update(*v1alpha1.BuildTemplate) (*v1alpha1.BuildTemplate, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.BuildTemplate, error)
	List(opts v1.ListOptions) (*v1alpha1.BuildTemplateList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.BuildTemplate, err error)
	BuildTemplateExpansion
}

// buildTemplates implements BuildTemplateInterface
type buildTemplates struct {
	client rest.Interface
	ns     string
}

// newBuildTemplates returns a BuildTemplates
func newBuildTemplates(c *BuildV1alpha1Client, namespace string) *buildTemplates {
	return &buildTemplates{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the buildTemplate, and returns the corresponding buildTemplate object, and an error if there is any.
func (c *buildTemplates) Get(name string, options v1.GetOptions) (result *v1alpha1.BuildTemplate, err error) {
	result = &v1alpha1.BuildTemplate{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("buildtemplates").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of BuildTemplates that match those selectors.
func (c *buildTemplates) List(opts v1.ListOptions) (result *v1alpha1.BuildTemplateList, err error) {
	result = &v1alpha1.BuildTemplateList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("buildtemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested buildTemplates.
func (c *buildTemplates) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("buildtemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a buildTemplate and creates it.  Returns the server's representation of the buildTemplate, and an error, if there is any.
func (c *buildTemplates) Create(buildTemplate *v1alpha1.BuildTemplate) (result *v1alpha1.BuildTemplate, err error) {
	result = &v1alpha1.BuildTemplate{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("buildtemplates").
		Body(buildTemplate).
		Do().
		Into(result)
	return
}

// Update takes the representation of a buildTemplate and updates it. Returns the server's representation of the buildTemplate, and an error, if there is any.
func (c *buildTemplates) Update(buildTemplate *v1alpha1.BuildTemplate) (result *v1alpha1.BuildTemplate, err error) {
	result = &v1alpha1.BuildTemplate{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("buildtemplates").
		Name(buildTemplate.Name).
		Body(buildTemplate).
		Do().
		Into(result)
	return
}

// Delete takes name of the buildTemplate and deletes it. Returns an error if one occurs.
func (c *buildTemplates) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("buildtemplates").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *buildTemplates) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("buildtemplates").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched buildTemplate.
func (c *buildTemplates) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.BuildTemplate, err error) {
	result = &v1alpha1.BuildTemplate{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("buildtemplates").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
