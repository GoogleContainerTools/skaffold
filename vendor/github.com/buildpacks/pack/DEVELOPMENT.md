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

Alternatively, you can use Gitpod to run pre-configured dev environment in the cloud right from your browser

[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/buildpacks/pack)


### Windows Caveats

* Symlinks - Some of our tests attempt to create symlinks. On Windows, this requires the [permission to be provided](https://stackoverflow.com/a/24353758).

## Tasks

### Building

To build pack:
```
make build
```

This will output the binary to the directory `out/`.

Options:

| ENV_VAR      | Description                                                            | Default |
|--------------|------------------------------------------------------------------------|---------|
| GOCMD        | Change the `go` executable. For example, [richgo][rgo] for testing.    | go      |
| PACK_BIN     | Change the name or location of the binary relative to `out/`.          | pack    |
| PACK_VERSION | Tell `pack` what version to consider itself                            | `dev`   |

[rgo]: https://github.com/kyoh86/richgo

_NOTE: This project uses [go modules](https://github.com/golang/go/wiki/Modules) for dependency management._

### Testing

To run unit and integration tests:
```shell
make unit
```
Test output will be streamed to your terminal and also saved to the file
out/unit

To run acceptance tests:
```shell
make acceptance
```
Test output will be streamed to your terminal and also saved to the file
out/acceptance

Alternately, to run all tests:
```shell
make test
```

To run our full acceptance suite (including cross-compatibility for n-1 `pack` and `lifecycle`):
```shell
make acceptance-all
```

### Tidy

To format the code:
```shell
make format
```

To tidy up the codebase and dependencies:
```shell
make tidy
```

### Verification

To verify formatting and code quality:
```shell
make verify
```

### Prepare for PR

Runs various checks to ensure compliance:
```shell
make prepare-for-pr
```

### Acceptance Tests
Some options users can provide to our acceptance tests are:

| ENV_VAR      | Description                                                            | Default |
|--------------|------------------------------------------------------------------------|---------|
| ACCEPTANCE_SUITE_CONFIG        | A set of configurations for how to run the acceptance tests, describing the version of `pack` used for testing, the version of `pack` used to create the builders used in the test, and the version of `lifecycle` binaries used to test with Github |  `[{"pack": "current", "pack_create_builder": "current", "lifecycle": "default"}]'`     |
| COMPILE_PACK_WITH_VERSION     | Tell `pack` what version to consider itself    | `dev`    |
| GITHUB_TOKEN | A Github Token, used when downloading `pack` and `lifecycle` releases from Github during the test setup | "" |
| LIFECYCLE_IMAGE        | Image reference to be used in untrusted builder workflows    | buildpacksio/lifecycle:<lifecycle version>  |
| LIFECYCLE_PATH        | Path to a `.tgz` file filled with a set of `lifecycle` binaries    | The Github release for the default version of lifecycle in `pack`  |
| PACK_PATH        | Path to a `pack` executable.  | A compiled version of the current branch      |
| PREVIOUS_LIFECYCLE_IMAGE        | Image reference to be used in untrusted builder workflows, used to test compatibility of `pack` with the n-1 version of the `lifecycle`    | buildpacksio/lifecycle:<PREVIOUS_LIFECYCLE_PATH lifecycle version>, buildpacksio/lifecycle:<n-1 lifecycle version>  |
| PREVIOUS_LIFECYCLE_PATH     |  Path to a `.tgz` file filled with a set of `lifecycle` binaries, used to test compatibility of `pack` with the n-1 version of the `lifecycle`    | The Github release for n-1 release of `lifecycle`     |
| PREVIOUS_PACK_FIXTURES_PATH | Path to a set of fixtures, used to override the most up-to-date fixtures, in case of changed functionality  | `acceptance/testdata/pack_previous_fixtures_overrides`   |
| PREVIOUS_PACK_PATH     | Path to a `pack` executable, used to test compatibility with n-1 version of `pack`          | The most recent release from `pack`'s Github release    |
