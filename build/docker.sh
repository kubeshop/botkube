set +x

BUILD_ROOT=$(dirname $0)
IMAGE_REPO=${1:-infracloud/kubeops}
IMAGE_TAG=${2:-latest}

pushd ${BUILD_ROOT}/..
docker build -t $IMAGE_REPO:$IMAGE_TAG -f ${BUILD_ROOT}/Dockerfile --no-cache .
popd
