---
title: "Installing Skaffold v2.0.0 [NEW]"
linkTitle: "Installing Skaffold v2.0.0 [NEW]"
weight: 10
aliases: [/docs/getting-started]
---
### Standalone binary

{{% tabs %}}

{{% tab "LINUX" %}}
The latest **stable** v2.0.0 beta binaries can be found here:
- Linux x86_64 (amd64): https://storage.googleapis.com/skaffold/releases/v2.0.0/skaffold-linux-amd64
- Linux ARMv8 (arm64): https://storage.googleapis.com/skaffold/releases/v2.0.0/skaffold-linux-arm64

Simply download the appropriate binary and add it to your `PATH`. Or, copy+paste one of the following commands in your terminal:

```bash
# For Linux x86_64 (amd64)
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0/skaffold-linux-amd64 && \
sudo install skaffold /usr/local/bin/
```

```bash
# For Linux ARMv8 (arm64)
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0/skaffold-linux-arm64 && \
sudo install skaffold /usr/local/bin/
```

We also release a v2.0.0 **bleeding edge** latest build, built from the latest commit:

- Linux x86_64 (amd64): https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-amd64
- Linux ARMv8 (arm64): https://storage.googleapis.com/skaffold/builds/latest/skaffold-linux-arm64

```bash
# For Linux on x86_64 (amd64)
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/v2.0.0/skaffold-linux-amd64 && \
sudo install skaffold /usr/local/bin/
```

```bash
# For Linux on ARMv8 (arm64)
curl -Lo skaffold https://storage.googleapis.com/skaffold/builds/v2.0.0/skaffold-linux-arm64 && \
sudo install skaffold /usr/local/bin/
```

{{% /tab %}}

{{% tab "MACOS" %}}

The latest **stable** binaries can be found here:

- Darwin x86_64 (amd64): https://storage.googleapis.com/skaffold/releases/v2.0.0/skaffold-darwin-amd64
- Darwin ARMv8 (arm64): https://storage.googleapis.com/skaffold/releases/v2.0.0/skaffold-darwin-arm64

Simply download the appropriate binary and add it to your `PATH`. Or, copy+paste one of the following commands in your terminal:

```bash
# For macOS on x86_64 (amd64)
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0/skaffold-darwin-amd64 && \
sudo install skaffold /usr/local/bin/
```

```bash
# For macOS on ARMv8 (arm64)
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v2.0.0/skaffold-darwin-arm64 && \
sudo install skaffold /usr/local/bin/
```

We also release a v2.0.0 **bleeding edge** build, built from the latest commit:

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
{{% /tab %}}


{{% tab "WINDOWS" %}}

The latest **stable** release binary can be found here:

https://storage.googleapis.com/skaffold/releases/v2.0.0/skaffold-windows-amd64.exe

Simply download it and place it in your `PATH` as `skaffold.exe`.

We also release a **bleeding edge** build, built from the latest commit:

https://storage.googleapis.com/skaffold/builds/latest/skaffold-windows-amd64.exe
{{% /tab %}}

{{% tab "DOCKER" %}}
### Stable binary

For the latest v2.0.0 beta **stable** release, you can use:

`docker run gcr.io/k8s-skaffold/skaffold/v2:latest skaffold <command>`

### Bleeding edge binary

For the latest v2.0.0 **bleeding edge** build:

`docker run gcr.io/k8s-skaffold/skaffold:edge skaffold <command>`

{{% /tab %}}

{{% /tabs %}}
