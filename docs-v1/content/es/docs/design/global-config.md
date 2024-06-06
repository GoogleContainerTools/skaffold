---
title: "Global Configuration"
linkTitle: "Global Configuration"
weight: 50
featureId: global\_config

---

Some context specific settings can be configured in a global configuration file, which defaults to `~/.skaffold/config`. Options can be configured globally or for specific Kubernetes contexts. Context name matching supports regex, e.g.: `.*-cluster.*-regex.*-test.*`

The options are:

| Option | Type | Description |
| ------ | ---- | ----------- |
| `default-repo` | string | The image registry where built artifact images are published (see [image name rewriting]({{< relref "/docs/environment/image-registries.md" >}})). |
| `multi-level-repo` | boolean | If true, do not replace '.' and '/' with '\_' in image name. |
| `debug-helpers-registry` | string | The image registry where debug support images are retrieved (see [debugging]({{< relref "/docs/workflows/debug.md" >}})). |
| `insecure-registries` | list of strings | A list of image registries that may be accessed without TLS. |
| `k3d-disable-load` | boolean | If true, do not use `k3d import image` to load images locally. |
| `kind-disable-load` | boolean | If true, do not use `kind load` to load images locally. |
| `local-cluster` | boolean | If true, do not try to push images after building. By default, contexts with names `docker-for-desktop`, `docker-desktop`, or `minikube` are treated as local. |

For example, to treat any context as local by default:

```bash
skaffold config set --global local-cluster true
```
This will create a global configuration file at `~/.skaffold/config` with `local-cluster` set to `true`.

{{% readfile file="samples/config/globalConfig.yaml" %}}
