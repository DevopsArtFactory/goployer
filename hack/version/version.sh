#!/bin/bash

version=$(cat version.txt)

echo "Current version is $version"

sed -i -e "s/LATEST_VERSION/v$version/g" pkg/version/version.go
