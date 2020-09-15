# Release Process

Pack follows a 6 week release cadence, composed of 3 phases:
  - [Development](#development)
  - [Feature Complete](#feature-complete)
  - [Release Finalization](#release-finalization)

## Development
Our development flow is detailed in [Development](DEVELOPMENT.md)

## Feature Complete
### Process
5 business days prior to a scheduled release, we enter `feature complete`. A release branch (in the form `release/<VERSION>`) is created, and used for User Acceptance testing (`UAT`).

During this period, relevant changes may be merged into the release branch, based on assessment by the `pack` [maintainers][maintainers] of the impact, effort and risk of including the changes. Any other change may get merged into `main` through the normal process, and will make it into the next release.

### Roles
#### Release Manager
One of the [maintainers][maintainers] is designated as the release manager. They communicate the release status to the working group meetings, schedule additional meetings with the `pack` [maintainers][maintainers] as needed, and finalize the release. They also take care of whatever release needs may arise.

## Release Finalization
The [release manager](#release-manager) will:
- Create a [github release][release], containing the **artifacts**, **release notes**, and a **migration guide** (if necessary), documenting breaking changes, and providing actions to migrate from prior versions.
- Tag the release branch as `v<version>`
- Merge the release branch into `main`
- Send out release notifications, if deemed necessary, on
  - The [cncf-buildpacks mailing list](https://lists.cncf.io/g/cncf-buildpacks)
  - Twitter

For more information, see the [release process RFC][release-process]

[maintainers]: https://github.com/buildpacks/community/blob/main/TEAMS.md#platform-team
[release-process]: https://github.com/buildpacks/rfcs/blob/main/text/0039-release-process.md#change-control-board
[release]: https://github.com/buildpacks/pack/releases
