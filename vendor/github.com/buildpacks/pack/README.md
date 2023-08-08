# pack - Buildpack CLI

[![Build results](https://github.com/buildpacks/pack/workflows/build/badge.svg)](https://github.com/buildpacks/pack/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/buildpacks/pack)](https://goreportcard.com/report/github.com/buildpacks/pack)
[![codecov](https://codecov.io/gh/buildpacks/pack/branch/main/graph/badge.svg)](https://codecov.io/gh/buildpacks/pack)
[![GoDoc](https://godoc.org/github.com/buildpacks/pack?status.svg)](https://godoc.org/github.com/buildpacks/pack)
[![GitHub license](https://img.shields.io/github/license/buildpacks/pack)](https://github.com/buildpacks/pack/blob/main/LICENSE)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/4748/badge)](https://bestpractices.coreinfrastructure.org/projects/4748)
[![Slack](https://img.shields.io/badge/slack-join-ff69b4.svg?logo=slack)](https://slack.buildpacks.io/)
[![Gitpod ready-to-code](https://img.shields.io/badge/Gitpod-ready--to--code-blue?logo=gitpod)](https://gitpod.io/#https://github.com/buildpacks/pack)

`pack` makes it easy for...
- [**App Developers**][app-dev] to use buildpacks to convert code into runnable images.
- [**Buildpack Authors**][bp-author] to develop and package buildpacks for distribution.
- [**Operators**][operator] to package buildpacks for distribution and maintain applications.

## Usage

<img src="resources/pack-build.gif" width="600px" />

## Getting Started
Get started by running through our tutorial: [An App’s Brief Journey from Source to Image][getting-started]

## Contributing
- [CONTRIBUTING](CONTRIBUTING.md) - Information on how to contribute, including the pull request process.
- [DEVELOPMENT](DEVELOPMENT.md) - Further detail to help you during the development process.
- [RELEASE](RELEASE.md) - Further details about our release process.

## Documentation
Check out the command line documentation [here][pack-docs]

## Specifications
`pack` is a CLI implementation of the [Platform Interface Specification][platform-spec] for [Cloud Native Buildpacks][buildpacks.io].

To learn more about the details, check out the [specs repository][specs].

[app-dev]: https://buildpacks.io/docs/app-developer-guide/
[bp-author]: https://buildpacks.io/docs/buildpack-author-guide/
[operator]: https://buildpacks.io/docs/operator-guide/
[buildpacks.io]: https://buildpacks.io/
[install-pack]: https://buildpacks.io/docs/install-pack/
[getting-started]: https://buildpacks.io/docs/app-journey
[specs]: https://github.com/buildpacks/spec/
[platform-spec]: https://github.com/buildpacks/spec/blob/main/platform.md
[pack-docs]: https://buildpacks.io/docs/tools/pack/cli/pack/
