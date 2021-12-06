### Example: ko builder

This example uses the
[`ko` builder](https://skaffold.dev/docs/pipeline-stages/builders/ko/)
to build a container image for a Go app.

The included [Cloud Build](https://cloud.google.com/build/docs) configuration
file shows how users can set up a simple pipeline using `skaffold build` and
`skaffold deploy`, without having to create a custom builder image.
