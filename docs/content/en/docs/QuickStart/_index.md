---
title: "Quickstart"
linkTitle: "Quickstart"
weight: 2
description: >
  You can easily start to use goployer.
---
Follow this tutorial to learn about Goployer on AWS Environment with autoscaling group and deploy the sample application
with goployer.

{{< alert title="Important Notice">}}
Goployer is a deployment tool which means you are going to `deploy the application with EC2 instance`. So **cost for AWS resources could be charged**! 
{{</alert>}}

In this quickstart, you will:

* Install goployer,
* Download a sample go app,
* Use `goployer dev` to build and deploy your app every time your code changes,
* Use `goployer run` to build and deploy your app once, similar to a CI/CD pipeline

## Before you begin

* [Install Goployer]({{< relref "/docs/install" >}})
* [Install kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [Install minikube](https://minikube.sigs.k8s.io/docs/start/)

{{< alert title="Note">}}
goployer will build the app using the Docker daemon hosted inside minikube. 
If you want to deploy against a different Kubernetes cluster, e.g. Kind, GKE clusters, you will have to install Docker to build this app.
{{</alert>}}

## Downloading the sample app

1. Clone the goployer repository:

    ```bash
    git clone https://github.com/GoogleContainerTools/goployer
    ```

1. Change to the `examples/getting-started` in goployer directory.

    ```bash
    cd goployer/examples/getting-started
    ```

## `goployer dev`: continuous build & deploy on code changes

Run `goployer dev` to build and deploy your app continuously.
You should see some outputs similar to the following entries:

```
Listing files to watch...
 - goployer-example
Generating tags...
 - goployer-example -> goployer-example:v1.1.0-113-g4649f2c16
Checking cache...
 - goployer-example: Not found. Building
Found [docker-desktop] context, using local docker daemon.
Building [goployer-example]...
Sending build context to Docker daemon  3.072kB
Step 1/6 : FROM golang:1.12.9-alpine3.10 as builder
 ---> e0d646523991
Step 2/6 : COPY main.go .
 ---> Using cache
 ---> e4788ffa88e7
Step 3/6 : RUN go build -o /app main.go
 ---> Using cache
 ---> 686396d9e9cc
Step 4/6 : FROM alpine:3.10
 ---> 965ea09ff2eb
Step 5/6 : CMD ["./app"]
 ---> Using cache
 ---> be0603b9d79e
Step 6/6 : COPY --from=builder /app .
 ---> Using cache
 ---> c827aa5a4b12
Successfully built c827aa5a4b12
Successfully tagged goployer-example:v1.1.0-113-g4649f2c16
Tags used in deployment:
 - goployer-example -> goployer-example:c827aa5a4b12e707163842b803d666eda11b8ec20c7a480198960cfdcb251042
   local images can't be referenced by digest. They are tagged and referenced by a unique ID instead
Starting deploy...
 - pod/getting-started created
Watching for changes...
[getting-started] Hello world!
[getting-started] Hello world!
[getting-started] Hello world!

```

`goployer dev` watches your local source code and executes your goployer pipeline
every time a change is detected. `goployer.yaml` provides specifications of the
workflow - in this example, the pipeline is

* Building a Docker image from the source using the Dockerfile
* Tagging the Docker image with the `sha256` hash of its contents
* Updating the Kubernetes manifest, `k8s-pod.yaml`, to use the image built previously
* Deploying the Kubernetes manifest using `kubectl apply -f`
* Streaming the logs back from the deployed app

Let's re-trigger the workflow just by a single code change!
Update `main.go` as follows:

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("Hello goployer!")
		time.Sleep(time.Second * 1)
	}
}
```

When you save the file, goployer will see this change and repeat the workflow described in
`goployer.yaml`, rebuilding and redeploying your application. Once the pipeline
is completed, you should see the changes reflected in the output in the terminal:

```
[getting-started] Hello goployer!
```

<span style="font-size: 36pt">✨</span>

## `goployer run`: build & deploy once 

If you prefer building and deploying once at a time, run `goployer run`.
goployer will perform the workflow described in `goployer.yaml` exactly once.

## What's next


:mega: **Please fill out our [quick 5-question survey](https://forms.gle/BMTbGQXLWSdn7vEs6)** to tell us how satisfied you are with goployer, and what improvements we should make. Thank you! :dancers:
