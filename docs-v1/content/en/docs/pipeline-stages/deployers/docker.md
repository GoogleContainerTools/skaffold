---
title: "Docker"
linkTitle: "Docker"
weight: 20
featureId: deploy.docker
---

{{< alert title="Note" >}}
This feature is currently experimental and subject to change.
{{< /alert >}}

## Deploying applications to a local Docker daemon

For simple container-based applications that don't rely on
Kubernetes resource types, Skaffold can "deploy" these applications
by running application containers directly in your local Docker daemon.
This enables application developers who are not yet ready to make the jump
to Kubernetes to take advantage of the streamlined development experience
Skaffold provides.

Additionally, deploying to Docker bypasses the overhead of pushing
images to a remote registry, and provides a faster time to running
application than traditional Kubernetes deployments.

### Configuration

To deploy to your local Docker daemon, specify the `docker` deploy type
in the `deploy` section of your `skaffold.yaml`.

The `docker` deploy type offers the following options:

{{< schema root="DockerDeploy" >}}

### Example

The following `deploy` section instructs Skaffold to deploy
the application image `my-image` to the local Docker daemon:

{{% readfile file="samples/deployers/docker.yaml" %}}

{{< alert title="Note" >}}
Images listed to be deployed with the `docker` deployer **must also have a corresponding build artifact built by Skaffold.**
{{< /alert >}}

## Deploying with Docker Compose

Skaffold can deploy your application using Docker Compose instead of individual containers.
This is useful when your application is already configured with a `docker-compose.yml` file
and you want to leverage Compose's features like service dependencies, networks, and volumes.

### Configuration

To deploy using Docker Compose, set `useCompose: true` in the `docker` deploy configuration:

```yaml
deploy:
  docker:
    useCompose: true
    images:
      - my-app
```

### How it works

When `useCompose` is enabled, Skaffold:

1. Reads your `docker-compose.yml` file (or a custom file specified via environment variable)
2. Builds images as specified in your `build` section
3. Automatically replaces image names in the compose file with the built image tags
4. Creates a temporary compose file with the updated images
5. Runs `docker compose up -d` with a unique project name (`skaffold-{runID}`)
6. On cleanup, runs `docker compose down --volumes --remove-orphans`

### Image name mapping

**Important**: For Skaffold to correctly replace images in your compose file, the image names
in your `docker-compose.yml` must match (or be contained in) the image names specified in
the `build.artifacts` section.

For example, if your `skaffold.yaml` has:

```yaml
build:
  artifacts:
    - image: gcr.io/my-project/frontend-app
    - image: gcr.io/my-project/backend-app
```

Your `docker-compose.yml` should use matching image names:

```yaml
version: '3.8'
services:
  frontend:
    image: frontend-app  # Matches the suffix of gcr.io/my-project/frontend-app
  backend:
    image: backend-app   # Matches the suffix of gcr.io/my-project/backend-app
```

Skaffold will replace `frontend-app` with `gcr.io/my-project/frontend-app:latest-abc123`
and `backend-app` with `gcr.io/my-project/backend-app:latest-def456`.

### Custom compose file location

By default, Skaffold looks for `docker-compose.yml` in the current directory.
You can specify a custom location using the `SKAFFOLD_COMPOSE_FILE` environment variable:

```bash
export SKAFFOLD_COMPOSE_FILE=path/to/my-compose.yml
skaffold dev
```

Or inline:

```bash
SKAFFOLD_COMPOSE_FILE=docker-compose.prod.yml skaffold run
```

### Example

Complete example configuration:

**skaffold.yaml:**
```yaml
apiVersion: skaffold/v4beta13
kind: Config
build:
  artifacts:
    - image: my-web-app
      docker:
        dockerfile: Dockerfile
deploy:
  docker:
    useCompose: true
    images:
      - my-web-app
```

**docker-compose.yml:**
```yaml
version: '3.8'
services:
  web:
    image: my-web-app
    ports:
      - "8080:8080"
    environment:
      - NODE_ENV=development
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

When you run `skaffold dev`, Skaffold will:
- Build `my-web-app` image
- Replace `my-web-app` in the compose file with the built tag (e.g., `my-web-app:latest-abc123`)
- Leave `redis:7-alpine` unchanged (not built by Skaffold)
- Deploy both services using `docker compose up`

### Limitations and Notes

- The compose file must have a valid `services` section
- Only images that are built by Skaffold will be replaced
- External images (like `postgres`, `redis`, etc.) are deployed as-is
- The compose project name is automatically generated as `skaffold-{runID}` to avoid conflicts
- Multiple Skaffold instances can run simultaneously without interfering with each other

For a complete working example, see [`examples/docker-compose-deploy`](https://github.com/GoogleContainerTools/skaffold/tree/main/examples/docker-compose-deploy).
