
---
title: "Goployer Documentation"
linkTitle: "Documentation"
weight: 20
layout: single
menu:
  main:
    weight: 20
---

Goployer is open-source deployment tool with AWS Autoscaling group. Goployer handles the whole cycle of deployment, and keep track of deployment
histories. This enables you to focus on server settings with the userdata and goployer will deploy your application safely with manifest configuration
 you made.
 
## Features

* Safe Deployment with versioning
  * **Blue/Green Deployment** - Goployer uses **blue/green** deployment by default in order to ensure the safe deployment. Goployer will create new autoscaling group and attach it to the target group.
  After checking all the instances of autoscaling group are healthy, then delete the previous autoscaling group.
  * **Versioning** - In the autoscaling group name, you can find the current version easily.
* Use most of autoscaling group feature
  * **Autoscaling Policy** - You can create autoscaling policy with AWS CloudWatch.
  * **Spot Instance** - You can make launch template with spot configuration.
  * **Mixed Instance Policy** - You can use on-demand and spot instance together with MixedInstancePolicy supported by autoscaling group. Autoscaling group will control spot request on behalf of you.
* Metric Enabled
  * **History Management** - You can make history records to AWS DynamoDB and keep track of deployment duration.

## Demo

![base](/images/base.gif)

## How goployer works

* Here are steps that goployer executes for a deployment.
1. Generate new version for current deployment.<br>
If other autoscaling groups of sample application already existed, for example `hello-v001`, then next version will be `hello-v002`
2. Create a new launch template. 
3. Create autoscaling group with launch template from the previous step. A newly created autoscaling group will be automatically attached to the target groups you specified in manifest.
4. Check all instances of the autoscaling group are healty. Until all of them pass healthchecking, it won't go to the next step.
5. (optional) If you add `autoscaling` in manifest, goployer creates autoscaling policies and put these to the autoscaling group. If you use `alarms` with autoscaling, then goployer will also create a cloudwatch alarm for autoscaling policy.
6. After step 5 is done, then goployer tries to delete previous versions of the same application.
   Launch templates of previous autoscaling groups are also going to be deleted.
7. If metric feature is enabled, then goployer create a history record of previous autoscaling group.

