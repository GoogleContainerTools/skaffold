# Arch Linux

There are two maintained packages by us and one official archlinux package:

- [pack-cli](https://archlinux.org/packages/extra/x86_64/pack-cli/): Official Archlinux package in the 'Extra' repo.
- [pack-cli-bin](https://aur.archlinux.org/packages/pack-cli-bin/): The latest release of `pack`, precompiled.
- [pack-cli-git](https://aur.archlinux.org/packages/pack-cli-git/): An unreleased version of `pack`, compiled from source of the `main` branch.


## Current State

The following depicts the current state of automation:

| package      | tested | distributed |
| ---          | ---    | ---         |
| pack-cli     | yes    | yes         |
| pack-cli-bin | yes    | yes         |
| pack-cli-git | yes    | yes         |

## Run Locally

> **CAUTION:** This makes changes directly to the published packages. To prevent changes, comment out `git push` in `publish-package.sh`.

```shell script
docker pull nektos/act-environments-ubuntu:18.04
docker pull archlinux:latest

export GITHUB_TOKEN="<YOUR_GH_TOKEN>"
export AUR_KEY="<AUR_KEY>"

act -P ubuntu-latest=nektos/act-environments-ubuntu:18.04 \
    -e .github/workflows/testdata/event-release.json \
    -s GITHUB_TOKEN -s AUR_KEY \
    -j <JOB_NAME>
```
