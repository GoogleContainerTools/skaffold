# Development

## Prerequisites

* [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)
    * macOS: _(built-in)_
    * Windows:
        * `choco install git -y`
        * `git config --global core.autocrlf false`
* [Go](https://golang.org/doc/install)
    * macOS: `brew install go`
    * Windows: `choco install golang -y`
* [Docker](https://www.docker.com/products/docker-desktop)
* Make (and build tools)
    * macOS: `xcode-select --install`
    * Windows:
        * `choco install cygwin make -y`
        * `[Environment]::SetEnvironmentVariable("PATH", "C:\tools\cygwin\bin;$ENV:PATH", "MACHINE")`
        
### Caveats

* The acceptance tests require the docker daemon to be able to communicate with a local containerized insecure registry. On Docker Desktop 3.3.x, this may result in failures such as: `Expected nil: push response: : Get http://localhost:<port>/v2/: dial tcp [::1]:<port>: connect: connection refused`. To fix these failures, it may be necessary to add the following to the Docker Desktop Engine config:
    * macOS: Docker > Preferences > Docker Engine:
```
  "insecure-registries": [
    "<my-host-ip>/32"
  ]
```

### Testing GitHub actions on forks

The lifecycle release process involves chaining a series of GitHub actions together such that:
* The "build" workflow creates the artifacts
  * .tgz files containing the lifecycle binaries, shasums for the .tgz files, a cosign public key, an SBOM, etc.
  * OCI images containing the lifecycle binaries, tagged with their commit sha (for more information, see RELEASE.md)
* The "draft-release" workflow finds the artifacts and downloads them, creating the draft release
* The "post-release" workflow re-tags the OCI images that were created during the "build" workflow with the release version

It can be rather cumbersome to test changes to these workflows, as they are heavily intertwined. Thus we recommend forking the buildpacks/lifecycle repository in GitHub and running through the entire release process end-to-end.
For the fork, it is necessary to add the following secrets:
* COSIGN_PASSWORD (see [cosign](https://github.com/sigstore/cosign#generate-a-keypair))
* COSIGN_PRIVATE_KEY
* DOCKER_PASSWORD (if not using ghcr.io)
* DOCKER_USERNAME (if not using ghcr.io)

The tools/test-fork.sh script can be used to update the source code to reflect the state of the fork. 
It can be invoked like so: `./tools/test-fork.sh <registry repo name> <path to cosign public key>`

## Tasks

To test, build, and package binaries into an archive, simply run:

```bash
$ make all
```
This will create archives at `out/lifecycle-<LIFECYCLE_VERSION>+linux.x86-64.tgz` and `out/lifecycle-<LIFECYCLE_VERSION>+windows.x86-64.tgz`.

`LIFECYCLE_VERSION` defaults to the value returned by `git describe --tags` if not on a release branch (for more information about the release process, see [RELEASE](RELEASE.md)). It can be changed by prepending `LIFECYCLE_VERSION=<some version>` to the
`make` command. For example:

```bash
$ LIFECYCLE_VERSION=1.2.3 make all
```

Steps can also be run individually as shown below.

### Test

Formats, vets, and tests the code.

```bash
$ make test
```

### Build

Builds binaries to `out/linux/lifecycle/` and `out/windows/lifecycle/`.

```bash
$ make build
```

> To clean the `out/` directory, run `make clean`.

### Package

Creates archives at `out/lifecycle-<LIFECYCLE_VERSION>+linux.x86-64.tgz` and `out/lifecycle-<LIFECYCLE_VERSION>+windows.x86-64.tgz`, using the contents of the
`out/linux/lifecycle/` directory, for the given (or default) `LIFECYCLE_VERSION`.

```bash
$ make package
```
