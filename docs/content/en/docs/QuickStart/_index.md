---
title: "Quickstart"
linkTitle: "Quickstart"
weight: 30
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
* Make manifest file for deployment
* Run goployer to deploy the application

## Before you begin

* [Install Goployer]({{< relref "/docs/install" >}})
* [Setup base infrastructure]({{< relref "/docs/baseinfrastructure" >}})

{{< alert title="Note">}}
Goployer will create AWS resources like Launch Template, Autoscaling group, DynamoDB, CloudWatch Alarm etc. 
So you need to have the right permissions to make these resources in your AWS credentials.
{{</alert>}}

## Preparing manifest and metric file

1. make new directory:
    ```bash
    $ mkdir goployer
    $ cd goployer
    ```

1. Create sample files with `goployer init`

    ```bash
    $ goployer init                                                                                                                                                                                                        
    What is application name: hello
    ```
   
1. Modify configurations in the file `manifests/hello.yaml` to your application.

    ```bash
    $ vim manifests/hello.yaml
    ```
   
1. Modify configurations in the file `metrics.yaml` .
* metrics.yaml file should be in the root directory where you run goployer command

    ```bash
    $ vim metrics.yaml
    ```
   
1. Run deploy command
* If you don't want to use metrics feature(Step 4), then use `--disable-metrics=true` 

    ```bash
   $ goployer deploy --manifest=manifests/hello.yaml --stack=<stack name> --region=ap-northeast-2 --slack-off=true --log-level=debug
    ```

## Logging   

```
$ goployer deploy --manifest=configs/hello.yaml --manifest-s3-region=ap-northeast-2 --stack=artd --region=ap-northeast-2 --slack-off=true --log-level=debug

INFO[0000] Beginning deployment: hello                  

============================================================
Target Stack Deployment Information
============================================================
name             : hello
env              : dev
timeout          : 60 min
polling-interval : 60 sec 
assume role      : 
extra tags       : 
============================================================
Stack
============================================================
[ artd ]
Account                 : dev
Environment             : dev
IAM Instance Profile    : app-hello-profile
Ansible tags            : all 
Capacity                : {Min:1 Max:2 Desired:1}
MixedInstancesPolicy
- Enabled                       : false
- Override                      : [c5.large c5.xlarge]
- OnDemandPercentage            : 20
- SpotAllocationStrategy        : lowest-price
- SpotInstancePools             : 3
- SpotMaxPrice                  : 0.3
        
============================================================
WARN[0000] no slack variables exists. [ SLACK_TOKEN, SLACK_CHANNEL ] 
DEBU[0000] create deployers for stacks                  
INFO[0000] Deploy Mode is BlueGreen                     
INFO[0000] Previous Versions : hello-dev_apnortheast2-v007 
INFO[0000] Current Version :8                           
INFO[0000] Successfully create new launch template : hello-dev_apnortheast2-v008-1595954238 
INFO[0001] Applied instance capacity - Min: 1, Desired: 1, Max: 2 
INFO[0002] Successfully create new autoscaling group : hello-dev_apnortheast2-v008 
DEBU[0002] Start Timestamp: 1595954238, timeout: 1h0m0s 
DEBU[0002] Healthchecking for region starts : ap-northeast-2 
INFO[0002] Healthy count does not meet the requirement(hello-dev_apnortheast2-v008) : 0/1 
INFO[0002] All stacks are not healthy... Please waiting to be deployed... 

(...skip...)

INFO[0124] {InstanceId:i-0b9187b2c4bc68a16 LifecycleState:InService TargetStatus:healthy HealthStatus:Healthy Healthy:true} 
INFO[0124] Healthy Count for hello-dev_apnortheast2-v009 : 1/1 
INFO[0124] All stacks are healthy                       
INFO[0124] Attaching autoscaling policies : ap-northeast-2 
INFO[0124] Metrics monitoring of autoscaling group is enabled : hello-dev_apnortheast2-v009 
INFO[0124] New metric alarm is created : scale_up_on_util / asg : hello-dev_apnortheast2-v009 
INFO[0124] New metric alarm is created : scale_down_on_util / asg : hello-dev_apnortheast2-v009 
DEBU[0124] run lifecycle callbacks before termination : [i-099f40b492a6a2da5] 
DEBU[0125] Delete Mode is BlueGreen                     
INFO[0125] [ap-northeast-2]The number of previous versions to delete is 2 
DEBU[0125] [Resizing to 0] target autoscaling group : hello-dev_apnortheast2-v007 
INFO[0125] Modifying the size of autoscaling group to 0 : hello-dev_apnortheast2-v007(artd) 
DEBU[0125] [Resizing to 0] target autoscaling group : hello-dev_apnortheast2-v008 
INFO[0125] Modifying the size of autoscaling group to 0 : hello-dev_apnortheast2-v008(artd) 
INFO[0125] Termination Checking for artd starts...      
INFO[0125] Checking Termination stack for region starts : ap-northeast-2 
INFO[0125] Waiting for instance termination in asg hello-dev_apnortheast2-v007 
DEBU[0125] Start deleting autoscaling group : hello-dev_apnortheast2-v007 
DEBU[0125] Autoscaling group is deleted : hello-dev_apnortheast2-v007 
DEBU[0125] Start deleting launch templates in hello-dev_apnortheast2-v007 
DEBU[0126] Launch templates are deleted in hello-dev_apnortheast2-v007 
INFO[0126] finished : hello-dev_apnortheast2-v007       
INFO[0126] Waiting for instance termination in asg hello-dev_apnortheast2-v008 
INFO[0126] 1 instance found : hello-dev_apnortheast2-v008 
INFO[0126] All stacks are not ready to be terminated... Please waiting... 

(...skip...)

INFO[0307] Waiting for instance termination in asg hello-dev_apnortheast2-v008 
DEBU[0307] Start deleting autoscaling group : hello-dev_apnortheast2-v008 
DEBU[0307] Autoscaling group is deleted : hello-dev_apnortheast2-v008 
DEBU[0307] Start deleting launch templates in hello-dev_apnortheast2-v008 
DEBU[0307] Launch templates are deleted in hello-dev_apnortheast2-v008 
INFO[0307] finished : hello-dev_apnortheast2-v008       
INFO[0307] All stacks are terminated!!        
```

## More examples
* [Examples]({{< relref "/docs/examples" >}})
