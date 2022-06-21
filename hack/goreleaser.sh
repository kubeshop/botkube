#!/bin/bash
# Copyright (c) 2021 InfraCloud Technologies
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


set -o errexit
set -o pipefail

IMAGE_REGISTRY="${IMAGE_REGISTRY:-ghcr.io/infracloudio}"
IMAGE_NAME="${IMAGE_NAME:-botkube}"

prepare() {
  export DOCKER_CLI_EXPERIMENTAL="enabled"
  docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
}

release_snapshot() {
  prepare
  export GORELEASER_CURRENT_TAG=v9.99.9-dev
  goreleaser release --rm-dist --snapshot --skip-publish
  # Push images
  docker push ${IMAGE_REGISTRY}/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-amd64
  docker push ${IMAGE_REGISTRY}/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-arm64
  docker push ${IMAGE_REGISTRY}/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-armv7
  # Create manifest
  docker manifest create ${IMAGE_REGISTRY}/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG} \
    --amend ${IMAGE_REGISTRY}/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-amd64 \
    --amend ${IMAGE_REGISTRY}/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-arm64 \
    --amend ${IMAGE_REGISTRY}/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-armv7
  docker manifest push ${IMAGE_REGISTRY}/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}
}

save_pr_image() {
  prepare

  if [ -z "${PR_NUMBER}" ]
  then
    echo "Missing PR_NUMBER."
    exit 1
  fi

  export GORELEASER_CURRENT_TAG=v${PR_NUMBER}-PR
  goreleaser release --rm-dist --snapshot --skip-publish

  # Re-tag with 'pr' prefix
  docker tag ${IMAGE_REGISTRY}/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-amd64 ${IMAGE_REGISTRY}/pr/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-amd64
  docker tag ${IMAGE_REGISTRY}/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-arm64 ${IMAGE_REGISTRY}/pr/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-arm64
  docker tag ${IMAGE_REGISTRY}/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-armv7 ${IMAGE_REGISTRY}/pr/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-armv7

  # Push images
  docker save ${IMAGE_REGISTRY}/pr/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-amd64 > /tmp/${IMAGE_NAME}-amd64.tar
  docker save ${IMAGE_REGISTRY}/pr/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-arm64 > /tmp/${IMAGE_NAME}-arm64.tar
  docker save ${IMAGE_REGISTRY}/pr/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-armv7 > /tmp/${IMAGE_NAME}-armv7.tar
}

push_pr_image() {
  prepare
  if [ -z "${PR_NUMBER}" ]
  then
    echo "Missing PR_NUMBER."
    exit 1
  fi

  export GORELEASER_CURRENT_TAG=v${PR_NUMBER}-PR

  # Load images
  docker load --input /tmp/${IMAGE_NAME}-amd64.tar
  docker load --input /tmp/${IMAGE_NAME}-arm64.tar
  docker load --input /tmp/${IMAGE_NAME}-armv7.tar

	# Push images
	docker push ${IMAGE_REGISTRY}/pr/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-amd64
	docker push ${IMAGE_REGISTRY}/pr/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-arm64
	docker push ${IMAGE_REGISTRY}/pr/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-armv7

  # Create manifest
  docker manifest create ${IMAGE_REGISTRY}/pr/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG} \
    --amend ${IMAGE_REGISTRY}/pr/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-amd64 \
    --amend ${IMAGE_REGISTRY}/pr/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-arm64 \
    --amend ${IMAGE_REGISTRY}/pr/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}-armv7
  docker manifest push ${IMAGE_REGISTRY}/pr/${IMAGE_NAME}:${GORELEASER_CURRENT_TAG}
}

build() {
  prepare
  docker run --rm --privileged \
    -v $PWD:/go/src/github.com/infracloudio/botkube \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -w /go/src/github.com/infracloudio/botkube \
    -e GORELEASER_CURRENT_TAG=v9.99.9-dev \
    goreleaser/goreleaser release --rm-dist --snapshot --skip-publish
}

release() {
  prepare
  if [ -z ${GITHUB_TOKEN} ]
  then
    echo "Missing GITHUB_TOKEN."
    exit 1
  fi
  goreleaser release --parallelism=1 --rm-dist
}


usage() {
    cat <<EOM
Usage: ${0} [build|release|release_snapshot]
Where,
  build: Builds project with goreleaser without pushing images.
  release_snapshot: Builds project without publishing release. It builds and pushes BotKube image with v9.99.9-dev image tag.
  release: Makes and published release to GitHub
EOM
    exit 1
}

[ ${#@} -gt 0 ] || usage
case "${1}" in
  build)
    build
    ;;
  release)
    release
    ;;
  release_snapshot)
    release_snapshot
    ;;
  save_pr_image)
    save_pr_image
    ;;
  push_pr_image)
    push_pr_image
    ;;
esac
