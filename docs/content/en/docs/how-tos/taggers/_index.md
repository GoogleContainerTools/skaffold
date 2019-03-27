---
title: "Taggers"
linkTitle: "Taggers"
weight: 20
---

This page discusses how to set up Skaffold to tag artifacts as you see fit.

Skaffold supports the following tagging policies:

* `gitCommit`: uses Git commit IDs as tags
* `sha256`: uses Sha256 hashes of contents as tags
* `envTemplate`: uses values of environment variables as tags
* `dateTime`: uses date and time values as tags

Tag policy is specified in the `tagPolicy` field of the `build` section of the
Skaffold configuration file, `skaffold.yaml`.

For a detailed discussion on Skaffold configuration, see
[Skaffold Concepts](/docs/concepts/#configuration) and
[skaffold.yaml References](/docs/references/yaml).

## `gitCommit`: uses Git commit IDs as tags

`gitCommit` is the default tag policy of Skaffold: if you do not specify the
`tagPolicy` field in the `build` section, Skaffold will use Git information
to tag artifacts.

The `gitCommit` tagger will look at the Git workspace that contains
the artifact's `context` directory and tag according to those rules:

 + If the workspace is on a Git tag, that tag is used to tag images
 + If the workspace is on a Git commit, the short commit is used
 + It the workspace has uncommited changes, a `-dirty` suffix is appended to the image tag

### Example

The following `build` section instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with the `gitCommit` tag policy
specified explicitly:

{{% readfile file="samples/taggers/git.yaml" %}}

### Configuration

`gitCommit` tag policy features no options.

## `sha256`: uses Sha256 hashes of contents as tags

`sha256` is a content-based tagging strategy: it uses the Sha256 hash of
your built image as the tag of the Docker image.

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
tag policy. The tag template is `{{.IMAGE_NAME}}:{{.FOO}}`; when Skaffold
finishes building the image, it will check the list of available environment
variables in the system for the variable `FOO`, and use its value to tag the
image.

{{< alert >}}
<b>Note</b><br>

<code>IMAGE_NAME</code> is a built-in variable whose value is the <code>imageName</code> field in
the <code>artifacts</code> part of the <code>build</code> section.
{{< /alert >}}

### Example

{{% readfile file="samples/taggers/envTemplate.yaml" %}}

Suppose the value of the `FOO` environment variable is `v1`, the image built
will be `gcr.io/k8s-skaffold/example:v1`.

### Configuration

The tag template uses the [Go Programming Language Syntax](https://golang.org/pkg/text/template/).
As showcased in the example, `envTemplate` tag policy features one
**required** parameter, `template`, which is the tag template to use. To learn more about templating support in Skaffold.yaml see [Templated fields](/docs/how-tos/templating)

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
[Go Programming Language Documentation: Time package/Format Function](https://golang.org/pkg/time/#Time.Format) and
[Go Programming Language Documentation: Time package/LoadLocation Function](https://golang.org/pkg/time/#LoadLocation) respectively. As showcased in the
example, `dateTime`
tag policy features two optional parameters: `format` and `timezone`.
