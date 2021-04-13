---
title: "Test"
linkTitle: "Test"
weight: 10
featureId: test
aliases: [/docs/how-tos/testers]
no_list: true
---

Skaffold has an integrated testing phase between the build and deploy phases of the pipeline. Skaffold supports the below types of tests.

| Skaffold testers|Description| 
|----------|-------|
| [Custom Test]({{< relref "/docs/pipeline-stages/testers/custom.md" >}}) | Enables users to run custom commands in the testing phase of the Skaffold pipeline | 
| [Container Structure Test]({{< relref "/docs/pipeline-stages/testers/structure.md" >}}) | Enables users to validate built container images before deploying them to our cluster | 
