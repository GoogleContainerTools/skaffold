---
title: "File sync"
linkTitle: "File sync"
weight: 40
---

This page discusses how to set up file sync for files that don't require full rebuild.

{{< alert title="Note" >}}
File sync is alpha and may change between releases.
{{< /alert >}}

Skaffold supports copying changed files to a deployed containers so as to avoid the need to
rebuild, redeploy, and restart the corresponding pod.  The file copying is enabled
by adding a `sync` section with _sync rules_ to the `artifact` in the `skaffold.yaml`.

The following example will cause any changes to JavaScript files under the _context_ directory
to be copied to the deployed container into the container's `WORKDIR`.

```yaml
apiVersion: skaffold/v1beta8
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/node-example
    context: node
    sync:
      '.filebaserc': .
      '*.html': static
      '**/*.png': assets
      '***/*.md': content
```
A double-asterisk (`**/`) applies recursively to all subdirectories but flattens the result,
stripping the subdirectory structure.
A triple-asterisk (`***/`) applies recursively to all subdirectories but retains
the subdirectory structure.  

Under the hood, Skaffold monitors and creates a tar file with changed files that match
the sync rules.  This tar file is sent and extracted on the corresponding containers. 

### Limitations

File sync has some limitations:

  - File sync can only update files that can be modified by the container's configured User ID.
  - File sync requires the `tar` command to be available in the container.
  - Only local source files can be synchronized: files created by the builder will not be copied.

{{% todo 1076 %}}
