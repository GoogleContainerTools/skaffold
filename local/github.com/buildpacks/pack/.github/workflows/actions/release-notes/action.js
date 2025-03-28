/*
 * This file is the main entrypoint for GitHub Actions (see action.yml)
 */

const core = require('@actions/core');
const github = require('@actions/github');
const releaseNotes = require('./release-notes.js');

try {
  const defaultConfigFile = "./.github/release-notes.yml";

  releaseNotes(
    github.getOctokit(core.getInput("github-token", {required: true})),
    `${github.context.repo.owner}/${github.context.repo.repo}`,
    core.getInput('milestone', {required: true}),
    core.getInput('configFile') || defaultConfigFile,
  )
    .then(contents => {
      console.log("GENERATED CHANGELOG\n=========================\n", contents);
      core.setOutput("contents", contents)
    })
    .catch(error => core.setFailed(error.message))
} catch (error) {
  core.setFailed(error.message);
}