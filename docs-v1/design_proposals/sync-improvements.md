# Sync improvements

* Author(s): Cornelius Weig (@corneliusweig)
* Design Shepherd: Tejal Desai (@tejal29)
* Date: 03/20/2019
* Status: Reviewed

## Background

Currently, skaffold config supports `artifact.sync` as a way to sync files
directly to pods. So far, artifact sync requires a specification of sync
patterns like

```yaml
sync:
  '*.js': app/
```

This is error prone and unnecessarily hard to use because the destination can often be determined by the builder. For example a destination path is
already contained in the Dockerfile for docker build. (see #1166, #1581).
In addition, the syncing needs to handle special cases for globbing and often
requires a long list of sync patterns (#1807).

Furthermore, builders should be able to inform Skaffold about additional sync paths (#1704).
This will result in much faster deploy cycles, without users having to bother about these optimizations.

#### Problems with existing syntax
The current `sync` config syntax has the following problems:

- The flattening of the file structure at the destination is surprising. For example:
  ```yaml
  sync:
    'src/**/*.js': dest/
  ```
  will put all `.js` files in the `dest` folder.
- The triple-star syntax is is quite implicit and a surprising extension of the standard globbing syntax. For example:
  ```yaml
  sync:
    'src/b/c/***/*.py': dest/
  ```
  will recreate subfolders _below_ `src/b/c` at `dest/` and sync `.py` files.
  The triple-star therefore triggers two actions which should not depend on each other:
  - do not flatten the directory structure
  - strip everything on the source path up to `***`
- The current syntax does not allow extension to further sync modes unless further magic strings are introduced.
  
The syntax should therefore be revised.

## Design

Skaffold sync shall have three different modes:

1. _manual_: the user specifies both the source and destination.
3. _auto-inferred destination_: the user specifies the syncable files, and Skaffold infers their destination in the container, e.g based on Dockerfile
2. _smart_: builder recommended sync rules for files, e.g. jib specifies what to sync and provides both the source and dest of files to sync. The user does not have to specify anything.

The scope of this design proposal is the _direct_ and _inferred_ mode.
The smart sync mode is out of scope for this document but may be mentioned at times.

### Goals
The sync mechanism in skaffold should be able to infer the destination location of local files, when the user opts-in to do so.

For example: given the following Dockerfile

```dockerfile
FROM nginx
WORKDIR /var
COPY static *.html www/
ADD nginx.conf .
ADD nginx.conf /etc/
```

The sync locations should be inferred as
- `static/page1.html` -> `/var/www/page1.html` (! no static folder at destination !)
- `static/assets/style.css` -> `/var/www/assets/style.css`
- `index.html` -> `/var/www/index.html`
- `nginx.conf` -> `/var/nginx.conf`, `/etc/nginx.conf`

A fundamental change compared to the current sync mechanism is that one source path may by sync'ed to multiple destinations in the future.

### Assumptions

Builders know where to put static files in the container.
This information needs to be pulled from the builders and made available in Skaffold.
Currently, we know how to do this for Docker artifacts (@r2d4 suggested this in #1166).
Jib transforms some input files and includes others as static resources.
The specific sync maps need to be detailed out but are straightforward.
Whether the bazel builder supports this, is unclear at the moment.

### Interface changes

The builder implementations need to fulfill a new requirement:
They need to provide a default sync map for changed or deleted files.
For that, the `Builder` interface (pkg/skaffold/build/build.go) will receive the new method `SyncMap`.

```go
type Builder interface {
   Labels() map[string]string

   Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]Artifact, error)

   DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) ([]string, error)

   // ADDED
   SyncMap(ctx context.Context, artifact *latest.Artifact) (map[string][]string, error)
}
```

`SyncMap` returns a map whose keys are paths of source files for the container.
These host paths will be relative to the workingdir of the respective artifact.
The map values list all absolute remote destination paths where the input files are placed in the container.
This must be a list, because a single input file may be copied to several locations in the container.
Note that this does not mean that every file will be sync'ed.
The intent to sync a file needs to be explicitly specified by user-provided sync rules (see config changes).
In the future, some builders may specify default sync rules, but this is out of scope for this document.

For example, given the above Dockerfile, `SyncMap` will return something like
```go
map[string][]string{
  "static/page1.html": []string{"/var/www/page1.html"},
  "static/assets/style.css": []string{"/var/www/assets/style.css"},
  "index.html": []string{"/var/www/index.html"},
  "nginx.conf": []string{"/var/nginx.conf", "/etc/nginx.conf"},
}
```

Unsupported builders will return a not-supported error.

### Config changes

The `artifact.sync` config needs to support additional requirements:

- Support existing use-cases for backwards compatibility, i.e.
  - Glob patterns on the host path
  - Transplant a folder structure, ~~or flatten the folder structure~~[this behavior was a bug and will be removed]
- Offer a way to use destination inference.
- Provide a concise and unambiguous configuration syntax to avoid the problems outlined [above](#Background).
- Be open for extension, such as the `smart` sync mode.

#### Suggestion
The `artifact.sync` config will be upgraded as follows:

```yaml
sync:
  # Existing use-case: Copy all js under 'src' into 'dest', re-creating the directory structure below 'src'
  # This corresponds to the current `'src/***/*.js': app/`
  manual:         # oneOf=sync
  - src: 'src/**/*.js' # required
    dest: app/         # required
    strip: 'src/'      # optional, default to ""

  # New use-case: Copy all python files and infer their destination.
  infer:          # oneOf=sync
  - '**/*.py'
    
  # New use-case: smart sync mode for jib (out of scope)
  smart: {}       # oneOf=sync
```

In the `sync` config, exactly one of the modes `manual` or `infer` may be specified, but not both.
Both `manual` and `infer` contain a list of sync rules.
When determining the destination for a given file, all sync rules will be considered and not just the first match.
Thus, a single file may be copied to multiple destinations.

Error cases:
- It should be an error to specify `strip` with a prefix which which does not match the static prefix of the `from` field.
  ```yaml
  - src: 'src/**/*.js'
    dest: dest/
    strip: src/sub        # ERR
  - src: 'src/**/*.js'
    dest: dest/
    strip: app/           # ERR
  ```

Open questions about the configuration:

**\<Is the syntax clear enough?\>**
Is the syntax clear?
Is the configuration intuitive to use?

Resolution:  YES

**\<Does it improve clarity to have the `inferTo` field?\>**
A different approach could be to fall back to inference, if the `to` field is left blank.
This would have the advantage that there needs to be no validation about having `to` and `inferTo` in the same rule, which should cause an error.

Resolution: Obsolete, the improved version excludes this error case.

**\<Error for `strip` and `flatten`\>**
Given the new sync config, users may try to combine various flags in unintended ways.
For example, should `strip` and `flatten` in the same rule always be an error?
My intuition would say, that `strip+flatten` is the same as `flatten`, but this should to be discussed.

Resolution: Obsolete.

**\<Do we need an `excludes` rule?\>**
The current config only suggests additive rules. For some applications it might be easier to also remove matched host paths from the syncable set. For example:
```yaml
sync:
# include all files below src...
- src: 'src/**/*'
  dest: app/
# ...but do not consider js files for sync
- src: 'src/**/*.js'
  exclude: true
```
Also see #1766.

Resolution: Descoped, open separate issue


#### Migration plan
An automatic schema upgrade shall be implemented.
The former default behavior to flatten directories at the destination will no longer be supported.
This is possible, because sync is still an alpha feature.
Users who are using incompatible sync patterns will receive a warning during upgrade.

### Open Issues/Question

**\<Overlapping copies\>**
Overlapping copies can be caused by at least two ways:

1. The `COPY/ADD` instructions in the Dockerfile may copy to the same destination. For example:
   ```dockerfile
   COPY foo /foo
   COPY bar/ /   # bar contains a file called "foo"
   ```
2. Two sync rules may copy files to the same destination. For example:
   ```yaml
   sync:
   - src: foo
     dest: /baz
   - src: bar
     dest: /baz
   ```

Should such cases be treated as errors, warnings, or not be detected at all?

Resolution: No detection which is the current behavior.

**\<Transitive copies\>**
Multi-stage Dockerfiles can lead to quite contrived scenarios. For example:
```dockerfile
FROM scratch
ADD foo baz
FROM scratch
COPY --from=0 baz bar
```
Ideally, the sync destination should be inferred as `foo -> bar`.

This is quite an edge case and can easily be circumvented.
However, it is quite complex to implement.
Should we bother about this special case?

Resolution: not now

**\<Other builders\>**
We don't know how to obtain a mapping of local paths to remote paths for other builders (jib, bazel).
This may even have to be implemented upstream.
Until we can support those builders, we to handle the case when a user tries to use inference with those builders.

- Option 1: bail out with an error
- Option 2: copy the file into the container with the destination set to the relative path wrt the context directory.

Resolution: Option 1, ideally during schema validation.

**\<Upgrade hint\>**
The path inference allows to simplify the sync specification considerably.
Besides, subdirectory syncing makes the sync functionality even feasible for large projects that need to sync many folders into the container.
Therefore, we could consider to advertise this new functionality when running Skaffold or when an old-style sync rule as found.

Resolution: Agree. We should support auto-fixing.

**\<SyncMap signature\>**
The new `SyncMap` interface provides some sync rules from the builder.
However, the suggested signature is `func() map[string]string` with the convention to not flatten directories.
To be more in line with the other sync functionality, it could also return `[]SyncRule`.
Although being clearer, this has the downside of introducing a dependency from the builder package to the pipeline config package.
If going that route, we should probably duplicate this struct in a proper place.
However, is it necessary at all, or is the current signature good enough?

Resolution: This is an implementation detail.

## Implementation plan

1. Upgrade `artifact.sync` to the new sync rule structure and test schema validation.
   Should already support multiple destination paths. (#1847)
2. Add inference logic for docker. (#2084)
3. Support support sync rules with inference. (former #1812, will become separate PR)
   - includes integration tests
   - includes doc update
   - includes example showcase
4. (out of scope) Smart sync mode by allowing builders to specify default sync rules.
   - Support smart sync for jib.
   - Possibly support smart sync for Dockerfile when `ADD` or `COPY` are not followed by a `RUN` command.
5. (future) Add sync map logic for jib and examples.
6. (future) Add sync map logic for bazel and examples.

## Integration test plan

- **implementation step 3** Change one of the existing sync examples so that it uses inference (e.g. #1826).
  The test should change a local source file. Expect that the change will be reflected in the container.

- **implementation step 3** Set up automatic destination syncing and delete a local input file.
  Expect that the input file is also deleted in the container.
  _Update_: This expectation cannot be met, because a deleted file is no longer contained in the inferred syncmap. Thus file deletion with inferred sync mode must trigger a rebuild.

- **implementation step 4** Add a test case that that features builder plugin sync patterns.

## Glossary

- _sync map_: specifies a mapping of source files to possibly multiple destinations in the container. It does not specify the intent to sync.
- _sync rule_: specifies the intent to sync files. For manual sync, it must also specify the destination of sync'ed files.
