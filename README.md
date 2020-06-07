# deployer
`deployer` is an application you can use for EC2 deployment. You can deploy in a blue/green mode. Deployer only
changes the autoscaling group so that you don't need to create another load balancer or manually attach autoscaling group to target group.
<br><br>
`deployer` is still **in development** and we need the feedback from you for faster improvements. Any feedback from any channel is welcome.
<br>
## Requirements
* You have to create a load balancer and target groups of it which deployer attach a new autoscaling group to. 
* If you want to setup loadbalancer and target group in terraform, then please check this [link](https://devops-art-factory.gitbook.io/devops-workshop/terraform/terraform-resource/computing/elb-+-ec2).
* Please understand how deployer really deploys application before applying to the real environment.
<br>

## how deployer works
* Here's the steps that deployer executes for deployment
1. Generate new version for current deployment.<br>
If the autoscaling of sample application is already launched, for example `hello-v001`, then next version will be `hello-v002`

2. Create launch template. 
3. Create autoscaling group with launch template from previous step. Newly created autoscaling group is automatically attached to the target groups you specified in manifest.
4. Check all instaces of all stacks are healty. Until all of them pass healthchecking, it won't go to the next step.
5. (optional) If you add `autoscaling` in manifest, deployer creates autoscaling policies and put these to the autoscaling group. If you use `alarms` with autoscaling, then deployer will also create cloudwatch alarm for autoscaling policy.
6. After all of stacks are deployed well, then deployer tries to delete previous versions of the same application.
   Launch templates of previous autoscaling groups are also going to be deleted.
   
<br>

## How to run deployer
* Before applying deployer, please make sure that you have made [manifest](#Manifest).
* Here's the options you can use with command line
    * `--manifest` : manifest file path
    * `--ami` : AMI ID
    * `--stack` : the stack value you want to use for deployment
    * `--region` : the ID of region to which you want to deploy instances
* If you sepcifies `--ami`, then you must have only one region in a stack or use `--region` option together.
```bash
go run deployer.go --manifest=configs/hello.yaml --ami=ami-01288945bd24ed49a --stack=<stack name> --region=ap-northeast-2
```
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
  - stack: dayonep

    # account alias
    account: prod

    # environment variable
    env: prod

    # assume_role for deployment
    assume_role: ""

    # Replacement type
    replacement_type: BlueGreen

    # IAM instance profile, not instance role
    iam_instance_profile: 'app-hello-profile'

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
      min: 1
      max: 2
      desired: 1

    # autoscaling means scaling policy of autoscaling group
    # You can find format in autoscaling block upside
    autoscaling: *autoscaling_policy

    # alarms means cloudwatch alarms for triggering autoscaling scaling policy
    # You can find format in alarms block upside
    alarms: *autoscaling_alarms

    # lifecycle callbacks
    lifecycle_callbacks:
      pre_terminate_past_clusters:
        - service hello stop

    # list of region
    # deployer will concurrently deploy across the region
    regions:
      - region: ap-northeast-2

        # instance type
        instance_type: m5.large

        # ssh_key for instances
        ssh_key: dayone-prod-master

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
        vpc: vpc-dayonep_apnortheast2

        # You can use security group id(sg-xxx)
        # If you specify the name of security group, then deployer will find the security group id with it.
        # In this case, only one security group should exist
        security_groups:
          - hello-dayonep_apnortheast2
          - default-dayonep_apnortheast2

        # You can use healthcheck target group
        healthcheck_target_group: hello-dayonepapne2-ext

        # If no availability zones specified, then all availability zones are selected by default.
        # If you want all availability zones, then please remove availability_zones key.
        availability_zones:
          - ap-northeast-2a
          - ap-northeast-2b
          - ap-northeast-2c

        # list of target groups.
        # The target group in the healthcheck_target_group should be included here.
        target_groups:
          - hello-dayonepapne2-ext


      - region: us-east-1
        ami_id: ami-09d95fab7fff3776c
        instance_type: t3.large
        ssh_key: dayone-prod-master
        use_public_subnets: true
        vpc: vpc-dayonep_useast1
        security_groups:
          - hello-dayonep_useast1
          - default-dayonep_useast1
        healthcheck_target_group: hello-dayonepuse1-ext
        target_groups:
          - hello-dayonepuse1-ext
``` 


