#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

export CONFIG_PATH=`pwd`

# Run unit and integration tests excluding dependencies
PACKAGES=$(go list ./... | grep -v '/vendor/')
for package in $PACKAGES; do
    go test -tags=test ${@} "$package" -v
done
