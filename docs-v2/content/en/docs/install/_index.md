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

To help prioritize features and work on improving Skaffold, we collect anonymized Skaffold usage data.
You can opt out of data collection by following [these instructions]({{<relref "/docs/resources/telemetry">}}).

Your use of this software is subject to the [Google Privacy Policy](https://policies.google.com/privacy)

{{< /alert >}}

### Managed IDE

{{% tabs %}}

{{% tab "CLOUD CODE" %}}

[Cloud Code](https://cloud.google.com/code) provides a managed experience of using Skaffold in supported IDEs. You can install the `Cloud Code` extension for [Visual Studio Code]([https://cloud.google.com/code/docs/vscode/quickstart-k8s#installing](https://cloud.google.com/code/docs/vscode/install#installing)) or the plugin for [JetBrains IDEs](https://cloud.google.com/code/docs/intellij/quickstart-k8s#installing_the_plugin). It manages and keeps Skaffold  up-to-date, along with other common dependencies, and works with any kubernetes cluster.

{{% /tab %}}

{{% tab "GOOGLE CLOUD SHELL" %}}

Google Cloud Platform's [_Cloud Shell_](http://cloud.google.com/shell)
provides a free [browser-based terminal/CLI and editor](https://cloud.google.com/shell#product-demo)
with Skaffold, Minikube, and Docker pre-installed.
(Requires a [Google Account](https://accounts.google.com/SignUp).)

Cloud Shell is a great way to try Skaffold out.

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?shellonly=true&cloudshell_git_repo=https%3A%2F%2Fgithub.com%2FGoogleContainerTools%2Fskaffold&cloudshell_working_dir=examples%2Fgetting-started)

{{% /tab %}}

{{% /tabs %}}

### Standalone binary

{{% tabs %}}

{{% tab "LINUX" %}}
The latest **stable** binaries can be found here:

- Linux x86_64 (amd64): https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64
- Linux ARMv8 (arm64): https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-arm64

Simply download the appropriate binary and add it to your `PATH`. Or, copy+paste one of the following commands in your terminal:

```bash
# For Linux x86_64 (amd64)
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64 && \
sudo install skaffold /usr/local/bin/
```

```bash
# For Linux ARMv8 (arm64)
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-arm64 && \
sudo install skaffold /usr/local/bin/
```

We also release a **bleeding edge** build, built from the latest commit:

- Linux x86_64 (amd64): https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-amd64
- Linux ARMv8 (arm64): https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-arm64

```bash
# For Linux on x86_64 (amd64)
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-amd64 && \
sudo install skaffold /usr/local/bin/
```

```bash
# For Linux on ARMv8 (arm64)
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-arm64 && \
sudo install skaffold /usr/local/bin/
```

{{% /tab %}}

{{% tab "MACOS" %}}

The latest **stable** binaries can be found here:

- Darwin x86_64 (amd64): https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-amd64
- Darwin ARMv8 (arm64): https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-arm64

Simply download the appropriate binary and add it to your `PATH`. Or, copy+paste one of the following commands in your terminal:

```bash
# For macOS on x86_64 (amd64)
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-amd64 && \
sudo install skaffold /usr/local/bin/
```

```bash
# For macOS on ARMv8 (arm64)
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-arm64 && \
sudo install skaffold /usr/local/bin/
```

We also release a **bleeding edge** build, built from the latest commit:

- Darwin x86_64 (amd64): https://storage.googleapis.com/skaffold/builds/latest/skaffold-darwin-amd64
- Darwin ARMv8 (arm64): https://storage.googleapis.com/skaffold/builds/latest/skaffold-darwin-arm64

```bash
# For macOS on x86_64 (amd64)
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-darwin-amd64 && \
sudo install skaffold /usr/local/bin/
```

```bash
# For macOS on ARMv8 (arm64)
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/latest/skaffold-darwin-arm64 && \
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

### Scoop

Skaffold can be installed using the [Scoop package manager](https://scoop.sh/)
from the [extras bucket](https://github.com/lukesampson/scoop-extras#readme).
This package is not maintained by the Skaffold team.

```powershell
scoop bucket add extras
scoop install skaffold
```

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

{{% tab "GCLOUD" %}}

If you have the Google Cloud SDK installed on your machine, you can quickly install Skaffold as a bundled component.

Make sure your gcloud installation and the components are up to date:

`gcloud components update`

Then, install Skaffold:

`gcloud components install skaffold`

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
