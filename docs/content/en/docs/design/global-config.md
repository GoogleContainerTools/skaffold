---
title: "Global Configuration"
linkTitle: "Global Configuration"
weight: 50
featureId: global_config

---

Some context specific settings can be configured in a global configuration file, which defaults to `~/.skaffold/config`. Options can be configured globally or for specific Kubernetes contexts. Context name matching supports regex, e.g.: `.*-cluster.*-regex.*-test.*`

The options are:

| Option | Type | Description |
| ------ | ---- | ----------- |
| `default-repo` | string | The image registry where images are published (See below). |
| `insecure-registries` | list of strings | A list of image registries that may be accesses without TLS. |
| `local-cluster` | boolean | If true, do not try to push images after building. By default, contexts with names `docker-for-desktop`, `docker-desktop`, or `minikube` are treated as local. |

For example, to treat any context as local by default:

```bash
skaffold config set --global local-cluster true
```
This will create a global configuration file at `~/.skaffold/config` with `local-cluster` set to `true`.

{{% readfile file="samples/config/globalConfig.yaml" %}}
