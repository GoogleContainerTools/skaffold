---
title: "Kube-context activation"
linkTitle: "Kube-context activation"
weight: 80
---

This page discusses how Skaffold selects the `kube-context`.


When interacting with a kubernetes cluster, Skaffold does so via a kube-context.
Thus, the selected kube-context determines the kubernetes cluster, the kubernetes user, and the default namespace.
By default, Skaffold uses the _current_ kube-context from your kube-config file.

You can override this default via

1. `--kube-context` flag

    ```bash
    skaffold dev --kube-context <myrepo>
    ```

1. Specify `deploy.kubeContext` configuration in `skaffold.yaml`

    ```yaml
    deploy:
      kubeContext: minikube
    ```

When both are given, the CLI flag always takes precedence.

### Kube-context activation and Skaffold profiles

The kube-context has a double role for Skaffold profiles:

1. A Skaffold profile may be auto-activated by the current kube-context (via `profiles.activation.kubeContext`).

1. A Skaffold profile may change the kube-context (via `profiles.deploy.kubeContext`).

Skaffold ensures that these two roles are not conflicting.
To that end, profile activation is done with the original kube-context.
If any profile is auto-activated by a matching kube-context, the resulting kube-context must remain unchanged.
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

It is illegal to activate both profiles here, because `profile-2` has an activation by `kube-context` and `profile-1` changes the effective `kube-context`.
This happens for

- `skaffold run -p profile-1,profile-2`
- `skaffold run -p profile-1` if the current kube-context is `minikube`

{{< alert title="Note" >}}
It is possible to activate conflicting profiles in conjunction with the CLI flag. So the following example is valid `skaffold run --kube-context minikube -p profile-1,profile-2`
{{< /alert >}}

### Limitations

It is not possible to change the kube-context of a running `skaffold dev` session.
To pick up the changes to `kubeContext`, you will need to quit and re-run `skaffold dev`.
