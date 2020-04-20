# Duck Types

Knative leverages duck-typing to interact with resources inside of Kubernetes
without explicit knowlage of the full resource shape. `knative/pkg` defines two
duck types that are used throughout Knative: `Addressable` and `Source`.

For APIs leveraging `ObjectReference`, the context of the resource in question
identifies the duck-type. To enable the case where no `ObjectRefrence` is used,
we have labeled the Custom Resource Definition with the duck-type. Those labels
are as follows:

| Label                               | Duck-Type                                                                     |
| ----------------------------------- | ----------------------------------------------------------------------------- |
| `duck.knative.dev/addressable=true` | [Addressable](https://godoc.org/knative.dev/pkg/apis/duck/v1#AddressableType) |
| `duck.knative.dev/binding=true`     | [Binding](https://godoc.org/knative.dev/pkg/apis/duck/v1alpha1#Binding)       |
| `duck.knative.dev/source=true`      | [Source](https://godoc.org/knative.dev/pkg/apis/duck/v1#Source)               |

## Addressable Shape

Addressable is expected to be the following shape:

```yaml
apiVersion: group/version
kind: Kind
status:
  address:
    url: http://host/path?query
```

## Binding Shape

Binding is expected to be in the following shape:

(with direct subject)

```yaml
apiVersion: group/version
kind: Kind
spec:
  subject:
    apiVersion: group/version
    kind: SomeKind
    namespace: the-namespace
    name: a-name
```

(with indirect subject)

```yaml
apiVersion: group/version
kind: Kind
spec:
  subject:
    apiVersion: group/version
    kind: SomeKind
    namespace: the-namespace
    selector:
      matchLabels:
        key: value
```

## Source Shape

Source is expected to be in the following shape:

(with ref sink)

```yaml
apiVersion: group/version
kind: Kind
spec:
  sink:
    ref:
      apiVersion: group/version
      kind: AnAddressableKind
      name: a-name
  ceOverrides:
    extensions:
      key: value
status:
  observedGeneration: 1
  conditions:
    - type: Ready
      status: "True"
  sinkUri: http://host
```

(with uri sink)

```yaml
apiVersion: group/version
kind: Kind
spec:
  sink:
    uri: http://host/path?query
  ceOverrides:
    extensions:
      key: value
status:
  observedGeneration: 1
  conditions:
    - type: Ready
      status: "True"
  sinkUri: http://host/path?query
```

(with ref and uri sink)

```yaml
apiVersion: group/version
kind: Kind
spec:
  sink:
    ref:
      apiVersion: group/version
      kind: AnAddressableKind
      name: a-name
    uri: /path?query
  ceOverrides:
    extensions:
      key: value
status:
  observedGeneration: 1
  conditions:
    - type: Ready
      status: "True"
  sinkUri: http://host/path?query
```
