# Copyright 2020 The Goployer Authors All rights reserved.
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

# Download goployer
FROM alpine:3.10 as download-goployer
ARG GOPLOYER_VERSION=1.0.0
ARG GOPLOYER_URL=https://github.com/DevopsArtFactory/goployer/releases/download/${GOPLOYER_VERSION}/goployer-linux-amd64
RUN wget -O goployer "${GOPLOYER_URL}"
RUN chmod +x goployer


FROM amazonlinux:latest as runtime_image

RUN yum update -y && \
    yum install -y \
    unzip \
    git \
    wget \
    openssl \
    java-1.8.0-openjdk-devel.x86_64

COPY --from=docker:18.09.6 /usr/local/bin/docker /usr/local/bin/
COPY --from=download-goployer goployer /usr/local/bin/

FROM runtime_image
COPY --from=golang:1.14 /usr/local/go /usr/local/go
ENV PATH /usr/local/go/bin:/root/go/bin:$PATH
