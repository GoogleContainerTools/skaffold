We're glad you are interested in contributing to this project. We hope that this
document helps you get started.

## Policies

This repository adheres to the following project policies:

- [Code of Conduct][code-of-conduct] - How we should act with each other.
- [Contributing][contributing] - General contributing standards.
- [Security][security] - Reporting security concerns.
- [Support][support] - Getting support.

## Contributing to this repository

### Development

Aside from the policies above, you may find [DEVELOPMENT.md](DEVELOPMENT.md) to provide specific helpful detail
to assist you while developing in this repository.

#### Preparing for a Pull Request

After making all the changes but before creating a [Pull Request][pull-request-process], you should run
`make prepare-for-pr`. This command runs a set of other tasks that resolve or report any simple issues that would
otherwise arise during the pull request review process.

### User Acceptance on a Pull Request

Running user acceptance on a pull request is just as critical as reviewing the code changes. It allows you, a contributor and user, direct insight into how a feature works and allows for you to provide feedback into what could be improved.

#### Downloading PR binaries

1. On GitHub's Pull Request view, click on the **Checks** tab.
2. On the top-right, click **Artifacts**.
3. Click on the zip file for the platform you are running.

#### Setup

1. Unzip binary:
    ```shell
    unzip pack-{{PLATFORM}}.zip
    ```
2. Enable execution of binary _(macOS/Linux only)_:
    ```shell
    chmod +x ./pack
    ```

    > For macOS, you might need to allow your terminal to be able to execute applications from unverified developers. See [Apple Support](https://support.apple.com/en-us/HT202491).
    > 
    > A quick solution is to add exception to the downloaded pack binary: `sudo spctl --add -v ./pack`
3. You should now be able to execute pack via:
    - macOS: `./pack`
    - Linux: `./pack`
    - Windows: `pack.exe`


#### Writing Feedback

When providing feedback please provide a succinct title, a summary of the observation, what you expected, and some output or screenshots.

Here's a simple template you can use:

```text

#### <!-- title -->

<!-- a summary of what you observed -->

###### Expected

<!-- describe what you expected -->

###### Output

<!-- output / logs / screenshots -->
```


[code-of-conduct]: https://github.com/buildpacks/.github/blob/master/CODE_OF_CONDUCT.md
[contributing]: https://github.com/buildpacks/.github/blob/master/CONTRIBUTING.md
[security]: https://github.com/buildpacks/.github/blob/master/SECURITY.md
[support]: https://github.com/buildpacks/.github/blob/master/SUPPORT.md
[pull-request-process]: https://github.com/buildpacks/.github/blob/master/CONTRIBUTIONS.md#pull-request-process
