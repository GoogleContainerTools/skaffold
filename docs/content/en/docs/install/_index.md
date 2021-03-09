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
The latest **stable** binaries can be found here:

https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64
https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-arm64

Simply download the appropriate binary and add it to your `PATH`. Or, copy+paste one of the following commands in your terminal:

```bash
# For Linux AMD64
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64 && \
sudo install skaffold /usr/local/bin/
```

```bash
# For Linux ARM64
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-arm64 && \
sudo install skaffold /usr/local/bin/
```

We also release a **bleeding edge** build, built from the latest commit:

https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-amd64
https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-arm64

```bash
# For Linux on AMD64
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-amd64 && \
sudo install skaffold /usr/local/bin/
```

```bash
# For Linux on ARM64
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-arm64 && \
sudo install skaffold /usr/local/bin/
```

{{% /tab %}}

{{% tab "MACOS" %}}

The latest **stable** binaries can be found here:

https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-amd64
https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-arm64

Simply download the appropriate binary and add it to your `PATH`. Or, copy+paste one of the following commands in your terminal:

```bash
# For macOS on AMD64
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-amd64 && \
sudo install skaffold /usr/local/bin/
```

```bash
# For macOS on ARM64
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-amd64 && \
sudo install skaffold /usr/local/bin/
```

We also release a **bleeding edge** build, built from the latest commit:

https://storage.googleapis.com/skaffold/builds/latest/skaffold-darwin-amd64
https://storage.googleapis.com/skaffold/builds/latest/skaffold-darwin-arm64

```bash
# For macOS on AMD64
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-darwin-amd64 && \
sudo install skaffold /usr/local/bin/
```

```bash
# For macOS on ARM64
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-darwin-amd64 && \
sudo install skaffold /usr/local/bin/
```

Skaffold is also kept up to date on a few central package managers:

### Homebrew

```bash
brew install skaffold
```

### MacPorts

```bash
sudo port install skaffold
```

{{% /tab %}}

{{% tab "WINDOWS" %}}

The latest **stable** release binary can be found here:

https://storage.googleapis.com/skaffold/releases/latest/skaffold-windows-amd64.exe

Simply download it and place it in your `PATH` as `skaffold.exe`.

We also release a **bleeding edge** build, built from the latest commit:

https://storage.googleapis.com/skaffold/builds/latest/skaffold-windows-amd64.exe

---

### Chocolatey

Skaffold can be installed using the [Chocolatey package manager](https://chocolatey.org/packages/skaffold).
This package is not maintained by the Skaffold team.

{{< alert title="Caution" >}}

Chocolatey's installation mechanism interferes with <kbd>Ctrl</kbd>+<kbd>C</kbd> handling
and [prevents Skaffold from cleaning up deployments](https://github.com/GoogleContainerTools/skaffold/issues/4815).
This cannot be fixed by Skaffold.
For more information about this defect see
[chocolatey/shimgen#32](https://github.com/chocolatey/shimgen/issues/32).

{{< /alert >}}

```bash
choco install -y skaffold
```

{{% /tab %}}

{{% tab "DOCKER" %}}

### Stable binary

For the latest **stable** release, you can use:

`docker run gcr.io/k8s-skaffold/skaffold:latest skaffold <command>`

### Bleeding edge binary

For the latest **bleeding edge** build:

`docker run gcr.io/k8s-skaffold/skaffold:edge skaffold <command>`

{{% /tab %}}

{{% tab "GCLOUD" %}}

If you have the Google Cloud SDK installed on your machine, you can quickly install Skaffold as a bundled component.

Make sure your gcloud installation and the components are up to date:

`gcloud components update`

Then, install Skaffold:

`gcloud components install skaffold`

{{% /tab %}}

{{% /tabs %}}
