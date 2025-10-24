# Example: Deploying with Docker Compose

This example demonstrates how to use Skaffold to build and deploy applications using Docker Compose.

## Prerequisites

- Docker and Docker Compose installed
- Skaffold installed

## Project Structure

- `skaffold.yaml` - Skaffold configuration with Docker Compose deployment
- `docker-compose.yml` - Docker Compose configuration
- `Dockerfile` - Simple application Docker image
- `main.go` - Simple Go web application

## How it Works

1. Skaffold builds the Docker image for the application
2. Skaffold updates the `docker-compose.yml` with the built image tag
3. Skaffold runs `docker compose up` to deploy the application
4. When you stop Skaffold, it runs `docker compose down` to clean up

## Usage

### Run the application

```bash
skaffold dev
```

This will:
- Build the application image
- Deploy it using Docker Compose
- Watch for changes and rebuild/redeploy automatically

### Deploy only

```bash
skaffold run
```

### Clean up

```bash
skaffold delete
```

Or simply press `Ctrl+C` when running `skaffold dev`.

## Configuration

The key part of the `skaffold.yaml` configuration is:

```yaml
deploy:
  docker:
    useCompose: true
    images:
      - compose-app
```

- `useCompose: true` - Enables Docker Compose deployment
- `images` - List of images to build and deploy

## Environment Variables

You can customize the Docker Compose file location:

```bash
export SKAFFOLD_COMPOSE_FILE=custom-compose.yml
skaffold dev
```

Default is `docker-compose.yml` in the current directory.

## Notes

- The Docker Compose project name will be `skaffold-{runID}`
- Skaffold automatically replaces image names in the compose file with the built tags
- Volumes and networks are automatically cleaned up on `skaffold delete`
