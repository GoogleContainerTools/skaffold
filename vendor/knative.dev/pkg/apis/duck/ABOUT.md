# Knative Duck Typing

![A Trojan Duck](images/Knative-Duck0.png)

**Figure 1:** How to integrate with Knative.

## Problem statement

In Knative, we want to support
[loose coupling](https://docs.google.com/presentation/d/1KxKAcIZyblkXbpdGVCgmzfDDIhqgUcwsa0zlABfvaXI/edit#slide=id.p)
of the building blocks we are releasing. We want users to be able to use these
building blocks together, but also support composing them with non-Knative
components as well.

Unlike Knative’s
[pluggability story](https://docs.google.com/presentation/d/10KWynvAJYuOEWy69VBa6bHJVCqIsz1TNdEKosNvcpPY/edit#slide=id.p)
(for replacing subsystems within a building block), we do not want to require
that the systems with which we compose have **identical** APIs (distinct
implementations). However, we do need a way of accessing (reading / writing)
certain **_pieces_** of information in a structured way.

**Enter [duck typing](https://en.wikipedia.org/wiki/Duck_typing)**. We will
define a partial schema, to which resource authors will adhere if they want to
participate within certain contexts of Knative.

For instance, consider the partial schema:

```yaml
foo:
  bar: <string>
```

Both of these resources implement the above duck type:

```yaml
baz: 1234
foo:
  bar: asdf
blah:
  blurp: true
```

```yaml
field: running out of ideas
foo:
  bar: a different string
another: you get the point
```

### Reading duck-typed data

At a high-level, reading duck-typed data is very straightforward: using the
partial object schema deserialize the resource ignoring unknown fields. The
fields we care about can then be accessed through the structured object that
represents the duck type.

### Writing duck-typed data

How to write duck-typed data is less straightforward because we do not want to
clobber every field we do not know about. To accomplish this, we will lean on
Kubernetes’ well established patching model.

First, we read the resource we intend to modify as our duck type. Keeping a copy
of the original, we then modify the fields of this duck typed resource to
reflect the change we want. Lastly, we synthesize a JSON Patch of the changes
between the original and the final version and issue a Patch to the Kubernetes
API with the delta.

Since the duck type inherently contains a subset of the fields in the resource,
the resulting JSON Patch can only contain fields relevant to the resource.

## Example: Reading Knative-style Conditions

In Knative, we follow the Kubernetes API principles of using `conditions` as a
key part of our resources’ status, but we go a step further in
[defining particular conventions](https://github.com/knative/serving/blob/master/docs/spec/errors.md#error-conditions-and-reporting)
on how these are used.

To support this, we define:

```golang
type KResource struct {
        metav1.TypeMeta   `json:",inline"`
        metav1.ObjectMeta `json:"metadata,omitempty"`

        Status KResourceStatus `json:"status"`
}

type KResourceStatus struct {
        Conditions Conditions `json:"conditions,omitempty"`
}

type Conditions []Condition

type Condition struct {
  // structure adhering to K8s API principles
  ...
}
```

We can now deserialize and reason about the status of any Knative-compatible
resource using this partial schema.

## Example: Mutating Knative CRD Generations

In Knative, all of our resources define a `.spec.generation` field, which we use
in place of `.metadata.generation` because the latter was not properly managed
by Kubernetes (prior to 1.11 with `/status` subresource). We manage bumping this
generation field in our webhook if and only if the `.spec` changed.

To support this, we define:

```golang
type Generational struct {
        metav1.TypeMeta   `json:",inline"`
        metav1.ObjectMeta `json:"metadata,omitempty"`

        Spec GenerationalSpec `json:"spec"`
}

type GenerationalSpec struct {
        Generation Generation `json:"generation,omitempty"`
}

type Generation int64
```

Using this our webhook can read the current resource’s generation, increment it,
and generate a patch to apply it.

## Example: Mutating Core Kubernetes Resources

Kubernetes already uses duck typing, in a way. Consider that `Deployment`,
`ReplicaSet`, `DaemonSet`, `StatefulSet`, and `Job` all embed a
`corev1.PodTemplateSpec` at the exact path: `.spec.template`.

Consider the example duck type:

```yaml
type PodSpecable corev1.PodTemplateSpec

type WithPod struct { metav1.TypeMeta   `json:",inline"` metav1.ObjectMeta
`json:"metadata,omitempty"`

Spec WithPodSpec `json:"spec,omitempty"` }

type WithPodSpec struct { Template PodSpecable `json:"template,omitempty"` }
```

Using this, we can access the PodSpec of arbitrary higher-level Kubernetes
resources in a very structured way and generate patches to mutate them.
[See examples](https://github.com/knative/pkg/blob/07104dad53e803457a95306e5b1322024bd69af3/apis/duck/podspec_test.go#L49-L53).

_You can also see a sample controller that reconciles duck-typed resources
[here](https://github.com/mattmoor/cachier)._

## Conventions

Each of our duck types will consist of a single structured field that must be
enclosed within the containing resource in a particular way.

1. This structured field will be named <code>Foo<strong>able</strong></code>,
2. <code>Fooable</code> will be directly included via a field named
   <code>fooable</code>,
3. Additional skeletal layers around <code>Fooable</code> will be defined to
   fully define <code>Fooable</code>’s position within complete resources.

_You can see parts of these in the examples above, however, those special cases
have been exempted from the first condition for legacy compatibility reasons._

For example:

1. `type Conditions []Condition`
2. <code>Conditions Conditions
   `json:"<strong>conditions</strong>,omitempty"`</code>
3. <code>KResource -> KResourceStatus -> Conditions</code>

## Supporting Mechanics

We will provide a number of tools to enable working with duck types without
blowing off feet.

### Verification

To verify that a particular resource implements a particular duck type, resource
authors are strongly encouraged to add the following as test code adjacent to
resource definitions.

`myresource_types.go`:

```golang
package v1alpha1

type MyResource struct {
   ...
}
```

`myresource_types_test.go`:

```golang
package v1alpha1

import (
    "testing"

    // This is where supporting tools for duck-typing will live.
    "github.com/knative/pkg/apis/duck"

    // This is where Knative-provided duck types will live.
    duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
)

// This verifies that MyResource contains all the necessary fields for the
// given implementable duck type.
func TestType(t *testing.T) {
   err := duck.VerifyType(&MyResource{}, &duckv1alpha1.Conditions{})
   if err != nil {
     t.Errorf("VerifyType() = %v", err)
   }
}
```

\_This call will create a fully populated instance of the skeletal resource
containing the Conditions and ensure that the fields can 100% roundtrip through
<code>MyResource</code>.</em>

### Patching

To produce a patch of a particular resource modification suitable for use with
<code>k8s.io/client-[go/dynamic](https://goto.google.com/dynamic)</code>,
developers can write:

```golang
before := …
after := before.DeepCopy()
// modify "after"

patch, err := duck.CreatePatch(before, after)
// check err

bytes, err := patch.MarshalJSON()
// check err

dynamicClient.Patch(bytes)
```

### Informers / Listers

To be able to efficiently access / monitor arbitrary duck-typed resources, we
want to be able to produce an Informer / Lister for interpreting particular
resource groups as a particular duck type.

To facilitate this, we provide several composable implementations of
`duck.InformerFactory`.

```golang
type InformerFactory interface {
        // Get an informer/lister pair for the given resource group.
        Get(GroupVersionResource) (SharedIndexInformer, GenericLister, error)
}


// This produces informer/lister pairs that interpret objects in the resource group
// as the provided duck "Type"
dif := &duck.TypedInformerFactory{
        Client:       dynaClient,
        Type:         &duckv1alpha1.Foo{},
        ResyncPeriod: 30 * time.Second,
        StopChannel:  stopCh,
}

// This registers the provided EventHandler with the informer each time an
// informer/lister pair is produced.
eif := &duck.EnqueueInformerFactory{
        Delegate: dif,
        EventHandler: cache.ResourceEventHandlerFuncs{
             AddFunc: impl.EnqueueControllerOf,
             UpdateFunc: controller.PassNew(impl.EnqueueControllerOf),
        },
}

// This caches informer/lister pairs so that we only produce one for each GVR.
cif := &duck.CachedInformerFactory{
        Delegate: eif,
}
```

### Trackers

Informers are great when you have something like an `OwnerReference` to key off
of for the association (e.g. `impl.EnqueueControllerOf`), however, when the
association is looser e.g. `corev1.ObjectReference`, then we need a way of
configuring a reconciliation trigger for the cross-reference.

For this (generally) we have the `knative/pkg/tracker` package. Here is how it
is used with duck types:

```golang
        c := &Reconciler{
                Base:             reconciler.NewBase(opt, controllerAgentName),
                ...
        }
        impl := controller.NewImpl(c, c.Logger, "Revisions")

        // Calls to Track create a 30 minute lease before they must be renewed.
        // Coordinate this value with controller resync periods.
        t := tracker.New(impl.EnqueueKey, 30*time.Minute)
        cif := &duck.CachedInformerFactory{
                Delegate: &duck.EnqueueInformerFactory{
                        Delegate: buildInformerFactory,
                        EventHandler: cache.ResourceEventHandlerFuncs{
                                AddFunc:    t.OnChanged,
                                UpdateFunc: controller.PassNew(t.OnChanged),
                        },
                },
        }

        // Now use: c.buildInformerFactory.Get() to access ObjectReferences.
        c.buildInformerFactory = buildInformerFactory

        // Now use: c.tracker.Track(rev.Spec.BuildRef, rev) to queue rev
        // each time rev.Spec.BuildRef changes.
        c.tracker = t
```
