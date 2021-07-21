---
title: "Resources"
linkTitle: "Resources"
weight: 130
no_list: true
---

## 2020 Roadmap

This now lives [on Github](https://github.com/GoogleContainerTools/skaffold/blob/master/ROADMAP.md).

## 2019 Roadmap

* Plugin model for builders
   * DONE - see custom artifacts
* IDE integration VSCode and IntelliJ Skaffold dev/build/run/deploy support, Skaffold Config code completion
   * DONE, see [Cloud Code](http://cloud.google.com/code)
* Debugging JVM applications 
    * DONE, we have Java, go, python and node for [debugging]({{<relref "/docs/workflows/debug">}})
* Skaffold keeps track of what it built, for faster restarts
    * DONE, artifact caching is enabled by default, can be controlled with the `--cache-artifacts` flag
* Pipeline CRD integration
    * DONE - we have Tekton pipeline generation in alpha, docs to come

In 2019 we also focused a major part of our efforts in fixing bugs, improve our triage, pull request and design processes, created better documentation, and continuously increased test coverage.

We reprioritized these items for next year: 

* Provide help with integration testing
* Automated Kubernetes manifest generation
* Infrastructure scaffolding for CI/CD on GCP/GKE
* Document end-to-end solutions
* Status dashboard for build (test) and deployment besides logging

## Contributing

See [Contributing Guide](https://github.com/GoogleContainerTools/skaffold/blob/main/CONTRIBUTING.md),
[Developing Guide](https://github.com/GoogleContainerTools/skaffold/blob/main/DEVELOPMENT.md),
and our [Code of Conduct](https://github.com/GoogleContainerTools/skaffold/blob/main/code-of-conduct.md)
on GitHub.

## Release Notes

See [Release Notes](https://github.com/GoogleContainerTools/skaffold/blob/main/CHANGELOG.md) on Github.

## Community

You can join the Skaffold community and discuss the project at:

* [Skaffold Mailing List](https://groups.google.com/forum#!forum/skaffold-users)
* [Skaffold Topic on Kubernetes Slack](https://kubernetes.slack.com/messages/CABQMSZA6/)
* [Give us feedback](feedback)

See our roadmap [in GitHub](https://github.com/GoogleContainerTools/skaffold/blob/main/ROADMAP.md).
