#!/bin/bash

set -o errexit
set -o pipefail

IMAGE_REGISTRY="${IMAGE_REGISTRY:-ghcr.io}"
IMAGE_REPOSITORY="${IMAGE_REPOSITORY:-mszostok/botkube}"
TEST_IMAGE_REPOSITORY="${TEST_IMAGE_REPOSITORY:-mszostok/botkube-test}"
IMAGE_SAVE_LOAD_DIR="${IMAGE_SAVE_LOAD_DIR:-/tmp/botkube-images}"
IMAGE_PLATFORM="${IMAGE_PLATFORM:-linux/amd64}"

prepare() {
  export DOCKER_CLI_EXPERIMENTAL="enabled"
  docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
}

release_snapshot() {
  prepare
  export GORELEASER_CURRENT_TAG=v9.99.9-dev
  goreleaser release --rm-dist --snapshot --skip-publish
  # Push images
  docker push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64
  docker push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64
  docker push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7
  docker push ${IMAGE_REGISTRY}/${TEST_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}
  # Create manifest
  docker manifest create ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG} \
    --amend ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64 \
    --amend ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64 \
    --amend ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7
  docker manifest push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}
}

save_images() {
  prepare

  if [ -z "${IMAGE_TAG}" ]
  then
    echo "Missing IMAGE_TAG."
    exit 1
  fi

  export GORELEASER_CURRENT_TAG=${IMAGE_TAG}
  goreleaser release --rm-dist --snapshot --skip-publish

  mkdir -p "${IMAGE_SAVE_LOAD_DIR}"

  # Save images
  IMAGE_FILE_NAME_PREFIX=$(echo "${IMAGE_REPOSITORY}" | tr "/" "-")
  docker save ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64 > ${IMAGE_SAVE_LOAD_DIR}/${IMAGE_FILE_NAME_PREFIX}-amd64.tar
  docker save ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64 > ${IMAGE_SAVE_LOAD_DIR}/${IMAGE_FILE_NAME_PREFIX}-arm64.tar
  docker save ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7 > ${IMAGE_SAVE_LOAD_DIR}/${IMAGE_FILE_NAME_PREFIX}-armv7.tar

  TEST_FILE_NAME=$(echo "${TEST_IMAGE_REPOSITORY}" | tr "/" "-")
  docker save ${IMAGE_REGISTRY}/${TEST_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG} > ${IMAGE_SAVE_LOAD_DIR}/${TEST_FILE_NAME}.tar
}

load_and_push_images() {
  prepare
  if [ -z "${IMAGE_TAG}" ]
  then
    echo "Missing IMAGE_TAG."
    exit 1
  fi

  export GORELEASER_CURRENT_TAG=${IMAGE_TAG}

  # Load images
  IMAGE_FILE_NAME_PREFIX=$(echo "${IMAGE_REPOSITORY}" | tr "/" "-")
  docker load --input ${IMAGE_SAVE_LOAD_DIR}/${IMAGE_FILE_NAME_PREFIX}-amd64.tar
  docker load --input ${IMAGE_SAVE_LOAD_DIR}/${IMAGE_FILE_NAME_PREFIX}-arm64.tar
  docker load --input ${IMAGE_SAVE_LOAD_DIR}/${IMAGE_FILE_NAME_PREFIX}-armv7.tar

  TEST_FILE_NAME=$(echo "${TEST_IMAGE_REPOSITORY}" | tr "/" "-")
  docker load --input ${IMAGE_SAVE_LOAD_DIR}/${TEST_FILE_NAME}.tar

	# Push images
  docker push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64
  docker push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64
  docker push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7
  docker push ${IMAGE_REGISTRY}/${TEST_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}

  # Create manifest
  docker manifest create ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG} \
    --amend ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-amd64 \
    --amend ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-arm64 \
    --amend ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}-armv7
  docker manifest push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}
}

build() {
  prepare
  docker run --rm --privileged \
    -v $PWD:/go/src/github.com/kubeshop/botkube \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -w /go/src/github.com/kubeshop/botkube \
    -e GORELEASER_CURRENT_TAG=v9.99.9-dev \
    -e ANALYTICS_API_KEY="${ANALYTICS_API_KEY}" \
    goreleaser/goreleaser release --rm-dist --snapshot --skip-publish
}

build_single() {
  export GORELEASER_CURRENT_TAG=v9.99.9-dev
  docker run --rm --privileged \
    -v "$PWD":/go/src/github.com/kubeshop/botkube \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -w /go/src/github.com/kubeshop/botkube \
    -e GORELEASER_CURRENT_TAG=${GORELEASER_CURRENT_TAG} \
    -e ANALYTICS_API_KEY="${ANALYTICS_API_KEY}" \
    goreleaser/goreleaser build --single-target --rm-dist --snapshot --id botkube -o "./botkube"
  docker build -f "$PWD/build/Dockerfile" --platform "${IMAGE_PLATFORM}" -t "${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}" .
  rm "$PWD/botkube"
}

build_single_e2e(){
  export GORELEASER_CURRENT_TAG=v9.99.9-dev
  docker run --rm --privileged \
    -v "$PWD":/go/src/github.com/kubeshop/botkube \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -w /go/src/github.com/kubeshop/botkube \
    -e GORELEASER_CURRENT_TAG=${GORELEASER_CURRENT_TAG} \
    goreleaser/goreleaser build --single-target --rm-dist --snapshot --id botkube-test -o "./botkube-e2e.test"
  docker build -f "$PWD/build/test.Dockerfile" --build-arg=TEST_NAME=botkube-e2e.test --platform "${IMAGE_PLATFORM}" -t "${IMAGE_REGISTRY}/${TEST_IMAGE_REPOSITORY}:${GORELEASER_CURRENT_TAG}" .
  rm "$PWD/botkube-e2e.test"
}

usage() {
    cat <<EOM
Usage: ${0} [build|release|release_snapshot]
Where,
  build: Builds project with goreleaser without pushing images.
  release_snapshot: Builds project without publishing release. It builds and pushes BotKube image with v9.99.9-dev image tag.
EOM
    exit 1
}

[ ${#@} -gt 0 ] || usage
case "${1}" in
  build)
    build
    ;;
  build_single)
    build_single
    ;;
  build_single_e2e)
    build_single_e2e
    ;;
  release_snapshot)
    release_snapshot
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
