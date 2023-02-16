# ko builder: Hot reloading of the Go binary in dev mode

* Author(s): Halvard Skogsrud (@halvards)
* Design Shepherd: \<skaffold-core-team-member\>
* Date: 2022-09-26
* Status: Draft
* go/skaffold-ko-devmode

## tl;dr

Make Go development on a Kubernetes cluster feel as fast and responsive as on
a developer workstation. Iterate on your code, and let Skaffold and ko rebuild
and reload the running binary, with latencies that are indistinguishable from
local development.

## Proposal

Implement
[auto sync](https://skaffold.dev/docs/filesync/#auto-sync-mode)
for the
[Skaffold ko builder](https://skaffold.dev/docs/builders/ko/) in
[`dev`](https://skaffold.dev/docs/references/cli/#skaffold-dev) mode.

On source code changes, Skaffold rebuilds the Go binary locally using ko, and
copies it directly to the running container. Once copied, a process
supervisor + file watcher
([`watchexec`](https://github.com/watchexec/watchexec)) reloads the
application.

The prototype implementation can reload a hello-world web application in two
seconds on a GKE cluster, and in about one second on a local Minikube cluster.

## Motivation

Local development can be setup to afford fast feedback with rapid rebuilding of
application code. However, when the code needs to run in a Kubernetes cluster,
this feedback loop slows down. Reasons for having to run on a Kubernetes cluster
during development and testing include having to interact with dependent
applications, and with other infrastructure components that are too heavyweight
or cumbersome to run on a developer workstation.

Skaffold has a `dev` mode that rebuilds and reloads the application running on a
Kubernetes cluster. In most situations, this involves:

1. Rebuilding the container image.
2. Pushing the image to a registry.
3. Creating a new pod with the updated image, replacing the existing pod.

These steps involve multiple network hops (developer workstation pushing to
registry, cluster node pulling from the registry) and the overhead of creating a
new image and a new pod. These constraints put a lower bound on the latency of
having the new code running, which in turn impacts an engineer's flow.

Avoiding the cost of creating a new pod can be especially beneficial for some
applications. An example is applications that pre-populate a cache on startup,
either via application logic, or in an `emptyDir` volume using an init
container.

## Background

The Skaffold
[Buildpacks builder](https://skaffold.dev/docs/builders/buildpacks/)
supports
[hot reloading](https://skaffold.dev/docs/filesync/#buildpacks)
with
[Google Cloud Buildpacks](https://github.com/GoogleCloudPlatform/buildpacks).

Skaffold watches for source file changes on the developer workstation and
copies them (as a tarball) to the running container. Another file watcher
(`watchexec`) runs in the container, and when it detects the copied source
files, it
[recompiles and relaunches the binary](https://gist.github.com/halvards/a3c1f9a48adc931a2dcdd9db083350c4).

This Buildpacks builder feature provides a great user experience for Node.js and
JVM development. However, for Go development,
[ko enables faster image builds than Cloud Native Buildpacks](https://cloud.google.com/blog/topics/developers-practitioners/ship-your-go-applications-faster-cloud-run-ko),
and ko doesn't require Docker.

The Buildpacks feature only sends source file changes over the network, but
the binary rebuilds in-container are constrained by the resources available to
the pod.

## Implementation, briefly

User runs `skaffold dev` for an artifact configured to use the ko builder, and
`sync.auto` is `true`:

1.  During the image build, check if `watchexec` exists in the
    [`kodata`](https://ko.build/features/static-assets/) directory (under the
    workspace). If it doesn't, download a release for the  tarball, extract
    the binary to the `kodata` directory (creating it if it doesn't exist).
    Also, add `watchexec` as an entry in a `.gitignore` file in the `kodata`
    directory.

2.  Rewrite the Kubernetes pod spec and specify watchexec as the container
    `command`. The original `command` and `args` will be added to `args`. To
    implement this, repurpose existing manifest rewriting logic from the
    Skaffold debug manifest rewriting implementation.

3.  When a change event takes place for local source code files, determine
    the platform for the rebuilt binary. Use the field from `skaffold.yaml` or
    `--platform` flag if present, if not, default to `linux/<host arch>`. Using
    the host architecture as the default helps Minikube and KinD users.

4.  Set the [`KOCACHE`](https://github.com/ko-build/ko/issues/264)
    environment variable if unset, so that ko builds the binary in a
    deterministic location. A clear contract from ko on this behavior would be
    helpful.

5.  When constructing the
    [ko build options](https://github.com/ko-build/ko/blob/main/pkg/build/options.go),
    ensure that ko doesn't download the base image again. Skaffold is only
    rebuilding the binary, so the base image isn't required. To achieve this,
    provide an
    [empty image](https://github.com/google/go-containerregistry/tree/main/pkg/v1/empty)
    as the ko base image.

6.  Use ko to build the new binary
    [src](https://github.com/ko-build/ko/blob/5e0452ad67230076340d0e28dd8488e4370675c2/pkg/build/gobuild.go#L967).

7.  Use the existing Skaffold sync feature to sync the rebuilt binary from
    the local file `$KOCACHE/bin/<import path>/linux/<arch>/out` to the
    container destination `/ko-app/<base name of import path>`.

## Open questions

1.  Should `sync.auto` default to `true` for the ko builder, as it does for
    the Buildpacks and Jib builders?

    Users must specify a default base image that works with watchexec, and
    the default ko base image doesn't. This means that if we default to
    `true`, `skaffold dev` will fail for ko users who don't specify a
    compatible base image.

    Options:

    -   Default to `true` for consistency with other builders. If
        skaffold dev fails, print an error message suggesting a compatible base
        image (e.g., `gcr.io/distroless/cc:debug`)

    -   Default to `false`, and print a message suggesting to set it to
        `true` when they run `skaffold dev`.

2.  Should the URL used to download `watchexec` be exposed as a field in the
    Skaffold schema?

    This can be helpful for two reasons:

    -   Users can specify a different version of `watchexec`, or the
        musl-based binary instead of the default glibc one.

    -   Some users work in environments with restricted internet access,
        and this would allow them to specify an internal HTTP server as an
        alternative.

    However, an alternative that we can document is to ask users to download
    `watchexec` out-of-band, and place it in the `kodata` directory before
    running `skaffold dev`. Skaffold checks for the presence of the
    `watchexec` binary in this directory and skips the download if it is
    present. This option avoids adding yet another field to the schema.

3.  Should the feature also cover the Docker deployer?

## Alternative implementation steps

1.  Download the `watchexec` binary in the container at startup (using `curl`).
    This avoids polluting the local filesystem with the `watchexec` binary,
    but it adds time to each image build (in `dev` mode). It also requires a
    wrapper script as the entrypoint. The pod requires network access to the
    location where `watchexec` can be downloaded.
