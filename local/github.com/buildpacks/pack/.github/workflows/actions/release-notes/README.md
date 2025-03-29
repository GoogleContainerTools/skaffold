## Changelog

A simple script that generates the changelog for pack based on a pack version (aka milestone).

### Usage

#### Config

This script takes a configuration file in the following format:

```yaml
labels:
  # labels are grouped based on order but displayed based on weight
  <label>:
    # title for the group of issues
    title: <string>
    # description for the group of issues
    description: <string>
    # description for the group of issues
    weight: <number>

sections:
  contributors:
    # title for the contributors section, hidden if empty
    title: <string>
    # description for the contributors section
    description: <string>
```

#### Github Action

```yaml
- name: Generate changelog
  uses: ./.github/workflows/actions/release-notes
  id: changelog
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    milestone: <milestone>
```

#### Local

To run/test locally:

```shell script
# install deps
npm install

# set required info
export GITHUB_TOKEN="<GITHUB_PAT_TOKEN>"

# run locally
npm run local -- <milestone> <config-path>
```

Notice that a file `changelog.md` is created as well for further inspection.

### Updating

This action is packaged for distribution without vendoring `npm_modules` with use of [ncc](https://github.com/vercel/ncc).

When making changes to the action, compile it and commit the changes.

```shell script
npm run-script build
```