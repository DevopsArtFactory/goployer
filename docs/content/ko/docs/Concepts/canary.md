---
title: "Weighted Canary Deployment"
linkTitle: "Weighted Canary"
weight: 45
description: >
  Load Balancer weighted target groups를 이용해 일부 트래픽만 canary Auto Scaling group으로 보냅니다.
---

Goployer의 canary 배포는 ALB/NLB weighted target groups를 사용합니다.
별도의 canary load balancer를 만들지 않고, 기존 load balancer listener의 default action을 수정합니다.

## 동작 방식

1. `healthcheck_target_group`으로 지정된 기존 target group을 복사해 `hello-dev-canary-v001` 같은 canary target group을 만듭니다.
2. 새 Auto Scaling group을 만들고 canary target group에만 붙입니다.
3. canary Auto Scaling group이 health check를 통과하면 기존 listener default action을 original target group과 canary target group의 weighted forward로 변경합니다.
4. `--complete-canary`를 실행하면 canary Auto Scaling group을 original target group에 붙이고, listener를 original target group으로 복구한 뒤 canary 전용 리소스를 정리합니다.

현재 구현은 listener default action만 지원합니다. host/path 기반 listener rule canary는 아직 지원하지 않습니다.

## Manifest

```yaml
replacement_type: Canary
regions:
  - region: ap-northeast-2
    healthcheck_target_group: hello-artdapne2-ext
    target_groups:
      - hello-artdapne2-ext
    canary:
      load_balancer: hello-artdapne2-ext
      listener_port: 443
      listener_protocol: HTTPS
      weight: 10
      bake_time: 10m
```

`canary.load_balancer`와 `canary.listener_port`는 현재 `healthcheck_target_group`으로 forward하고 있는 listener를 찾는 데 사용합니다.
`canary.listener_protocol`은 같은 port/protocol을 명확히 표시하고 싶을 때 지정합니다. 기존 방식처럼 `canary.listener_arn`을 직접 지정할 수도 있습니다.
`canary.weight`는 canary target group으로 보낼 트래픽 비율입니다. 생략하거나 `0`이면 기본값 `10`을 사용합니다.
`canary.bake_time`은 canary traffic을 유지하며 배포 명령이 기다릴 시간입니다. 생략하면 기다리지 않습니다.

## 제약사항

- ALB와 NLB listener default action만 지원합니다.
- NLB는 같은 listener 안에서 TCP/TLS/UDP target group을 섞을 수 없습니다.
- weighted target groups는 같은 IP address type의 target group끼리 사용해야 합니다.
- NLB weight 변경은 기존 연결이 아니라 새 flow부터 반영됩니다.
- Goployer의 `canary.weight`는 percentage 값이라 `0`부터 `90`까지만 허용합니다.
- `canary.bake_time`은 시간 대기만 수행합니다. CloudWatch alarm이나 API test 기반 자동 승격 판단은 별도 승격 조건으로 구성해야 합니다.

## Commands

Canary 배포 시작:

```bash
goployer deploy --manifest examples/manifests/canary-example.yaml --stack artd
```

Canary 배포 완료:

```bash
goployer deploy --manifest examples/manifests/canary-example.yaml --stack artd --complete-canary
```
