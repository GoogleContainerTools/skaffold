# Lifecycle

[![Build Status](https://travis-ci.org/buildpacks/lifecycle.svg?branch=master)](https://travis-ci.org/buildpack/lifecycle)
[![GoDoc](https://godoc.org/github.com/buildpacks/lifecycle?status.svg)](https://godoc.org/github.com/buildpacks/lifecycle)

A reference implementation of [Buildpack API v3](https://github.com/buildpacks/spec).

## Commands

### Build

* `detector` - chooses buildpacks (via `/bin/detect`)
* `analyzer` - restores launch layer metadata from the previous build
* `builder` -  executes buildpacks (via `/bin/build`)
* `exporter` - remotely patches images with new layers (via rebase & append)
* `launcher` - invokes choice of process

### Develop

* `detector` - chooses buildpacks (via `/bin/detect`)
* `developer` - executes buildpacks (via `/bin/develop`)
* `launcher` - invokes choice of process

### Cache

* `restorer` - restores cache
* `cacher` - updates cache

### Rebase

* `rebaser` - remotely patches images with new base image

## Notes

Cache implementations (`restorer` and `cacher`) are intended to be interchangeable and platform-specific.
A platform may choose not to deduplicate cache layers.

## Development
To test, build, and package binaries into an archive, simply run:

```bash
$ make all
```
This will create an archive at `out/lifecycle-<LIFECYCLE_VERSION>+linux.x86-64.tgz`.

By default, `LIFECYCLE_VERSION` is `0.0.0`. It can be changed by prepending `LIFECYCLE_VERSION=<some version>` to the
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

Builds binaries to `out/lifecycle/`.

```bash
$ make build
```

> To clean the `out/` directory, run `make clean`.

### Package

Creates an archive at `out/lifecycle-<LIFECYCLE_VERSION>+linux.x86-64.tgz`, using the contents of the
`out/lifecycle/` directory, for the given (or default) `LIFECYCLE_VERSION`.

```bash
$ make package
```
