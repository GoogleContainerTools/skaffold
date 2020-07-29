---
title: "Tag"
linkTitle: "Tag"
weight: 15
featureId: tagpolicy
aliases: [/docs/how-tos/taggers]
---

Skaffold supports multiple taggers or tag policies for tagging images:

 + the `gitCommit` tagger uses git commits/references to tag images.
 + the `sha256` tagger uses `latest` to tag images.
 + the `envTemplate` tagger uses environment variables to tag images.
 + the `datetime` tagger uses current date and time, with a configurable pattern.

The default tagger, if none is specified in the `skaffold.yaml`, is the `gitCommit` tagger.

### Configuration

The tag policy is specified in the `tagPolicy` field of the `build` section
of the `skaffold.yaml` configuration file.

{{% readfile file="samples/taggers/git.yaml" %}}

For a detailed discussion on Skaffold configuration, see
[Skaffold Concepts]({{< relref "/docs/design/config.md" >}}) and
[skaffold.yaml References]({{< relref "/docs/references/yaml" >}}).

### How tagging works

 + Image tags are computed before the images are built.
 + No matter the tagger, Skaffold always uses immutable references in Kubernetes manifests.
   Which reference is used depends on whether the images are pushed to a registry or loaded directly into the cluster (such as via the Docker daemon):
     + **When images are pushed**, their immutable digest is available. Skaffold then references
       images both by tag and digest. Something like `image:tag@sha256:abacabac...`.
       Using both the tag and the digest seems superfluous but it guarantees immutability
       and helps users quickly see which version of the image is used.
     + **When images are loaded directly into the cluster**, such as loading into the cluster's Docker daemon, digests are not available. We have the tags and the
       imageIDs. Since imageIDs can't be used in Kubernetes manifests, Skaffold creates
       an additional immutable, local only, tag with the same name as the imageID and uses that in manifests.
       Something like `image:abecfabecfabecf...`.
 + Skaffold never references images just by their tags because those tags are mutable and
   can lead to cases where Kubernetes will use an outdated version of the image.

## `gitCommit`: uses git commits/references as tags

`gitCommit` is the default tag policy of Skaffold: if you do not specify the
`tagPolicy` field in the `build` section, Skaffold will use Git information
to tag artifacts.

The `gitCommit` tagger will look at the Git workspace that contains
the artifact's `context` directory and tag according to those rules:

 + If the workspace is on a Git tag, that tag is used to tag images
 + If the workspace is on a Git commit, the short commit is used
 + If the workspace has uncommitted changes, a `-dirty` suffix is appended to the image tag

### Example

The following `build` section instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with the `gitCommit` tag policy
specified explicitly:

{{% readfile file="samples/taggers/git.yaml" %}}

### Configuration

{{< schema root="GitTagger" >}}

## `sha256`: uses `latest` to tag images

`sha256` is a misleading name. It is named like that because, in the end, when Skaffold
deploys to a remote cluster, the image's `sha256` digest is used in addition to `:latest`
in order to create an immutable image reference.

### Example

The following `build` section instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with the `sha256` tag policy:

{{% readfile file="samples/taggers/sha256.yaml" %}}

### Configuration

`sha256` tag policy features no options.

## `envTemplate`: uses values of environment variables as tags

`envTemplate` allows you to use environment variables in tags. This
policy requires that you specify a tag template, where part of template
can be replaced with values of environment variables during the tagging
process.

The following `build` section, for example, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with the `envTemplate`
tag policy. The tag template is `{{.FOO}}`; when Skaffold
finishes building the image, it will check the list of available environment
variables in the system for the variable `FOO`, and use its value to tag the
image.

{{< alert >}}
<b>Deprecated</b><br>

The use of `IMAGE_NAME` as a built-in variable whose value is the `imageName` field in the `artifacts` part of the `build` section has been deprecated. Please use the envTemplate to express solely the tag value for the image.
{{< /alert >}}

### Example

{{% readfile file="samples/taggers/envTemplate.yaml" %}}

Suppose the value of the `FOO` environment variable is `v1`, the image built
will be `gcr.io/k8s-skaffold/example:v1`.

### Configuration

The tag template uses the [Go Programming Language Syntax](https://golang.org/pkg/text/template/).
As showcased in the example, `envTemplate` tag policy features one
**required** parameter, `template`, which is the tag template to use. To learn more about templating support in Skaffold.yaml see [Templated fields]({{< relref "../environment/templating.md" >}})

## `dateTime`: uses data and time values as tags

`dateTime` uses the time when Skaffold starts building artifacts as the
tag. You can choose which format and timezone Skaffold should use. By default,
Skaffold uses the time format `2006-01-02_15-04-05.999_MST` and the local
timezone.

### Example

The following `build` section, for example, instructs Skaffold to build a Docker
image `gcr.io/k8s-skaffold/example` with the `dateTime`
tag policy:

{{% readfile file="samples/taggers/dateTime.yaml" %}}

Suppose current time is `15:04:09.999 January 2nd, 2006` and current time zone
is `MST` (`US Mountain Standard Time`), the image built will
be `gcr.io/k8s-skaffold/example:2006-01-02_15-04-05.999_MST`.

### Configuration

You can learn more about what time format and time zone you can use in
[Go Programming Language Documentation: Time package/Format Function](https://golang.org/pkg/time#Time.Format) and
[Go Programming Language Documentation: Time package/LoadLocation Function](https://golang.org/pkg/time#LoadLocation) respectively. As showcased in the
example, `dateTime`
tag policy features two optional parameters: `format` and `timezone`.
