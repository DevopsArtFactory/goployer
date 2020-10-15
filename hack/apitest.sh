# Copyright 2020 The Goployer Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/bin/bash
set -e

BUILD_DIR="build"
TEST_DIR="test"
URL="https://hello.devops-art-factory.com"

if [[ ! -d $BUILD_DIR ]]; then
    echo "creating new directory [ $BUILD_DIR ]"
    mkdir -p $BUILD_DIR
fi

# clean first
make clean

# process local test
make test

# build goployer
GOOS=darwin CGO_ENABLED=1 go build -o ./$BUILD_DIR/goployer cmd/goployer/main.go
if [[ $? -ne 0 ]];then
    echo "error occurred when building binary file"
    exit 1
fi

# Multistack
echo "test with multistack"
./$BUILD_DIR/goployer deploy --auto-apply --manifest=$TEST_DIR/test_manifest.yaml --slack-off=true --log-level=debug --region=ap-northeast-2 --polling-interval=20s

echo "API deployment test is done"

echo  "delete test autoscaling group"
./$BUILD_DIR/goployer delete --auto-apply --manifest=$TEST_DIR/test_manifest.yaml --slack-off=true --log-level=debug --region=ap-northeast-2 --polling-interval=20s

rm -rf $BUILD_DIR

