set +x

BUILD_ROOT=$(dirname $0)
IMAGE_REPO=${1:-infracloud/botkube}
IMAGE_TAG=${2:-latest}

[ ! -z $(go env GOOS) ] && [ ! -z $(go env GOARCH) ] && \
	GOOS=$(go env GOOS) && GOARCH=$(go env GOARCH) || \
	echo "Couldn't determine the system architecture."

pushd ${BUILD_ROOT}/..
docker build --build-arg GOOS_VAL=${GOOS} --build-arg GOARCH_VAL=${GOARCH} -t $IMAGE_REPO:$IMAGE_TAG -f ${BUILD_ROOT}/Dockerfile --no-cache .
popd
