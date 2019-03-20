# Sync improvements

* Author(s): Cornelius Weig (@corneliusweig)
* Design Shepherd: Tejal Desai (@tejal29)
* Date: 03/20/2019
* Status: Draft

## Background

Currently, skaffold config supports `artifact.sync` as a way to sync files
directly to pods. So far, artifact sync requires a specification of sync
patterns like

```yaml
sync:
  '*.js': app/
```

This is error prone and unnecessarily hard to use, because the destination is
already contained in the Dockerfile for docker build. (see #1166, #1581).
In addition, the syncing needs to handle special cases for globbing and often
requires a long list of sync patterns (#1807).

Furthermore, builders should be able to inform Skaffold about additional sync paths (#1704).
This will result in much faster deploy cycles, without users having to bother about these optimizations.

## Design

### Goals
The sync mechanism in skaffold should be able to infer the destination location of local files.

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
Whether jib and bazel builders can provide that information too, is unclear at the moment.

### Interface changes

There are at least two relevant changes

1. When builders are queried for their dependencies, they must provide a list of destinations together with local source paths. The signature of `Builder.GetDependencies` (pkg/skaffold/build/build.go) therefore needs to change to:
   ```go
   type Builder interface {
       Labels() map[string]string

       Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]Artifact, error)

       // OLD
       // DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) ([]string, error)
       // NEW
       DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) (map[string][]string, error)
   }
   ```
   
   So far, `DependenciesForArtifact` returned a list of local paths (relative to the working directory of the artifact) that are input files for the container.
   Now, `DependenciesForArtifact` serves two purposes. As before, the map keys are relative local paths of input files for the container.
   In addition, the map values list all absolute remote destination paths where the input files are placed in the container.
   This has to be a list, because a single input file may be copied to several locations in the container.
   
   Thus, `DependenciesForArtifact` serves two purposes: firstly, it lists all input files to watch for changes.
   Secondly, it provides default sync destinations for changed or deleted files.
   Note that this does not mean, that every file will be sync'ed. If a file is entitled for syncing depends on the user-provided sync rules.
   
   For example, given the above Dockerfile, it will return something like
   ```go
   map[string][]string{
    "static/page1.html": []string{"/var/www/page1.html"},
    "static/assets/style.css": []string{"/var/www/assets/style.css"},
    "index.html": []string{"/var/www/index.html"},
    "nginx.conf": []string{"/var/nginx.conf", "/etc/nginx.conf"},
   }
   ```
   
2. To support additional sync maps from builders, the builder interface will need an additional method `SyncMap`:
   ```go
   type Builder interface {
       Labels() map[string]string

       Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]Artifact, error)

       DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) (map[string][]string, error)
    
       // ADDED
       SyncMap() map[string]string
   }
   ```
   
   The new `SyncMap` function shall return pairs of
   
   - **key**: a glob pattern of local files to sync into the container
   - **value**: a destination path in the container
   
   By convention, the sync rule from a builder does never flatten directories but does subdir syncing instead (also see open question below).

### Config changes

The `artifact.sync` config needs to support additional requirements:

- Support existing use-cases for backwards compatibility, i.e.
  - Glob patterns on the host path
  - Transplant a folder structure, or flatten the folder structure
- Offer a way to use destination inference.
- Avoid ambiguity

#### Problems with existing syntax
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

#### Suggestion
These above problems can be made less surprising by a more explicit configuration syntax.
The `artifact.sync` will be upgraded to a list of *sync rules*.

```yaml
sync:
# Existing use-case: Copy all js files under 'src' flat into 'app'
# This corresponds to the current `'src/**/*.js': app/`
- from: 'src/**/*.js'
  to: app/
  flatten: true  # default to false
  
# Existing use-case: Copy all js under 'src' into 'dest', re-creating the directory structure below 'src'  
# This corresponds to the current `'src/***/*.js': dest/`
- from: 'src/**/*.js'
  to: dest/
  strip: 'src'   # default to ""
  
# New use-case: Copy all python files and infer their destination.
- from: '**/*.py'
  inferTo: true   # <- tbd, default to false
```

When determining the destination for a given file, all sync rules will be considered.
Thus, a single file may be copied to multiple destinations.

Error cases:
- It should be an error to specify `inferTo=true` together with `to` in the same sync rule.
  ```yaml
  - from: '*.js'
    to: dest/
    inferTo: true
  ```
- It should be an error to specify either of `flatten` or `strip` together with `inferTo`.
  ```yaml
  - from: '*.js'
    inferTo: true
    flatten: true        # ERR
  - from: 'src/**/*.js'
    inferTo: true
    strip: src/          # ERR
  ```
- It should be an error to specify `strip` with a prefix which which does not match the static prefix of the `from` field.
  ```yaml
  - from: 'src/**/*.js'
    to: dest/
    strip: src/sub        # ERR
  - from: 'src/**/*.js'
    to: dest/
    strip: app/           # ERR
  ```

Open questions about the configuration:

**\<Is the syntax clear enough?\>**
Is the syntax clear?
Is the configuration intuitive to use?
Is there any ambiguity which needs attention?

Resolution: __Not Yet Resolved__

**\<Does it improve clarity to have the `inferTo` field?\>**
A different approach could be to fall back to inference, if the `to` field is left blank.
This would have the advantage that there needs to be no validation about having `to` and `inferTo` in the same rule, which should cause an error.

Resolution: __Not Yet Resolved__
   
**\<Error for `strip` and `flatten`\>**
Given the new sync config, users may try to combine various flags in unintended ways.
For example, should `strip` and `flatten` in the same rule always be an error?
My intuition would say, that `strip+flatten` is the same as `flatten`, but this should to be discussed.

Resolution: __Not Yet Resolved__
   
   
#### Migration plan
The new configuration supports all existing use-cases and is fully backwards compatible.
An automatic schema upgrade shall be implemented.

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
   - from: foo
     to: /baz
   - from: bar
     to: /baz
   ```
   
Should such cases be treated as errors, warnings, or not be detected at all?

Resolution: __Not Yet Resolved__

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
   
Resolution: __Not Yet Resolved__

**\<Other builders\>**
We don't know how to obtain a mapping of local paths to remote paths for other builders (jib, bazel).
This may even have to be implemented upstream.
Until we can support those builders, we to handle the case when a user tries to use inference with those builders.

- Option 1: bail out with an error
- Option 2: copy the file into the container with the destination set to the relative path wrt the context directory.

Resolution: __Not Yet Resolved__

**\<Upgrade hint\>**
The path inference allows to simplify the sync specification considerably.
Besides, subdirectory syncing makes the sync functionality even feasible for large projects that need to sync many folders into the container.
Therefore, we could consider to advertise this new functionality when running Skaffold or when an old-style sync map as found.

Resolution: __Not Yet Resolved__

**\<SyncMap signature\>** 
The new `SyncMap` interface provides some sync rules from the builder.
However, the suggested signature is `func() map[string]string` with the convention to not flatten directories.
To be more in line with the other sync functionality, it could also return `[]SyncRule`.
Although being clearer, this has the downside of introducing a dependency from the builder package to the pipeline config package.
If going that route, we should probably duplicate this struct in a proper place.
However, is it necessary at all, or is the current signature good enough?


## Implementation plan

1. Upgrade `artifact.sync` to the new sync rule structure and test schema validation.
   Should already support multiple destination paths. (#1847)
2. Add inference logic for docker and examples. (#1812)
3. Support support sync rules with inference. (former #1812, will become separate PR)
4. Finally, support builder plugins to add sync patterns.
5. (future) Add inference logic for jib and examples.
6. (future) Add inference logic for bazel and examples.

## Integration test plan

- **step 3** Change one of the existing sync examples so that it uses inference (e.g. #1826).
  The test should change a local source file. Expect that the change will be reflected in the container.

- **step 3** Set up automatic destination syncing and delete a local input file.
  Expect that the input file is also deleted in the container.

- **step 4** Add a test case that that features builder plugin sync patterns.
