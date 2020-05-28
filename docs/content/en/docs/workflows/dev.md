---
title: "skaffold dev"
linkTitle: "Continuous Development"
featureId: dev
weight: 20
---

`skaffold dev` enables continuous local development on an application.
While in `dev` mode, Skaffold will watch an application's source files, and when it detects changes,
will rebuild your images (or sync files to your running containers), push any new images, and redeploy the application to your cluster.

`skaffold dev` is considered Skaffold's main mode of operation, as it allows you
to leverage all of the features of Skaffold in a continuous way while iterating
on your application.

## Dev loop

When `skaffold dev` is run, Skaffold will first do a full build and deploy of all artifacts specified in the `skaffold.yaml`, similar to `skaffold run`. Upon successful build and deploy, Skaffold will start watching all source file dependencies for all artifacts specified in the project. As changes are made to these source files, Skaffold will rebuild the associated artifacts, and redeploy the new changes to your cluster.

The dev loop will run until the user cancels the Skaffold process with `Ctrl+C`. Upon receiving this signal, Skaffold will clean up all deployed artifacts on the active cluster, meaning that Skaffold won't abandon any Kubernetes resources that it created throughout the lifecycle of the run. This can be optionally disabled by using the `--no-prune` flag.

## Precedence of Actions

The actions performed by Skaffold during the dev loop have precedence over one another, so that behavior is always predictable. The order of actions is:

1. File Sync
1. Build
1. Deploy

## File Watcher and Watch Modes

Skaffold computes the dependencies for each artifact based on the builder being used, and the root directory of the artifact. Once all source file dependencies are computed, in `dev` mode, Skaffold will continuously watch these files for changes in the background, and conditionally re-run the loop when changes are detected.

By default, Skaffold uses `fsnotify` to monitor events on the local filesystem. Skaffold also supports a `polling` mode where the filesystem is checked for changes on a configurable interval, or a `manual` mode, where Skaffold waits for user input to check for file changes. These watch modes can be configured through the `--trigger` flag.

## Control API

By default, the dev loop will carry out all actions (as needed) each time a file is changed locally, with the exception of operating in `manual` trigger mode. However, individual actions can be gated off by user input through the Skaffold API.

With this API, users can selectively turn off the automatic dev loop and can tell Skaffold to wait for user input before performing any of these actions, even if the requisite files were changed on the filesystem. By doing so, users can "queue up" changes while they are iterating locally, and then have Skaffold rebuild and redeploy only when asked. This can be very useful when builds are happening more frequently than desired, when builds or deploys take a long time or are otherwise very costly, or when users want to integrate other tools with `skaffold dev`.

For more documentation, see the [Skaffold API Docs]({{<relref "/docs/design/api" >}}).
