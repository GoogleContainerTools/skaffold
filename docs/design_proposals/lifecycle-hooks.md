# Design Proposal: 
Skaffold lifecycle hooks 

**authors:** Gaurav Ghosh (gaghosh@)  
**status:** **approved**  
**approval date:** 07-14-2020  
**proposed on:** 07-06-2020  
**approvers:**

<table>
<thead>
<tr>
<th><strong>LDAP</strong></th>
<th><strong>LGTM Date</strong></th>
</tr>
</thead>
<tbody>
<tr>
<td>bdealwis@</td>
<td>2020-07-09</td>
</tr>
<tr>
<td>nkubala@</td>
<td>07/14/20</td>
</tr>
<tr>
<td>tejaldesai@</td>
<td>07/08/2020</td>
</tr>
</tbody>
</table>

# Background

Supporting callbacks for lifecycle hooks is a heavily requested feature in the Skaffold community with some of the top voted issues being:

1. [Issue #2425](https://github.com/GoogleContainerTools/skaffold/issues/2425): Users get away with wrapping skaffold in scripts that execute additional actions but want to move away from that 
1. [Issue #3475](https://github.com/GoogleContainerTools/skaffold/issues/3475): Users want to execute tests prior to build 
1. [Issue #1441](https://github.com/GoogleContainerTools/skaffold/issues/1441): Wrapping skaffold in a script doesn't solve the problem for iterative development using skaffold dev or debug where you'd want some custom action to repeat on every dev loop.
1. [Issue #3737](https://github.com/GoogleContainerTools/skaffold/issues/3737): Users want to be able to calculate and export environment variables in the build step and reference it in subsequent steps. Currently there is no work around for this.

In the past there have been discussions that prioritized implementing targeted features over a generic script execution that, while making the platform more flexible, renders it somewhat blind to the specific action that the user is trying to accomplish. However as evidenced by user feedback there are scenarios both in local development and CICD that can be readily solved by opening up hooks into Skaffold.

# Overview

There are three broad scenarios:

1. Host hooks: Being able to execute a script on the host machine before and after every build/sync/deploy step.
1. Container hooks: Being able to execute a script within a launched container after every sync or deploy step.
1. Being able to export environment variables from a script executed as part of these lifecycle callbacks and reference them later at other steps. 

# Detailed Design

## Schema

```yaml
hooks:
   before:
     - command: [ “sleep”, “5”  ]
       os: [ “linux”, “darwin” ]

     - command: [ “timeout”, “5”  ]
       os: [ “windows” ]

     - containerCommand:  [ “echo”, “foo”  ]
       # containerName is optional for artifact scoped hooks like in build and sync
       containerName: foo
       # podPrefix is optional for artifact scoped hooks like in build and sync
       podPrefix: bar

   after:
     - command:  [ “echo”, “foo”  ]
```

This can be nested under build, deploy and sync stages (described in examples below):

-  Build: Hooks are defined per artifact build definition in skaffold.yaml
-  Sync: Hooks are defined per artifact sync definition in skaffold.yaml
-  Deploy: Hooks are defined per deployment type definition in skaffold.yaml

Possible values for `os` field are all golang [recognised](https://github.com/golang/go/blob/master/src/go/build/syslist.go#L10) platforms. Missing value implies all.  
If the command points to files, those don't get added to the change monitoring for dev loop. Inline commands by virtue of being part of the skaffold.yaml file would be subject to dev loop reload on change.

## Sample implementations 

### Example 1: Docker context composition

For multiple microservices in a repo sharing multiple common libraries it may not be ideal for the repo root to be the Dockerfile context. We use pre-build hook to copy over the necessary files prior to the build

```yaml
apiVersion: skaffold/vX
kind: Config
metadata:
 name: microservices

build:
 artifacts:
   - image: leeroy-web
     context: ./leeroy-web/
     hooks:
       before:
         - command: [ “cp”, “./shared/package1.py”, “./leeroy-web/pkg/” ]
         - command: [ “./setup.sh” ]

   - image: leeroy-app
     context: ./leeroy-app/
deploy: ...
```

### Example 2: Running tests

We might want to run tests to validate artifacts or deployment. If it exits with non-zero status code then depending on the run type it'll stop the execution

```yaml
apiVersion: skaffold/vX
kind: Config
metadata:
 name: microservices
build:
 artifacts:
   - image: leeroy-web
     context: ./leeroy-web/
   - image: leeroy-app
     context: ./leeroy-app/

deploy:
 kubectl:
   manifests:
     - ./leeroy-web/kubernetes/*
     - ./leeroy-app/kubernetes/*
   hooks:
     before:
       - command: [ “make”, “pre-deployment-tests”  ]
     after:
       - command: [ “make”, “post-deployment-tests”  ]
```

### Example 3: Run command on file sync

We want to run say javascript minification post every file sync. 

```yaml
apiVersion: skaffold/vX
kind: Config
metadata:
 name: node-example

build:
 artifacts:
   - image: node-example
     context: ./node/
     sync: 
        manual:
          - src: ‘src/**/*.js’
          - dest: ‘./raw/’
        hooks: 
          after: 
             - container-command: [ “./minify-script.sh’, “./raw”, “./min/” ]

deploy: ...
```

## Information contract

<table>
<thead>
<tr>
<th><strong>Environment variable</strong></th>
<th><strong>Description</strong></th>
<th><strong>Availability</strong></th>
</tr>
</thead>
<tbody>
<tr>
<td>$IMAGE</td>
<td>The fully qualified image name. For example, "gcr.io/image1:tag"</td>
<td>Pre-Build; Post-Build</td>
</tr>
<tr>
<td>$PUSH_IMAGE</td>
<td>Set to true if the image in $IMAGE is expected to exist in a remote registry. Set to false if the image is expected to exist locally.</td>
<td>Pre-Build; Post-Build</td>
</tr>
<tr>
<td>$IMAGE_REPO</td>
<td>The image repo. For example, "gcr.io/image1"</td>
<td>Pre-Build; Post-Build</td>
</tr>
<tr>
<td>$IMAGE_TAG</td>
<td>The image tag. For example, "tag"</td>
<td>Pre-Build; Post-Build</td>
</tr>
<tr>
<td>$BUILD_CONTEXT</td>
<td>An absolute path to the directory this artifact is meant to be built from. Specified by artifact context in the skaffold.yaml.</td>
<td>Pre-Build; Post-Build</td>
</tr>
<tr>
<td>$SYNC_FILES</td>
<td>Semi-colon delimited list of absolute path to all files synced or to be synced in current dev loop</td>
<td>Pre-Sync; Post-Sync</td>
</tr>
<tr>
<td>$SKAFFOLD_RUN_ID</td>
<td>Run specific UUID label for deployed or to be deployed resources</td>
<td>Pre-Deploy; Post-Deploy</td>
</tr>
<tr>
<td>$SKAFFOLD_DEFAULT_REPO</td>
<td>The resolved default repository</td>
<td>All</td>
</tr>
<tr>
<td>$SKAFFOLD_RPC_PORT</td>
<td>TCP port to expose event API</td>
<td>All</td>
</tr>
<tr>
<td>$SKAFFOLD_HTTP_PORT</td>
<td>TCP port to expose event REST API over HTTP</td>
<td>All</td>
</tr>
<tr>
<td>$SKAFFOLD_KUBE_CONTEXT</td>
<td>The resolved Kubernetes context</td>
<td>All</td>
</tr>
<tr>
<td>$SKAFFOLD_NAMESPACES</td>
<td>Comma separated list of Kubernetes namespaces</td>
<td>All</td>
</tr>
<tr>
<td>$SKAFFOLD_WORK_DIR</td>
<td>The workspace root directory</td>
<td>All</td>
</tr>
<tr>
<td>$SKAFFOLD_PROFILES</td>
<td>Comma separated list of activated profiles</td>
<td>All</td>
</tr>
<tr>
<td>Local environment variables</td>
<td>The current state of the local environment (e.g. $HOST, $PATH). Determined by the golang <a href="https://golang.org/pkg/os#Environ">os.Environ</a> function.</td>
<td>All</td>
</tr>
</tbody>
</table>

## ENV propagation

It might be helpful to be able to pass variables between multiple hooks callback functions. This can be achieved by setting a pattern to dump key-value pairs into standard output that are then parsed, stored and supplied in subsequent hooks command execution.

For example, running:

```
echo "::set-env name=FOO::BAR"
```

will set the environment variable FOO to the value BAR

```yaml
hooks:
   after:
       - command: [ “/bin/echo”, “::set-env”, “name=FOO::BAR”]
       - command: [ “/bin/echo”, “The value of FOO is ${FOO}”]
```

This is only scoped to propagate values across lifecycle hooks commands only but across several dev loops.

Note: There is a related issue [#4106](https://github.com/GoogleContainerTools/skaffold/issues/4106) requesting a feature of being able to read environment variables from files and substituting them in the skaffold template. If this feature is available then users can get away with modifying the environment variable file to propagate environment variables across stages without needing this implementation.

# Testing

In addition to regular unit tests we'll add integration tests against new example projects that showcase using hooks for the scenarios:

1. Pre and post build
1. Pre and post sync
1. Pre and post deploy
1. Env variable propagation across hooks

# Metrics

There are currently no metrics collected within skaffold. However events are reported on the event API server. We'll add event notifications for the following phases:

<table>
<thead>
<tr>
<th><strong>Event</strong></th>
<th><strong>Params</strong></th>
</tr>
</thead>
<tbody>
<tr>
<td>HostCallbackEvent</td>
<td>Type(pre/post - build/sync/deploy)<br>
Status(InProgress, Completed, Failed)<br>
error message</td>
</tr>
<tr>
<td>ContainerCallbackEvent</td>
<td>Type(post - sync/deploy)<br>
Status(InProgress, Completed, Failed)<br>
error message</td>
</tr>
</tbody>
</table>

Additionally we'll append the count of each type of hooks defined in skaffold.yaml to the corresponding sections of the MetaEvent for Build, Sync and Deploy

# 

# Implementation breakdown

<table>
<thead>
<tr>
<th><strong>Priority</strong></th>
<th><strong>Feature / Requirement</strong></th>
<th><strong>Notes</strong></th>
</tr>
</thead>
<tbody>
<tr>
<td></td>
<td><u>Config changes</u></td>
<td></td>
</tr>
<tr>
<td>P0</td>
<td>Users can define pre-hooks and post-hooks in the build, deploy and sync sections of skaffold.yaml</td>
<td></td>
</tr>
<tr>
<td></td>
<td><u>Hooks Runner</u></td>
<td></td>
</tr>
<tr>
<td>P0</td>
<td>Generic Host Hooks runner implemented and tested via unit tests</td>
<td></td>
</tr>
<tr>
<td>P0</td>
<td>Generic Container Hooks runner implemented and tested via unit tests</td>
<td></td>
</tr>
<tr>
<td></td>
<td><u>Dev loop integration</u></td>
<td></td>
</tr>
<tr>
<td>P0</td>
<td>Runner integrated with pre and post build dev loop</td>
<td></td>
</tr>
<tr>
<td>P0</td>
<td>Runner integrated with pre and post sync dev loop</td>
<td></td>
</tr>
<tr>
<td>P0</td>
<td>Runner integrated with pre and post deploy dev loop</td>
<td></td>
</tr>
<tr>
<td>P0</td>
<td>Integration examples with tests for all lifecycle hooks</td>
<td></td>
</tr>
<tr>
<td></td>
<td><u>ENV propagation</u></td>
<td></td>
</tr>
<tr>
<td>P1</td>
<td>Enable environment variable propagation across hooks</td>
<td></td>
</tr>
<tr>
<td>P1</td>
<td>Integration example for env propagation</td>
<td></td>
</tr>
</tbody>
</table>

# Alternatives Considered

Schema Alternative 1 (too verbose, no explicit container-command)

```yaml
hooks:
   pre:
     - exec:
         command: [ “sleep”, “5”  ]
         os: [ “linux”, “darwin” ]
     - exec:
         command:  [ “timeout”, “5”  ]
         os:  [ “windows” ]

   post:
     - exec:
         command:  [ “echo”, “foo”  ]
```

Schema Alternative 2 (less verbose, dash-casing over camelCasing)

```yaml
pre-hooks:
   - command: [ “sleep”, “5”  ]
     os: [ “linux”, “darwin” ]

   - command: [ “timeout”, “5”  ]
     os: [ “windows” ]

   - container-command:  [ “echo”, “foo”  ]
     # container-name is optional for artifact scoped hooks like in build and sync
     container-name: foo
     # pod-prefix is optional for artifact scoped hooks like in build and sync
     pod-prefix: bar

 post-hooks:
     - command:  [ “echo”, “foo”  ]

```