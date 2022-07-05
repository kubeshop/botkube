#!/bin/bash

set -e

install_kind() {
  echo "Installing KIND cluster"
  go install sigs.k8s.io/kind@v0.13.0
}

create_kind_cluster() {
  install_kind
  echo "creating KIND cluster"
  kind create cluster --name kind-cicd
}

destroy_kind_cluster() {
  echo "destroying KIND cluster"
  kind delete cluster --name kind-cicd
}

help() {
  usage="$(basename "$0") [option] -- Script to create or destroy KIND cluster.
  Available options are destroy-kind, create-kind or help"
  echo $usage
}


if [ $# -gt 1 ]; then help ;fi
case "${1}" in
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

