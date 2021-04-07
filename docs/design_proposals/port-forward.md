# Make Skaffold port-forwarding less surprising

* Author(s): Brian de Alwis
* Date: 2020-09-22
* Late updated: 2021-03-16
* Status: Reviewed/Cancelled/**Under implementation**/Complete

## Objective

Align Skaffold port-forwarding behaviour with user expectations.


## Background

Skaffold supports establishing port-forwarding to deployed Kubernetes resources through
several mechanisms:

- A user can configure one or more [port forwards in their `skaffold.yaml`](https://skaffold.dev/docs/references/yaml/#portForward)
  to connect a local port to a remote port on a pod, a service, or some workload resource that
  has a pod-spec (e.g., Deployment, ReplicaSet, Job).
- Skaffold automatically configures port-forwards for services.
- When in _debug_ mode (`skaffold debug`), Skaffold automatically configures port-forwards to pods too.

Port-forwards are only established when a user runs Skaffold with `--port-forward`.  This was a conscious
choice in [#969](https://github.com/GoogleContainerTools/skaffold/issues/969) as Skaffold had been
forwarding all containerPorts defined on pods, and doing so led to conflicts
in ports chosen.  Users with many services encountered confusion when dealing with many services
([#1564](https://github.com/GoogleContainerTools/skaffold/issues/1564)).

But putting all port-forwards behind the `--port-forward` flag has resulted in new user confusion
(e.g., [#4163](https://github.com/GoogleContainerTools/skaffold/issues/4163),
[#4818](https://github.com/GoogleContainerTools/skaffold/issues/4818)):

> *discr33t* 12:32 PM: I’m having a problem getting Skaffold to port-forward to my service after
> doing a Helm deploy. If I use the --port-forward flag it will port-forward all my services. But
> if I define portForward: anywhere in my `skaffold.yaml` nothing is working.

Forwarding all `containerPort`s on pods in `skaffold debug` has also resulted in odd situations
such as container-ports being allocated before service ports.  For example, the Skaffold
[`examples/react-reload`](https://github.com/GoogleContainerTools/skaffold/tree/HEAD/examples/react-reload/) example
has a [service on port 8080](https://github.com/GoogleContainerTools/skaffold/tree/HEAD/examples/react-reload/k8s/deployment.yaml#L7)
and a [deployment/pod on port 8080](https://github.com/GoogleContainerTools/skaffold/blob/master/examples/react-reload/k8s/deployment.yaml#L29).



## Design

This document proposes to change Skaffold port-forwarding's defaults in a similar fashion as proposed
by @corneliusweig in https://github.com/GoogleContainerTools/skaffold/issues/1564#issuecomment-473528574.
These changes are intended to be backwards-compatible with the previous version
of Skaffold except where noted.


### Command-Line Changes

> **NOTE**: Ideally we would re-purpose Skaffold's existing `--port-forward` CLI argument in
> a backward-compatible manner (still under investigation).  This document assumes it is
> not possible.

~Skaffold's `--port-forward` argument should be changed from a binary true/false option to~

Skaffold will introduce a new `--port-forward-modes` argument that takes
a set of comma-separated values with the following defined modes:

   - `user`: user-defined port-forwards as defined in the `skaffold.yaml`
   - `services`: service ports are forwarded
   - `pods`: `containerPort`s defined on Pods and Kubernetes workload objects that have pod-specs
     are forwarded (Deployment, ReplicaSet, StatefulSet, DaemonSet, Job, CronJob)
   - `debug`: an internal strategy to forward debugging-related ports on Pods as set up
     from `skaffold debug`
   - `off`: no ports are ever forwarded

#### Changes to port-forwarding defaults

Skaffold will default to enabling port-forwarding for establishing user-defined port-forwards
as defined in the `skaffold.yaml`.  Skaffold's `dev` and `debug` will be changed to the following defaults.

Command-line                        | v1.15.0 default modes          | New default modes
----------------------------------- | ---------------- | -------------------
`skaffold dev`                        | off            | user
`skaffold dev --port-forward`         | user, services | user, services (no change)
`skaffold dev --port-forward=false`   | off            | off (no change)
`skaffold debug`                      | off            | user, debug
`skaffold debug --port-forward`       | user, services, pods | user, services, debug
`skaffold debug --port-forward=false` | off            | off (no change)

**Compatibility Change:** The behaviour of `skaffold debug --port-forward` no longer forwards all
`containerPort`s defined on pods.  Container ports were forwarded as there was no other option to forward
just debug ports.  With this proposed change, the `debug` mode will only select debug ports.  Forwarding all
`containerPort`s was the cause of some confusion in earlier iterations of port-forwarding.


### Open Issues/Questions

Please list any open questions here in the following format:

**\<Question\>**

Resolution: Please list the resolution if resolved during the design process or
specify __Not Yet Resolved__


## Implementation plan

1. Separate user-defined forwarding from [ResourceForwarder](https://github.com/GoogleContainerTools/skaffold/blob/master/pkg/skaffold/kubernetes/portforward/resource_forwarder.go)
2. Amend `WatchingPodForwarder` to take a pod/port-filter to allow determining if a port is
   a debug port, as determined by the `debug.cloud.google.com` annotation.
3. Somehow turn `--port-forward` into string slice argument, if possible!
4. Add support to the skaffold.yaml schema
5. Add support to the global configuration schema.

___


## Integration test plan

Please describe what new test cases you are going to consider.

1. `skaffold dev` should forward user-defined ports (no services, not container ports).
2. `skaffold dev --port-forward` should forward services (no container ports).
3. `skaffold dev --port-forward-modes=off` should forward nothing.
4. `skaffold dev --port-forward-modes=X` for X={user,debug,pods,services}` should only forward those items.
5. `skaffold debug` should forward user-defined ports and debug ports (no services, no other container ports).
6. `skaffold debug --port-forward` should forward user-defined ports, debug ports, and services (no container ports).
7. `skaffold debug --port-forward-modes=off` should forward nothing.
8. `skaffold debug --port-forward-modes=X` for X={user,debug,pods,services}` should only forward those items.
