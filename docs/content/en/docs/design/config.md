---
title: "Skaffold Pipeline"
linkTitle: "Skaffold Pipeline"
weight: 40
aliases: [/docs/concepts/config]
---

You can configure Skaffold with the Skaffold configuration file,
`skaffold.yaml`.  A single configuration consists of several different components:

| Component  | Description |
| ---------- | ------------|
| `apiVersion` | The Skaffold API version you would like to use. The current API version is {{< skaffold-version >}}. |
| `kind`  |  The Skaffold configuration file has the kind `Config`.  |
| `metadata`  |  Holds additional properties like the `name` of this configuration.  |
| `build`  |  Specifies how Skaffold builds artifacts. You have control over what tool Skaffold can use, how Skaffold tags artifacts and how Skaffold pushes artifacts. Skaffold supports using local Docker daemon, Google Cloud Build, Kaniko, or Bazel to build artifacts. See [Builders](/docs/pipeline-stages/builders) and [Taggers]({{< relref "/docs/pipeline-stages/taggers" >}}) for more information. |
| `test` |  Specifies how Skaffold tests artifacts. Skaffold supports [container-structure-tests](https://github.com/GoogleContainerTools/container-structure-test) to test built artifacts and custom tests to run custom commands as part of the development pipeline. See [Testers]({{< relref "/docs/pipeline-stages/testers" >}}) for more information. |
| `deploy` |  Specifies how Skaffold deploys artifacts. Skaffold supports using `kubectl`, `helm`, or `kustomize` to deploy artifacts. See [Deployers]({{< relref "/docs/pipeline-stages/deployers" >}}) for more information. |
| `profiles`|  Profile is a set of settings that, when activated, overrides the current configuration. You can use Profile to override the `build`, `test` and `deploy` sections. |
| `requires`|  Specifies a list of other skaffold configurations to import into the current config |

You can [learn more]({{< relref "/docs/references/yaml" >}}) about the syntax of `skaffold.yaml`.

Skaffold normally expects to find the configuration file as
`skaffold.yaml` in the current directory, but the location can be
overridden with the `--filename` flag.

### File resolution

The Skaffold configuration file often references other files and
directories.  These files and directories are resolved
relative to the current directory _and not to the location of
the Skaffold configuration file_.  There are two important exceptions:
1. Files referenced from a build artifact definition are resolved relative to the build artifact's _context_ directory.
   When omitted, the context directory defaults to the current directory.
2. For [configurations resolved as dependencies](#configuration-dependencies"), paths are always resolved relative to the directory containing the imported configuration file.

For example, consider a project with the following layout:
```
.
├── frontend
│   └── Dockerfile
├── helm
│   └── project
│       └── dev-values.yaml
└── skaffold.yaml
```

The config file might look like:
```yaml
apiVersion: skaffold/v2beta11
kind: Config
build:
  artifacts:
  - image: app
    context: frontend
    docker:
      dockerfile: "Dockerfile"
deploy:
  helm:
    releases:
    - name: project
      chartPath: helm/project
      valuesFiles:
      - "helm/project/dev-values.yaml"
      artifactOverrides:
        image: app
```

In this example, the `Dockerfile` for building `app`
is resolved relative to `app`'s context directory,
whereas the the Helm chart's location and its values-files are
relative to the current directory in `helm/project`.

We generally recommend placing the configuration file in the root directory of the Skaffold project.

## Multiple configuration support

A single `skaffold.yaml` file can define multiple skaffold configurations in the schema described above using the separator `---`. If these configuration objects define the `metadata.name` property then we consider them as `modules`, that can then be activated by name.

Consider a `skaffold.yaml` defined as:

```yaml
apiVersion: skaffold/vX
kind: Config
metadata:
  name: cfg1
build:
  # build definition
deploy:
  # deploy definition

---

apiVersion: skaffold/vX
kind: Config
metadata:
  name: cfg2
build:
  # build definition
deploy:
  # deploy definition
```

Here `cfg1` and `cfg2` are independent skaffold modules. Running `skaffold dev` for instance will execute actions from both these modules. You could also run `skaffold dev --module cfg1` to only activate the `cfg1` module and skip `cfg2`.

## Configuration dependencies

In addition to authoring configurations in a `skaffold.yaml` file, we can also import other existing configurations as dependencies. Skaffold manages all imported and defined configurations in the same session. It also ensures all artifacts in a required config are built prior to those in current config (provided the artifacts have dependencies defined); and all deploys in required configs are applied prior to those in current config.

{{< alert title="Note:" >}}
Running `skaffold <command> --module <config-name>` will filter to the specified target module, but also include the transitive closure of all other configurations in its dependency graph. For instance, if a module `cfg1` imported another module `cfg2` as a dependency while `cfg2` imported `cfg3` and `cfg4`, then running `skaffold dev --module cfg1` would activate all of `cfg1`, `cfg2`, `cfg3` and `cfg4` and execute them in dependency order.
{{< /alert >}}

### Local config dependency

Consider the same `skaffold.yaml` defined above. Modules `cfg1` and `cfg2` from the above file can be imported as dependencies in your current config definition, via:

```yaml
apiVersion: skaffold/v2beta11
kind: Config
requires:
  - configs: ["cfg1", "cfg2"]
    path: path/to/other/skaffold.yaml 
build:
  # build definition
deploy:
  # deploy definition
```

If the `configs` list isn't defined then it imports all the configs defined in the file pointed by `path`. Additionally, if the `path` to the configuration isn't defined it assumes that all the required configs are defined in the same file as the current config.

{{< alert title="Note:" >}}
In imported configurations, files are resolved relative to the location of imported Skaffold configuration file.
{{< /alert >}}

### Remote config dependency

The required skaffold config can live in a remote git repository:

```yaml
apiVersion: skaffold/v2beta12
kind: Config
requires:
  - configs: ["cfg1", "cfg2"]
    git:
      repo: http://github.com/GoogleContainerTools/skaffold.git
      path: getting-started/skaffold.yaml
      ref: main
```

The environment variable `SKAFFOLD_REMOTE_CACHE_DIR` or flag `--remote-cache-dir` specifies the download location for all remote repos. If undefined then it defaults to `~/.skaffold/repos`. 
The repo root directory name is a hash of the repo `uri` and the `branch/ref`.
Every execution of a remote module resets the cached repo to the referenced ref. The default ref is `master`. If `master` is not defined then it defaults to `main`.
The remote config gets treated like a local config after substituting the path with the actual path in the cache directory.

### Profile Activation in required configs

Profiles specified by the `--profile` flag are also propagated to all  configurations imported as dependencies, if they define them. This behavior can be disabled by setting the `--propagate-profiles` flag to `false`.

You can additionally set up more granular and conditional profile activations across dependencies through the `activeProfiles` stanza:

```yaml
apiVersion: skaffold/v2beta11
kind: Config
metadata:
    name: cfg
requires:
  - path: ./path/to/required/skaffold.yaml
    configs: [cfg1, cfg2]                 
    activeProfiles:                                     
     - name: profile1                               
       activatedBy: [profile2, profile3] 
```

Here, `profile1` is a profile that needs to exist in both configs `cfg1` and `cfg2`; while `profile2` and `profile3` are profiles defined in the current config `cfg`. If the current config is activated with either `profile2` or `profile3` then the required configs `cfg1` and `cfg2` are imported with `profile1` applied. If the `activatedBy` clause is omitted then that `profile1` always gets applied for the imported configs.



{{< alert title="Follow up" >}}
Take a look at the [tutorial]({{< relref "/docs/tutorials/config-dependencies" >}}) section to see this in action.
{{< /alert >}}
