# Skaffold deprecation policy

Skaffold adopts the [Kubernetes deprecation policy for admin facing components](https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-flag-or-cli). In summary, deprecations to a flag or CLI command require the following notification periods, depending on the release track:

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
    1. [Document](./docs) changes in relevant sections. These docs will be
        published to [offical skaffold website](https://skaffold.dev/docs/)
    2. Release notes
    3. Command help
    4. Log messages
    5. https://skaffold.dev/docs/references/yaml/
    6. [deprecation policy](/deprecation-policy.md)

2. if applicable, [from the kubernetes policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-flag-or-cli):
    > Rule #6: Deprecated CLI elements must emit warnings (optionally disable) when used.

# Current maturity of skaffold

## Skaffold.yaml (pipeline config)

The pipeline config, i.e. `skaffold.yaml` is **beta**.

This means that you can safely depend on the skaffold config with the assumption that skaffold will auto-upgrade to the latest version:

- Removal and non-upgradable changes are subject to the deprecation policy for all (even new) features under the config.
- Auto-upgradable changes are not considered breaking changes.

## Skaffold components

We are committed to design for auto-upgradeable changes in the config.
However the **behavior** of individual component might suffer breaking changes depending on maturity.

- Filewatcher: beta
- Builders
  - local: beta
  - googleCloudBuild: beta
  - kaniko: beta
  - plugins gcb: alpha
- Artifact types:
  - Dockerfile: beta
  - Bazel: beta
  - jibMaven: alpha
  - jibGradle: alpha
- Filesync: alpha
- Port-forwarding: alpha
- Taggers: beta
  - gitCommit : beta
  - sha256: beta
  - dateTime : beta
  - envTagger: beta
- Testers: alpha
  - Structure tests: alpha
- Deployers: beta
    - Helm: beta
    - Kustomize: beta
    - Kubectl: beta
- Profiles: beta
- Debug: alpha

## Skaffold commands

Commands and their flags are subject to the deprecation policy based on the following table list:

- build:  beta
- completion:  beta
- config:  alpha
- debug: alpha
- delete:  beta
- deploy:  beta
- dev:  beta
- diagnose:  beta
- fix:  beta
- help:  beta
- init:  alpha
- run:  beta
- version:  beta


## Current deprecation notices


03/15/2019: With release v0.25.0 we mark for deprecation the `flags` field in kaniko (`KanikoArtifact.AdditionalFlags`) , instead Kaniko's additional flags will now be represented as unique fields under `kaniko` per artifact (`KanikoArtifact` type).
This flag will will be removed earliest 06/15/2019.

02/15/2019: With  release v0.23.0 we mark for deprecation the following env variables in the `envTemplate` tagger:
- `DIGEST`
- `DIGEST_ALGO`
- `DIGEST_HEX`
Currently these variables resolve to `_DEPRECATED_<envvar>_`, and the new tagging mechanism adds a digest to the image name thus it shouldn't break existing configurations.
This backward compatibility behavior will be removed earliest 05/14/2019.
