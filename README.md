<!-- github does not support `width` with markdown images-->
<img src="logo/skaffold.png" width="220">

---------------------

[![Code Coverage](https://codecov.io/gh/GoogleContainerTools/skaffold/branch/main/graph/badge.svg)](https://codecov.io/gh/GoogleContainerTools/skaffold)
[![LICENSE](https://img.shields.io/github/license/GoogleContainerTools/skaffold.svg)](https://github.com/GoogleContainerTools/skaffold/blob/main/LICENSE)
[![Releases](https://img.shields.io/github/release-pre/GoogleContainerTools/skaffold.svg)](https://github.com/GoogleContainerTools/skaffold/releases)

Skaffold is a command line tool that facilitates continuous development for
Kubernetes applications. You can iterate on your application source code
locally then deploy to local or remote Kubernetes clusters. Skaffold handles
the workflow for building, pushing and deploying your application. It also
provides building blocks and describe customizations for a CI/CD pipeline.

---------------------

## [Install Skaffold](https://skaffold.dev/docs/install/)

Or, check out our [Github Releases](https://github.com/GoogleContainerTools/skaffold/releases) page for release info or to install a specific version.

![Demo](docs/static/images/intro.gif)

## Features

* Blazing fast local development
  * **optimized source-to-deploy** - Skaffold detects changes in your source code and handles the pipeline to
  **build**, **push**, and **deploy** your application automatically with **policy based image tagging**
  * **continuous feedback** - Skaffold automatically aggregates logs from deployed resources and forwards container ports to your local machine
* Project portability
  * **share with other developers** - Skaffold is the easiest way to **share your project** with the world: `git clone` and `skaffold run`
  * **context aware** - use Skaffold profiles, user level config, environment variables and flags to describe differences in environments
  * **CI/CD building blocks** - use `skaffold run` end-to-end, or use individual Skaffold phases to build up your CI/CD pipeline. `skaffold render` outputs hydrated Kubernetes manifests that can be used in GitOps workflows.
* Pluggable, declarative configuration for your project
  * **skaffold init** - Skaffold discovers your files and generates its own config file
  * **multi-component apps** - Skaffold supports applications consisting of multiple components
  * **bring your own tools** - Skaffold has a pluggable architecture to integrate with any build or deploy tool
* Lightweight
  * **client-side only** - Skaffold has no cluster-side component, so there is no overhead or maintenance burden
  * **minimal pipeline** - Skaffold provides an opinionated, minimal pipeline to keep things simple

### Check out our [examples page](./examples) for more complex workflows!

## IDE integrations

For a managed experience of Skaffold you can install the Google `Cloud Code` extensions:
- for [Visual Studio Code](https://cloud.google.com/code/docs/vscode/quickstart-k8s#installing)
- for [JetBrains IDEs](https://cloud.google.com/code/docs/intellij/quickstart-k8s#installing_the_plugin). 

It can manage and keep Skaffold  up-to-date while providing a more guided startup experience, along with providing and managing other common dependencies, and works with any kubernetes cluster. 

## Contributing to Skaffold

We welcome any contributions from the community with open arms - Skaffold wouldn't be where it is today without contributions from the community! Have a look at our [contribution guide](./CONTRIBUTING.md) for more information on how to get started on sending your first PR.

## Community

* [#skaffold on Kubernetes Slack](https://kubernetes.slack.com/messages/CABQMSZA6/)
* [skaffold-users mailing list](https://groups.google.com/forum/#!forum/skaffold-users)

## Support 

Skaffold is generally available and considered production ready.
Detailed feature maturity information and how we deprecate features are described in our [Deprecation Policy](https://skaffold.dev/docs/references/deprecation).

## Security Disclosures

Please see our [security disclosure process](SECURITY.md).  All [security advisories](https://github.com/GoogleContainerTools/skaffold/security/advisories) are managed on Github.
