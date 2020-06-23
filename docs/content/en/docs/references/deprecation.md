---
title: "Deprecation Policy"
linkTitle: "Deprecation Policy"
weight: 60
---

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
    

2. if applicable, [inspired by the Kubernetes policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-flag-or-cli):
    > Rule #6: Deprecated CLI elements must emit warnings (optionally disable) when used.

# Current maturity of skaffold

Skaffold and its features are considered GA unless specified (in this document, CLI reference, config YAML reference or in docs in skaffold.dev).  
Skaffold is constantly evolving with the tools space, we want to be able to experiment and sometimes change things. 
In order to be transparent about the maturity of feature areas and things that might change we offer the feature level maturity matrix that we keep up to date.

## Skaffold.yaml (pipeline config)

You can safely depend on the skaffold config with the assumption that skaffold will auto-upgrade to the latest version:

- Removal and other non-upgradeable changes are subject to the deprecation policy for all (even new) features under the config.
- Auto-upgradeable changes are not considered breaking changes.

## Skaffold features

We are committed to design for auto-upgradeable changes in the config.
However the **behavior** of individual component might suffer breaking changes depending on maturity.

The following is the maturity of the larger feature areas: 

{{< maturity-table >}}

## Exceptions 

No policy can cover every possible situation. 
This policy is a living document, and will evolve over time. 
In practice, there will be situations that do not fit neatly into this policy, or for which this policy becomes a serious impediment. 
Examples could be getting fixes fast for a serious vulnerability, a destructive bug or requirements that might be imposed by third parties (such as legal requirements).
Such situations should be discussed on the given bugs / feature requests and during Skaffold Office Hours, always bearing in mind that Skaffold is committed to being a stable system that, as much as possible, never breaks users. 
Exceptions will always be announced in all relevant release notes.

## Current deprecation notices

10/21/2019: With release v0.41.0 we mark for deprecation the `$IMAGES` environment variable passed to custom builders. Variable `$IMAGE` should be used instead.
This environment variable flag will be removed no earlier than 01/21/2020.

## Past deprecation notices

03/15/2019: With release v0.25.0 we mark for deprecation the `flags` field in kaniko (`KanikoArtifact.AdditionalFlags`) , instead Kaniko's additional flags will now be represented as unique fields under `kaniko` per artifact (`KanikoArtifact` type).
This flag will will be removed earliest 06/15/2019.

02/15/2019: With  release v0.23.0 we mark for deprecation the following env variables in the `envTemplate` tagger:
- `DIGEST`
- `DIGEST_ALGO`
- `DIGEST_HEX`
Currently these variables resolve to `_DEPRECATED_<envvar>_`, and the new tagging mechanism adds a digest to the image name thus it shouldn't break existing configurations.
This backward compatibility behavior will be removed earliest 05/14/2019.
