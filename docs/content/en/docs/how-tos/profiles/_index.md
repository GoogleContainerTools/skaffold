
---
title: "Using Profiles"
linkTitle: "Using Profiles"
weight: 70
---

This page discusses Skaffold profiles.

Skaffold profiles allow you to define build and deployment
configurations for different contexts. Different contexts are typically different environments in your app's lifecycle, like Production or Development. 

You can create profiles in the `profiles` section of `skaffold.yaml`. For a
detailed discussion on Skaffold configuration,
see [Skaffold Concepts: Configuration](/concepts/config) and
[skaffold.yaml References](/references/config).

## Profiles (`profiles`)

Each profile has three parts:

* Name (`name`): The name of the profile.
* Build configuration (`build`)
* Deploy configuration (`deploy`)

Once activated, the specified build and deploy configuration
in the profile will override the `build` and `deploy` section declared
in `skaffold.yaml`. The build and deploy configuration in the `profiles`
section use the same syntax as the `build` and `deploy` section of
`skaffold.yaml`; for more information, see [Using Builders](/how-tos/builders),
[Using Taggers](/how-tos/taggers), and [Using Deployers](/how-tos/deployers).

You can activate a profile with the `-p` (`--profile`) parameter in the
`skaffold dev` and `skaffold run` commands.

The following example, showcases a `skaffold.yaml` with one profile, `gcb`,
for building with Google Cloud Build:

```
apiVersion: skaffold/v1alpha2
kind: Config
build:
    artifacts:
    - imageName: gcr.io/k8s-skaffold/skaffold-example
    deploy:
        kubectl:
        manifests:
        - k8s-pod
    profiles:
    - name: test-env
      build:
        googleCloudBuild:
            projectId: k8s-skaffold
```

With no profile activated, Skaffold will build the artifact
`gcr.io/k8s-skaffold/skaffold-example` using local Docker daemon and deploy it
with `kubectl`. However, if you run Skaffold with the following command:

`skaffold dev -p test-env` (or `skaffold run -p test-env`)

Skaffold will switch to Google Cloud Build for building artifacts. Note that
since the `gcb` profile does not specify a deploy configuration, Skaffold will
continue using `kubectl` for deployments.
