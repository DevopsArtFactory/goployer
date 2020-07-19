# Copyright 2020 The Goployer Authors All rights reserved.

# Download goployer
FROM alpine:3.10 as download-goployer
ENV KUBECTL_VERSION v1.0.0
ENV KUBECTL_URL https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
RUN wget -O kubectl "${KUBECTL_URL}"
RUN chmod +x kubectl


FROM amazonlinux:latest as runtime_deps

RUN yum update -y && \
    yum install -y \
    unzip \
    git \
    wget \
    openssl \
    java-1.8.0-openjdk-devel.x86_64

COPY --from=docker:18.09.6 /usr/local/bin/docker /usr/local/bin/
COPY --from=download-goployerr goloyer /usr/local/bin/

FROM runtime_deps
COPY --from=golang:1.14 /usr/local/go /usr/local/go
ENV PATH /usr/local/go/bin:/root/go/bin:$PATH
