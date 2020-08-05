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

<br>

Total Deployment Process:
* [goployer deploy](#goployer-deploy) - to deploy a new application
* [goployer delete](#goployer-delete) - to delete previous applications

## goployer init
- setup goployer project
```
Examples:
  # Minimum argument
  goployer init

  # See log
  goployer init --log-level=debug

Options:
      --log-level string                Level of logging
```

- All project files will be created in current directory.
  - manifests
  - scripts
  - metric.yaml
<br>

## goployer add
- add new goployer manifest file 
```
Examples:
  # Minimum argument
  goployer add 

  # You can specify application name from command
  goployer add hello

Options:
      --log-level string                Level of logging
```
<br>


## goployer status
-  Retrieve information of the specific deployment
```
Examples:
  # Minimum argument
  goployer status hello --region=ap-northeast-2

Options:
      --region string                  Region of autoscaling group
```
<br>

## goployer deploy
- Deploy a new application

```

Examples:
  # Minimum argument
  goployer deploy --manifest=configs/hello.yaml --stack=artd --region=ap-northeast-2

  # Use manifest from s3
  goployer deploy --manifest=s3://goployer/manifest/hello.yaml --manifest-s3-region=ap-northeast-2 --stack=artd --region=ap-northeast-2

  # Turn off slack notification
  goployer deploy --manifest=configs/hello.yaml --stack=artd --region=ap-northeast-2 --slack-off=true

  # Control polling interval for healthcheck
  goployer deploy --manifest=configs/hello.yaml --stack=artd --region=ap-northeast-2 --polling-interval=30s

Options:
  -m, --manifest string                 The manifest configuration file to use. (required)
      --stack string                    stack that should be deployed.(required)
      --manifest-s3-region string       Region of bucket containing the manifest configuration file to use. (required if –manifest starts with s3://)
      --ami string                      Amazon AMI to use.
      --ansible-extra-vars string       Extra variables for ansible
      --assume-role string              The Role ARN to assume into.
      --disable-metrics                 Disable gathering metrics.
      --env string                      The environment that is being deployed into.
      --extra-tags string               Extra tags to add to autoscaling group tags
      --force-manifest-capacity         Force-apply the capacity of instances in the manifest file
      --log-level string                Level of logging
      --override-instance-type string   Instance Type to override
      --polling-interval duration       Time to interval for polling health check (default 60s) (default 1m0s)
      --region string                   The region to deploy into, if undefined, then the deployment will run against all regions for the given environment.
      --release-notes string            Release note for the current deployment
      --release-notes-base64 string     Base64 encoded string of release note for the current deployment
      --slack-off                       Turn off slack alarm
      --timeout duration                Time to wait for deploy to finish before timing out (default 60m) (default 1h0m0s)
      --auto-apply                      Apply command without confirmation from local terminal
```
<br>

### Further information
* If you specifies `--ami`, then you must have only one region in a stack or use `--region` option together.

## goployer delete
- Delete previous applications
```

Examples:
  # Minimum argument
  goployer delete --manifest=configs/hello.yaml --stack=artd --region=ap-northeast-2

  # Use manifest from s3
  goployer delete --manifest=s3://goployer/manifest/hello.yaml --manifest-s3-region=ap-northeast-2 --stack=artd --region=ap-northeast-2

  # Control polling interval for healthcheck
  goployer delete --manifest=configs/hello.yaml --stack=artd --region=ap-northeast-2 --polling-interval=30s

Options:
  -m, --manifest string                 The manifest configuration file to use. (required)
      --stack string                    stack that should be deployed.(required)
      --manifest-s3-region string       Region of bucket containing the manifest configuration file to use. (required if –manifest starts with s3://)
      --region string                   The region to deploy into, if undefined, then the deployment will run against all regions for the given environment.
      --assume-role string              The Role ARN to assume into.
      --disable-metrics                 Disable gathering metrics.
      --env string                      The environment that is being deployed into.
      --log-level string                Level of logging
      --polling-interval duration       Time to interval for polling health check (default 60s) (default 1m0s)
      --timeout duration                Time to wait for deploy to finish before timing out (default 60m) (default 1h0m0s)
      --auto-apply                      Apply command without confirmation from local terminal
```
<br>

