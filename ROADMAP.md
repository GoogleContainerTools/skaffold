# 2020 Roadmap
Last updated: 04/10/2020

**What this roadmap is**

A list of features and concepts that we’d love to see in Skaffold over the next year. Some of these are motivated by direct community support, and some of them are things that we’re personally excited about and feel would enhance the experience of using Skaffold.

**What this roadmap is not**

A list of promises, or a hard timeline commitment. Some items on this list are still under consideration, and may be removed from the roadmap if we decide that they're not worth the engineering investment. At the time of writing this, though, we feel good about everything on this list and want to start a public conversation about it, so that we can use the community’s voice to help us drive the direction of Skaffold.

**Why we’re sharing this**

Feedback! We want to hear from each and every one of you: what are you personally excited about in Skaffold? What are the features that you most want to see? What would make your life easier? This is meant to be a public, living roadmap, where anyone can voice their opinions and we can get a true idea of where the broadest community support lies.

## New Features

### Helm 3 Support - P0
Helm continues to be the definitive Kubernetes package manager, and Skaffold will always provide a first-class experience for developing applications with Helm. With the release of Helm 3, we’ll make sure that we make the transition from Helm 2 to Helm 3 straightforward for developers, and also ensure that we’re always backwards compatible between the two versions.

### Project Modules - P0
Currently, image artifacts in a project are handled by Skaffold individually. We'll add support for grouping artifacts into "modules", which will allow for much better dependency management in multi-artifact projects, and give the flexibility needed to more accurately translate existing projects into a Skaffold pipeline. This will allow for
* Using an artifact as a base image for another artifact
* Iterating on only one service, while deploying all other services in a project with a predefined tag
* Debugging individual services without redeploying all other services in a project

### Extensibility Hooks - P1
We plan to provide pre- and post-stage hooks in Skaffold. This will allow users to extend the different stages of the Skaffold pipeline past their normal lifecycle, to provide more flexibility and extensibility in Skaffold workflows.

### Buildpacks - P1
CNCF Buildpacks allow for zero-configuration container image builds. We're already working to provide first-class support for buildpacks in Skaffold, as we continue to ensure that Skaffold provides the best source-to-deployment development experience.

### Kubernetes Manifest Generation - P1
Another very important part of the source-to-deploy getting started experience in particular is removing the configuration burden from the user. We'll add simple Kubernetes manifest generation into `skaffold init`, giving users a pathway through Skaffold to directly migrate their existing applications over to Kubernetes.

### Private Registry Support - P2
We recognize that while many projects are set up to push images to privately hosted registries with a variety of configurations, Skaffold does not always provide the best experience with some of these setups. We're dedicated to fixing issues around this, to ensure that whatever your registry set up might be, Skaffold will never get in your way.

### Secret Management - P3
Secrets are essential in real-world applications, but managing secrets is a major point of friction for many Kubernetes developers. Users sometimes have to work around Skaffold to handle secrets, but we want users to work **with** Skaffold instead. We'll provide native support for secret management within Skaffold to ensure that this adds as little time to the development cycle as possible.

### Interactive UI - P3
We want to provide an excellent user interface for interacting with Skaffold, either in the terminal or hosted locally in the browser. This will be especially helpful for debugging complex applications, but can also be useful for interacting with the different phases of a Skaffold pipeline in a visual way.

## Existing Feature Maturity and Improvement

### Actionable Error Messages - P0
Skaffold will always tell you when something goes wrong, but it's not always easy to tell what went wrong. We’d like to see how we can overhaul the error messaging surfaced through Skaffold to make sure that when something doesn't go as expected, users can always have actionable feedback to speed up their development.

### Debug goes GA - P0
Debugging applications during development is an integral part of a typical dev workflow, and we see a lot of value in continuing to invest in the `skaffold debug` experience. We’d love to be able to express confidence in the maturity of the debugging functionality in Skaffold by the end of the year.

### Init goes beta - P0
As part of our commitment to providing an excellent getting started experience for new Skaffold users, we’d like to shift the `skaffold init` command into beta. There are still many moving parts to this functionality that are changing, but as we flesh this out we’ll be growing much more confident in the onboarding experience that this provides.

### Render goes beta - P1
`skaffold render` is the crucial command backing our support for GitOps development workflows. We’ll be working to make sure that this command works natively with all of our supported deployers, and that the feature focuses on ease-of-use as part of our efforts to make Skaffold more compatible with the CI/CD stage of dev workflows.

### GitOps - P2
We want to make sure that Skaffold also works very well in projects configured with GitOps. Skaffold will support Kubernetes manifest rendering and project snapshotting to provide users with the artifacts they need for GitOps-based CI/CD workflows.
