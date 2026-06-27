---
title: "Weighted Canary Deployment"
linkTitle: "Weighted Canary"
weight: 45
description: >
  Shift a small percentage of load balancer traffic to a canary Auto Scaling group.
---

Goployer can run canary deployments with ALB/NLB weighted target groups.
This mode reuses the existing load balancer listener default action instead of creating a separate canary load balancer.

## How it works

1. Goployer copies the configured `healthcheck_target_group` into a new canary target group named like `hello-dev-canary-v001`.
2. Goployer creates a new Auto Scaling group and attaches it only to the canary target group.
3. After the canary Auto Scaling group passes health checks, Goployer changes the existing listener default action to split traffic between the original target group and the canary target group.
4. `--complete-canary` attaches the canary Auto Scaling group back to the original target group, restores the listener to the original target group, and removes canary-only resources.

This implementation supports listener default actions. Listener rule based canary routing is not supported yet.

## Health check rollback

If the canary Auto Scaling group fails health checks or the health check step times out after the deploy step, Goployer automatically rolls the canary back before returning the error:

1. Restore the listener default action to the original target group.
2. Detach and delete the canary target group.
3. Scale the failed canary Auto Scaling group to `0`.
4. Wait until the failed canary Auto Scaling group has no instances.
5. Delete the failed canary Auto Scaling group and its launch template.

When rollback succeeds, Goployer clears the canary deploy step status so later finish/cleanup phases do not promote the failed canary.

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

`canary.load_balancer` and `canary.listener_port` identify the listener whose default action currently forwards to `healthcheck_target_group`.
`canary.listener_protocol` is optional and useful when you want to make the listener match explicit. You can still set `canary.listener_arn` directly for backward compatibility.
`canary.weight` is the percentage of traffic sent to the canary target group. If it is omitted or set to `0`, Goployer uses `10`.
`canary.bake_time` keeps the canary traffic active before the deploy command exits. If omitted, Goployer does not wait.

## Constraints

- ALB and NLB listener default actions are supported.
- NLB weighted listeners cannot mix TCP, TLS, and UDP target groups on the same listener.
- Weighted target groups must use the same IP address type.
- NLB weight changes apply to new flows, not existing connections.
- Goployer treats `canary.weight` as a percentage, so valid values are `0` through `90`.
- `canary.bake_time` only waits. CloudWatch alarm or API test based promotion gates should be added separately.
- Automatic rollback is tied to Goployer health check failures. It does not monitor CloudWatch alarms after the deploy command exits.

## Commands

Start canary deployment:

```bash
goployer deploy --manifest examples/manifests/canary-example.yaml --stack artd
```

Complete canary deployment:

```bash
goployer deploy --manifest examples/manifests/canary-example.yaml --stack artd --complete-canary
```
