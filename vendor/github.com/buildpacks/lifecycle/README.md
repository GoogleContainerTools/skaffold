# Lifecycle

[![Build Status](https://github.com/buildpacks/lifecycle/workflows/build/badge.svg)](https://github.com/buildpacks/lifecycle/actions)
[![GoDoc](https://godoc.org/github.com/buildpacks/lifecycle?status.svg)](https://godoc.org/github.com/buildpacks/lifecycle)
[![codecov](https://codecov.io/gh/buildpacks/lifecycle/branch/main/graph/badge.svg)](https://codecov.io/gh/buildpacks/lifecycle/tree/main)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/4748/badge)](https://bestpractices.coreinfrastructure.org/projects/4748)
 [![Gitpod ready-to-code](https://img.shields.io/badge/Gitpod-ready--to--code-blue?logo=gitpod)](https://gitpod.io/#https://github.com/buildpacks/lifecycle)

A reference implementation of the [Cloud Native Buildpacks specification](https://github.com/buildpacks/spec).

## Supported APIs
| Lifecycle Version | Platform APIs                                                                                                                                    | Buildpack APIs                                                                                                                 |
|-------------------|--------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------|
| 0.17.x*           | [0.3][p/0.3], [0.4][p/0.4], [0.5][p/0.5], [0.6][p/0.6], [0.7][p/0.7], [0.8][p/0.8], [0.9][p/0.9], [0.10][p/0.10], [0.11][p/0.11], [0.12][p/0.12] | [0.2][b/0.2], [0.3][b/0.3], [0.4][b/0.4], [0.5][b/0.5], [0.6][b/0.6], [0.7][b/0.7], [0.8][b/0.8], [0.9][b/0.9], [0.10][b/0.10] |
| 0.16.x            | [0.3][p/0.3], [0.4][p/0.4], [0.5][p/0.5], [0.6][p/0.6], [0.7][p/0.7], [0.8][p/0.8], [0.9][p/0.9], [0.10][p/0.10], [0.11][p/0.11]                 | [0.2][b/0.2], [0.3][b/0.3], [0.4][b/0.4], [0.5][b/0.5], [0.6][b/0.6], [0.7][b/0.7], [0.8][b/0.8], [0.9][b/0.9]                 |
| 0.15.x            | [0.3][p/0.3], [0.4][p/0.4], [0.5][p/0.5], [0.6][p/0.6], [0.7][p/0.7], [0.8][p/0.8], [0.9][p/0.9], [0.10][p/0.10]                                 | [0.2][b/0.2], [0.3][b/0.3], [0.4][b/0.4], [0.5][b/0.5], [0.6][b/0.6], [0.7][b/0.7], [0.8][b/0.8], [0.9][b/0.9]                 |
| 0.14.x            | [0.3][p/0.3], [0.4][p/0.4], [0.5][p/0.5], [0.6][p/0.6], [0.7][p/0.7], [0.8][p/0.8], [0.9][p/0.9]                                                 | [0.2][b/0.2], [0.3][b/0.3], [0.4][b/0.4], [0.5][b/0.5], [0.6][b/0.6], [0.7][b/0.7], [0.8][b/0.8]                               |
| 0.13.x            | [0.3][p/0.3], [0.4][p/0.4], [0.5][p/0.5], [0.6][p/0.6], [0.7][p/0.7], [0.8][p/0.8]                                                               | [0.2][b/0.2], [0.3][b/0.3], [0.4][b/0.4], [0.5][b/0.5], [0.6][b/0.6], [0.7][b/0.7]                                             |
| 0.12.x            | [0.3][p/0.3], [0.4][p/0.4], [0.5][p/0.5], [0.6][p/0.6], [0.7][p/0.7]                                                                             | [0.2][b/0.2], [0.3][b/0.3], [0.4][b/0.4], [0.5][b/0.5], [0.6][b/0.6]                                                           |
| 0.11.x            | [0.3][p/0.3], [0.4][p/0.4], [0.5][p/0.5], [0.6][p/0.6]                                                                                           | [0.2][b/0.2], [0.3][b/0.3], [0.4][b/0.4], [0.5][b/0.5], [0.6][b/0.6]                                                           |
| 0.10.x            | [0.3][p/0.3], [0.4][p/0.4], [0.5][p/0.5]                                                                                                         | [0.2][b/0.2], [0.3][b/0.3], [0.4][b/0.4], [0.5][b/0.5]                                                                         |
| 0.9.x             | [0.3][p/0.3], [0.4][p/0.4]                                                                                                                       | [0.2][b/0.2], [0.3][b/0.3], [0.4][b/0.4]                                                                                       |
| 0.8.x             | [0.3][p/0.3]                                                                                                                                     | [0.2][b/0.2]                                                                                                                   |
| 0.7.x             | [0.2][p/0.2]                                                                                                                                     | [0.2][b/0.2]                                                                                                                   |
| 0.6.x             | [0.2][p/0.2]                                                                                                                                     | [0.2][b/0.2]                                                                                                                   |

[b/0.2]: https://github.com/buildpacks/spec/blob/buildpack/v0.2/buildpack.md
[b/0.3]: https://github.com/buildpacks/spec/tree/buildpack/v0.3/buildpack.md
[b/0.4]: https://github.com/buildpacks/spec/tree/buildpack/v0.4/buildpack.md
[b/0.5]: https://github.com/buildpacks/spec/tree/buildpack/v0.5/buildpack.md
[b/0.6]: https://github.com/buildpacks/spec/tree/buildpack/v0.6/buildpack.md
[b/0.7]: https://github.com/buildpacks/spec/tree/buildpack/v0.7/buildpack.md
[b/0.8]: https://github.com/buildpacks/spec/tree/buildpack/v0.8/buildpack.md
[b/0.9]: https://github.com/buildpacks/spec/tree/buildpack/v0.9/buildpack.md
[b/0.10]: https://github.com/buildpacks/spec/tree/buildpack/v0.10/buildpack.md
[p/0.2]: https://github.com/buildpacks/spec/blob/platform/v0.2/platform.md
[p/0.3]: https://github.com/buildpacks/spec/blob/platform/v0.3/platform.md
[p/0.4]: https://github.com/buildpacks/spec/blob/platform/v0.4/platform.md
[p/0.5]: https://github.com/buildpacks/spec/blob/platform/v0.5/platform.md
[p/0.6]: https://github.com/buildpacks/spec/blob/platform/v0.6/platform.md
[p/0.7]: https://github.com/buildpacks/spec/blob/platform/v0.7/platform.md
[p/0.8]: https://github.com/buildpacks/spec/blob/platform/v0.8/platform.md
[p/0.9]: https://github.com/buildpacks/spec/blob/platform/v0.9/platform.md
[p/0.10]: https://github.com/buildpacks/spec/blob/platform/v0.10/platform.md
[p/0.11]: https://github.com/buildpacks/spec/blob/platform/v0.11/platform.md
[p/0.12]: https://github.com/buildpacks/spec/blob/platform/v0.12/platform.md

\* denotes unreleased version

## Usage

### Build

Either:
* `analyzer` - Reads metadata from the previous image and ensures registry access.
* `detector` - Chooses buildpacks (via `/bin/detect`) and produces a build plan.
* `restorer` - Restores layer metadata from the previous image and from the cache, and restores cached layers.
* `builder` -  Executes buildpacks (via `/bin/build`).
* `exporter` - Creates an image and caches layers.

Or:
* `creator` - Runs the five phases listed above in order.

### Run

* `launcher` - Invokes a chosen process.

### Rebase

* `rebaser` - Creates an image from a previous image with updated base layers.

## Contributing
- [CONTRIBUTING](CONTRIBUTING.md) - Information on how to contribute and grow your understanding of the lifecycle.
- [DEVELOPMENT](DEVELOPMENT.md) - Further detail to help you during the development process.
- [RELEASE](RELEASE.md) - Further details about our release process.
