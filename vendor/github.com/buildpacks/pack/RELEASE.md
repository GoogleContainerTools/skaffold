# Release Process

Pack follows a 6 week release cadence, composed of 3 phases:
  - [Development](#development)
  - [Feature Complete](#feature-complete)
  - [Release Finalization](#release-finalization)

## Roles

#### Release Manager

One of the [maintainers][maintainers] is designated as the release manager. They communicate the release status to the working group meetings, schedule additional meetings with the `pack` [maintainers][maintainers] as needed, and finalize the release. They also take care of whatever release needs may arise.

## Phases

### Development

Our development flow is detailed in [Development](DEVELOPMENT.md).

### Feature Complete

5 business days prior to a scheduled release, we enter `feature complete`.

During this period, a **Release Candidate** (RC) is published and used for further User Acceptance testing (`UAT`). Furthermore, additional RCs may be published based on assessment by the `pack` [maintainers][maintainers] of the **impact**, **effort** and **risk** of including the changes in the upcoming release. Any other changes may be merged into the `main` branch through the normal process, and will make it into the next release.

To produce the release candidate the [release manager](#release-manager) will:
- Create a new release branch in form `release/<VERSION>-rc<NUMBER>` yielding a draft GitHub release to be published. 
- Publish the [GitHub release][release]:
    - Tag release branch as `v<VERSION>-rc<NUMBER>`.
    - Release should be marked as "pre-release".
    - The GitHub release will contain the following:
        - **artifacts**
        - **release notes**
    - The release notes should be edited and cleaned
- Merge the release branch into `main`.

### Release Finalization

The [release manager](#release-manager) will:
- Create a new release branch in form `release/<VERSION>` yielding a draft GitHub release to be published. 
- Publish the [GitHub release][release] while tagging the release branch as `v<VERSION>`.
    - Tag release branch as `v<VERSION>`.
    - The GitHub release will contain the following:
        - **artifacts**
        - **release notes**
        - **migration guide** (if necessary)
- Merge the release branch into `main`.
- Create a new [milestone](https://github.com/buildpacks/pack/milestones) for the next version, and set the delivery time in 6 weeks.
- Move all still open PRs/issues in the delivered milestone to the new milestone
- Close the delivered milestone
- Send out release notifications, if deemed necessary, on
  - The [cncf-buildpacks mailing list](https://lists.cncf.io/g/cncf-buildpacks)
  - Twitter
- Post release, you should be able to remove any acceptance test constraints (in [acceptance/invoke/pack.go](acceptance/invoke/pack.go)) in the `featureTests` struct. Create a PR removing them, in order to ensure our acceptance tests are clean.

And with that, you're done!

## Manual Releasing

We release pack to a number of systems, including `homebrew`, `docker`, and `archlinux`. All of our delivery pipelines
have workflow_dispatch triggers, if a maintainer needs to manually trigger them. To activate it, go to the
[actions page](https://github.com/buildpacks/pack/actions), and select the desired workflow. Run it by providing the pack
version to release, in the format `v<version>`.

_For more information, see the [release process RFC][release-process]_

[maintainers]: https://github.com/buildpacks/community/blob/main/TEAMS.md#platform-team
[release-process]: https://github.com/buildpacks/rfcs/blob/main/text/0039-release-process.md#change-control-board
[release]: https://github.com/buildpacks/pack/releases
