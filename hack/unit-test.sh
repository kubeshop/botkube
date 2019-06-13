#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

PACKAGES=$(go list ./... | grep -v '/vendor/')

for package in $PACKAGES; do
    go test ${@} "$package"
done
