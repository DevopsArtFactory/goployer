#!/bin/bash
set -e -o pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BIN=${DIR}/bin
VERSION=1.27.0

function install_linter() {
  echo "Installing GolangCI-Lint"
  ${DIR}/install_golint.sh -b ${BIN} v$VERSION
}

if ! [ -x "$(command -v ${BIN}/golangci-lint)" ] ; then
  install_linter
elif [[ $(${BIN}/golangci-lint --version | grep -c " $VERSION ") -eq 0 ]]
then
  echo "required golangci-lint: v$VERSION"
  echo "current version: $(golangci-lint --version)"
  echo "reinstalling..."
  rm $(which ${BIN}/golangci-lint)
  install_linter
fi

FLAGS=""
if [[ "${CI}" == "true" ]]; then
    FLAGS="-v --print-resources-usage"
fi

${BIN}/golangci-lint run ${FLAGS} -c ${DIR}/golangci.yml \
    | awk '/out of memory/ || /Timeout exceeded/ {failed = 1}; {print}; END {exit failed}'
