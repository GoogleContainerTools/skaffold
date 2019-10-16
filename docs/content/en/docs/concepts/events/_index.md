---
title: "Skaffold Events"
linkTitle: "Skaffold Events"
weight: 40
---

This page discusses the Skaffold Events.

Skaffold provides a continous development mode [`skaffold dev`](../modes/) which builds, deploys
your application on changes. In a single development loop, one or more container images
may be built and deployed. The time taken for the changes to deploy varies.

Skaffold exposes events for users to get notified when phases within a development loop
complete. 
You can use these events to automate next steps in your development workflow. 

e.g: when making a change to port-forwarded frontend service, reload the 
browser url after the service is deployed and running to test changes.

## Using the Events API
To get Skaffold events curl the [events API endpoint](#events) on port `50052`

s
