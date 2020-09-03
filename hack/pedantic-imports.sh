#!/usr/bin/env bash
EXIT_CODE=0

for gofile in $(find . -name *.go | grep -v '/vendor/'); do
    awk '{
        if ($0 == "import (") {inImport=1}
        if (inImport && $0 == "") {blankLines++}
        if ($0 == ")") {inImport=0; exit}
    } END {
        if (blankLines > 2) {exit 1}
    }' "${gofile}"
    if [[ $? -ne 0 ]]; then
        echo "${gofile} contains more than 3 groups of imports"
        EXIT_CODE=1
    fi
    
    awk '{
        if ($0 == "import (") {inImport=1}
        if (inImport && $0 == "") {blankLines++}
        if (inImport && $0 != ")") {last=$0}
        if ($0 == ")") {inImport=0; exit}
    } END {
        if (blankLines == 2 && index(last, "github.com/DevopsArtFactory") == 0) {exit 1}
    }' "${gofile}"
    if [[ $? -ne 0 ]]; then
        echo "${gofile} should have DevopsArtFactory imports last"
        EXIT_CODE=1
    fi
done

exit ${EXIT_CODE}
