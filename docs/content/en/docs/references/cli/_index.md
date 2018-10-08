
---
title: "CLI References"
linkTitle: "CLI References"
weight: 110
---

Skaffold command-line interface provides the following commands:


* [`skaffold run`](#skaffold-run) - to build & deploy once
* [`skaffold dev`](#skaffold-dev) - to trigger the watch loop build & deploy workflow with cleanup on exit

* [`skaffold build`](#skaffold-build) - to just build and tag your image(s)
* [`skaffold deploy`](#skaffold-deploy) - to deploy the given image(s) 
* [`skaffold delete`](#skaffold-delete) - to cleanup the deployed artifacts

* [`skaffold init`](#skaffold-init) - to bootstrap skaffold.yaml
* [`skaffold fix`](#skaffold-fix) - to upgrade from 

* [`skaffold help`](#skaffold-help) - print help
* [`skaffold version`](#skaffold-version) - get Skaffold version
* [`skaffold completion`](#skaffold-completion) - setup tab completion for the CLI 
* [`skaffold config`](#skaffold-config) - manage context specific parameters

## Global flags

<table>
    <thead>
        <tr>
            <th>Flag</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td><code>-h, --help</code></td>
            <td>
                Prints the HELP file for the current command.
            </td>
        </tr>
        <tr>
            <td><code>-v, --verbosity LOG-LEVEL</code></td>
            <td>
                Uses a specific log level.
                <p>Available log levels are <code>debug</code>, <code>info</code>, <code>warn</code>, <code>error</code>, <code>fatal</code>, and <code>panic</code>.</p>
                <p>Default value is <code>warn</code>.</p>
            </td>
        </tr>
    </tbody>
<table>

## `skaffold build`

`skaffold build` builds the artifacts without deploying them.

### Usage

`skaffold build [flags]`

### Flags

<table>
    <thead>
        <tr>
            <th>Flag</th>
            <th>Environment variable</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td><code>-f, --filename PATH</code></td>
            <td><code>SKAFFOLD_FILENAME</code></td>
            <td>
                PATH (Filename or URL) to the Skaffold configuration file, <code>skaffold.yaml</code>.
                <p>Default value is `skaffold.yaml`.</p>
            </td>
        </tr>
        <tr>
            <td><code>-o, --output TEMPLATE</code></td>
            <td><code>SKAFFOLD_OUTPUT</code></td>
            <td>
                Formats output with a Go template.
                <p>Default value is <code>{{range .Builds}}{{.ImageName}} -> {{.Tag}}{{end}}</code>.</p>
            </td>
        </tr>
        <tr>
            <td><code>-p, --profile PROFILE</code></td>
            <td><code>SKAFFOLD_PROFILE</code></td>
            <td>
                Activates a Skaffold profile.
            </td>
        </tr>
        <tr>
            <td><code>-q, --quiet</code></td>
            <td><code>SKAFFOLD_PROFILE</code></td>
            <td>
                Enables quite mode. Skaffold will suppress outputs from the builing tool.
            </td>
        </tr>
        <tr>
            <td><code>--toot</code></td>
            <td><code>SKAFFOLD_TOOT</code></td>
            <td>
                Beeps when the building is completed.
            </td>
        </tr>
    </tbody>
<table>

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

`skaffold config` helps you view and modify Kubernetes-related Skaffold
configuration. It has two sub-commands: `list` and `set`. 

{{< alert >}}
**Note** 

**The Skaffold configuration specified here is different from the settings
in the Skaffold configuration file, `skaffold.yaml`**.

With `skaffold config`, you can control which Kubernetes context and default
source repository Skaffold should use.
{{< /alert >}}

`skaffold config list` list all values set in Kubernetes-related Skaffold
configuration.

`skaffold config set` set a value in Kubernetes-related Skaffold
configuration.

{{< alert >}}
**Note** 

By default, `skaffold config` views and modifies Kubernetes-related Skaffold
configuration **in the global scale**.
{{< /alert >}}

### Usage

`skaffold config list [flags]`

`skaffold config set [flags] FIELD VALUE`

There are two fields to list and set:

* `kube-context`: The Kubernetes context Skaffold uses.
* `default-repo`: The default source repository.

### Flags

Below are flags for `skaffold config list`:

<table>
    <thead>
        <tr>
            <th>Flag</th>
            <th>Environment variable</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td><code>-a, --all</code></td>
            <td><code>SKAFFOLD_ALL</code></td>
            <td>
                Show all available Kubernetes contexts.
            </td>
        </tr>
        <tr>
            <td><code>-c, --config PATH</code></td>
            <td><code>SKAFFOLD_CONFIG</code></td>
            <td>
                Path to Kuberentes-related Skaffold configuration.
            </td>
        </tr>
        <tr>
            <td><code>-k, --kube-context CONTEXT</code></td>
            <td><code>SKAFFOLD_KUBE_CONTEXT</code></td>
            <td>
                Lists Kubernetes-related Skaffold configuration in a specific Kuberentes context.
            </td>
        </tr>
    </tbody>
<table>

Below are flags for `skaffold config set`:

<table>
    <thead>
        <tr>
            <th>Flag</th>
            <th>Environment variable</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td><code>-g, -global</code></td>
            <td><code>SKAFFOLD_KUBE_GLOBAL</code></td>
            <td>
                Show all available Kubernetes contexts.
            </td>
        </tr>
        <tr>
            <td><code>-c, --config PATH</code></td>
             <td><code>SKAFFOLD_CONFIG</code></td>
            <td>
                Path to Kuberentes-related Skaffold configuration.
            </td>
        </tr>
        <tr>
            <td><code>-k, --kube-context CONTEXT</code></td>
             <td><code>SKAFFOLD_KUBE_CONTEXT</code></td>
            <td>
                Sets Kubernetes-related Skaffold configuration in a specific Kuberentes context.
            </td>
        </tr>
    </tbody>
<table>

## `skaffold delete`

`skaffold delete` helps delete deployed resources.

### Usage

`skaffold delete [flags]`

### Flags

<table>
    <thead>
        <tr>
            <th>Flag</th>
            <th>Environment variable</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td><code>-f, --filename PATH</code></td>
            <td><code>SKAFFOLD_FILENAME</code></td>
            <td>
                PATH (Filename or URL) to the Skaffold configuration file, <code>skaffold.yaml</code>.
                <p>Default value is `skaffold.yaml`.</p>
            </td>
        </tr>
        <tr>
            <td><code>-p, --profile PROFILE</code></td>
            <td><code>SKAFFOLD_PROFILE</code></td>
            <td>
                Activates a Skaffold profile.
            </td>
        </tr>
        <tr>
            <td><code>--toot</code></td>
            <td><code>SKAFFOLD_TOOT</code></td>
            <td>
                Beeps when the building is completed.
            </td>
        </tr>
    </tbody>
<table>

## `skaffold dev`

`skaffold dev` starts Skaffold in continuous development mode.

### Usage

`skaffold dev [flags]`

### Flags

<table>
    <thead>
        <tr>
            <th>Flag</th>
            <th>Environment variable</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td><code>--cleanup</code></td>
            <td><code>SKAFFOLD_CLEANUP</code></td>
            <td>
                Deletes deployments if the workflow is interrupted.
                <p>Default value is `true`.</p>
            </td>
        </tr>
        <tr>
            <td><code>-f, --filename PATH</code></td>
            <td><code>SKAFFOLD_FILENAME</code></td>
            <td>
                PATH (Filename or URL) to the Skaffold configuration file, <code>skaffold.yaml</code>.
                <p>Default value is `skaffold.yaml`.</p>
            </td>
        </tr>
        <tr>
            <td><code>-n, --namespace NAMESPACE</code></td>
            <td><code>SKAFFOLD_NAMESPACE, SKAFFOLD_DEPLOY_NAMESPACE (deprecated) </code></td>
            <td>
                Run Helm deployments in the specified namespace.
            </td>
        </tr>
        <tr>
            <td><code>-p, --profile PROFILE</code></td>
            <td><code>SKAFFOLD_PROFILE</code></td>
            <td>
                Activates a Skaffold profile.
            </td>
        </tr>
        <tr>
            <td><code>--toot</code></td>
            <td><code>SKAFFOLD_TOOT</code></td>
            <td>
                Beeps when the deployment is completed.
            </td>
        </tr>
        <tr>
            <td><code>-w, --watch-image IMAGES</code></td>
            <td><code>SKAFFOLD_WATCH_IMAGE</code></td>
            <td>
                Watches (monitors) the source code of specific artifacts.
                <p>Use <code>=</code> as the delimiter. For example, <code>--watch-images=/web/Dockerfile.web=gcr.io/web-project/image</code>.</p>
                <p>Default value is to watch (monitor) the source code of all artifacts.</p>
            </td>
        </tr>
    </tbody>
<table>


## `skaffold fix`

`skaffold fix` converts old version of `skaffold` to the latest version.

### Usage

`skaffold fix [flags]`

### Flags

<table>
    <thead>
        <tr>
            <th>Flag</th>
            <th>Environment variable</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td><code>-f, --filename PATH</code></td>
            <td><code>SKAFFOLD_FILENAME</code></td>
            <td>
                PATH (Filename or URL) to the Skaffold configuration file, <code>skaffold.yaml</code>.
                <p>Default value is `skaffold.yaml`.</p>
            </td>
        </tr>
        <tr>
            <td><code>--overwrite</code></td>
            <td><code>SKAFFOLD_OVERWRITE</code></td>
            <td>
                Overwrites the original file.
            </td>
        </tr>
    </tbody>
<table>

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

<table>
    <thead>
        <tr>
            <th>Flag</th>
            <th>Environment variable</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td><code>-f, --file PATH/code></td>
            <td><code>SKAFFOLD_PATH</code></td>
            <td>
                PATH to write the initialized Skaffold configuration.
                <p>Default value is `skaffold.yaml`.</p>
            </td>
        </tr>
        <tr>
            <td><code>-a, --artifacts ARTIFACT-LIST</code></td>
            <td><code>SKAFFOLD_ARTIFACTS</code></td>
            <td>
                Lists of artifacts to build.
                <p>Use <code>=</code> as the delimiter. For example, <code>--artifact=/web/Dockerfile.web=gcr.io/web-project/image</code>.</p>
            </td>
        </tr>
        <tr>
            <td><code>--skip-build</code></td>
            <td><code>SKAFFOLD_SKIP_BUILD</code></td>
            <td>
                Skips generating the list of artifacts.
            </td>
        </tr>
    </tbody>
<table>

## `skaffold run`

`skaffold run` starts Skaffold in standard mode. Skaffold will build and deploy
your application for exactly once.

### Usage

`skaffold run`

### Flags

<table>
    <thead>
        <tr>
            <th>Flag</th>
            <th>Environment variable</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td><code>-f, --filename PATH</code></td>
            <td><code>SKAFFOLD_FILENAME</code></td>
            <td>
                PATH (Filename or URL) to the Skaffold configuration file, <code>skaffold.yaml</code>.
                <p>Default value is `skaffold.yaml`.</p>
            </td>
        </tr>
        <tr>
            <td><code>-n, --namespace NAMESPACE</code></td>            
            <td><code>SKAFFOLD_NAMESPACE, SKAFFOLD_DEPLOY_NAMESPACE (deprecated) </code></td>                                                           
            <td>
                Run Helm deployments in the specified namespace.
            </td>
        </tr>
        <tr>
            <td><code>-p, --profile PROFILE</code></td>
              <td><code>SKAFFOLD_PROFILE</code></td>
            <td>
                Activates a Skaffold profile.
            </td>
        </tr>
        <tr>
            <td><code>--toot</code></td>
            <td><code>SKAFFOLD_TOOT</code></td>
            <td>
                Beeps when the deployment is completed.
            </td>
        </tr>
        <tr>
            <td><code>--tail</code></td>
            <td>
                Streams logs from deployed targets.
            </td>
        </tr>
        <tr>
            <td><code>-t, --tag TAG</code></td>
            <td>
                Uses a custom tag that overrides the tag policy settings in the configuration file.
            </td>
        </tr>
    </tbody>
<table>

## `skaffold version`

`skaffold version` prints the version number of Skaffold.

### Usage

`skaffold version [flags]`

### Flags

<table>
    <thead>
        <tr>
            <th>Flag</th>
            <th>Environment variable</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td><code>-o, --output TEMPLATE</code></td>
            <td><code>SKAFFOLD_OUTPUT</code></td>
            <td>
                Formats output with a Go template.
                <p>Default value is <code>{{.Version}}</code>.</p>
            </td>
        </tr>
    </tbody>
<table>
