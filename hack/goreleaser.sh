#!/bin/bash

set -o errexit
set -o pipefail

IMAGE_REGISTRY="${IMAGE_REGISTRY:-ghcr.io}"
IMAGE_REPOSITORY="${IMAGE_REPOSITORY:-kubeshop/botkube}"
CFG_EXPORTER_IMAGE_REPOSITORY="${CFG_EXPORTER_IMAGE_REPOSITORY:-kubeshop/botkube-config-exporter}"
IMAGE_SAVE_LOAD_DIR="${IMAGE_SAVE_LOAD_DIR:-/tmp/botkube-images}"
IMAGE_PLATFORM="${IMAGE_PLATFORM:-linux/amd64}"
GORELEASER_CURRENT_TAG="${GORELEASER_CURRENT_TAG:-v9.99.9-dev}"

prepare() {
  export DOCKER_CLI_EXPERIMENTAL="enabled"
  docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
}

release_snapshot() {
  prepare
  goreleaser release --clean --snapshot --skip-publish

  # Push images
  docker push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64
  docker push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64
  docker push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7

  docker push ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64
  docker push ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64
  docker push ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7

  # Create manifest
  docker manifest create ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG} \
    --amend ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64 \
    --amend ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64 \
    --amend ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7
  docker manifest push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}

  # Create Config Exporter manifest
   docker manifest create ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG} \
      --amend ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64 \
      --amend ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64 \
      --amend ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7
    docker manifest push ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}
}

build_single_arch_cli() {
  export GORELEASER_CURRENT_TAG=v9.99.9-dev
  goreleaser build --clean --snapshot --id botkube-cli --single-target
}

save_images() {
  prepare

  if [ -z "${IMAGE_TAG}" ]; then
    echo "Missing IMAGE_TAG."
    exit 1
  fi

  export GORELEASER_CURRENT_TAG=${IMAGE_TAG}

  GORELEASER_FILE="$(prepare_goreleaser)"
  goreleaser release --clean --snapshot --skip-publish --config="${GORELEASER_FILE}"

  mkdir -p "${IMAGE_SAVE_LOAD_DIR}"

  # Save images
  if [[ -z "$BUILD_TARGETS" || ",$BUILD_TARGETS," == *",botkube-agent,"* ]]; then
      IMAGE_FILE_NAME_PREFIX=$(echo "${IMAGE_REPOSITORY}" | tr "/" "-")
      docker save ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64 >${IMAGE_SAVE_LOAD_DIR}/${IMAGE_FILE_NAME_PREFIX}-amd64.tar
      docker save ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64 >${IMAGE_SAVE_LOAD_DIR}/${IMAGE_FILE_NAME_PREFIX}-arm64.tar
      docker save ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7 >${IMAGE_SAVE_LOAD_DIR}/${IMAGE_FILE_NAME_PREFIX}-armv7.tar
  fi

  if [[ -z "$BUILD_TARGETS" || ",$BUILD_TARGETS," == *",botkube-config-exporter,"* ]]; then
      CFG_EXPORTER_IMAGE_FILE_NAME_PREFIX=$(echo "${CFG_EXPORTER_IMAGE_REPOSITORY}" | tr "/" "-")
      docker save ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64 >${IMAGE_SAVE_LOAD_DIR}/${CFG_EXPORTER_IMAGE_FILE_NAME_PREFIX}-amd64.tar
      docker save ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64 >${IMAGE_SAVE_LOAD_DIR}/${CFG_EXPORTER_IMAGE_FILE_NAME_PREFIX}-arm64.tar
      docker save ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7 >${IMAGE_SAVE_LOAD_DIR}/${CFG_EXPORTER_IMAGE_FILE_NAME_PREFIX}-armv7.tar
  fi

}

load_and_push_images() {
  prepare
  if [ -z "${IMAGE_TAG}" ]; then
    echo "Missing IMAGE_TAG."
    exit 1
  fi

  export GORELEASER_CURRENT_TAG=${IMAGE_TAG}

  # Load images
  if [[ -z "$BUILD_TARGETS" || ",$BUILD_TARGETS," == *",botkube-agent,"* ]]; then
      IMAGE_FILE_NAME_PREFIX=$(echo "${IMAGE_REPOSITORY}" | tr "/" "-")
      docker load --input ${IMAGE_SAVE_LOAD_DIR}/${IMAGE_FILE_NAME_PREFIX}-amd64.tar
      docker load --input ${IMAGE_SAVE_LOAD_DIR}/${IMAGE_FILE_NAME_PREFIX}-arm64.tar
      docker load --input ${IMAGE_SAVE_LOAD_DIR}/${IMAGE_FILE_NAME_PREFIX}-armv7.tar
  fi

  if [[ -z "$BUILD_TARGETS" || ",$BUILD_TARGETS," == *",botkube-config-exporter,"* ]]; then
      CFG_EXPORTER_IMAGE_FILE_NAME_PREFIX=$(echo "${CFG_EXPORTER_IMAGE_REPOSITORY}" | tr "/" "-")
      docker load --input ${IMAGE_SAVE_LOAD_DIR}/${CFG_EXPORTER_IMAGE_FILE_NAME_PREFIX}-amd64.tar
      docker load --input ${IMAGE_SAVE_LOAD_DIR}/${CFG_EXPORTER_IMAGE_FILE_NAME_PREFIX}-arm64.tar
      docker load --input ${IMAGE_SAVE_LOAD_DIR}/${CFG_EXPORTER_IMAGE_FILE_NAME_PREFIX}-armv7.tar
  fi

  # Push images
  if [[ -z "$BUILD_TARGETS" || ",$BUILD_TARGETS," == *",botkube-agent,"* ]]; then
      docker push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64
      docker push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64
      docker push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7
  fi

  if [[ -z "$BUILD_TARGETS" || ",$BUILD_TARGETS," == *",botkube-config-exporter,"* ]]; then
      docker push ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64
      docker push ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64
      docker push ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7
  fi

  # Create manifest
  if [[ -z "$BUILD_TARGETS" || ",$BUILD_TARGETS," == *",botkube-agent,"* ]]; then
      docker manifest create ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG} \
        --amend ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64 \
        --amend ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64 \
        --amend ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7
      docker manifest push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}
  fi

  if [[ -z "$BUILD_TARGETS" || ",$BUILD_TARGETS," == *",botkube-config-exporter,"* ]]; then
      docker manifest create ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG} \
        --amend ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64 \
        --amend ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64 \
        --amend ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7
      docker manifest push ${IMAGE_REGISTRY}/${CFG_EXPORTER_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}
  fi
}

build() {
  prepare
  docker run --rm --privileged \
    -v $PWD:/go/src/github.com/kubeshop/botkube \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -w /go/src/github.com/kubeshop/botkube \
    -e GORELEASER_CURRENT_TAG=v9.99.9-dev \
    -e ANALYTICS_API_KEY="${ANALYTICS_API_KEY}" \
    -e CLI_ANALYTICS_API_KEY="${CLI_ANALYTICS_API_KEY}" \
    goreleaser/goreleaser release --clean --snapshot --skip-publish
}

build_plugins_command() {
  local command="goreleaser build -f .goreleaser.plugin.yaml --clean --snapshot"

  local targets=()
  if [ -n "$PLUGIN_TARGETS" ]; then
    IFS=',' read -ra targets <<<"$PLUGIN_TARGETS"
  fi

  for target in "${targets[@]}"; do
    command+=" --id $target"
  done

  echo "$command"
}

build_plugins() {
  eval "$(build_plugins_command)"
}

build_plugins_single() {
  command+="$(build_plugins_command) --single-target"
  eval "$command"
}

prepare_goreleaser() {
  if [ -z "${BUILD_TARGETS}" ]; then
    echo ".goreleaser.yml"
    exit 0
  fi

  cp .goreleaser.yml .goreleaser_temp.yaml

  # Filter the builds section
  for build_id in $(yq e '.builds[].id' .goreleaser_temp.yaml); do
      if [[ ! ",$BUILD_TARGETS," == *",$build_id,"* ]]; then
          yq e "del(.builds[] | select(.id == \"$build_id\"))" -i .goreleaser_temp.yaml
      fi
  done

  # Filter the dockers section
  for docker_id in $(yq e '.dockers[].id' .goreleaser_temp.yaml); do
      build_name=$(echo "$docker_id" | rev | cut -d'-' -f2- | rev)
      if [[ ! ",$BUILD_TARGETS," == *",$build_name,"* ]]; then
          yq e "del(.dockers[] | select(.id == \"$docker_id\"))" -i .goreleaser_temp.yaml
      fi
  done

  # Filter the archives section
  for archive_id in $(yq e '.archives[].id' .goreleaser_temp.yaml); do
      if [[ ! ",$BUILD_TARGETS," == *",$archive_id,"* ]]; then
          yq e "del(.archives[] | select(.id == \"$archive_id\"))" -i .goreleaser_temp.yaml
      fi
  done

  # Filter the brews section
  DEFAULT_BREW_NAME="botkube"
  BOTKUBE_CLI_ID="botkube-cli"
  if [[ ! ",$BUILD_TARGETS," == *",$BOTKUBE_CLI_ID,"* ]]; then
      yq e "del(.brews[] | select(.name == \"$DEFAULT_BREW_NAME\"))" -i .goreleaser_temp.yaml
  fi

  if [[ "${SINGLE_PLATFORM}" == "true" ]]; then
    CURRENT_OS=$(go env GOOS)
    CURRENT_ARCH=$(go env GOARCH)

    # Remove the goarm from the YAML file if it's not Darwin
    if [ "$CURRENT_OS" != "darwin" ]; then
      yq eval 'del(.builds[].goos[] | select(. == "darwin"))' .goreleaser_temp.yaml -i
      yq eval 'del(.builds[].goarm)' .goreleaser_temp.yaml -i
    fi

    if [ -n "$CURRENT_OS" ]; then
      yq eval "del(.builds[].goos[] | select(. == \"$CURRENT_OS\"))" .goreleaser_temp.yaml -i
    fi

    if [ -n "$CURRENT_ARCH" ]; then
      yq eval "del(.builds[].goarch[] | select(. == \"$CURRENT_ARCH\"))" .goreleaser_temp.yaml -i
    fi
  fi

  echo ".goreleaser_temp.yaml"
}

build_single() {
  export IMAGE_TAG=v9.99.9-dev
  docker run --rm --privileged \
    -v "$PWD":/go/src/github.com/kubeshop/botkube \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -w /go/src/github.com/kubeshop/botkube \
    -e IMAGE_TAG=${IMAGE_TAG} \
    -e ANALYTICS_API_KEY="${ANALYTICS_API_KEY}" \
    -e CLI_ANALYTICS_API_KEY="${CLI_ANALYTICS_API_KEY}" \
    goreleaser/goreleaser build --single-target --clean --snapshot --id botkube-agent -o "./botkube-agent"
  docker build -f "$PWD/build/Dockerfile" --platform "${IMAGE_PLATFORM}" -t "${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${IMAGE_TAG}" .
  rm "$PWD/botkube-agent"
}

usage() {
  cat <<EOM
Usage: ${0} [build|release|release_snapshot|build_single_arch_cli]
Where,
  build: Builds project with goreleaser without pushing images.
  release_snapshot: Builds project without publishing release. It builds and pushes Botkube image with v9.99.9-dev image tag.
EOM
  exit 1
}

[ ${#@} -gt 0 ] || usage
case "${1}" in
build)
  build
  ;;
build_plugins)
  build_plugins
  ;;
build_plugins_single)
  build_plugins_single
  ;;
build_single)
  build_single
  ;;
release_snapshot)
  release_snapshot
  ;;
build_single_arch_cli)
  build_single_arch_cli
  ;;
save_images)
  save_images
  ;;
save_pr_image)
  save_pr_image
  ;;
load_and_push_images)
  load_and_push_images
  ;;
esac
