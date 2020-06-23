---
title: "Installing Skaffold"
linkTitle: "Installing Skaffold"
weight: 10
aliases: [/docs/getting-started]
---

{{< alert title="Note" >}}
To keep Skaffold up to date, update checks are made to Google servers to see if a new version of
Skaffold is available.

You can turn this update check off by following [these instructions]({{<relref "/docs/references/privacy#update-check">}}).

Your use of this software is subject to the [Google Privacy Policy](https://policies.google.com/privacy)

{{< /alert >}}


{{% tabs %}}
{{% tab "LINUX" %}}
### Stable binary
For the latest **stable** release download and place it in your `PATH`:

https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64

Run these commands to download and place the binary in your /usr/local/bin folder:

```bash
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64
sudo install skaffold /usr/local/bin/
```

### Latest bleeding edge binary

For the latest **bleeding edge** build, download and place it in your `PATH`:

https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-amd64

Run these commands to download and place the binary in your /usr/local/bin folder:

```bash
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-amd64
sudo install skaffold /usr/local/bin/
```

{{% /tab %}}

{{% tab "MACOS" %}}

### Homebrew

```bash
brew install skaffold
```

### MacPorts

```bash
sudo port install skaffold
```

### Stable binary
For the latest **stable** release download and place it in your `PATH`:

https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-amd64

Run these commands to download and place the binary in your /usr/local/bin folder:

```bash
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-amd64
sudo install skaffold /usr/local/bin/
```

### Bleeding edge binary

For the latest **bleeding edge** build, download and place it in your `PATH`:

https://storage.googleapis.com/skaffold/builds/latest/skaffold-darwin-amd64

Run these commands to download and place the binary in your /usr/local/bin folder:

```bash
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-darwin-amd64
sudo install skaffold /usr/local/bin/
```
{{% /tab %}}

{{% tab "WINDOWS" %}}

### Chocolatey

```bash
choco install -y skaffold
```

### Stable binary

For the latest **stable** release download and place it in your `PATH` as `skaffold.exe`:

https://storage.googleapis.com/skaffold/releases/latest/skaffold-windows-amd64.exe

### Bleeding edge binary

For the latest **bleeding edge** build, download and place it in your `PATH` as `skaffold.exe`:

https://storage.googleapis.com/skaffold/builds/latest/skaffold-windows-amd64.exe

{{% /tab %}}


{{% tab "DOCKER" %}}

### Stable binary

For the latest **stable** release, you can use: 

`docker run gcr.io/k8s-skaffold/skaffold:latest skaffold <command>`

### Bleeding edge binary

For the latest **bleeding edge** build:

`docker run gcr.io/k8s-skaffold/skaffold:edge skaffold <command>`

{{% /tab %}}

{{% /tabs %}}

