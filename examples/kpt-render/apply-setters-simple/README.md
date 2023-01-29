# apply-setters: Simple Example

### Overview

In this example, we will see how to apply desired setter values to the 
resource fields parameterized by `kpt-set` comments.

### Fetch the example package

Get the example package by running the following commands:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt-functions-catalog.git/examples/apply-setters-simple@apply-setters/v0.2
```

We use `ConfigMap` to configure the `apply-setters` function. The desired
setter values are provided as key-value pairs using `data` field where key is
the name of the setter(as seen in the reference comments) and value is the new
desired value for the tagged field.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: apply-setters-fn-config
data:
  replicas: "3"
  role: |
    - dev
    - prod
  tag: 1.16.2
```

### Function invocation

Invoke the function by running the following command:

```shell
$ kpt fn render apply-setters-simple
```

### Expected result

1. Check the value of field `replicas` is set to `3` in `Deployment` resource.
2. Check the value of field `image` is set to value `nginx:1.16.2` in `Deployment` resource.
3. Check the value of field `environments` is set to value `[dev, prod]` in `MyKind` resource.

#### Note:

Refer to the `create-setters` function documentation for information about creating setters.
