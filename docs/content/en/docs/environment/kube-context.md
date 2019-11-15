---
title: "Kube-context Activation"
linkTitle: "Kube-context Activation"
weight: 80
featureId: deploy.kubecontext
---

When interacting with a Kubernetes cluster, just like any other Kubernetes-native tool,
Skaffold requires a valid Kubernetes context to be configured.
The selected kube-context determines the Kubernetes cluster, the Kubernetes user, and the default namespace.
By default, Skaffold uses the _current_ kube-context from your kube-config file.

You can override this default one of two ways:

1. `--kube-context` flag

    ```bash
    skaffold dev --kube-context <myrepo>
    ```

1. Specify `deploy.kubeContext` configuration in `skaffold.yaml`

    ```yaml
    deploy:
      kubeContext: minikube
    ```

The CLI flag always takes precedence over the config field in the `skaffold.yaml`.

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

## Kubeconfig selection

The kubeconfig file is only loaded once during Skaffold's startup phase.

1. If the `--kubeconfig` flag is set, then only that file is loaded.
2. If `$KUBECONFIG` environment variable is set, then it is used as a list of paths (normal path delimiting rules for your system). These paths are merged.
3. Otherwise, ${HOME}/.kube/config is used.
4. If neither `--kubeconfig` or `--kube-context` are given and no kubeconfig file is found, Skaffold will try to guess an in-cluster
   configuration using the secrets stored in `/var/run/secrets/kubernetes.io/serviceaccount/`. This is useful when Skaffold runs inside
   a kubernetes Pod and should deploy to the same cluster.
