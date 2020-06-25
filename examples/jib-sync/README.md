### Example: Jib Sync

[Jib](https://github.com/GoogleContainerTools/jib) is one of the supported builders in Skaffold. Jib
has special sync support using the `auto` configuration.

## Running the example

Run the maven or gradle version of the example with port forwarding on.

#### Gradle
```
$ skaffold dev -f skaffold-gradle.yaml --port-forward
```

#### Maven
```
$ skaffold dev -f skaffold-maven.yaml --port-forward
```

You can now see sync in action:
1. See the original response from the `HelloController` in the spring application 
  ```
  $ curl localhost:8080
  text-to-replace
  ```
1. Edit the hello controller `src/main/java/hello/HelloController.java`
  ```diff
  +       return "some-new-text\n";
  -       return "text-to-replace\n";
  ```
1. Give skaffold a few seconds to synchronize the file to the container, and give Spring
   Boot Developer Tools a chance to reload your application.
1. See the updated response from the `HelloController`
  ```
  $ curl localhost:8080
  some-new-text
  ```
1. You've now seen auto sync in action!

## How it works

This example contains both maven and gradle build configs and separate skaffold.yamls.

- **gradle**: `skaffold-gradle.yaml`
- **maven**: `skaffold-maven.yaml`

use the `-f` flag to specify the correct buildfile when running (or rename your preferred option to `skaffold.yaml`)
```
$ skaffold -f skaffold-gradle.yaml ...
```

We configure it in `skaffold.yaml`, by enabling sync on the jib artifact.

```yaml
build:
  artifacts:
  - image: skaffold-example
    context: .
    jib: {}
    sync: 
      auto: {}
```

This example is designed around the functionality available in [Spring Boot Developer Tools](https://docs.spring.io/spring-boot/docs/current/reference/html/using-spring-boot.html#using-boot-devtools) for developing against running applications.

Some additional steps in your java build are required for this to work:
- Sync requires `tar` on the running container to copy files over. The default base image that Jib uses `gcr.io/distroless/java` does not include `tar` or any utilities. During development you must use a base image that includes `tar`, in this example we use the `debug` flavor of distroless: `gcr.io/distroless/java:debug` 

`maven`
```xml
<plugin>
  <groupId>com.google.cloud.tools</groupId>
  <artifactId>jib-maven-plugin</artifactId>
  <version>${jib.maven-plugin-version}</version>
  <configuration>
    ...
    <from>
      <image>gcr.io/distroless/java:debug</image>
    </from>
  </configuration>
</plugin>
```

`gradle`
```groovy
jib {
  ...
  from {
    image = "gcr.io/distroless/java:debug"
  }
}
```

- You must include the `spring-boot-devtools` dependency at the `compile/implementation` scope, which is contrary to the configuration outlined in the [official docs](https://docs.spring.io/spring-boot/docs/current/reference/html/using-spring-boot.html#using-boot-devtools). Because jib is unaware of any special spring only configuration in your builds, we recommend using profiles to turn on or off devtools support in your jib container builds.

`maven`
```xml
<profiles>
  <profile>
    <id>sync<id>
    <dependencies>
      <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-devtools</artifactId>
        <!-- <optional>true</optional> not required -->
      </dependency>
    </dependencies>
  </profile>
</profiles>
```

`gradle`
```groovy
dependencies {
  ...
  if (project.hasProperty('sync')) {
    implementation "org.springframework.boot:spring-boot-devtools"
    // do not use developmentOnly
  }
}
```

To activate these profiles, we add `-Psync` to the maven/gradle build command. This can be done directly in the artifact configuration

`skaffold.yaml`
```
build:
  artifacts:
  - image: skaffold-example
    context: .
    jib: 
      args: 
      - -Psync
    sync: 
      auto: {}
```

You can also take advantage of [skaffold profiles](https://skaffold.dev/docs/environment/profiles/) to control when to activate sync on your project.

`skaffold.yaml`
```
build:
  artifacts:
  - image: test-file-sync
    jib: {}

profiles:
- name: sync
  patches:
    # assuming jib is the first artifact (index 0) in your build.artifacts list
  - op: add
    # we want to activate sync on our skaffold artifact
    path: /build/artifacts/0/sync
    value:
      - auto: {}
  - op: add
    # we activate the sync profile in our java builds
    path: /build/artifacts/0/jib/args
    value:
    - -Psync
```

skaffold profiles can be activated using the the `-p` flag when running

```
$ skaffold -p sync ...
```
