### Example: inferred file sync using the ko builder

This example uses
[inferred file sync](https://skaffold.dev/docs/pipeline-stages/filesync/#inferred-sync-mode)
for static assets with the
[`ko` builder](https://skaffold.dev/docs/pipeline-stages/builders/ko/)
for a Go web app.

To observe the behavior of file sync, run this command:

```shell
skaffold dev
```

Try changing the HTML file in the `kodata` directory to see how Skaffold
syncs the file.

If change the the `main.go` file, Skaffold will rebuild and redeploy the image.
