---
title: "In cluster build"
linkTitle: "In cluster"
weight: 20
---

Skaffold supports building in cluster via [Kaniko]({{< relref "/docs/pipeline-stages/builders/builder-types/docker#dockerfile-in-cluster-with-kaniko" >}}) 
or [Custom Build Script]({{<relref "/docs/pipeline-stages/builders/builder-types/custom#custom-build-script-in-cluster" >}}).

## Configuration

To configure in-cluster Build, add build type `cluster` to the build section of `skaffold.yaml`. 

```yaml
build:
  cluster: {}
```

The following options can optionally be configured:

{{< schema root="ClusterDetails" >}}

## Faster builds

Skaffold can build multiple artifacts in parallel, by settings a value higher than `1` to `concurrency`.
For in-cluster builds, the default is to build all the artifacts in parallel. If your cluster is too
small, you might want to reduce the `concurrency`. Setting `concurrency` to `1` will cause artifacts to be built sequentially.

{{<alert title="Note">}}
When artifacts are built in parallel, the build logs are still printed in sequence to make them easier to read.
{{</alert>}}
