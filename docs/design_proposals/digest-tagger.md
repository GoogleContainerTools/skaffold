# Improve taggers

* Author(s): David Gageot (@dgageot)
* Date: 4 September 2019

## Background

So far, Skaffold supports multiple taggers or tag policies:

 + the `git` tagger uses git commits/references to tag images.
 + the `sha256` tagger uses `latest` or the tag specified on the artifact's image name.
 + the `envTemplate` tagger uses environment variables to tag images.
 + the `datetime` tagger uses current date and time, with a configurable pattern.

The default tagger, if none is specified in the `skaffold.yaml`, is the `git` tagger.

Here are some rules about how tagging currently works:

 + **Image tags are computed before the images are built**. In early versions of Skaffold, we tried
   to compute tags after the build. It made the process super complex with a lot of retagging.
   It also produced images that were tagged with their own digest or imageID which is superfluous
   since those can be used to reference the images directly.
 + **No matter the tagger, Skaffold always uses immutable references in Kubernetes manifests**.
   Which reference is used depends on whether the images are pushed or not:
     + **When images are pushed**, their immutable digest is available. Skaffold then references
       images both by tag and digest. Something like `image:tag@sha256:abacabac...`.
       Using both the tag and the digest seems superfluous but it guarantees immutability
       and helps users quickly see which version of the image is used.
     + **When images are not pushed**, digests are not available. We have the tags and the
       imageIDs. Since imageIDs can't be used in Kubernetes manifests, Skaffold creates
       an additional immutable tag with the same name as the imageID and uses that in manifests.
       Something like `image:abecfabecfabecf...`.
 + **Skaffold never references images just by their tags** because those tags are mutable and
   can lead to cases where Kubernetes will use an outdated version of the image.

## Issues

 + the `git` tagger requires users to install `git`. It also requires the project to be
   a git project, which is typically not the case when users just try to get started.
   **So the `git` tagger seems like a wrong choice for a default tagger**.
 + `sha256` is a misleading name. It is named like that because, in the end, when Skaffold
   deploys to a remote cluster, the image's sha256 digest is used as the immutable tag.
   **Users are confused with this name and behavior**.
 + the `sha256` used to be able to use the image tags provided in the artifact definition,
   instead of `latest`. This was not documented and is not possible anymore because artifact
   definitions are now considered invalid if images names have a tag.
   **The new way to achieve that goal is to use the `envTemplate` tagger**
 + the `envTemplate` tagger used to be able to replace `{{.DIGEST}}` with the image's imageID
   or digest. **This was buggy and is not possible anymore since tags are computed before the
   images are built and those digest are only available after the image is built or pushed.**
 + users have asked for a tagger that uses the inputs' digest as a tag. **They think that's
   what the `sha256` tagger should do.**
   
## Proposal

 + we introduce a `latest` tagger that tags images with `:latest`.
 + the `latest` tagger is used by default instead of the `git` tagger.
 + `sha256` tagger is deprecated.
 + `datetime` tagger is kept as is.
 + `git` tagger is kept as is but is no longer the default.
 + An `inputDigest` is added. It uses the digest of the artifact's inputs as the tag.
   [#2301](https://github.com/GoogleContainerTools/skaffold/pull/2301) tried to implement
   such tagger by computing the digest of the whole workspace. We should instead compute
   the digest of the artifact's dependencies, including the artifact's configuration. This
   is exactly what the caching mechanism currently does.
 + `envTemplate` learns how to replace `{{.DIGEST}}` with a digest of the artifact's
    inputs as computed by the `inputDigest` tagger.
 + **No matter the tagger, Skaffold will keep on using immutable references in manifests**.
   + by tag and digest, when images are pushed.
   + by imageID (used as a tag), when images are not pushed.

## Open Issues/Questions

 + How do we handle users who didn't configure a tagger and were happy with the default
   being the `git` tagger?
 + How do we handle users who were happy with `sha256` tagger?

When no `tagPolicy` is used or when a deprecated tagger is used, we will have to
show a warning to the user.

For all the above changes we need clear communication in the release notes.
