#!/bin/bash
RED='\033[0;31m'
GREEN='\033[0;32m'
RESET='\033[0m'

echo "Running linters..."
scripts=(
    "hack/boilerplate.sh"
    "hack/gofmt.sh"
    "hack/pedantic-imports.sh"
    "hack/golangci-lint.sh"
)
fail=0
for s in "${scripts[@]}"; do
    echo "RUN ${s}"

    start=$(date +%s)
    ./$s
    result=$?
    end=$(date +%s)

    if [[ $result -eq 0 ]]; then
        echo -e "${GREEN}PASSED${RESET} ${s} in $((end-start))s"
    else
        echo -e "${RED}FAILED${RESET} ${s}"
        fail=1
    fi
done
exit $fail
