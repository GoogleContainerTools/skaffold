<!-- 🎉🎉🎉 Thank you for the PR!!! 🎉🎉🎉 -->


Relates to _in case of new feature, this should point to issue/(s) which describes the feature_

Fixes `#<issue number>`. _in case of a bug fix, this should point to a bug and any other related issue(s)_

Should merge before : _list any PRs that depend on this PR_

Should merge after : _list any PRs that are prerequisites to this PR_

**Description**

<!-- Describe your changes here- ideally you can get that description straight from
your descriptive commit message(s)! -->

**User facing changes**

Write n/a if not output or log lines changed and no behavior is changed

**Before**

If log/output changes: Paste the current relevant skaffold output
If behavior changes: describe succinctly the current behavior

**After**

If log/output changes: Paste skaffold output after your change
If behavior changes: describe succintly the behavior after your change

**Next PRs.**

In this section describe a list of follow up PRs if the current PR is a part of big feature change.

See example #2811

Write n/a if not applicable.


**Submitter Checklist**

These are the criteria that every PR should meet, please check them off as you
review them:

- [ ] Includes [unit tests](../DEVELOPMENT.md#creating-a-pr)
- [ ] Mentions any output changes.
- [ ] Adds documentation as needed: user docs, YAML reference, CLI reference.
- [ ] Adds integration tests if needed.

_See [the contribution guide](../CONTRIBUTING.md) for more details._

Double check this list of stuff that's easy to miss:

- If you are adding [a example to the `examples` dir](https://github.com/GoogleContainerTools/skaffold/tree/master/examples), please copy them to [`integration/examples`](https://github.com/GoogleContainerTools/skaffold/tree/master/integration/examples)
- Every new example added in [`integration/examples` dir](https://github.com/GoogleContainerTools/skaffold/tree/master/integration/examples), should be tested in [integration test](https://github.com/GoogleContainerTools/skaffold/tree/master/integration)

**Reviewer Notes**

- [ ] The code flow looks good. 
- [ ] Unit test added.
- [ ] User facing changes look good.


**Release Notes**

Describe any user facing changes here so maintainer can include it in the release notes, or delete this block.

```
Examples of user facing changes:
- Skaffold config changes like
  e.g. "Add buildArgs to `Kustomize` deployer skaffold config."
- Bug fixes
  e.g. "Improve skaffold init behaviour when tags are used in manifests"
- Any changes in skaffold behavior
  e.g. "Artiface cachine is turned on by default."

```
