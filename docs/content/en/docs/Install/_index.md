---
title: "Install goployer"
linkTitle: "Install goployer"
weight: 1
description: >
  Here's where your user finds out if your project is for them.
---


{{< alert title="Note" >}}

To keep goployer up to date, update checks are made to Google servers to see if a new version of
goployer is available.


Your use of this software is subject to the [Google Privacy Policy](https://policies.google.com/privacy)

{{< /alert >}}


{{% tabs %}}
{{% tab "LINUX" %}}
The latest **stable** binary can be found here:

https://storage.googleapis.com/goployer/releases/latest/goployer-linux-amd64

Simply download it and add it to your `PATH`. Or, copy+paste this command in your terminal:

```bash
curl -Lo goployer https://storage.googleapis.com/goployer/releases/latest/goployer-linux-amd64 && \
sudo install goployer /usr/local/bin/
```

We also release a **bleeding edge** build, built from the latest commit:

https://storage.googleapis.com/goployer/builds/latest/goployer-linux-amd64

```bash
curl -Lo goployer https://storage.googleapis.com/goployer/builds/latest/goployer-linux-amd64 && \
sudo install goployer /usr/local/bin/
```

{{% /tab %}}

{{% tab "MACOS" %}}

The latest **stable** binary can be found here:

https://storage.googleapis.com/goployer/releases/latest/goployer-darwin-amd64

Simply download it and add it to your `PATH`. Or, copy+paste this command in your terminal:

```bash
curl -Lo goployer https://storage.googleapis.com/goployer/releases/latest/goployer-darwin-amd64 && \
sudo install goployer /usr/local/bin/
```

We also release a **bleeding edge** build, built from the latest commit:

https://storage.googleapis.com/goployer/builds/latest/goployer-darwin-amd64

```bash
curl -Lo goployer https://storage.googleapis.com/goployer/builds/latest/goployer-darwin-amd64 && \
sudo install goployer /usr/local/bin/
```

goployer is also kept up to date on a few central package managers:

### Homebrew

```bash
brew install goployer
```

### MacPorts

```bash
sudo port install goployer
```

{{% /tab %}}

{{% tab "WINDOWS" %}}

The latest **stable** release binary can be found here:

https://storage.googleapis.com/goployer/releases/latest/goployer-windows-amd64.exe

Simply download it and place it in your `PATH` as `goployer.exe`.

We also release a **bleeding edge** build, built from the latest commit:

https://storage.googleapis.com/goployer/builds/latest/goployer-windows-amd64.exe


### Chocolatey

```bash
choco install -y goployer
```

{{% /tab %}}

{{% tab "DOCKER" %}}

### Stable binary

For the latest **stable** release, you can use:

`docker run gcr.io/k8s-goployer/goployer:latest goployer <command>`

### Bleeding edge binary

For the latest **bleeding edge** build:

`docker run gcr.io/k8s-goployer/goployer:edge goployer <command>`

{{% /tab %}}

{{% tab "GCLOUD" %}}

If you have the Google Cloud SDK installed on your machine, you can quickly install goployer as a bundled component.

Make sure your gcloud installation and the components are up to date:

`gcloud components update`

Then, install goployer:

`gcloud components install goployer`

{{% /tab %}}

{{% /tabs %}}
