# Title

* Author(s): Appu Goundan (@loosebazooka)
* Design Shepherd: \<skaffold-core-team-member\>
* Date: 09/17/2019
* Status: New

## Background

Currently skaffold does not support `sync` for files that are generated
during a build. For example when syncing java files to a container, one
would normally expect `.class` files to be sync'd, but skaffold is
really only aware of `.java` files in a build.

1. Why is this required?
  - Having a sync mode that has knowledge of files that need to be built before
    sync'ing and how to build them will allow us to cover use cases like
    springboot dev-tools.

2. If this is a redesign, what are the drawbacks of the current implementation?
  - This is not a redesign, but a new sync mode that may or may not be available
    in the skaffold.yaml directly to users. It could simply be an internal API
    that is usuable by builders like Jib.
  - This is similar to `_smart_` described in [sync-improvements](sync-improvements.md)

3. Is there any another workaround, and if so, what are its drawbacks?
  - Currently one can create a docker file that only copies specific build
    results into the container and relies on a local build to generate those
    intermediate artifacts. This requires a user to trigger a first build
    manually before docker kicks in to containerize the application. While
    it may be possible to automate this (for example: gradle --continuous), it
    is not an acceptable solution to require manual external processes for
    a build to succeed.

4. Mention related issues, if there are any.
  - This is not trying to solve the problem of dealing with a multistage
    dockerbuild. Intermediate build artifacts might still be possible to
    determine, however that would require an extra mechanism to do a partial
    docker build and sync files from a built container to a running container --
    something we do not intend to cover here.

#### Problems with current API/config
The current `sync` system has the following problems:
1. No way to trigger local external processes - for example, in jib, to build a
   container, one would run `./gradlew jib`, but to update class files so the
   system may sync them, one would only be required to run `./gradlew classes`.
2. No way to tell the system to sync non build inputs. Skaffold is
   normally only watching `.java` files for a build, in the sync case, we want
   it to watch `.java` files, trigger a partial build, and sync `.class` files
   so a remote server can pick up and reload the changes.

## Design

#### Hack it
To get close to the functionality we want, without modifying skaffold at all, a
Dockerfile which depends on java build outputs could be used, like:
```
FROM openjdk:8
COPY build/dependencies/ /app/dependencies
COPY build/classes/java/main/ /app/classes

CMD ["java", "-cp", "/app/classes:/app/dependencies/*", "hello.Application"]
```

with a skaffold sync block that looks like:
```
sync:
  manual:
    - src: "build/classes/java/main/**/*.class"
      dest: "/app/classes"
      strip: "build/classes/java/main/"
```

A user's devloop then looks like this:

1. run `./gradlew classes copyDependencies`
2. run `skaffold dev`
3. *make changes to some java file*
4. run `./gradlew classes`
5. *skaffold syncs files*

which is far from ideal.

#### Fix it?

What we might want, is an API that can accept configuration that looks like:

```
sync: auto {
  indirect:
  - command: "./gradlew jib classes"
    inputs:
      - src/main/java
      - src/main/resources
    files:
      - src: "target/classes"
        dest: "/app/classes",
      - src: "target/resources.
        dest: "/app/resources",
  - command: "compile_c_stuff.sh"
    inputs:
    - asdf.c
    files:
     - src: "asdf.o"
       dest: "/libs/asdf.o"
  direct:
    - src: "local/computer/extrafiles"
      dest: "/extrafiles"
  includeTools: true
}
```

Lets breakdown the different parts

1. `sync: auto` - just a name for the mode, can be anything
2. `indirect` - this means, you can't sync files directly, you need a command to
   tranform them before sync. Indirect is also a list that can handle multiple
   indirect types.
     1. `command` - the command to run to generate syncable files
     2. `inputs` - the inputs to watch to trigger `command`, if left empty, this
        should just be considered the full list of inputs that are normally
        watched for this build minus anything in `direct` (below).
     3. `files` - the built files to sync (and where to sync them)
3. `direct` - same as manual sync, just copy them directly, don't wait on some
   command to transform them
4. `includeTools` - useful for containers that don't contain, for example, `tar`
   and similar to debug, we can use some init container to inject the necessary
   tooling for sync.

### Open Issues/Questions

Please list any open questions here in the following format:

**How does a tool, like Jib, configure this**

Jib will export the right data, similar to how `skaffoldFiles` currently exports
files to watch. Other builders integrating with the API might need to do the
same things.

**Should we even allow multiple `indirect` configs?**

*Not Yet Resolved*

## Implementation plan

*TBD*

## Integration test plan

*TBD*
