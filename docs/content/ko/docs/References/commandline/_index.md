---
title: "명령어 인자"
linkTitle: "명령어 인자"
weight: 10
---

<br>

전체 배포 과정 실행:
* [goployer deploy](#goployer-deploy) - 배포 실행

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

Flags:
      --ami string                      Amazon AMI to use.
      --ansible-extra-vars string       Extra variables for ansible
      --assume-role string              The Role ARN to assume into.
      --disable-metrics                 Disable gathering metrics.
      --env string                      The environment that is being deployed into.
      --extra-tags string               Extra tags to add to autoscaling group tags
      --force-manifest-capacity         Force-apply the capacity of instances in the manifest file
  -h, --help                            help for deploy
      --log-level string                Level of logging
  -m, --manifest string                 The manifest configuration file to use. (required)
      --manifest-s3-region string       Region of bucket containing the manifest configuration file to use. (required if –manifest starts with s3://)
      --override-instance-type string   Instance Type to override
      --polling-interval duration       Time to interval for polling health check (default 60s) (default 1m0s)
      --region string                   The region to deploy into, if undefined, then the deployment will run against all regions for the given environment.
      --release-notes string            Release note for the current deployment
      --release-notes-base64 string     Base64 encoded string of release note for the current deployment
      --slack-off                       Turn off slack alarm
      --stack string                    stack that should be deployed.(required)
      --timeout duration                Time to wait for deploy to finish before timing out (default 60m) (default 1h0m0s)
```
<br>

## 추가 정보
* 만약 `--ami`를 명시한 경우에는 stack에 하나의 리전만 있거나 `--region`을 통해 리전을 명시해주어야 합니다.
