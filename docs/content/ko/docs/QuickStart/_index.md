---
title: "실습하기"
linkTitle: "실습하기"
weight: 3
description: >
  본 페이지를 통해 손쉽게 실습을 진행하실 수 있습니다.
---
본 가이드를 따라하시면서 어떻게 Goployer를 활용하여 AWS 환경에 어플리케이션을 배포할 수 있는지 익히실 수 있습니다.

{{< alert title="필독 사항">}}
Goployer는 어플리케이션 배포 툴로 `AWS의 다양한 리소스를 생성`합니다. 따라서 **AWS 비용이 발생할 수 있습니다.**
{{</alert>}}

본 가이드에서 여러분은 아래의 과정을 실습할 예정입니다.

* Goployer를 설치합니다.
* 배포를 위한 매니페스트 파일을 생성합니다.
* Goployer를 통해 어플리케이션을 배포합니다.

## 시작하기 전 준비사항

* [Goployer 설치하기]({{< relref "/docs/install" >}})
* [기본 인프라 세팅]({{< relref "/docs/baseinfrastructure" >}})

{{< alert title="주의">}}
Goployer는 AWS Launch Template, Autoscaling Group, DynamoDB, CloudWatch Alarm 등 다양한 AWS 리소스를 생성합니다. 
<br>따라서 이에 필요한 **IAM 권한을 가지고 있어야 합니다.**
{{</alert>}}

## 매니페스트 및 매트릭 관련 파일 작업

1. goployer를 위한 샘플 폴더를 생성합니다.
    ```bash
    $ mkdir goployer
    $ cd goployer
    ```

1. `goployer init` 명령어를 사용해 샘플 파일들을 생성합니다.

    ```bash
    $ goployer init                                                                                                                                                                                                        
    What is application name: hello
    ```
   
1. `manifests/hello.yaml` 파일을 열어 설정을 변경합니다.

    ```bash
    $ vim manifests/hello.yaml
    ```
   
1. `metrics.yaml` 파일을 수정합니다.
* metrics.yaml 파일은 반드시 goployer가 실행되는 루트 디렉토리에 있어야 합니다.
    ```bash
    $ vim metrics.yaml
    ```
   
1. goployer를 실행합니다.
* 4번에서 세팅한 Metric 기능을 사용하고 싶지 않으신 경우에는, 명령어 실행 시 `--disable-metrics=true` 을 사용하시기 바랍니다.
    ```bash
   goployer deploy --manifest=manifests/hello.yaml --stack=<stack name> --region=ap-northeast-2 --slack-off=true --log-level=debug --disable-metrics=true
    ```

## 로그 확인   

```
$ goployer deploy --manifest=configs/hello.yaml --manifest-s3-region=ap-northeast-2 --stack=artd --region=ap-northeast-2 --slack-off=true --log-level=debug --disable-metrics=true
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


## 예시 더보기
* [Examples]({{< relref "/docs/examples" >}})

