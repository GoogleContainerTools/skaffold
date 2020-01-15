---
title: "Image Repository Handling"
linkTitle: "Image Repository Handling"
weight: 70
featureId: default_repo
aliases: [/docs/concepts/image_repositories]
---

Often, a Kubernetes manifest (or `skaffold.yaml`) makes references to images that push to
registries that we might not have access to. Modifying these individual image names manually
is tedious, so Skaffold supports automatically prefixing these image names with a registry
specified by the user. Using this, any project configured with Skaffold can be run by any user
with minimal configuration, and no manual YAML editing!

This is accomplished through the `default-repo` functionality, and can be used one of three ways:

1. `--default-repo` flag

    ```bash
    skaffold dev --default-repo <myrepo>
    ```

1. `SKAFFOLD_DEFAULT_REPO` environment variable

    ```bash
    SKAFFOLD_DEFAULT_REPO=<myrepo> skaffold dev
    ```

1. Skaffold's global config

    ```bash
    skaffold config set default-repo <myrepo>
    ```

If no `default-repo` is provided by the user, there is no automated image name rewriting, and Skaffold will
try to push the image as provided in the yaml.

The image name rewriting strategies are designed to be *conflict-free*:
the full image name is rewritten on top of the default-repo so similar image names don't collide in the base namespace (e.g.: repo1/example and repo2/example would collide in the target_namespace/example without this)

Automated image name rewriting strategies are determined based on the default-repo and the original image repository:

* default-repo does not begin with gcr.io
  * **strategy**: 		escape & concat & truncate to 256

    ```
     original image: 	gcr.io/k8s-skaffold/skaffold-example1
     default-repo:      aws_account_id.dkr.ecr.region.amazonaws.com
     rewritten image:   aws_account_id.dkr.ecr.region.amazonaws.com/gcr_io_k8s-skaffold_skaffold-example1
    ```

* default-repo begins with "gcr.io" (special case - as GCR allows for infinite deep image repo names)
  * **strategy**: concat unless prefix matches
  * **example1**: prefix doesn't match:

    ```
      original image: 	gcr.io/k8s-skaffold/skaffold-example1
      default-repo: 	gcr.io/myproject/myimage
      rewritten image:  gcr.io/myproject/myimage/gcr.io/k8s-skaffold/skaffold-example1
    ```
  * **example2**: prefix matches:

    ```
      original image: 	gcr.io/k8s-skaffold/skaffold-example1
      default-repo: 	gcr.io/k8s-skaffold
      rewritten image:  gcr.io/k8s-skaffold/skaffold-example1	
    ```
  * **example3**: shared prefix:

    ```
      original image: 	gcr.io/k8s-skaffold/skaffold-example1
      default-repo: 	gcr.io/k8s-skaffold/myimage
      rewritten image:  gcr.io/k8s-skaffold/myimage/skaffold-example1	
    ```

## Insecure image registries

During development you may be forced to push images to a registry that does not support HTTPS.
By itself, Skaffold will never try to downgrade a connection to a registry to plain HTTP.
In order to access insecure registries, this has to be explicitly configured per registry name.

There are several levels of granularity to allow insecure communication with some registry:

1. Per Skaffold run via the repeatable `--insecure-registry` flag

    ```bash
    skaffold dev --insecure-registry insecure1.io --insecure-registry insecure2.io
    ```
    
1. Per Skaffold run via `SKAFFOLD_INSECURE_REGISTRY` environment variable

    ```bash
    SKAFFOLD_INSECURE_REGISTRY='insecure1.io,insecure2.io' skaffold dev
    ```
    
1. Per project via the Skaffold pipeline config `skaffold.yaml`
    
    ```yaml
    build:
        insecureRegistries:
        - insecure1.io
        - insecure2.io
    ```

1. Per user via Skaffold's global config

    ```bash
    skaffold config set insecure-registries insecure1.io           # for the current kube-context
    skaffold config set --global insecure-registries insecure2.io  # for any kube-context
    ```
    
    Note that multiple set commands _add_ to the existing list of insecure registries.
    To clear the list, run `skaffold config unset insecure-registries`.
    
Skaffold will join the lists of insecure registries, if configured via multiple sources.
