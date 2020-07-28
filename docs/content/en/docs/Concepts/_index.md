---
title: "Concepts"
linkTitle: "Concepts"
weight: 4
description: >
    Understanding concept of goployer is so important
---


## How Goployer works
* Here's the steps that goployer executes for deployment
1. Generate new version for current deployment.<br>
If other autoscaling groups of sample application already existed, for example `hello-v001`, then next version will be `hello-v002`
2. Create a new launch template. 
3. Create autoscaling group with launch template from the previous step. A newly created autoscaling group will be automatically attached to the target groups you specified in manifest.
4. Check all instances of all stacks are healty. Until all of them pass healthchecking, it won't go to the next step.
5. (optional) If you add `autoscaling` in manifest, goployer creates autoscaling policies and put these to the autoscaling group. If you use `alarms` with autoscaling, then goployer will also create a cloudwatch alarm for autoscaling policy.
6. After all stacks are deployed, then goployer tries to delete previous versions of the same application.
   Launch templates of previous autoscaling groups are also going to be deleted.


