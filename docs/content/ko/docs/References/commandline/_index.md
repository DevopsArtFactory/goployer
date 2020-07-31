---
title: "명령어 인자"
linkTitle: "명령어 인자"
weight: 10
---
현재 Goployer는 배포명령어 밖에 없습니다.<br>
*나중에, Goployer에 다양한 명령어가 추가될 예정입니다.*
<br>

전체 배포 과정 실행:
* [goployer](#argument) - 배포 실행

## 예시
```

Examples:
  # Minimum argument
  goployer --manifest=configs/hello.yaml --stack=artd --region=ap-northeast-2

  # Use manifest from s3
  goployer --manifest=s3://goployer/manifest/hello.yaml --manifest-s3-region=ap-northeast-2 --stack=artd --region=ap-northeast-2

  # Turn off slack notification
  goployer --manifest=configs/hello.yaml --stack=artd --region=ap-northeast-2 --slack-off=true

  # Control polling interval for healthcheck
  goployer --manifest=configs/hello.yaml --stack=artd --region=ap-northeast-2 --polling-interval=30s
```
<br>

## Arguments:
* Here are options you can use with command line
  * `--manifest` : manifest file path (required)
  * `--manifest-s3-region` : region of S3 bucket containing manifest file 
      (required if --manifest starts with s3://)
  * `--stack` : the stack value you want to use for deployment (required)
  * `--region` : the ID of region to which you want to deploy instances
  * `--ami` : AMI ID
  * `--assume-role` : arn of IAM role you want to assume
  * `--timeout` : timeout duration of total deployment process (default: 60m)
  * `--slack-off` : whether turning off slack alarm or not. (default: false)
  * `--log-level` : level of Log (debug, info, error)
  * `--extra-tags` : extra tags to set from command line. comma-delimited string(no space between tags)
      -  ex) --extra-tags=key1=value1,key2=value2
  * `--ansible-extra-vars` : extra variables to be used in ansible. Will be added to tag with `ansible-extra-vars` key.
  * `--override-instance-type` : instance type you want to override when running goployer command.
  * `--release-notes` : Release notes for deployment.
  * `--release-notes-base64` : Release notes for deployment encoded with base64
  * `--polling-interval` : Time to interval for polling health check (default 60s) 
<br>

## 추가 정보
* 만약 `--ami`를 명시한 경우에는 stack에 하나의 리전만 있거나 `--region`을 통해 리전을 명시해주어야 합니다.
