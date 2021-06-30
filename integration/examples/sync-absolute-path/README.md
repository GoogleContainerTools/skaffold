### Example: Getting started with a simple go app

This is a simple example based on:

* **building** a single Go file app and with a multistage `Dockerfile` using local docker to build
* **tagging** using the default tagPolicy (`gitCommit`)
* **deploying** a single container pod using `kubectl`
* **manualsync** manual file sync with absolute path

> Notice: /home/test path should exist and should be at least 1 file there