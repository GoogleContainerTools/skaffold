### Example: using the envTemplate tag policy

This example uses an environment variable `FOO` to tag the image.
The way you configure it in `skaffold.yaml` is the following build stanza:

```yaml
build:
     artifacts:
     - image: skaffold-example
     tagPolicy:
       envTemplate:
         template: "{{.FOO}}"
```

1. define `tagPolicy` to be `envTemplate`
2. use [go templates](https://golang.org/pkg/text/template) syntax