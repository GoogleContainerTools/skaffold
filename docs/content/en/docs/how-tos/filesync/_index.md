---
title: "File sync"
linkTitle: "File sync"
weight: 40
---

This page discusses how to set up file sync for files that don't require full rebuild.

{{< alert title="Note" >}}
File sync is alpha and may change between releases.
{{< /alert >}}

Skaffold supports copying changed files to a deployed container so as to avoid the need to rebuild, redeploy, and restart the corresponding pod.
The file copying is enabled by adding a `sync` section with _sync rules_ to the `artifact` in the `skaffold.yaml`.
Under the hood, Skaffold creates a tar file with changed files that match the sync rules.
This tar file is sent to and extracted on the corresponding containers. 

### Manual sync mode

A manual sync rule must specify the `src` and `dest` field.
The `src` field is a glob pattern to match files relative to the artifact _context_ directory, which may contain `**` to match nested files.
The `dest` field is the absolute or relative destination path in the container.
If the destination is a relative path, an absolute path will be inferred by prepending the path with the container's `WORKDIR`.
By default, matched files are transplanted with their whole directory hierarchy below the artifact context directory onto the destination.
The optional `strip` field can cut off some levels from the directory hierarchy.
The following example showcases manual filesync:

{{% readfile file="samples/filesync/filesync.yaml" %}}

- The first rule synchronizes the file `.filebaserc` to the `/etc` folder in the container.
- The second rule synchronizes all `html` files in the `static-html` folder into the `<WORKDIR>/static` folder in the container.
  Note that this pattern does not match files in sub-folders below `static-html` (e.g. `static-html/a.html` but not `static-html/sub/a.html`).
- The third rule synchronizes all `png` files from any sub-folder into the `assets` folder on the container.
  For example, `img.png` ↷ `assets/img.png` or `sub/img.png` ↷ `assets/sub/img.png`.
- The last rule synchronizes all `md` files below the `content/en` directory into the `content` folder on the container.
  The `strip` directive ensures that only the directory hierarchy below `content/en` is re-created at the destination.
  For example, `content/en/index.md` ↷ `content/index.md` or `content/en/sub/index.md` ↷ `content/sub/index.md`.

Currently, there is only manual filesync mode, but a mode with destination inference is already in the making.

## Limitations

File sync has some limitations:

  - File sync can only update files that can be modified by the container's configured User ID.
  - File sync requires the `tar` command to be available in the container.
  - Only local source files can be synchronized: files created by the builder will not be copied.
