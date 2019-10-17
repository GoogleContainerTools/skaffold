---
title: "Global configuration"
linkTitle: "Global configuration"
weight: 50
---

This section discusses global configuration options for Skaffold (`~/.skaffold/config`). These options are saved per user and apply to all Skaffold pipelines.


Some context specific settings can be configured in a global configuration file, defaulting to `~/.skaffold/config`. Options can be configured globally or for specific contexts.

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
