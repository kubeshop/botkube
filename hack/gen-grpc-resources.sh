#!/usr/bin/env bash

# standard bash error handling
set -o nounset # treat unset variables as an error and exit immediately.
set -o errexit # exit immediately when a command fails.
set -E         # needs to be set if we want the ERR trap

CURRENT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT_DIR=$(cd "${CURRENT_DIR}/.." && pwd)

readonly CURRENT_DIR
readonly REPO_ROOT_DIR

readonly STABLE_PROTOC_VERSION=24.0
readonly STABLE_PROTOC_GEN_GO_GRPC_VERSION=1.3.0
readonly STABLE_PROTOC_GEN_GO_VERSION=v1.31.0

readonly GREEN='\033[0;32m'
readonly NC='\033[0m' # No Color

host::os() {
  local host_os
  case "$(uname -s)" in
    Darwin)
      host_os=osx
      ;;
    Linux)
      host_os=linux
      ;;
    *)
      echo "Unsupported host OS. Must be Linux or Mac OS X."
      exit 1
      ;;
  esac
  echo "${host_os}"
}

host::arch() {
  local host_arch
  case "$(uname -m)" in
    x86_64*)
      host_arch=x86_64
      ;;
    i?86_64*)
      host_arch=x86_64
      ;;
    amd64*)
      host_arch=x86_64
      ;;
    aarch64*)
      host_arch=aarch_64
      ;;
    arm64*)
      host_arch=aarch_64
      ;;
    *)
      echo "Unsupported host arch."
      exit 1
      ;;
  esac
  echo "${host_arch}"
}

host::install::protoc() {
  echo "Install the protoc ${STABLE_PROTOC_VERSION} locally to ./bin..."

  readonly INSTALL_DIR="${REPO_ROOT_DIR}/bin"
  mkdir -p "$INSTALL_DIR"
  pushd "$INSTALL_DIR" >/dev/null

  export GOBIN="$INSTALL_DIR"
  export PATH="${INSTALL_DIR}:${PATH}"

  readonly os=$(host::os)
  readonly arch=$(host::arch)

  # download the release
  readonly name="protoc-${STABLE_PROTOC_VERSION}-${os}-${arch}"
  curl -L -O "https://github.com/protocolbuffers/protobuf/releases/download/v${STABLE_PROTOC_VERSION}/${name}.zip"

  # extract the archive and binary
  unzip -o "${name}".zip >/dev/null
  mv "$INSTALL_DIR/bin/protoc" .
  echo -e "${GREEN}√ install protoc${NC}"

  # Install Go plugins
  go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc@v${STABLE_PROTOC_GEN_GO_GRPC_VERSION}"
  echo -e "${GREEN}√ install protoc-gen-go-grpc${NC}"
  go install "google.golang.org/protobuf/cmd/protoc-gen-go@${STABLE_PROTOC_GEN_GO_VERSION}"
  echo -e "${GREEN}√ install protoc-gen-go${NC}"

  popd >/dev/null
}

protoc::gen() {
  echo "Generating gRPC APIs..."
  protoc -I="${REPO_ROOT_DIR}/proto/" \
    -I="${REPO_ROOT_DIR}/bin/include" \
    --go-grpc_out="." \
    --go_out="." \
    "${REPO_ROOT_DIR}"/proto/*.proto

  echo "Generation completed successfully."
}

main() {
  host::install::protoc

  protoc::gen
}

main
