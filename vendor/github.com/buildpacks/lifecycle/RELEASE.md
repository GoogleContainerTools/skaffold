## Release Finalization

To cut a pre-release:
1. If applicable, ensure the README is updated with the latest supported apis (example PR: https://github.com/buildpacks/lifecycle/pull/550).
1. Create a release branch in the format `release/0.99.0-rc.1`. New commits to this branch will trigger the `build` workflow and produce a lifecycle image: `buildpacksio/lifecycle:<commit sha>`.
1. When ready to cut the release, manually trigger the `draft-release` workflow: Actions -> draft-release -> Run workflow -> Use workflow from branch: `release/0.99.0-rc.1`. This will create a draft release on GitHub using the artifacts from the `build` workflow run for the latest commit on the release branch.
1. Edit the release notes as necessary.
1. Perform any manual validation of the artifacts (see below).
1. When ready to publish the release, edit the release page and click "Publish release". This will trigger the `post-release` workflow that will re-tag the lifecycle image from `buildpacksio/lifecycle:<commit sha>` to `buildpacksio/lifecycle:0.99.0` but will NOT update the `latest` tag.

To cut a release:
1. Ensure the relevant spec APIs have been released.
1. Ensure the `lifecycle/0.99.0` milestone on the [docs repo](https://github.com/buildpacks/docs/blob/main/RELEASE.md#lump-changes) is complete, such that every new feature in the lifecycle is fully explained in the `release/lifecycle/0.99` branch on the docs repo, and [migration guides](https://github.com/buildpacks/docs/tree/main/content/docs/reference/spec/migration) (if relevant) are included.
1. Create a release branch in the format `release/0.99.0`. New commits to this branch will trigger the `build` workflow and produce a lifecycle image: `buildpacksio/lifecycle:<commit sha>`.
1. If applicable, ensure the README is updated with the latest supported apis (example PR: https://github.com/buildpacks/lifecycle/pull/550) and remove the pre-release note for the latest apis.
1. When ready to cut the release, manually trigger the `draft-release` workflow: Actions -> draft-release -> Run workflow -> Use workflow from branch: `release/0.99.0`. This will create a draft release on GitHub using the artifacts from the `build` workflow run for the latest commit on the release branch.
1. Edit the release notes as necessary.
1. Perform any manual validation of the artifacts.
1. When ready to publish the release, edit the release page and click "Publish release". This will trigger the `post-release` workflow that will re-tag the lifecycle image from `buildpacksio/lifecycle:<commit sha>` to `buildpacksio/lifecycle:0.99.0` and `buildpacksio/lifecycle:latest`.
1. Once released
- Update the `main` branch to remove the pre-release note in [README.md](https://github.com/buildpacks/lifecycle/blob/main/README.md) and/or merge `release/0.99.0` into `main`.
- Ask the learning team to merge the `release/lifecycle/0.99` branch into `main` on the docs repo.
