---
title: "Kube-context activation"
linkTitle: "Kube-context activation"
weight: 80
---

This page discusses how Skaffold selects the kube-context.


When interacting with a kubernetes cluster, Skaffold does so via a kube-context.
Thus, the selected kube-context determines the kubernetes cluster, the kubernetes user, and the default namespace.
By default, Skaffold uses the _current_ kube-context from your kube-config file.

To override this default, Skaffold offers two options:

1. Via the `kube-context` flag

    ```bash
    skaffold dev --kube-context <myrepo>
    ```

1. Via the `deploy.kubeContext` option in `skaffold.yaml`

    ```yaml
    deploy:
      kubeContext: minikube
    ```

When both are given, the CLI flag always takes precedence.

### Kube-context activation and Skaffold profiles

The kube-context has a double role for Skaffold profiles:

1. Profiles may be auto-activated by a given kube-context.

1. It is possible to change the kube-context through a `deploy.kubeContext` option in a Skaffold profile.

For clarity, Skaffold does the whole profile activation with the original kube-context.
In addition, when a profile is auto-activated by a matching kube-context, the resulting kube-context must remain unchanged.
This rule prevents profile-specific settings for one context to be deployed into a different context.


For example, given the following profiles:
```yaml
profiles:
  - name: profile-1
    deploy:
      kubeContext: docker-for-desktop

  - name: profile-2
    activation:
      - kubeContext: minikube
```

It is illegal to activate both profiles which happens when

- `skaffold run -p profile-1,profile-2`
- `skaffold run -p profile-1` if the current kube-context is `minikube`

{{< alert title="Note" >}}
It is possible to activate conflicting profiles in conjunction with the CLI flag:

    skaffold run --kube-context minikube -p profile-1,profile-2

{{< /alert >}}

### Limitations

It is not possible to change the kube-context of a running `skaffold dev` session.
In order to pick up the new kube-context from `skaffold.yaml`, Skaffold has to be restarted.
