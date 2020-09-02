---
title: "Goployer 설치하기"
linkTitle: "Goployer 설치하기"
weight: 1
description: >
   여기서 최신 Goployer 버전을 다운로드 받으실 수 있습니다.
---

{{% tabs %}}
{{% tab "LINUX" %}}
최신 **stable** 버전의 바이너리 파일은 아래 경로에서 다운로드하실 수 있습니다.

https://goployer.s3.ap-northeast-2.amazonaws.com/releases/latest/goployer-linux-amd64

다운도르를 하신 후에 `PATH` 경로로 옮기시거나, 아래 명령어를 복사 + 붙여넣기하여 실행하시기 바랍니다.

```bash
curl -Lo goployer https://goployer.s3.ap-northeast-2.amazonaws.com/releases/latest/goployer-linux-amd64 && \
sudo install goployer /usr/local/bin/
```

가장 최신에 개발된 기능까지 포함해서 사용하고 싶으신 경우에는 **edge** 버전을 사용하시면 됩니다.

https://goployer.s3.ap-northeast-2.amazonaws.com/edge/latest/goployer-linux-amd64

```bash
curl -Lo goployer https://goployer.s3.ap-northeast-2.amazonaws.com/edge/latest/goployer-linux-amd64 && \
sudo install goployer /usr/local/bin/
```

{{% /tab %}}

{{% tab "MACOS" %}}

최신 **stable** 버전의 바이너리 파일은 아래 경로에서 다운로드 하실 수 있습니다.

https://goployer.s3.ap-northeast-2.amazonaws.com/edge/latest/goployer-darwin-amd64

다운도르를 하신 후에 `PATH` 경로로 옮기시거나, 아래 명령어를 복사 + 붙여넣기하여 실행하시기 바랍니다.

```bash
curl -Lo goployer https://goployer.s3.ap-northeast-2.amazonaws.com/releases/latest/goployer-darwin-amd64 && \
sudo install goployer /usr/local/bin/
```

가장 최신에 개발된 기능까지 포함해서 사용하고 싶으신 경우에는 **edge** 버전을 사용하시면 됩니다.

https://goployer.s3.ap-northeast-2.amazonaws.com/edge/latest/goployer-darwin-amd64

```bash
curl -Lo goployer https://goployer.s3.ap-northeast-2.amazonaws.com/edge/latest/goployer-darwin-amd64 && \
sudo install goployer /usr/local/bin/
```

Goployer는 몇몇 패키지 관리자를 통해서도 다운로드 받으실 수 있습니다.

### Homebrew

```bash
brew tap devopsartfactory/devopsart
brew install goployer
```

{{% /tab %}}

{{% tab "WINDOWS" %}}

최신 **stable** 버전의 바이너리 파일은 아래 경로에서 다운로드 하실 수 있습니다.

https://goployer.s3.ap-northeast-2.amazonaws.com/releases/latest/goployer-windows-amd64.exe

Simply download it and place it in your `PATH` as `goployer.exe`.
다운도르를 하신 후에 `PATH`에 `goployer.exe`라는 이름으로 저장하시기 바랍니다.

가장 최신에 개발된 기능까지 포함해서 사용하고 싶으신 경우에는 **edge** 버전을 사용하시면 됩니다.

https://goployer.s3.ap-northeast-2.amazonaws.com/edge/latest/goployer-windows-amd64.exe

{{% /tab %}}

{{% tab "DOCKER" %}}

### Stable binary

최신 **stable** 버전의 바이너리 파일이 설치된 도커 이미지는 아래 경로에서 다운로드 하실 수 있습니다.

`docker run devopsart/goployer:latest goployer <command>`

### Bleeding edge binary

가장 최신에 개발된 기능까지 포함해서 사용하고 싶으신 경우에는 **edge** 버전을 사용하시면 됩니다.

`docker run devopsart/goployer:edge goployer <command>`

{{% /tab %}}

{{% /tabs %}}
