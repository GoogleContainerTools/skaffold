---
title: "Telemetry"
linkTitle: "Telemetry"
weight: 400
aliases: [/docs/metrics]
---

<script type="module" src="main.js"></script>

To help improve the quality of this product, we collect anonymized usage data from Skaffold.

You will be *opted-in* by default and you can opt-out anytime via a `skaffold config` command

Usage data does not include argument values, or any personal information.

In order to disable sending usage data, run the following command after you have installed skaffold:

```bash
skaffold config set --global collect-metrics false
```

The breakdown of data we collect is as follows
<ul id="metrics-list"></ul>

#### Example
```bash
skaffold dev -v trace --port-forward --cache-artifacts=false --filename=./skaffold.yaml
```
Running the above in the [microservices example](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/microservices)
after a couple of builds/deploys results in the following metrics being collected
```json
[{
    "ExitCode": 0,
    "BuildArtifacts": 3,
    "Command": "dev",
    "Version": "v1.19.0",
    "OS": "darwin",
    "Arch": "amd64",
    "PlatformType": "local",
    "Deployers": ["kubectl"],
    "EnumFlags": {
        "cache-artifacts": "false",
        "port-forward": "true"
    },
    "Builders": {
        "docker": 3
    },
    "SyncType": {},
    "DevIterations": [{
        "Intent": "build",
        "ErrorCode": 0
    }, {
        "Intent": "build",
        "ErrorCode": 104
    }, {
        "Intent": "build",
        "ErrorCode": 0
    }, {
        "Intent": "deploy",
        "ErrorCode": 300
    }, {
        "Intent": "deploy",
        "ErrorCode": 0
    }],
    "StartTime": "2021-01-25T16:24:38.615012-05:00",
    "Duration": 176315222939,
    "ErrorCode": 0
}]
```

This data is handled in accordance with our privacy policy [https://policies.google.com/privacy](https://policies.google.com/privacy).
