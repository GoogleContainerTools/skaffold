---
title: "File Sync"
linkTitle: "File Sync"
weight: 40
featureId: sync
aliases: [/docs/how-tos/filesync]
---

Skaffold supports copying changed files to a deployed container so as to avoid the need to rebuild, redeploy, and restart the corresponding pod.
The file copying is enabled by adding a `sync` section with _sync rules_ to the `artifact` in the `skaffold.yaml`.
Under the hood, Skaffold creates a tar file with changed files that match the sync rules.
This tar file is sent to and extracted on the corresponding containers.

Multiple types of sync are supported by Skaffold:

 + `manual`: The user must specify both the files in their local workspace and the destination in the running container.
   This is supported by every type of artifact.

 + `infer`: The destinations for each changed file is inferred from instructions in a Dockerfile.
   This is supported by docker and kaniko artifacts and also for custom artifacts that declare a
   dependency on a Dockerfile.

 + `auto`: Skaffold automatically configures the sync.  This is only supported by Jib and Buildpacks artifacts.

### Manual sync mode

A manual sync rule must specify the `src` and `dest` field.
The `src` field is a glob pattern to match files relative to the artifact _context_ directory, which may contain `**` to match nested files.
The `dest` field is the absolute or relative destination path in the container.
If the destination is a relative path, an absolute path will be inferred by prepending the path with the container's `WORKDIR`.
By default, matched files are transplanted with their whole directory hierarchy below the artifact context directory onto the destination.
The optional `strip` field can cut off some levels from the directory hierarchy.
The following example showcases manual filesync:

{{% readfile file="samples/filesync/filesync-manual.yaml" %}}

- The first rule synchronizes the file `.filebaserc` to the `/etc` folder in the container.
- The second rule synchronizes all `html` files in the `static-html` folder into the `<WORKDIR>/static` folder in the container.
  Note that this pattern does not match files in sub-folders below `static-html` (e.g. `static-html/a.html` but not `static-html/sub/a.html`).
- The third rule synchronizes all `png` files from any sub-folder into the `assets` folder on the container.
  For example, `img.png` ↷ `assets/img.png` or `sub/img.png` ↷ `assets/sub/img.png`.
- The last rule synchronizes all `md` files below the `content/en` directory into the `content` folder on the container.
  The `strip` directive ensures that only the directory hierarchy below `content/en` is re-created at the destination.
  For example, `content/en/index.md` ↷ `content/index.md` or `content/en/sub/index.md` ↷ `content/sub/index.md`.

### Inferred sync mode

For docker artifacts, Skaffold knows how to infer the desired destination from the artifact's `Dockerfile`.
To enable syncing, you only need to specify which files are eligible for syncing in the sync rules.
The sync rules for inferred sync mode is just a list of glob patterns.
The following example showcases this filesync mode:

Given a Dockerfile such as

```Dockerfile
FROM hugo

ADD .filebaserc /etc/
ADD assets assets/
COPY static-html static/
COPY content/en content/
```

And a `skaffold.yaml` with the following sync configuration:

{{% readfile file="samples/filesync/filesync-infer.yaml" %}}

- The first rule synchronizes the file `.filebaserc` to `/etc/.filebaserc` in the container.
- The second rule synchronizes all `html` files in the `static-html` folder into the `<WORKDIR>/static` folder in the container.
  Note that this pattern does not match files in sub-folders below `static-html` (e.g. `static-html/a.html` but not `static-html/sub/a.html`).
- The third rule synchronizes any `png`. For example if `assest/img.png` ↷ `assets/img.png` or `static-html/imgs/demo.png` ↷ `static/imgs/demo.png`.
- The last rule enables synchronization for all `md` files below the `content/en`.
  For example, `content/en/sub/index.md` ↷ `content/sub/index.md` but _not_ `content/en_GB/index.md`.
  
Inferred sync mode only applies to modified and added files.
File deletion will always cause a complete rebuild.

### Auto sync mode

#### Buildpacks

Skaffold requires special collaboration from the Buildpacks for the `auto` sync to work.
The [gcr.io/buildpacks/builder:v1](https://github.com/GoogleCloudPlatform/buildpacks) supports Skaffold
out of the box, currently for Go and NodeJS.

Cloud Native Buildpacks set a `io.buildpacks.build.metadata` label on the images they create.
This labels points to json description of the [Bill-of-Materials, aka BOM](https://github.com/buildpacks/spec/blob/master/buildpack.md#bill-of-materials-toml) of the build.
In the BOM, under the `metadata.devmode.sync` key, Buildpacks that want to collaborate with Skaffold
have to output the sync rules based on their exploration of the source and the build process they had to apply to it.
Those sync rules will then be used by Skaffold without the user having to configure them manually.

Another thing the Buildpacks have to do is support the `GOOGLE_DEVMODE` environment variable. Skaffold will
set it to `1` when running `skaffold dev` with sync configured to `auto: {}`. The Buildpacks can then use that
signal to change the way the application is built so that it reloads the changes or rebuilds the app on each change.

#### Jib

Jib integration with Skaffold allows for zero-config `auto` sync. In this mode, Jib will sync your class files, resource files, and Jib's "extra directories" files to a remote container as changes are made. It can only be used with Jib in the default build mode (exploded) for non-WAR applications. It was primarily designed around [Spring Boot Developer Tools](https://docs.spring.io/spring-boot/docs/current/reference/html/using-spring-boot.html#using-boot-devtools), but can work with any embedded server that can reload/restart.

Check out the [Jib Sync example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/jib-sync) for more details.

## Limitations

File sync has some limitations:

  - File sync can only update files that can be modified by the container's configured User ID.
  - File sync requires the `tar` command to be available in the container.
  - Only local source files can be synchronized: files created by the builder will not be copied.
  - It is currently not allowed to mix `manual`, `infer` and `auto` sync modes.
    If you have a use-case for this, please let us know!
