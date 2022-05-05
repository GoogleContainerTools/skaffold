---
title: "Raw YAML"
linkTitle: "Raw YAML"
weight: 20
featureId: render
---

## Rendering with raw YAML

In the case that your project does not currently use a render engine 
(helm, kustomize, kpt, etc), the `rawYaml` renderer should be used.  This instructs
skaffold to only do it's own yaml field replacement (`image:` and `labels:` modifications) and 
not to use any additional render engine.

### Configuration

To use `rawYaml`, add render type `rawYaml` to the `manifests` section of
`skaffold.yaml`.

The `rawYaml` configuration accepts a list of paths to your manifests with glob syntax supported. 

### Example

The following `manifests` section instructs Skaffold to render
artifacts using `rawYaml`.   Each entry should point to YAML manifest file and supports glob syntax:

{{% readfile file="samples/renderers/rawYaml.yaml" %}}