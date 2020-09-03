#!/bin/bash
set -e

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
BOILERPLATEDIR=$DIR/boilerplate

files=$(python ${BOILERPLATEDIR}/boilerplate.py --rootdir . --boilerplate-dir ${BOILERPLATEDIR})

if [[ ! -z ${files} ]]; then
	echo "Boilerplate missing in:"
    echo "${files}"
	exit 1
fi
