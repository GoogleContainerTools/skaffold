# Consistent Templating

* Author(s): Christoph Girstenbrei, Brian Topping
* Design Shepherd:
* Date: 2023-09-05
* Status: Reviewed
* Related: #9063, #9062, (minor) #8872

## Background

Currently, multiple renderers & deployers support templating in some of their
fields. One example is the helm deployer.

Which fields support templating depends on the deployer and which values are
available for templating differs from field to field.

Additionally which templating logic is allowed in which field is not consistent.

**Example**

```yaml
apiVersion: skaffold/v4beta6
kind: Config
metadata:
  name: skaffold

deploy:
  helm:
    releases:
      # Assume env variable 'NAME' with value 'example' set
      - name: "{{ .NAME }}"
        chartPath: helm
        setValueTemplates:
          image.tag: "{{ .IMAGE_TAG_myFirstImage }}"

```

The resulting helm command in this example is:

```
helm --kube-context example-namespace install example helm --post-renderer /usr/local/bin/skaffold --set image.tag=<no value>
```

This will in the best case result in an error, in the worst case in a deployed helm chart with `image.tag` value set to `<no value>`.

Trying to set defaults for both variables results in the following:

```yaml
apiVersion: skaffold/v4beta6
kind: Config
metadata:
  name: skaffold

deploy:
  helm:
    releases:
      # Assume env variable 'NAME' is not set here
      - name: '{{ default "example" .NAME }}'
        chartPath: helm
        setValueTemplates:
          image.tag: '{{ default "latest" .IMAGE_TAG_myFirstImage }}'
```

The error here is:

```
cannot expand release name "{{ default \"latest\" .NAME }}": template: envTemplate:1:20: executing "envTemplate" at <.NAME>: map has no entry for key "NAME"
```

**Conclusion**

To a user, this behavior seems confusing. Why can the `default` function be used in one field and not in another? Why are some values only available in `setValueTemplates` (e.g. `.IMAGE_*`)? Why does templating render `<no value>` in a field?

**Goals**

1. Provide an inherently consistent user experience of the golang templating language.\
   This includes having the same templating functions available in all fields and
   a consistent error behavior across fields.
2. Provide a consistent set of values to template with inside a specific pipeline stage.\
   This includes having access to all template values available to the pipeline
   stage in question.
3. Provide values to template with in a structured way to downstream pipeline stages.\
   This includes having access to build information in deploy stages.
___

## Design

The design is split according to the goals set.

### Phase 1: Consistent templating UX
Currently, templating is mainly happening via
[`util.ExpandEnvTemplate(...)` @ util/env_template.go#43](https://github.com/GoogleContainerTools/skaffold/blob/bca77a4f2d421f487adf20c6b34af0daba08c2f7/pkg/skaffold/util/env_template.go#L43). It is a single unit of code execution,
you provided it with a template string and a map of variables, it gives you back
a finished, templated string or an error. [`util.ExpandEnvTemplateOrFail(...)` @ util/env_template.go#52](https://github.com/GoogleContainerTools/skaffold/blob/bca77a4f2d421f487adf20c6b34af0daba08c2f7/pkg/skaffold/util/env_template.go#L52)
works based on the same design, but returning an error on a missing key.

This design prevents any caching between template execution. Currently, templates
[are parsed @ util/env_template.go#44](https://github.com/GoogleContainerTools/skaffold/blob/bca77a4f2d421f487adf20c6b34af0daba08c2f7/pkg/skaffold/util/env_template.go#L44)
environment variables are parsed [every invocation @ util/env_template.go#69](https://github.com/GoogleContainerTools/skaffold/blob/bca77a4f2d421f487adf20c6b34af0daba08c2f7/pkg/skaffold/util/env_template.go#L69-L72) of those methods. \
To solve this, provide a `util.Templater` object. This should be able to:

 * be instantiated
 * parse environment variables once
 * add additional variables on instantiation
 * evaluate those templates with additional values per evaluation

This can then be used by every location in code which needs templating.
A somewhat similar struct is already in use by [tag templating @ tag/env_template.go#30](https://github.com/GoogleContainerTools/skaffold/blob/bca77a4f2d421f487adf20c6b34af0daba08c2f7/pkg/skaffold/tag/env_template.go#L30)


#### Open Issues/Questions

**\<Should parsed templates be cached?\>** \
To be able to to do so effectively would
require to instantiate the `Templater` in a very central place and then cache
templates based on e.g. YQ path names. While this might benefit performance,
uniqueness of set cache keys (e.g. path names) would have to be guaranteed to
avoid accidental misuse  of templates in other fields.

Resolution: __Not Yet Resolved__

**\<Should empty values be disallowed?\>** \
One could argue, that if a field is present in the `skaffold.yaml`, it should
always have some value other than the empty string. An empty string is could
in most cases be related to an unset variable or missing default. This would
allow us to set `missingkey=zero` and error out if a template after templating
is `== ""`. Optionally this restriction could only apply to some fields.
Using `missingkey=error` also errors if a variable is not set but a default would
be set, making it a non-option.

Resolution: __Not Yet Resolved__


### Phase 2: Stage value consistency
Currently, not all fields in a single stage have access to the same values. E.g.
image build information is available in the helm deployer only in `setValueTemplates`
but not other templateable fields. \
As a user, this makes it difficult to remember which values are available to
which fields.

Although one might discuss the sensibility of having some information available in
some fields, the decision what is and what is not sensible in the current
setup should be left to the user.

To implement this, for every component using templating:

1. Centralize value instantiation to once at the start of each component execution.\
   Those values can then be re-used during execution for all templates.
2. Implement instantiation consistency across components.\
   E.g. all deployers should have the same values to template with available to them,
   providing a user the same templating UX switching between different components
   of the same pipeline stage.

This should implementation should only lift current existing restrictions,
allowing the use of values consistently across fields and components of the same
pipeline stage. All in al templating should feel more useful and uniform without
breaking any current existing functionality.

#### Open Issues/Questions

None.

### Phase 3: Downstream value passing
Currently, a pipeline stage downstream from another is responsible for computing
their own values to template with. This is e.g. done from `builds []graph.Artifact`
and especially useful to access build tags for this skaffold execution in
subsequent deploy and render stages.

Having each component responsible to compute those values on their own opens
up the implementation for inconsistencies. Centralizing the computation logic
of available template values can prevent that.

To do so, one can view those values available to a user as an API which should
have a specification. A draft specification can be the following:

```yaml
"$schema": https://json-schema.org/draft/2020-12/schema
type: object
properties:
  
  Env:
    description: |
      All environment variables available to the skaffold process on startup are
      available to the template user under the .Env key.
    type: object
    additionalProperties: 
      type: string

  Config:
    description: |
      The configuration of skaffold contains some global values which are not
      subject to templating. These are accessible to the templating user under
      the .Config key.
    type: object
    properties:
      name:
        description: The name of the skaffold configuration running
        type: String
      labels:
        description: Labels attached to this skaffold configuration
        type: object
        additionalProperties:
          type: string
      annotations:
        description: Annotations attached to this skaffold configuration
        type: object
        additionalProperties:
          type: string
    additionalProperties: false

  Builds:
    description: |
      In all stages after tagging, the information how builds where tagged is
      available to the template user under the .Builds key.
    type: array
    items:
      type: object
      properties:
        imageName:
          description: The name of a build as referred to in the configuration
          type: string
        tag:
          description: The name of the build as tagged by skaffold
          type: string
        additionalProperties: false

# Build information might not be available in all stages
required:
  - Env
  - Config

# TBD, see open questions
additionalProperties: false

```
#### Open Issues/Questions

**\<Do we need less or more values?\>** \
The above values are a proposal and motivated by ideas from looking at the current
source code and what one could do with it. They might miss out on some thing
while being overkill for others. One such open value would be if the current
running profile should be available.

Resolution: __Not Yet Resolved__

**\<Is the schema adequate?\>** \
The shape of available data as proposed above is inspired by accessing Helm
values. This might not be the most intuitive solution here.

Resolution: __Not Yet Resolved__

**\<Where to compute those values?\>** \
Where is the correct place in code to do this value computation? 

Resolution: __Not Yet Resolved__

**\<Does the current available values stay?\>** \
Currently values are available in a different way (e.g. the `IMAGE_REPO_*` format).
This could be kept for backwards compatibility reasons but would introduce additional
maintenance and testing burden. Removing them is definitely a breaking change and
would therefore require a major version change.

Resolution: __Not Yet Resolved__


**\<Value clashing?\>** \
The root property naming convention is adhering to the helm naming convention.
This naming convention is different than the all-caps format usually used by
environment variables and the current variables. This whoever does not guarantee
uniqueness, as, of course, a user can  already have an environment variable
e.g. named `Env`.

Resolution: __Not Yet Resolved__

## Implementation plan

To keep PR size manageable, the implementation should be split into at least 3
parts following the three different design goals. These could be split further
into the sub-list items.

1. Consistent templating UX
   1. Implement `templater` and switch only one component
   2. Switch the rest to the new implementation to make them consistent
2. Stage value consistency
   1. Centralize value instantiation for one component
   2. Switch the rest to the new implementation to make them consistent
3. Downstream value passing
   1. Implement objects (e.g. `.Env`, `.Config`, `.Builds`) one by one instead
      of all of them at once.

## Integration test plan

For phase 1 & 2, functionality should only expand. All currently passing tests
should still pass. New code has to have associated tests with it.
This should ensure we don't brake current functionality as well as make new
code as solid as we can.

For phase 3, centralizing the generation of those values should provide a point
of leverage for testing. This can be done by directly testing the generating
code or using a specification validation setup.
