# Improve taggers

* Author(s): David Gageot (@dgageot)
* Date: 4 September 2019

## Background

So far, Skaffold supports multiple taggers or tag policies:

 + the `git` tagger will use git commits/references to tag images
 + the `sha256` tagger always uses `latest` or the tag specified on the artifact's image name
 + the `envTemplate` tagger that can use environment variables to tag images

The default tagger, is none is specified in the skaffold.yaml, is the git tagger.

There are multiple issues:

 + the `git` tagger requires users to have the `git` binary installed. It also requires
   the project to be a git project, which is typically not the case when users just try
   to get started.
 + the `sha256` has a wrong name. It is named like that because, in the end, when Skaffold
   deploys to a remote cluster, the image's sha256 digest will be used as the immutable tag.
   Users are very confused with this name and behaviour.
 + the `sha256` used to be able to use the image tags provided in the artifact definition,
   instead of `latest`. This was not documented and is not possible anymore because artifact
   definitions are now considered invalid if images names have a tag.
 + the `envTemplate` tagger used to be able to replace `{{.DIGEST}}` with the image's imageID
   or digest. This was buggy and is not possible anymore since tags are computed before the
   images are built.
 + users want a tagger that uses the inputs' digest as a tag. They think that's what the
   `sha256` tagger should do.
   
## Proposal

 + we introduce a `latest` tagger tag tags images with `:latest`.
 + the `latest` tagger is used by default instead of the `git` tagger.
 + `sha256` is completely changed to use a digest of the artifact's inputs as the tag.
   Something like https://github.com/GoogleContainerTools/skaffold/pull/2301
 + `envTemplate` learns how to replace `{{.DIGEST}}` with a digest of the artifact's
    inputs as the tag.

## Open Issues/Questions

 + How do we handle users who didn't configure a tagger and were happy with the default
   being the `git` tagger?
 + How do we handle users who were happy with `sha256` tagger?
