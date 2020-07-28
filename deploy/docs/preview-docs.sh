#!/usr/bin/env bash
set -e

readonly CURRENT_DIR=$(pwd)
readonly DOCS_DIR="${CURRENT_DIR}/docs"

MOUNTS="-v ${CURRENT_DIR}/.git:/app/.git:ro"
MOUNTS="${MOUNTS} -v ${DOCS_DIR}/config.toml:/app/docs/config.toml:ro"

for dir in $(find ${DOCS_DIR} -mindepth 1 -maxdepth 1 -type d | grep -v themes | grep -v public | grep -v resources | grep -v node_modules); do
    MOUNTS="${MOUNTS} -v $dir:/app/docs/$(basename $dir):ro"
done

docker build -t goployer-docs-previewer deploy/docs
docker run --rm -ti -p 1313:1313 ${MOUNTS} goployer-docs-previewer $@
