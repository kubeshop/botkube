#!/bin/bash
# Copyright (c) 2019 InfraCloud Technologies
#
# Permission is hereby granted, free of charge, to any person obtaining a copy of
# this software and associated documentation files (the "Software"), to deal in
# the Software without restriction, including without limitation the rights to
# use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
# the Software, and to permit persons to whom the Software is furnished to do so,
# subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
# FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
# COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
# IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
# CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

set -e

install_kind() {
    echo "Installing KIND cluster"
    GO111MODULE="on" go get sigs.k8s.io/kind@v0.9.0
}

create_kind_cluster() {
  echo "creating KIND cluster"
  kind create cluster
}

destroy_kind_cluster() {
  echo "destroying KIND cluster"
  kind delete clusters --all
}

help() {
  usage="$(basename "$0") [option] -- Script to create or destroy KIND cluster.
  Available option are install-kind, destroy-kind or create-kind"
  echo $usage
}


if [ $# -gt 1 ]; then help ;fi
case "${1}" in
        install-kind)
            install_kind
        ;;
        create-kind)
            create_kind_cluster
            ;;
        destroy-kind)
            destroy_kind_cluster
        ;;
        *)
            help
            exit 1
esac
