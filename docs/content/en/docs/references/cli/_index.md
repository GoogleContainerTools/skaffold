
---
title: "CLI References"
linkTitle: "CLI References"
weight: 110
---

Skaffold command-line interface provides the following commands:


* [skaffold run](#skaffold-run) - to build & deploy once
* [skaffold dev](#skaffold-dev) - to trigger the watch loop build & deploy workflow with cleanup on exit

* [skaffold build](#skaffold-build) - to just build and tag your image(s)
* [skaffold deploy](#skaffold-deploy) - to deploy the given image(s) 
* [skaffold delete](#skaffold-delete) - to cleanup the deployed artifacts

* [skaffold init](#skaffold-init) - to bootstrap skaffold.yaml
* [skaffold fix](#skaffold-fix) - to upgrade from 

* [skaffold help](#skaffold-help) - print help
* [skaffold version](#skaffold-version) - get Skaffold version
* [skaffold completion](#skaffold-completion) - setup tab completion for the CLI 
* [skaffold config](#skaffold-config) - manage context specific parameters


## Global flags

| Flag | Description |
|------- |---------------|
|`-h, --help`| Prints the HELP file for the current command.|
|`-v, --verbosity LOG-LEVEL` | Uses a specific log level. Available log levels are `info`, `warn`, `error`, `fatal`. Default value is `warn`.|

## Global environment variables

| Flag | Description |
|------- |---------------|
|`SKAFFOLD_UPDATE_CHECK`|Enables checking for latest version of the skaffold binary. By default it's `true`. |       
    

## Skaffold commands

## `skaffold build`

`skaffold build` builds the artifacts without deploying them.

### Usage

`skaffold build [flags]`

### Flags

| Flag | Environment variable | Description |
|------- |---------------| ---- |
|`-f, --filename PATH`| `SKAFFOLD_FILENAME` |  PATH (Filename or URL) to the Skaffold configuration file, `skaffold.yaml`. Default value is `skaffold.yaml`. |
|`-o, --output TEMPLATE`|`SKAFFOLD_OUTPUT`| Formats output with a Go template. Default value is `{{range .Builds}}{{.ImageName}} -> {{.Tag}}{{end}}`. |
|`-p, --profile PROFILE`|`SKAFFOLD_PROFILE`|Activates a Skaffold profile.|
|`-q, --quiet`|`SKAFFOLD_PROFILE`|Enables quite mode. Skaffold will suppress outputs from the builing tool.|
|`--toot`|`SKAFFOLD_TOOT`|Beeps when the building is completed.|

## `skaffold completion`

`skaffold completion` prints a `bash` script that, after running, enables
shell completion for the `skaffold` command.

You can run this script with `eval "$(skaffold completion bash)"`.

Too run this script automatically at the time of startup for all of your
sessions, add the following line to the end of `~/.bashrc` or `~/.bash_profile`:

`eval "$(skaffold completion bash)"`

### Usage

`skaffold completion`

### Flags

No flags available.

## `skaffold config`

{{% todo 1060 "this needs more work" %}}

`skaffold config` helps you view and modify _contextual_ Skaffold configuration.
It has two sub-commands: `list` and `set`.
There is one global configuration that is always applicable. 
This global config can be overriden for each kubecontext.  

{{< alert title="Note" >}} 

<b>The Skaffold configuration specified here is different from the settings
in the Skaffold configuration file, `skaffold.yaml`</b>.

With `skaffold config`, you can control context specific things, for example the default image repo to push to.
{{< /alert >}}

`skaffold config list` list values in the contextual configurations

`skaffold config set <key> <value>` set a `key` to `value` in the contextual config 


{{< alert title="Note" >}} 

By default, `skaffold config` views and modifies Kubernetes-related Skaffold
configuration <b>in the global scale</b>.
{{< /alert >}}

### Usage

`skaffold config list [flags]`

`skaffold config set [flags] FIELD VALUE`

There are two fields to list and set:

* `kube-context`: The Kubernetes context Skaffold uses.
* `default-repo`: The default source repository.

### Flags

Below are flags for `skaffold config list`:

| Flag | Environment variable | Description |
|------- |---------------| ---- |
|`-a, --all`|`SKAFFOLD_ALL`|Show all available Kubernetes contexts.|
|`-c, --config PATH`|`SKAFFOLD_CONFIG`|Path to Kuberentes-related Skaffold configuration.|
|`-k, --kube-context CONTEXT`|`SKAFFOLD_KUBE_CONTEXT`|Lists Kubernetes-related Skaffold configuration in a specific Kuberentes context.|

Below are flags for `skaffold config set`:

| Flag | Environment variable | Description |
|------- |---------------| ---- |
|`-g, -global`|`SKAFFOLD_KUBE_GLOBAL`|Show all available Kubernetes contexts.|
|`-c, --config PATH`|`SKAFFOLD_CONFIG`|Path to Kuberentes-related Skaffold configuration.|
|`-k, --kube-context CONTEXT`|`SKAFFOLD_KUBE_CONTEXT`|Sets Kubernetes-related Skaffold configuration in a specific Kuberentes context.|

## `skaffold delete`

`skaffold delete` helps delete deployed resources.

### Usage

`skaffold delete [flags]`

### Flags

| Flag | Environment variable | Description |
|------- |---------------| ---- |
|`-f, --filename PATH`|`SKAFFOLD_FILENAME`| PATH (Filename or URL) to the Skaffold configuration file, `skaffold.yaml`. Default value is `skaffold.yaml`.  |
|`-p, --profile PROFILE`|`SKAFFOLD_PROFILE`|Activates a Skaffold profile.|
|`--toot`|`SKAFFOLD_TOOT`| Beeps when the building is completed.|

## `skaffold dev`

`skaffold dev` starts Skaffold in continuous development mode.

### Usage

`skaffold dev [flags]`

### Flags

| Flag | Environment variable | Description |
|------- |---------------| ---- |
|`--cleanup`|`SKAFFOLD_CLEANUP`| Deletes deployments if the workflow is interrupted. Default value is `true`. |
|`-f, --filename PATH`|`SKAFFOLD_FILENAME`| PATH (Filename or URL) to the Skaffold configuration file, `skaffold.yaml`. Default value is `skaffold.yaml`. |
|`-n, --namespace NAMESPACE`|`SKAFFOLD_NAMESPACE, SKAFFOLD_DEPLOY_NAMESPACE (deprecated) `|Run Helm deployments in the specified namespace.|
|`-p, --profile PROFILE`|`SKAFFOLD_PROFILE`|Activates a Skaffold profile.|
|`--toot`|`SKAFFOLD_TOOT`|Beeps when the deployment is completed.|
|`-w, --watch-image IMAGES`|`SKAFFOLD_WATCH_IMAGE`| Watches (monitors) the source code of specific artifacts. Use `=` as the delimiter. For example, <p>`--watch-images /web/Dockerfile.web=gcr.io/web-project/image`.<p> Default value is to watch (monitor) the source code of all artifacts.|


## `skaffold fix`

`skaffold fix` converts old version of `skaffold` to the latest version.

### Usage

`skaffold fix [flags]`

### Flags

| Flag | Environment variable | Description |
|------- |---------------| ---- |
|`-f, --filename PATH`|`SKAFFOLD_FILENAME`|  PATH (Filename or URL) to the Skaffold configuration file, `skaffold.yaml`. Default value is `skaffold.yaml`. |
|`--overwrite`|`SKAFFOLD_OVERWRITE`|Overwrites the original file.|

## `skaffold help`

`skaffold help` prints the HELP file for `skaffold` commands.

### Usage

`skaffold help COMMAND`

### Flags

No flags available.

## `skaffold init`

`skaffold init` initializes Skaffold configuration.

### Usage

`skaffold init [flags]`

### Flags

| Flag | Environment variable | Description |
|------- |---------------| ---- |
|`-f, --file PATH/code>|`SKAFFOLD_PATH`| PATH to write the initialized Skaffold configuration. Default value is `skaffold.yaml`. |
|`-a, --artifacts ARTIFACT-LIST`|`SKAFFOLD_ARTIFACTS`| Lists of artifacts to build. Use `=` as the delimiter. For example, `--artifact=/web/Dockerfile.web=gcr.io/web-project/image`. |
|`--skip-build`|`SKAFFOLD_SKIP_BUILD`|Skips generating the list of artifacts.|

## `skaffold run`

`skaffold run` starts Skaffold in standard mode. Skaffold will build and deploy
your application for exactly once.

### Usage

`skaffold run`

### Flags

| Flag | Environment variable | Description |
|------- |---------------| ---- |
|`-f, --filename PATH`|`SKAFFOLD_FILENAME`| PATH (Filename or URL) to the Skaffold configuration file, `skaffold.yaml`. Default value is `skaffold.yaml`. | 
|`-n, --namespace NAMESPACE`|`SKAFFOLD_NAMESPACE, SKAFFOLD_DEPLOY_NAMESPACE (deprecated) `| Run Helm deployments in the specified namespace.|
|`-p, --profile PROFILE`|`SKAFFOLD_PROFILE`|Activates a Skaffold profile.|
|`--toot`|`SKAFFOLD_TOOT`|Beeps when the deployment is completed.|
|`--tail`| Streams logs from deployed targets. |
|`-t, --tag TAG`| Uses a custom tag that overrides the tag policy settings in the configuration file.|

## `skaffold version`

`skaffold version` prints the version number of Skaffold.

### Usage

`skaffold version [flags]`

### Flags

| Flag | Environment variable | Description |
|------- |---------------| ---- |
|`-o, --output TEMPLATE`|`SKAFFOLD_OUTPUT`| Formats output with a Go template. Default value is `{{.Version}}`. |
        
