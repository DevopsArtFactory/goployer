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

https://goployer.s3.ap-northeast-2.amazonaws.com/releases/latest/goployer-linux-amd64

Simply download it and add it to your `PATH`. Or, copy+paste this command in your terminal:

```bash
curl -Lo goployer https://goployer.s3.ap-northeast-2.amazonaws.com/releases/latest/goployer-linux-amd64 && \
sudo install goployer /usr/local/bin/
```

We also release a **bleeding edge** build, built from the latest commit:

https://goployer.s3.ap-northeast-2.amazonaws.com/edge/latest/goployer-linux-amd64

```bash
curl -Lo goployer https://goployer.s3.ap-northeast-2.amazonaws.com/edge/latest/goployer-linux-amd64 && \
sudo install goployer /usr/local/bin/
```

{{% /tab %}}

{{% tab "MACOS" %}}

The latest **stable** binary can be found here:

https://storage.googleapis.com/goployer/releases/latest/goployer-darwin-amd64

Simply download it and add it to your `PATH`. Or, copy+paste this command in your terminal:

```bash
curl -Lo goployer https://goployer.s3.ap-northeast-2.amazonaws.com/releases/latest/goployer-darwin-amd64 && \
sudo install goployer /usr/local/bin/
```

We also release a **bleeding edge** build, built from the latest commit:

https://storage.googleapis.com/goployer/edge/latest/goployer-darwin-amd64

```bash
curl -Lo goployer https://goployer.s3.ap-northeast-2.amazonaws.com/edge/latest/goployer-darwin-amd64 && \
sudo install goployer /usr/local/bin/
```

goployer is also kept up to date on a few central package managers:

### Homebrew

```bash
brew install goployer
```

{{% /tab %}}

{{% tab "WINDOWS" %}}

The latest **stable** release binary can be found here:

https://goployer.s3.ap-northeast-2.amazonaws.com/releases/latest/goployer-windows-amd64.exe

Simply download it and place it in your `PATH` as `goployer.exe`.

We also release a **bleeding edge** build, built from the latest commit:

https://goployer.s3.ap-northeast-2.amazonaws.com/edge/latest/goployer-windows-amd64.exe

{{% /tab %}}

{{% tab "DOCKER" %}}

### Stable binary

For the latest **stable** release, you can use:

`docker run devopsart/goployer:latest goployer <command>`

### Bleeding edge binary

For the latest **bleeding edge** build:

`docker run devopsart/goployer:edge goployer <command>`

{{% /tab %}}

{{% /tabs %}}
