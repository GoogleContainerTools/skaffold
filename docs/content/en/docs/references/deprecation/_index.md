---
title: "Deprecation Policy"
linkTitle: "Deprecation policy"
weight: 300
---

# Skaffold deprecation policy

This document sets out the deprecation policy for Skaffold, and outlines how the Skaffold project will approach the introduction of breaking changes over time.

Deprecation policy applies only to Stable Builds. Bleeding Edge builds may have less stable implementations.  

Deprecations to a flag or CLI command require the following notification periods, depending on the release track:

| Release Track | Deprecation Period |
| -------- | -------- |
| Alpha (experimental)    |0 releases     |
| Beta (pre-release) | 3 months or 1 release (whichever is longer)|
| GA (generally available)   | 6 months or 1 release (whichever is longer) |

**Breaking changes**
A breaking change is when the primary functionality of a feature changes in a way that the user has to make changes to their workflows/configuration.
- **Breaking config change**:  In case of Skaffold's pipeline config (skaffold.yaml) a breaking change between an old and new version occurs when the skaffold binary cannot parse the input yaml with auto-upgrade. This can happen when the new version removes a feature or when the new version introduces a mandatory field with no default value
- **Breaking functional change**: functional changes that force user workflow changes even when the config is the same or upgradeable.

## How do we deprecate things?

A "deprecation event" would coincide with a release.

1. We document the deprecation in the following places if applicable
    1. deprecation policy - this document 
    1. [Document on this site]({{< relref "/docs" >}}) changes in relevant sections
    1. [Release notes](https://github.com/GoogleContainerTools/skaffold/blob/master/CHANGELOG.md)
    1. [Command help]({{< relref "/docs/references/cli" >}})
    1. Log messages
    1. [skaffold yaml reference]({{< relref "/docs/references/yaml" >}})
    

2. if applicable, [inspired by the kubernetes policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-flag-or-cli):
    > Rule #6: Deprecated CLI elements must emit warnings (optionally disable) when used.

# Current maturity of skaffold
Skaffold and its features are considered Beta unless specified (in this document, CLI reference, config YAML reference or in docs in skaffold.dev).  
Skaffold is constantly evolving with the tools space, we want to be able to experiment and sometimes change things. 
In order to be transparent about the maturity of feature areas and things that might change we offer the feature level maturity matrix that we keep up to date.

## Skaffold.yaml (pipeline config)

The pipeline config, i.e. `skaffold.yaml` is **beta**.

This means that you can safely depend on the skaffold config with the assumption that skaffold will auto-upgrade to the latest version:

- Removal and non-upgradable changes are subject to the deprecation policy for all (even new) features under the config.
- Auto-upgradable changes are not considered breaking changes.

## Skaffold features

We are committed to design for auto-upgradeable changes in the config.
However the **behavior** of individual component might suffer breaking changes depending on maturity.

The following is the maturity of the larger feature areas: 

|area|state|description|
|----|----|----|
[Build]({{< relref "/docs/how-tos/builders" >}})|beta |Build images based on multiple build tools in a configurable way
Control API |alpha|Applications can control sync, build and deployment during instead of automated sync, build and deploy
[Debug]({{< relref "/docs/how-tos/debug" >}})|alpha|Language-aware reconfiguration of containers on the fly to become debuggable
[Default-repo]({{< relref "/docs/concepts/image_repositories" >}})|alpha|specify a default image repository & rewrite image names to default repo
Delete|beta |delete everything deployed by skaffold run from the cluster
[Deploy] ({{< relref "docs/how-tos/deployers" >}})|beta |Deploy a set of deployables as your applications and replace the image name with the built images
Dev|beta |Continuous development
Diagnose|beta |Diagnose the current project and its configuration
Event API v1|alpha|Publish events and state of the application on gRPC and HTTP
[Filesync]({{< relref "/docs/how-tos/filesync" >}})|alpha|Instead of rebuilding, copy the changed files in the running container
[Global config]({{< relref "/docs/concepts/config" >}})|alpha|store user preferences in a separate preferences file
Init|alpha|Initialize a skaffold.yaml file based on the contents of the current directory
Insecure registry handling|alpha |Target registries for built images which are not secure
[Port-forwarding]({{< relref "/docs/how-tos/portforward" >}})|alpha |Port forward application to localhost
[Profiles]({{< relref "/docs/how-tos/profiles" >}})|beta |Create different pipeline configurations based on overrides and patches defined in one or more profiles
skaffold build |beta |run skaffold build separately
skaffold fix|beta |Upgrade an older skaffold config to the current version
skaffold run|beta |One-off build & deployment of the skaffold application
[Tagpolicy]({{< relref "/docs/how-tos/taggers" >}})|beta |Automated tagging
[Test]({{< relref "/docs/how-tos/testers" >}})|alpha |Run tests as part of your pipeline
Trigger|alpha |Feature area: Trigger configured actions when source files change
version|beta|get the version string of the current skaffold binary
[Templating]({{< relref "/docs/how-tos/templating" >}})|alpha|certain fields of skaffold.yaml can be parametrized with environment and built-in variables

Within a feature area we do have certain features that are expected to change: 

|area|feature|state|description|
|----|----|----|----|
Debug|debug python apps|alpha|debug python apps
Debug|debug node apps|alpha|debug node apps
Debug|debug java apps|alpha|debug java apps
Default-repo|preconcatentation strategy|beta|collision free rewriting strategy
Tagpolicy|latest tagger|alpha|tag with latest, use image digest / image ID for deployment
Tagpolicy|contentDigest tagger|alpha|reintroduce DIGEST and content based digest tag
Filesync|sync.infer|alpha|mark files as "syncable" - infer the destinations based on the Dockerfile
Init|json based |alpha|skaffold init JSON based API for IDE integrations
Init|interactive|alpha|skaffold init interactive for CLI users
Init|init for k8s manifests|alpha|skaffold init recognizes k8s manifest and the image names in them
Init|init for Dockerfiles |alpha|skaffold init recognizes Dockerfiles


## Current deprecation notices

No active deprecation notices.

## Past deprecation notices

03/15/2019: With release v0.25.0 we mark for deprecation the `flags` field in kaniko (`KanikoArtifact.AdditionalFlags`) , instead Kaniko's additional flags will now be represented as unique fields under `kaniko` per artifact (`KanikoArtifact` type).
This flag will will be removed earliest 06/15/2019.

02/15/2019: With  release v0.23.0 we mark for deprecation the following env variables in the `envTemplate` tagger:
- `DIGEST`
- `DIGEST_ALGO`
- `DIGEST_HEX`
Currently these variables resolve to `_DEPRECATED_<envvar>_`, and the new tagging mechanism adds a digest to the image name thus it shouldn't break existing configurations.
This backward compatibility behavior will be removed earliest 05/14/2019.