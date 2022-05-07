---
title: "Lifecycle Hooks"
linkTitle: "Lifecycle Hooks"
weight: 50
featureId: hooks
aliases: [/docs/how-tos/hooks]
---

This page describes how to use the lifecycle hook framework to run code triggered by different events during the skaffold process lifecycle.

## Overview

We identify three distinct phases in skaffold - `build`, `sync` and `deploy`. Skaffold can trigger a hook `before` or `after` executing each phase. There are two types of `hooks` that can be defined - `host` hooks and `container` hooks.

## Host hooks

Host hooks are executed on the runner and can be defined for the following phases:

### `before-build` and `after-build`

Build hooks are executed before and after each artifact is built. 
If an artifact is not built, such as happens when the image was found in the Skaffold image cache, then the build hooks will not be executed.
To force the build hooks, run Skaffold with `--cache-artifacts=false` option.

Example: _skaffold.yaml_ snippet
```yaml
build:
  artifacts:
  - image: hooks-example
    hooks:
      before:
        - command: ["sh", "-c", "./hook.sh"]
          os: [darwin, linux]
        - command: ["cmd.exe", "/C", "hook.bat"]
          os: [windows]
      after:
        - command: ["sh", "-c", "./hook.sh"]
          os: [darwin, linux]
        - command: ["cmd.exe", "/C", "hook.bat"]
          os: [windows]
```
This config snippet defines that `hook.sh` (for `darwin` or `linux` OS) or `hook.bat` (for `windows` OS) will be executed `before` and `after` each build for artifact `hooks-example`.

### `before-sync` and `after-sync`

Example: _skaffold.yaml_ snippet
```yaml
build:
  artifacts:
  - image: hooks-example
    sync: 
      auto: {}
      hooks:
        before:
          - host:
              command: ["sh", "-c", "./hook.sh"]
              os: [darwin, linux]
          - host:
              command: ["cmd.exe", "/C", "hook.bat"]
              os: [windows]
        after:
          - host:
              command: ["sh", "-c", "./hook.sh"]
              os: [darwin, linux]
          - host:
              command: ["cmd.exe", "/C", "hook.bat"]
              os: [windows]
```
This config snippet defines that `hook.sh` (for `darwin` or `linux` OS) or `hook.bat` (for `windows` OS) will be executed `before` and `after` each file sync operation for artifact `hooks-example`.

### `before-deploy` and `after-deploy`

Example: _skaffold.yaml_ snippet
```yaml
deploy:
  kubectl:
    manifests:
      - deployment.yaml
    hooks:
      before:
        - host:
            command: ["sh", "-c", "echo pre-deploy host hook running on $(hostname)!"]
            os: [darwin, linux]
      after:
        - host:
            command: ["sh", "-c", "echo post-deploy host hook running on $(hostname)!"]
```
This config snippet defines a simple `echo` command to run before and after each `kubectl` deploy.

### Environment variables

The following environment variables will be available for the corresponding phase host hooks, that can be resolved in both inline commands or scripts.

Environment variable | Description | Availability
-- | -- | --
$SKAFFOLD_IMAGE | The fully qualified image name. For example, “gcr.io/image1:tag” | Build, Sync
$SKAFFOLD_PUSH_IMAGE | Set to true if the image in $IMAGE is expected to exist in a remote registry. Set to false if the image is expected to exist locally. | Build
$SKAFFOLD_IMAGE_REPO | The image repo. For example, “gcr.io/image1” | Build
$SKAFFOLD_IMAGE_TAG | The image tag. For example, “tag” | Build
$SKAFFOLD_BUILD_CONTEXT | An absolute path to the directory this artifact is meant to be built from. Specified by artifact context in the skaffold.yaml. | Build
$SKAFFOLD_FILES_ADDED_OR_MODIFIED | Semi-colon delimited list of absolute path to all files synced or to be synced in current dev loop that have been added or modified | Sync
$SKAFFOLD_FILES_DELETED | Semi-colon delimited list of absolute path to all files synced or to be synced in current dev loop that have been deleted | Sync
$SKAFFOLD_RUN_ID | Run specific UUID label for deployed or to be deployed resources | Deploy
$SKAFFOLD_DEFAULT_REPO | The resolved default repository | All
$SKAFFOLD_RPC_PORT | TCP port to expose event API | All
$SKAFFOLD_HTTP_PORT | TCP port to expose event REST API over HTTP | All
$SKAFFOLD_KUBE_CONTEXT | The resolved Kubernetes context | Sync, Deploy
$SKAFFOLD_MULTI_LEVEL_REPO | The multi-level support of the repository | All
$SKAFFOLD_NAMESPACES | Comma separated list of Kubernetes namespaces | Sync, Deploy
$SKAFFOLD_WORK_DIR | The workspace root directory | All
Local environment variables | The current state of the local environment (e.g. $HOST, $PATH). Determined by the golang os.Environ function. | All

## Container hooks
Container hooks are executed on a target container and can be defined on the following phases:

### `before-sync` and `after-sync`

Example: _skaffold.yaml_ snippet
```yaml
build:
  artifacts:
  - image: hooks-example
    sync: 
      auto: {}
      hooks:
        before:
          - container:
              command: ["sh", "-c", "echo before sync hook"]
        after:
          - container:
              command: ["sh", "-c", "echo after sync hook"]
```
This config snippet defines a command to run inside the container corresponding to the artifact `hooks-example` image, `before` and `after` each file sync operation.

### `before-deploy` and `after-deploy`

Example: _skaffold.yaml_ snippet
```yaml
deploy:
  kubectl:
    manifests:
      - deployment.yaml
    hooks:
      before:
        - container:
            # this will only run when there's a matching container from a previous deploy iteration like in `skaffold dev` 
            command: ["sh", "-c", "echo pre-deploy container hook running on $(hostname)!"]
            containerName: hooks-example*
            podName: hooks-example-deployment*
      after:
        - container:
            command: ["sh", "-c", "echo post-deploy container hook running on $(hostname)!"]
            containerName: hooks-example* # use a glob pattern to prefix-match the container name and pod name for deployments, stateful-sets, etc.
            podName: hooks-example-deployment*
```
This config snippet defines a simple `echo` command to run inside the containers that match `podName` and `containerName`, before and after each `kubectl` deploy. The `after` container commands are only run after the [deployment status checks]({{< relref "/docs/pipeline-stages/status-check" >}}) on the deployment are complete. Also, unlike the `sync` container hooks, skaffold cannot determine the target container from just the config definition, and needs the `podName` and `containerName`.
