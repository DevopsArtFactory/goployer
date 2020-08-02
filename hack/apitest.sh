#!/bin/bash
set -e

BUILD_DIR="build"
TEST_DIR="test"
URL="https://hello.devops-art-factory.com"

stacks=("artd" "artd-spot" "artd-mixed" "artd-without-targetgroup")

if [[ ! -d $BUILD_DIR ]]; then
    echo "creating new directory [ $BUILD_DIR ]"
    mkdir -p $BUILD_DIR
fi

# process local test
make test

# build goployer
GOOS=darwin CGO_ENABLED=1 go build -o ./$BUILD_DIR/goployer cmd/goployer/main.go
if [[ $? -ne 0 ]];then
    echo "error occurred when building binary file"
    exit 1
fi

# api test
lastStack=""
for stack in "${stacks[@]}"; do
    ./$BUILD_DIR/goployer deploy --manifest=$TEST_DIR/test_manifest.yaml --stack=$stack --slack-off=true --log-level=debug --region=ap-northeast-2 --polling-interval=20s
    if [[ $? -eq 0 ]]; then
        echo "$stack is deployed"
        for ((i=1;i<=10;i++)); do
            curl -s $URL > /dev/null
            if [[ $? -ne 0 ]]; then
                echo "error occurred"
                exit 1
            fi
            echo "done $stack $i"
            sleep 1
        done
        echo "healthcheck is done"
    fi
    lastStack=$stack
    sleep 30
done
echo "API test is done"

echo  "delete test autoscaling group"
./$BUILD_DIR/goployer delete --manifest=$TEST_DIR/test_manifest.yaml --stack=$lastStack --slack-off=true --log-level=debug --region=ap-northeast-2 --polling-interval=20s

rm -rf $BUILD_DIR

