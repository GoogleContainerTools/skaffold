---
title: "Importing configuration as dependencies"
linkTitle: "Config Dependencies"
weight: 100
---

Skaffold's [config dependencies]({{<relref "/docs/design/config#configuration-dependencies" >}}) feature allows you to import configurations defined across multiple `skaffold.yaml` files (even across repositories) into a single configuration as its dependencies. This helps more readily integrate Skaffold with multi-microservice and multi-repository applications.

For a guided Cloud Shell tutorial on setting up [local dependencies]({{<relref "/docs/design/config#local-config-dependency">}}), follow:

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/gsquared94/bank-of-anthos-demo&cloudshell_workspace=.&cloudshell_tutorial=tutorial.md)

For a guided Cloud Shell tutorial on setting up [remote git dependencies]({{<relref "/docs/design/config#remote-config-dependency">}}), follow:

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/gsquared94/skaffold-remote-configs-demo&cloudshell_workspace=.&cloudshell_tutorial=tutorial.md)