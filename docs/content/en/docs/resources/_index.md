---
title: "Resources"
linkTitle: "Resources"
weight: 130
---

## 2020 Roadmap

* Web UI for Skaffold 
* Skaffold modules for dependency management 
* Extensibility, hooks
* Composability of Skaffold config
* Better tutorials 
* Better GitOps workflows with Skaffold
* Kustomize and Helm support for `skaffold init` 
* Buildpacks support
  
{{< alert title="Note" >}}
The roadmap is subject to change and aspirational but we would like to share our plans with the user and contributor community.
{{< /alert >}}

## 2019 Roadmap

* Plugin model for builders - DONE - see custom artifacts
* IDE integration - VSCode and IntelliJ Skaffold dev/build/run/deploy support, Skaffold Config code completion - DONE, see Cloud Code
* Debugging JVM applications - DONE, we have Java, go, python and node
* Skaffold keeps track of what it built, for faster restarts - DONE, artifact caching is implemented
* Pipeline CRD integration - DONE - we have Tekton pipeline generation

We reprioritized these items: 

* Provide help with integration testing 
* Automated Kubernetes manifest generation
* Infrastructure scaffolding for CI/CD on GCP/GKE
* Document end-to-end solutions
* Status dashboard for build (test) and deployment besides logging

## Contributing

See [Contributing Guide](https://github.com/GoogleContainerTools/skaffold/blob/master/CONTRIBUTING.md),
[Developing Guide](https://github.com/GoogleContainerTools/skaffold/blob/master/DEVELOPMENT.md),
and our [Code of Conduct](https://github.com/GoogleContainerTools/skaffold/blob/master/code-of-conduct.md)
on GitHub.

## Release Notes

See [Release Notes](https://github.com/GoogleContainerTools/skaffold/blob/master/CHANGELOG.md) on Github.

## Community

You can join the Skaffold community and discuss the product at:

* [Skaffold Mailing List](https://groups.google.com/forum#!forum/skaffold-users)
* [Skaffold Topic on Kubernetes Slack](https://kubernetes.slack.com/messages/CABQMSZA6/)
* [Give us feedback](feedback)

The Skaffold Project also holds a bi-weekly meeting at 9:30am PST on Google
Hangouts. Everyone is welcome to add suggestions to the [Meeting Agenda](https://docs.google.com/document/d/1mnCC_fAI3pmg3Vb2nMJyPk8Qtjjuapw_BTyqI_dX7sk/edit)
and [attend the meeting](https://hangouts.google.com/hangouts/_/google.com/skaffold).
If you join the Skaffold Mailing List, a calendar invite will be sent to your Google
Calendar.
