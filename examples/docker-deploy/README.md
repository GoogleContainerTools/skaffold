### Example: Deploying a simple go app to Docker

This is a simple example based on:

* **building** a two single Go file apps, each with a multistage `Dockerfile` using local docker to build
* **tagging** using the default tagPolicy (`gitCommit`)
* **deploying** to docker by simply running two individual containers
