### Example: using the envTemplate tag policy

This example reuses the image name and uses an environment variable `FOO` to tag the image.
The way you configure it in `skaffold.yaml` is the following build stanza:

```yaml
build:
     artifacts:
     - image: skaffold-example
     tagPolicy:
       envTemplate:
         template: "{{.IMAGE_NAME}}:{{.FOO}}"
```

1. define `tagPolicy` to be `envTemplate`
2. use [go templates](https://golang.org/pkg/text/template) syntax
3. The `IMAGE_NAME` variable is built-in and reuses the value defined in the artifacts' `image`.