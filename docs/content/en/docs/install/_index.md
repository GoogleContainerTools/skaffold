---
title: "Installing Skaffold"
linkTitle: "Installing Skaffold"
weight: 5
---

{{< alert title="Note" >}}
To keep Skaffold up to date, update checks are made to Google servers to see if a new version of
Skaffold is available.

You can turn this update check off by following <a href=/docs/references/privacy#update-check>these instructions</a>.


Your use of this software is subject to the <a href=https://policies.google.com/privacy>Google Privacy Policy</a>
{{< /alert >}}


{{% tabs %}}
{{% tab "LINUX" %}}
### Stable binary
For the latest **stable** release download and place it in your `PATH`:

https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64

Run these commands to download and place the binary in your /usr/local/bin folder:

```bash
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64
chmod +x skaffold
sudo mv skaffold /usr/local/bin
```

### Latest bleeding edge binary

For the latest **bleeding edge** build, download and place it in your `PATH`:

https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-amd64

Run these commands to download and place the binary in your /usr/local/bin folder:

```bash
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-amd64
chmod +x skaffold
sudo mv skaffold /usr/local/bin
```

{{% /tab %}}

{{% tab "MACOS" %}}

### Homebrew

```bash
brew install skaffold
```

### Stable binary
For the latest **stable** release download and place it in your `PATH`:

https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-amd64

Run these commands to download and place the binary in your /usr/local/bin folder:

```bash
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-amd64
chmod +x skaffold
sudo mv skaffold /usr/local/bin
```

### Bleeding edge binary

For the latest **bleeding edge** build, download and place it in your `PATH`:

https://storage.googleapis.com/skaffold/builds/latest/skaffold-darwin-amd64

Run these commands to download and place the binary in your /usr/local/bin folder:

```bash
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-darwin-amd64
chmod +x skaffold
sudo mv skaffold /usr/local/bin
```
{{% /tab %}}

{{% tab "WINDOWS" %}}

### Chocolatey

```bash
choco install skaffold
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

