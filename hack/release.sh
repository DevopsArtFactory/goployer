#!/usr/bin/env bash
set -euo pipefail

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd)

if [[ ${#} -eq 0 ]]; then
    echo "No argument"
fi

git tag $1
git commit -m "Release the new version $1"
git push --tags
