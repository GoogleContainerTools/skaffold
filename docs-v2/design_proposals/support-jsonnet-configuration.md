# Jsonnet Configuration File

* Author(s): Sören Bohn (@S-Bohn)
<!-- * Design Shepherd: -->
* Date: 5 October 2020
* Status: Shelved Temporarily
* Reason: While reviewing the DD, the skaffold team saw few issues e.g. `skaffold fix`. Team acknowledges this is a good feature to have. @S-Bohn and skaffold team could probably pick this up after skaffold modules work is in.

## Background

When using Skaffold the creation of the configuration file sometimes requires a lot of repetition. This can be mitigated partly by using YAML anchors but may still require writing almost identical pieces of code multiple times. 

[Jsonnet](https://jsonnet.org) is a data templating language which extends json by additional constructs like inheritance and functions to make it more expressive and reduce complexity. See [the official site](https://jsonnet.org/articles/comparisons.html) for further comparison with other languages.

Example 1: The use case is a project which consists of multiple very similar but still distinct small services. Each implement the same interface but implements a different external provider. This requires us to write an artifact block for each of them. Every block only differs in the name of the image and a few parameters passed to a common dockerfile. Additionally, for each service, a test entry needs to be created which again only differs in the image name. Adding a new service is a tedious and error-prone task. Using Jsonnet we could define those similar pieces as functions, create a list of services, and use array comprehension to create the artifact list. The result would be much cleaner, easier to read, shorter, and less likely subject to introduce mistakes.

Example 2: We have a project where we currently generate multiple variants of the Kubernetes deployment manifests. We sometimes require to change a small particular option based on operational needs. To avoid creating constantly code changes we generate those different variants beforehand and create multiple (manual) continuous deployment jobs to bring them into production when needed. It is easy and convenient to using Kustomize to create the overlays and Skaffold to render the manifest variations. We create a profile for each of them which creates some line noise because we have to add each variant with only very few differences. Once all variants are written one has to repeat them for the staging environment. With Jsonnet this shrinks to a simple array comprehension.
```
 $ wc skaffold.jsonnet skaffold.yaml
  69  137 1634 skaffold.jsonnet
 289  423 6719 skaffold.yaml
```

## Workarounds
* Write the Skaffold.yaml directly file and repeat the required pieces for every new module. Requires a lot of boring and error-prone manual work.
* Use Jsonnet (or any other template language) to generate the `skaffold.yaml` file. Requires a manual step before using Skaffold.
* Use YAML anchors to define and reuse repeated parts. Already possible and reduces the amount of code, but still requires copy and paste existent of code to introduce new services.

## Design

The proposal is to allow Jsonnet as an allowed language to write the configuration file. Jsonnet returns by default pure JSON. Given that YAML is a superset of JSON the integration can be handled seamlessly during the loading of the configuration file. The parser can be reused without requiring a change.

From a user perspective, the only change is that the configuration file can be written in a different language. No further change in the configuration required. No interface changes required.

### Open Issues/Questions

**Jsonnet allows specification of top-level and external arguments using the command line interface. Is this something Skaffold should support?**

Resolution: __Not Yet Resolved__

**How should Jsonnet library search paths be specified?**

Resolution: __Not Yet Resolved__

**I am unsure about the integration tests. Given that no real change besides a different language dialect is used. Just rewrite the quickstart example in Jsonnet?**

Resolution: __Not Yet Resolved__

## Implementation plan

1. Add basic support by introducing `go-jsonnet` as a new dependency and use it to render Jsonnet configuration files during the load step. Could be made explicit by requiring that the command line option for configuration file (`-f`) points to a file with `.jsonnet` file extension.
2. Allow specification of additional library search paths.
3. Extend the basic support by adding native helper functions: `parseJSON` and `parseYAML`. This would allow further modularization by giving a simple way to use YAML for static settings a extend it for repetitive portions in Jsonnet.
   ```
   local common = std.native('parseYaml')(importstr 'skaffold-common.yaml');
   local artifacts = ...; // list of artifact objects
   common + {
   	build: {artifacts: artifacts}
   }
   ```
4. Maybe: Add support for top-level arguments / external variables.

## Outlook

* [jsonnet-bundler](https://github.com/jsonnet-bundler/jsonnet-bundler) is a package manager for Jsonnet. Native support could be beneficial for reusing components.
* Adding support for [Tanka](https://tanka.dev/) as a native deployer could be nicely fit to define deployment manifests in the same language and benefit from similar language bindings.

## Integration test plan

Please describe what new test cases you are going to consider.
