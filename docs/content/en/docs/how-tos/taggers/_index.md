
---
title: "Taggers"
linkTitle: "Taggers"
weight: 30
---

This page discusses how to set up Skaffold to tag artifacts as you see fit.

Skaffold supports the following tagging policies:

* Using Git commit IDs as tags (`gitCommit`)
* Using Sha256 hashes of contents as tags (`sha256`)
* Using values of environment variables as tags (`envTemplate`)
* Using date and time values as tags (`dateTime`)

Tag policy is specified in the `tagPolicy` field of the `build` section of the
Skaffold configuration file, `skaffold.yaml`. For a detailed discussion on
Skaffold configuration,
see [Skaffold Concepts: Configuration](/docs/concepts/#configuration) and
[skaffold.yaml References](https://github.com/GoogleContainerTools/skaffold/blob/master/examples/annotated-skaffold.yaml).

## `gitCommit`: using Git commit IDs as tags

`gitCommit` is the default tag policy of Skaffold: if you do not specify the
`tagPolicy` field in the `build` section, Skaffold will tag artifacts with
the Git commit IDs of the repository.

The following `build` section, for example, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with the `gitCommit` tag policy
specified explicitly:

```yaml
build:
    artifacts:
    - image: gcr.io/k8s-skaffold/example
    tagPolicy:
        gitCommit: {}
```

`gitCommit` tag policy features no options.

## `sha256`: using Sha256 hashes of contents as tags

`sha256` is a content-based tagging strategy: it uses the Sha256 hash of
your built image as the tag of the Docker image.

{{< alert title="Note" >}} 

It is recommended that you use `sha256` tag policy during development, as
it allows Kubernetes to re-deploy images every time your source code changes.
{{< /alert >}}

The following `build` section, for example, instructs Skaffold to build a
Docker image `gcr.io/k8s-skaffold/example` with the `sha256` tag policy:

```yaml
build:
    artifacts:
    - image: gcr.io/k8s-skaffold/example
    tagPolicy:
        sha256: {}
```

`sha256` tag policy features no options.

## `envTemplate`: using values of environment variables as tags

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

```yaml
build:
    artifacts:
    - image: gcr.io/k8s-skaffold/example
    tagPolicy:
        envTemplate:
            template: "{{.IMAGE_NAME}}:{{.FOO}}"
```

Suppose the value of the `FOO` environment variable is `v1`, the image built
will be `gcr.io/k8s-skaffold/example:v1`.

The tag template uses the [Go Programming Language Syntax](https://golang.org/pkg/text/template/).
As showcased in the example, `envTemplate` tag policy features one
**required** parameter, `template`, which is the tag template to use. To learn more about templating support in Skaffold.yaml see [Templated fields](/docs/how-tos/templating)

## `dateTime`: using data and time values as tags

`dateTime` uses the time when Skaffold starts building artifacts as the
tag. You can choose which format and timezone Skaffold should use. By default,
Skaffold uses the time format `2006-01-02_15-04-05.999_MST` and the local
timezone.

The following `build` section, for example, instructs Skaffold to build a Docker
image `gcr.io/k8s-skaffold/example` with the `dateTime`
tag policy:

```yaml
build:
    artifacts:
    - image: gcr.io/k8s-skaffold/example
    tagPolicy:
        dateTime:
            format: "2006-01-02_15-04-05.999_MST"
            timezone: "Local"
# The build section above is equal to
# build:
#   artifacts:
#   - image: gcr.io/k8s-skaffold/example
#   tagPolicy:
#       dateTime: {}
#   local: {}
```

Suppose current time is `15:04:09.999 January 2nd, 2006` and current time zone
is `MST` (`US Mountain Standard Time`), the image built will
be `gcr.io/k8s-skaffold/example:2006-01-02_15-04-05.999_MST`.

You can learn more about what time format and time zone you can use in
[Go Programming Language Documentation: Time package/Format Function](https://golang.org/pkg/time/#Time.Format) and
[Go Programming Language Documentation: Time package/LoadLocation Function](https://golang.org/pkg/time/#LoadLocation) respectively. As showcased in the
example, `dateTime`
tag policy features two optional parameters: `format` and `timezone`.
