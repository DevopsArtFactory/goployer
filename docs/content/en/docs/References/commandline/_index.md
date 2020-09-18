---
title: "CLI Argument"
linkTitle: "CLI Argument"
weight: 10
---

<br>

Initiate project:
* [goployer init](#goployer-init) - setup goployer project
* [goployer add](#goployer-add) - add goployer new manifest file 

<br>

Retrieve and Modify deployment:
* [goployer status](#goployer-status) -  Retrieve information of the specific deployment
* [goployer update](#goployer-update) -  Update configuration of deployment without re-deployment

<br>

Total Deployment Process:
* [goployer deploy](#goployer-deploy) - to deploy a new application
* [goployer delete](#goployer-delete) - to delete previous applications

## goployer init
- setup goployer project

```bash
Examples:
  # Minimum argument
  goployer init

  # See log
  goployer init --log-level=debug

Flags:
  -h, --help             help for init
  -p, --profile string   Profile configuration of AWS

Global Flags:
  -v, --log-level string   Log level (debug, info, warn, error, fatal, panic) (default "warning")
```

- All project files will be created in the current directory.
  - manifests
  - scripts
  - metric.yaml
<br>

## goployer add
- add new goployer manifest file 

```bash
Examples:
  # Minimum argument
  goployer add 

  # You can specify application name from command
  goployer add hello

Flags:
  -h, --help             help for add
  -p, --profile string   Profile configuration of AWS

Global Flags:
  -v, --log-level string   Log level (debug, info, warn, error, fatal, panic) (default "warning")
```
<br>


## goployer status
-  Retrieve information of the specific deployment

```bash
Examples:
  # Minimum argument
  goployer status hello 

  # With region
  goployer status hello --region=ap-northeast-2

Usage:
  goployer status [flags]

Flags:
  -h, --help             help for status
  -p, --profile string   Profile configuration of AWS
      --region string    Region of autoscaling group

Global Flags:
  -v, --log-level string   Log level (debug, info, warn, error, fatal, panic) (default "warning")
```

```bash
$ goployer status hello
? Choose autoscaling group: hello-dev_apnortheast2-v003
Name:           hello-dev_apnortheast2-v003
Created Time:   2020-09-16 10:29:21.169 +0000 UTC

📦 Capacity
MINIMUM    DESIRED    MAXIMUM
1          1          2

🖥 Instance Statistics
 ∙ t3.medium: 1

⚓ Tags
 ∙ Name=hello-dev_apnortheast2-v003
 ∙ ansible-tags=all
 ∙ app=hello
 ∙ project=test
 ∙ repo=hello-deploy
 ∙ stack=_apnortheast2
 ∙ stack-name=artd
 ∙ test=test
```
<br>

## goployer update
-  Update configuration of deployment without re-deployment
  - Capacity modification: change value of min/desired/max

```bash
Examples:
  # Minimum argument
  # at least one of `--min, --max, --desired` is needed
  goployer update hello --desired=1 --min=0 --max=1

  # Auto apply without confirmation
  goployer update hello --desired=1 --auto-apply

  # Update with other options
  goployer update hello --desired=1 --region=ap-northeast-2 --auto-apply --polling-interval=20s

Usage:
  goployer update name-prefix [flags] 

Flags:
      --auto-apply                  Apply command without confirmation from local terminal
      --desired int                 Desired instance capacity you want to update with (default -1)
  -h, --help                        help for update
      --max int                     Maximum instance capacity you want to update with (default -1)
      --min int                     Minimum instance capacity you want to update with (default -1)
      --polling-interval duration   Time to interval for polling health check (default 60s) (default 1m0s)
  -p, --profile string              Profile configuration of AWS
      --region string               Region of autoscaling group
      --timeout duration            Time to wait for deploy to finish before timing out (default 60m) (default 1h0m0s)

Global Flags:
  -v, --log-level string   Log level (debug, info, warn, error, fatal, panic) (default "warning")
```
<br>


## goployer deploy
- Deploy a new application

```bash
Examples:
  # Minimum argument
  goployer deploy --manifest=configs/hello.yaml --stack=artd --region=ap-northeast-2

  # Use manifest from s3
  goployer deploy --manifest=s3://goployer/manifest/hello.yaml --manifest-s3-region=ap-northeast-2 --stack=artd --region=ap-northeast-2

  # Turn off slack notification
  goployer deploy --manifest=configs/hello.yaml --stack=artd --region=ap-northeast-2 --slack-off=true

  # Control polling interval for healthcheck
  goployer deploy --manifest=configs/hello.yaml --stack=artd --region=ap-northeast-2 --polling-interval=30s

Flags:
      --ami string                      Amazon AMI to use.
      --ansible-extra-vars string       Extra variables for ansible
      --assume-role string              The Role ARN to assume into.
      --auto-apply                      Apply command without confirmation from local terminal
      --disable-metrics                 Disable gathering metrics.
      --env string                      The environment that is being deployed into.
      --extra-tags string               Extra tags to add to autoscaling group tags
      --force-manifest-capacity         Force-apply the capacity of instances in the manifest file
  -h, --help                            help for deploy
  -m, --manifest string                 The manifest configuration file to use. (required)
      --manifest-s3-region string       Region of bucket containing the manifest configuration file to use. (required if –manifest starts with s3://)
      --override-instance-type string   Instance Type to override
      --polling-interval duration       Time to interval for polling health check (default 60s) (default 1m0s)
  -p, --profile string                  Profile configuration of AWS
      --region string                   The region to deploy into, if undefined, then the deployment will run against all regions for the given environment.
      --release-notes string            Release note for the current deployment
      --release-notes-base64 string     Base64 encoded string of release note for the current deployment
      --slack-off                       Turn off slack alarm
      --stack string                    stack that should be deployed.(required)
      --timeout duration                Time to wait for deploy to finish before timing out (default 60m) (default 1h0m0s)

Global Flags:
  -v, --log-level string   Log level (debug, info, warn, error, fatal, panic) (default "warning")
```
<br>

### Further information
* If you specifies `--ami`, then you must have only one region in a stack or use `--region` option together.

## goployer delete
- Delete previous applications

```bash
Examples:
  # Minimum argument
  goployer delete --manifest=configs/hello.yaml --stack=artd --region=ap-northeast-2

  # Use manifest from s3
  goployer delete --manifest=s3://goployer/manifest/hello.yaml --manifest-s3-region=ap-northeast-2 --stack=artd --region=ap-northeast-2

  # Control polling interval for healthcheck
  goployer delete --manifest=configs/hello.yaml --stack=artd --region=ap-northeast-2 --polling-interval=30s

Flags:
      --ami string                      Amazon AMI to use.
      --ansible-extra-vars string       Extra variables for ansible
      --assume-role string              The Role ARN to assume into.
      --auto-apply                      Apply command without confirmation from local terminal
      --disable-metrics                 Disable gathering metrics.
      --env string                      The environment that is being deployed into.
      --extra-tags string               Extra tags to add to autoscaling group tags
      --force-manifest-capacity         Force-apply the capacity of instances in the manifest file
  -h, --help                            help for delete
  -m, --manifest string                 The manifest configuration file to use. (required)
      --manifest-s3-region string       Region of bucket containing the manifest configuration file to use. (required if –manifest starts with s3://)
      --override-instance-type string   Instance Type to override
      --polling-interval duration       Time to interval for polling health check (default 60s) (default 1m0s)
  -p, --profile string                  Profile configuration of AWS
      --region string                   The region to deploy into, if undefined, then the deployment will run against all regions for the given environment.
      --release-notes string            Release note for the current deployment
      --release-notes-base64 string     Base64 encoded string of release note for the current deployment
      --slack-off                       Turn off slack alarm
      --stack string                    stack that should be deployed.(required)
      --timeout duration                Time to wait for deploy to finish before timing out (default 60m) (default 1h0m0s)

Global Flags:
  -v, --log-level string   Log level (debug, info, warn, error, fatal, panic) (default "warning")
```
<br>

