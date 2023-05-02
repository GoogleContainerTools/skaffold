## Policies

This repository adheres to the following project policies:

- [Code of Conduct][code-of-conduct] - How we should act with each other.
- [Contributing][contributing] - General contributing standards.
- [Security][security] - Reporting security concerns.
- [Support][support] - Getting support.

## Contributing to this repository

### Welcome

We welcome contributions to this repository! To get a sense of what the team is currently focusing on, check out our [milestones](https://github.com/buildpacks/lifecycle/milestones). Issues labeled [good first issue](https://github.com/buildpacks/lifecycle/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) and issues in our [docs repo](https://github.com/buildpacks/docs/issues?q=is%3Aissue+is%3Aopen+label%3Ateam%2Fimplementations) are great places to get started, but you are welcome to work on any issue that interests you. For issues requiring a greater degree of coordination, such as those labeled `status/needs-discussion` or that are part of larger epics, please reach out in the #implementation channel in [Slack](https://slack.buildpacks.io/).

### Development

Aside from the policies above, you may find [DEVELOPMENT.md](DEVELOPMENT.md) helpful in developing in this repository.

### Background

Here are some topics that might be helpful in further understanding the lifecycle:

* Cloud Native Buildpacks platform api spec
  * Example platforms: [pack CLI](https://github.com/buildpack/pack), [Tekton](https://github.com/tektoncd/catalog/blob/master/task/buildpacks/0.1/README.md)
* Cloud Native Buildpacks buildpack api spec
  * Example buildpack providers: [Google](https://github.com/GoogleCloudPlatform/buildpacks), [Heroku](https://www.heroku.com/), [Paketo](https://paketo.io/)
* The Open Container Initiative (OCI) and [OCI image spec](https://github.com/opencontainers/image-spec)
* Questions to deepen understanding:
  * What are the different [lifecycle phases](https://buildpacks.io/docs/concepts/components/lifecycle/)? What is the purpose of each phase?
  * What is a [builder](https://buildpacks.io/docs/concepts/components/builder/)? Is it required to run the lifecycle?
  * What is the [untrusted builder workflow](https://medium.com/buildpacks/faster-more-secure-builds-with-pack-0-11-0-4d0c633ca619)? Why do we have this flow?
  * What is the [launcher](https://github.com/buildpacks/spec/blob/main/platform.md#launch)? Why do we have a launcher?
  * What does a [buildpack](https://buildpacks.io/docs/concepts/components/buildpack/) do? Where does it write data? How does it communicate with the lifecycle?
  * What does a [platform](https://buildpacks.io/docs/concepts/components/platform/) do? What things does it know about that the lifecycle does not? How does it communicate with the lifecycle?
  * What is a [stack](https://buildpacks.io/docs/concepts/components/stack/)? Who produces stacks? Why is the stack concept important for the lifecycle?

[code-of-conduct]: https://github.com/buildpacks/.github/blob/main/CODE_OF_CONDUCT.md
[contributing]: https://github.com/buildpacks/.github/blob/main/CONTRIBUTING.md
[security]: https://github.com/buildpacks/.github/blob/main/SECURITY.md
[support]: https://github.com/buildpacks/.github/blob/main/SUPPORT.md
[pull-request-process]: https://github.com/buildpacks/.github/blob/main/CONTRIBUTIONS.md#pull-request-process
