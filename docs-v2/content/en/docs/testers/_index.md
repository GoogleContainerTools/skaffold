---
title: "Test"
linkTitle: "Test"
weight: 42
featureId: test
aliases: [/docs/how-tos/testers, /docs/pipeline-stages/testers/]
no_list: true
---

Skaffold has an integrated testing phase between the build and deploy phases of the pipeline. Skaffold supports the below types of tests.

| Skaffold testers|Description| 
|----------|-------|
| [Custom Test]({{< relref "/docs/testers/custom.md" >}}) | Enables users to run custom commands in the testing phase of the Skaffold pipeline | 
| [Container Structure Test]({{< relref "/docs/testers/structure.md" >}}) | Enables users to validate built container images before deploying them to their cluster | 
