# goployer
`goployer` is an application you can use for EC2 deployment. You can deploy in a blue/green mode. goployer only
changes the autoscaling group so that you don't need to create another load balancer or manually attach autoscaling group to target group.
<br><br>
`goployer` is still **in development** and we need the feedback from you for faster improvements. Any feedback from any channel is welcome.
<br><br>

## # Requirements
* You have to create a load balancer and target groups of it which goployer attach a new autoscaling group to. 
* If you want to setup loadbalancer and target group with terraform, then please check this [devopsart workshop](https://devops-art-factory.gitbook.io/devops-workshop/terraform/terraform-resource/computing/elb-+-ec2).
* Please understand how goployer really deploys application before applying to the real environment.
<br>

## # How goployer works
* Here's the steps that goployer executes for deployment
1. Generate new version for current deployment.<br>
If other autoscaling groups of sample application already existed, for example `hello-v001`, then next version will be `hello-v002`
2. Create a new launch template. 
3. Create autoscaling group with launch template from the previous step. A newly created autoscaling group will be automatically attached to the target groups you specified in manifest.
4. Check all instances of all stacks are healty. Until all of them pass healthchecking, it won't go to the next step.
5. (optional) If you add `autoscaling` in manifest, goployer creates autoscaling policies and put these to the autoscaling group. If you use `alarms` with autoscaling, then goployer will also create a cloudwatch alarm for autoscaling policy.
6. After all stacks are deployed, then goployer tries to delete previous versions of the same application.
   Launch templates of previous autoscaling groups are also going to be deleted.
   
<br>

## # How to run goployer
* Before applying goployer, please make sure that you have made [manifest](#Manifest).
* Here are options you can use with command line
    * `--manifest` : manifest file path
    * `--ami` : AMI ID
    * `--assume-role` : arn of IAM role you want to assume
    * `--stack` : the stack value you want to use for deployment
    * `--region` : the ID of region to which you want to deploy instances
    * `--slack-off` : whether turning off slack alarm or not. (default: false)
    * `--log-level` : level of Log (debug, info, error)
    * `--extra-tags` : extra tags to set from command line. comma-delimited string(no space between tags)
        -  ex) `--extra-tags=key1=value1,key2=value2`
    * `--ansible-extra-vars` : extra variables to be used in ansible. Will be added to tag with `ansible-extra-vars` key.
* If you sepcifies `--ami`, then you must have only one region in a stack or use `--region` option together.
* You *cannot run goployer from local environment* for security & management issue.
```bash
$ make build 
$ ./bin/goployer --manifest=configs/hello.yaml --ami=ami-01288945bd24ed49a --stack=<stack name> --region=ap-northeast-2
```
<br>

## # Spot Instance
* You can use `spot instance` option with goployer.
* There are two possible ways to use `spot instance`.


`instance_market_options` : You can set spot instance options and with this, you will only use spot instances.
```yaml
    instance_market_options:
      market_type: spot
      spot_options:
        block_duration_minutes: 180
        instance_interruption_behavior: terminate # terminate / stop / hibernate
        max_price: 0.2
        spot_instance_type: one-time # one-time or persistent
```
<br>  
  
`mixed_instances_policy` : You can mix `on-demand` and `spot` together with this setting. 
  
```yaml
    mixed_instances_policy:
      enabled: true
      override_instance_types:
        - c5.large
        - c5.xlarge
      on_demand_percentage: 20
      spot_allocation_strategy: lowest-price
      spot_instance_pools: 3
      spot_max_price: 0.3
```
 
You can see the detailed information in [manifest](#Manifest) section.

<br>

## Manifest
Manifest file is the configurations for application deployment. You need to set at least one stack for each application. You can find the example manifest file in `config/hello.yaml`.
```yaml
---
name: hello
userdata:
  type: local
  path: scripts/userdata.sh

autoscaling: &autoscaling_policy
  - name: scale_up
    adjustment_type: ChangeInCapacity
    scaling_adjustment: 1
    cooldown: 60
  - name: scale_down
    adjustment_type: ChangeInCapacity
    scaling_adjustment: -1
    cooldown: 180

alarms: &autoscaling_alarms
  - name: scale_up_on_util
    namespace: AWS/EC2
    metric: CPUUtilization
    statistic: Average
    comparison: GreaterThanOrEqualToThreshold
    threshold: 50
    period: 120
    evaluation_periods: 2
    alarm_actions:
      - scale_up
  - name: scale_down_on_util
    namespace: AWS/EC2
    metric: CPUUtilization
    statistic: Average
    comparison: LessThanOrEqualToThreshold
    threshold: 30
    period: 300
    evaluation_periods: 3
    alarm_actions:
      - scale_down

# Tags should be like "key=value"
tags:
  - project=test
  - app=hello
  - repo=hello-deploy

stacks:
  - stack: artp

    # account alias
    account: prod

    # environment variable
    env: prod

    # assume_role for deployment
    assume_role: ""

    # Replacement type
    replacement_type: BlueGreen

    # IAM instance profile, not IAM role
    iam_instance_profile: app-hello-profile

    # Ansible tags
    ansible_tags: all
    extra_vars: ""
    ebs_optimized: true

    # instance_market_options is for spot usage
    # You only can choose spot as market_type.
    # If you want to set customized stop options, then please write spot_options correctly.
    instance_market_options:
      market_type: spot
      spot_options:
        block_duration_minutes: 180
        instance_interruption_behavior: terminate # terminate / stop / hibernate
        max_price: 0.2
        spot_instance_type: one-time # one-time or persistent

    # MixedInstancesPolicy
    # You can set autoscaling mixedInstancePolicy to use on demand and spot instances together.
    # if mixed_instance_policy is set, then `instance_market_options` will be ignored.
    mixed_instances_policy:
      enabled: true

      # instance type list to override the instance types in launch template.
      override_instance_types:
        - c5.large
        - c5.xlarge

      # Proportion of on-demand instances.
      # By default, this value  will be 100 which means no spot instance.
      on_demand_percentage: 20

      # spot_allocation_strategy means in what strategy you want to allocate spot instances.
      # options could be either `lowest-price` or `capacity-optimized`.
      # by default, `low-price` strategy will be applied.
      spot_allocation_strategy: lowest-price

      # The number of spot instances pool.
      # This will be set among instance types in `override` fields
      # This will be valid only if the `spot_allocation_strategy` is low-price.
      spot_instance_pools: 3

      # Spot price.
      # By default, on-demand price will be automatically applied.
      spot_max_price: 0.3

  # block_devices is the list of ebs volumes you can use for ec2
    # device_name is required
    # If you do not set volume_size, it would be 16.
    # If you do not set volume_type, it would be gp2.
    block_devices:
      - device_name: /dev/xvda
        volume_size: 100
        volume_type: "gp2"
      - device_name: /dev/xvdb
        volume_type: "st1"
        volume_size: 500

    # capacity
    capacity:
      min: 10
      max: 10
      desired: 10

    # autoscaling means scaling policy of autoscaling group
    # You can find format in autoscaling block upside
    autoscaling: *autoscaling_policy

    # alarms means cloudwatch alarms for triggering autoscaling scaling policy
    # You can find format in alarms block upside
    alarms: *autoscaling_alarms

    # lifecycle callbacks
    lifecycle_callbacks:
      pre_terminate_past_clusters:
        - echo test
        - service hello stop

    # lifecycle hooks
    lifecycle_hooks:
      # Lifecycle hooks for launching new instances
      launch_transition:
        - lifecycle_hook_name: hello-launch-lifecycle-hook

          # The maximum time, in seconds, that can elapse before the lifecycle hook times
          # out.
          #
          # If the lifecycle hook times out, Amazon EC2 Auto Scaling performs the action
          # that you specified in the `default_result` parameter.
          heartbeat_timeout: 30

          # Defines the action the Auto Scaling group should take when the lifecycle
          # hook timeout elapses or if an unexpected failure occurs. The valid values
          # are CONTINUE and ABANDON. The default value is ABANDON.
          default_result: CONTINUE

          # Additional information that you want to include any time Amazon EC2 AutoScaling sends a message to the notification target.
          notification_metadata: "this is test for launching"

          # The ARN of the target that Amazon EC2 Auto Scaling sends notifications to
          # when an instance is in the transition state for the lifecycle hook. The notification
          # target can be either an SQS queue or an SNS topic.
          notification_target_arn: arn:aws:sns:ap-northeast-2:816736805842:test

          # The ARN of the IAM role that allows the Auto Scaling group to publish to
          # the specified notification target, for example, an Amazon SNS topic or an
          # Amazon SQS queue.
          # This is required if `notification_target_arn is not empty
          role_arn: arn:aws:iam::816736805842:role/test-autoscaling-role


    # list of region
    # deployer will concurrently deploy across the region
    regions:
      - region: ap-northeast-2

        # instance type
        instance_type: m5.large

        # ssh_key for instances
        ssh_key: test-master-key

        # ami_id
        # You can override this value via command line `--ami`
        ami_id: ami-01288945bd24ed49a

        # Whether you want to use public subnet or not
        # By Default, deployer selects private subnets
        # If you want to use public subnet, then you should set this value to ture.
        use_public_subnets: true

        # You can use VPC id(vpc-xxx)
        # If you specify the name of VPC, then deployer will find the VPC id with it.
        # In this case, only one VPC should exist.
        vpc: vpc-artp_apnortheast2

        # You can use security group id(sg-xxx)
        # If you specify the name of security group, then deployer will find the security group id with it.
        # In this case, only one security group should exist
        security_groups:
          - hello-artp_apnortheast2
          - default-artp_apnortheast2

        # You can use healthcheck target group
        healthcheck_target_group: hello-artpapne2-ext

        # If no availability zones specified, then all availability zones are selected by default.
        # If you want all availability zones, then please remove availability_zones key.
        availability_zones:
          - ap-northeast-2a
          - ap-northeast-2b
          - ap-northeast-2c

        # list of target groups.
        # The target group in the healthcheck_target_group should be included here.
        target_groups:
          - hello-artpapne2-ext


      - region: us-east-1
        ami_id: ami-09d95fab7fff3776c
        instance_type: t3.large
        ssh_key: art-prod-master
        use_public_subnets: true
        vpc: vpc-artp_useast1
        security_groups:
          - hello-artp_useast1
          - default-artp_useast1
        healthcheck_target_group: hello-artpuse1-ext
        target_groups:
          - hello-artpuse1-ext
``` 


