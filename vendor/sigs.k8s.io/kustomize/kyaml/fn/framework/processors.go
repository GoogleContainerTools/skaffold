// Copyright 2021 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"strings"

	validationErrors "k8s.io/kube-openapi/pkg/validation/errors"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"k8s.io/kube-openapi/pkg/validation/strfmt"
	"k8s.io/kube-openapi/pkg/validation/validate"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	k8syaml "sigs.k8s.io/yaml"
)

// SimpleProcessor processes a ResourceList by loading the FunctionConfig into
// the given Config type and then running the provided Filter on the ResourceList.
// The provided Config MAY implement Defaulter and Validator to have Default and Validate
// respectively called between unmarshalling and filter execution.
//
// Typical uses include functions that do not actually require config, and simple functions built
// with a filter that closes over the Config instance to access ResourceList.functionConfig values.
type SimpleProcessor struct {
	// Filter is the kio.Filter that will be used to process the ResourceList's items.
	// Note that kio.FilterFunc is available to transform a compatible func into a kio.Filter.
	Filter kio.Filter
	// Config must be a struct capable of receiving the data from ResourceList.functionConfig.
	// Filter functions may close over this struct to access its data.
	Config interface{}
}

// Process makes SimpleProcessor implement the ResourceListProcessor interface.
// It loads the ResourceList.functionConfig into the provided Config type, applying
// defaulting and validation if supported by Config. It then executes the processor's filter.
func (p SimpleProcessor) Process(rl *ResourceList) error {
	if err := LoadFunctionConfig(rl.FunctionConfig, p.Config); err != nil {
		return errors.WrapPrefixf(err, "loading function config")
	}
	return errors.WrapPrefixf(rl.Filter(p.Filter), "processing filter")
}

// GVKFilterMap is a FilterProvider that resolves Filters through a simple lookup in a map.
// It is intended for use in VersionedAPIProcessor.
type GVKFilterMap map[string]map[string]kio.Filter

// ProviderFor makes GVKFilterMap implement the FilterProvider interface.
// It uses the given apiVersion and kind to do a simple lookup in the map and
// returns an error if no exact match is found.
func (m GVKFilterMap) ProviderFor(apiVersion, kind string) (kio.Filter, error) {
	if kind == "" {
		return nil, errors.Errorf("kind is required")
	}
	if apiVersion == "" {
		return nil, errors.Errorf("apiVersion is required")
	}

	var ok bool
	var versionMap map[string]kio.Filter
	if versionMap, ok = m[kind]; !ok {
		return nil, errors.Errorf("kind %q is not supported", kind)
	}

	var p kio.Filter
	if p, ok = versionMap[apiVersion]; !ok {
		return nil, errors.Errorf("apiVersion %q is not supported for kind %q", apiVersion, kind)
	}
	return p, nil
}

// FilterProvider is implemented by types that provide a way to look up which Filter
// should be used to process a ResourceList based on the ApiVersion and Kind of the
// ResourceList.functionConfig in the input. FilterProviders are intended to be used
// as part of VersionedAPIProcessor.
type FilterProvider interface {
	// ProviderFor returns the appropriate filter for the given APIVersion and Kind.
	ProviderFor(apiVersion, kind string) (kio.Filter, error)
}

// FilterProviderFunc converts a compatible function to a FilterProvider.
type FilterProviderFunc func(apiVersion, kind string) (kio.Filter, error)

// ProviderFor makes FilterProviderFunc implement FilterProvider.
func (f FilterProviderFunc) ProviderFor(apiVersion, kind string) (kio.Filter, error) {
	return f(apiVersion, kind)
}

// VersionedAPIProcessor selects the appropriate kio.Filter based on the ApiVersion
// and Kind of the ResourceList.functionConfig in the input.
// It can be used to implement configuration function APIs that evolve over time,
// or create processors that support multiple configuration APIs with a single entrypoint.
// All provided Filters MUST be structs capable of receiving ResourceList.functionConfig data.
// Provided Filters MAY implement Defaulter and Validator to have Default and Validate
// respectively called between unmarshalling and filter execution.
type VersionedAPIProcessor struct {
	// FilterProvider resolves a kio.Filter for each supported API, based on its APIVersion and Kind.
	// GVKFilterMap is a simple FilterProvider implementation for use here.
	FilterProvider FilterProvider
}

// Process makes VersionedAPIProcessor implement the ResourceListProcessor interface.
// It looks up the configuration object to use based on the ApiVersion and Kind of the
// input ResourceList.functionConfig, loads ResourceList.functionConfig into that object,
// invokes Validate and Default if supported, and finally invokes Filter.
func (p *VersionedAPIProcessor) Process(rl *ResourceList) error {
	api, err := p.FilterProvider.ProviderFor(extractGVK(rl.FunctionConfig))
	if err != nil {
		return errors.WrapPrefixf(err, "unable to identify provider for resource")
	}
	if err := LoadFunctionConfig(rl.FunctionConfig, api); err != nil {
		return errors.Wrap(err)
	}
	return errors.Wrap(rl.Filter(api))
}

// extractGVK returns the apiVersion and kind fields from the given RNodes if it contains
// valid TypeMeta. It returns an empty string if a value is not found.
func extractGVK(src *yaml.RNode) (apiVersion, kind string) {
	if src == nil {
		return "", ""
	}
	if versionNode := src.Field("apiVersion"); versionNode != nil {
		if a, err := versionNode.Value.String(); err == nil {
			apiVersion = strings.TrimSpace(a)
		}
	}
	if kindNode := src.Field("kind"); kindNode != nil {
		if k, err := kindNode.Value.String(); err == nil {
			kind = strings.TrimSpace(k)
		}
	}
	return apiVersion, kind
}

// LoadFunctionConfig reads a configuration resource from YAML into the provided data structure
// and then prepares it for use by running defaulting and validation on it, if supported.
// ResourceListProcessors should use this function to load ResourceList.functionConfig.
func LoadFunctionConfig(src *yaml.RNode, api interface{}) error {
	if api == nil {
		return nil
	}
	// Run this before unmarshalling to avoid nasty unmarshal failure error messages
	var schemaValidationError error
	if s, ok := api.(ValidationSchemaProvider); ok {
		schema, err := s.Schema()
		if err != nil {
			return errors.WrapPrefixf(err, "loading provided schema")
		}
		schemaValidationError = errors.Wrap(validate.AgainstSchema(schema, src, strfmt.Default))
		// don't return it yet--try to make it to custom validation stage to combine errors
	}

	// using sigs.k8s.io/yaml here lets the custom types embed core types
	// that only have json tags, notably types from k8s.io/apimachinery/pkg/apis/meta/v1
	if err := k8syaml.Unmarshal([]byte(src.MustString()), api); err != nil {
		if schemaValidationError != nil {
			// if we got a validation error, report it instead as it is likely a nicer version of the same message
			return schemaValidationError
		}
		return errors.Wrap(err)
	}

	if d, ok := api.(Defaulter); ok {
		if err := d.Default(); err != nil {
			return errors.Wrap(err)
		}
	}

	if v, ok := api.(Validator); ok {
		return combineErrors(schemaValidationError, v.Validate())
	}
	return schemaValidationError
}

// combineErrors produces a CompositeValidationError for the given schemaErr and givenErr.
// If either is already a CompsiteError, its constituent errors become part of the new
// composite error. If both given errors are nil, this function returns nil.
func combineErrors(schemaErr, customErr error) error {
	combined := validationErrors.CompositeValidationError()
	if compositeSchemaErr, ok := schemaErr.(*validationErrors.CompositeError); ok {
		combined.Errors = append(combined.Errors, compositeSchemaErr.Errors...)
	} else if schemaErr != nil {
		combined.Errors = append(combined.Errors, schemaErr)
	}
	if compositeCustomErr, ok := customErr.(*validationErrors.CompositeError); ok {
		combined.Errors = append(combined.Errors, compositeCustomErr.Errors...)
	} else if customErr != nil {
		combined.Errors = append(combined.Errors, customErr)
	}
	if len(combined.Errors) > 0 {
		return combined
	}
	return nil
}

// TemplateProcessor is a ResourceListProcessor based on rendering templates with the data in
// ResourceList.functionConfig. It works as follows:
// - loads ResourceList.functionConfig into TemplateData
// - runs PreProcessFilters
// - renders ResourceTemplates and adds them to ResourceList.items
// - renders PatchTemplates and applies them to ResourceList.items
// - executes a merge on ResourceList.items if configured to
// - runs PostProcessFilters
// The TemplateData struct MAY implement Defaulter and Validator to have Default and Validate
// respectively called between unmarshalling and filter execution.
//
// TemplateProcessor also implements kio.Filter directly and can be used in the construction of
// higher-level processors. For example, you might use TemplateProcessors as the filters for each
// API supported by a VersionedAPIProcessor (see VersionedAPIProcessor examples).
type TemplateProcessor struct {
	// TemplateData will will be exposed to all the templates in the processor (unless explicitly
	// overridden for a template).
	// If TemplateProcessor is used directly as a ResourceListProcessor, TemplateData will contain the
	// value of ResourceList.functionConfig.
	TemplateData interface{}

	// ResourceTemplates returns a list of templates to render into resources.
	// If MergeResources is set, any matching resources in ResourceList.items will be used as patches
	// modifying the rendered templates. Otherwise, the rendered resources will be appended to
	// the input resources as-is.
	ResourceTemplates []ResourceTemplate

	// PatchTemplates is a list of templates to render into patches that apply to ResourceList.items.
	// ResourcePatchTemplate can be used here to patch entire resources.
	// ContainerPatchTemplate can be used here to patch specific containers within resources.
	PatchTemplates []PatchTemplate

	// MergeResources, if set to true, will cause the resources in ResourceList.items to be
	// applied as patches on any matching resources generated by ResourceTemplates.
	MergeResources bool

	// PreProcessFilters provides a hook to manipulate the ResourceList's items or config after
	// TemplateData has been populated but before template-based filters are applied.
	PreProcessFilters []kio.Filter

	// PostProcessFilters provides a hook to manipulate the ResourceList's items after template
	// filters are applied.
	PostProcessFilters []kio.Filter

	// AdditionalSchemas is a function that returns a list of schema definitions to add to openapi.
	// This enables correct merging of custom resource fields.
	AdditionalSchemas SchemaParser
}

type SchemaParser interface {
	Parse() ([]*spec.Definitions, error)
}

type SchemaParserFunc func() ([]*spec.Definitions, error)

func (s SchemaParserFunc) Parse() ([]*spec.Definitions, error) {
	return s()
}

// Filter implements the kio.Filter interface, enabling you to use TemplateProcessor
// as part of a higher-level ResourceListProcessor like VersionedAPIProcessor.
// It sets up all the features of TemplateProcessors as a pipeline of filters and executes them.
func (tp TemplateProcessor) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	if tp.AdditionalSchemas != nil {
		defs, err := tp.AdditionalSchemas.Parse()
		if err != nil {
			return nil, errors.WrapPrefixf(err, "parsing AdditionalSchemas")
		}
		defer openapi.ResetOpenAPI()
		for i := range defs {
			openapi.AddDefinitions(*defs[i])
		}
	}

	buf := &kio.PackageBuffer{Nodes: items}
	pipeline := kio.Pipeline{
		Inputs: []kio.Reader{buf},
		Filters: []kio.Filter{
			kio.FilterFunc(tp.doPreProcess),
			kio.FilterFunc(tp.doResourceTemplates),
			kio.FilterFunc(tp.doPatchTemplates),
			kio.FilterFunc(tp.doMerge),
			kio.FilterFunc(tp.doPostProcess),
		},
		Outputs:               []kio.Writer{buf},
		ContinueOnEmptyResult: true,
	}
	if err := pipeline.Execute(); err != nil {
		return nil, err
	}

	return buf.Nodes, nil
}

// Process implements the ResourceListProcessor interface, enabling you to use TemplateProcessor
// directly as a processor. As a Processor, it loads the ResourceList.functionConfig into the
// TemplateData field, exposing it to all templates by default.
func (tp TemplateProcessor) Process(rl *ResourceList) error {
	if err := LoadFunctionConfig(rl.FunctionConfig, tp.TemplateData); err != nil {
		return errors.Wrap(err)
	}
	return errors.Wrap(rl.Filter(tp))
}

// PatchTemplate is implemented by kio.Filters that work by rendering patches and applying them to
// the given resource nodes.
type PatchTemplate interface {
	// Filter is a kio.Filter-compliant function that applies PatchTemplate's templates as patches
	// on the given resource nodes.
	Filter(items []*yaml.RNode) ([]*yaml.RNode, error)
	// DefaultTemplateData accepts default data to be used in template rendering when no template
	// data was explicitly provided to the PatchTemplate.
	DefaultTemplateData(interface{})
}

func (tp *TemplateProcessor) doPreProcess(items []*yaml.RNode) ([]*yaml.RNode, error) {
	if tp.PreProcessFilters == nil {
		return items, nil
	}
	for i := range tp.PreProcessFilters {
		filter := tp.PreProcessFilters[i]
		var err error
		items, err = filter.Filter(items)
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (tp *TemplateProcessor) doMerge(items []*yaml.RNode) ([]*yaml.RNode, error) {
	var err error
	if tp.MergeResources {
		items, err = filters.MergeFilter{}.Filter(items)
	}
	return items, err
}

func (tp *TemplateProcessor) doPostProcess(items []*yaml.RNode) ([]*yaml.RNode, error) {
	if tp.PostProcessFilters == nil {
		return items, nil
	}
	for i := range tp.PostProcessFilters {
		filter := tp.PostProcessFilters[i]
		var err error
		items, err = filter.Filter(items)
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (tp *TemplateProcessor) doResourceTemplates(items []*yaml.RNode) ([]*yaml.RNode, error) {
	if tp.ResourceTemplates == nil {
		return items, nil
	}

	for i := range tp.ResourceTemplates {
		tp.ResourceTemplates[i].DefaultTemplateData(tp.TemplateData)
		newItems, err := tp.ResourceTemplates[i].Render()
		if err != nil {
			return nil, err
		}
		if tp.MergeResources {
			// apply inputs as patches -- add the new items to the front of the list
			items = append(newItems, items...)
		} else {
			// assume these are new unique resources--append to the list
			items = append(items, newItems...)
		}
	}
	return items, nil
}

func (tp *TemplateProcessor) doPatchTemplates(items []*yaml.RNode) ([]*yaml.RNode, error) {
	if tp.PatchTemplates == nil {
		return items, nil
	}

	for i := range tp.PatchTemplates {
		// Default the template data for the patch to the processor's data
		tp.PatchTemplates[i].DefaultTemplateData(tp.TemplateData)
		var err error
		if items, err = tp.PatchTemplates[i].Filter(items); err != nil {
			return nil, err
		}
	}
	return items, nil
}
