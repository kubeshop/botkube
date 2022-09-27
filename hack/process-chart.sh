#!/usr/bin/env bash
#
# This scripts runs linters to ensure the correctness of the BotKube Helm chart.
#

# standard bash error handling
set -o nounset # treat unset variables as an error and exit immediately.
set -o errexit # exit immediately when a command fails.
set -E         # needs to be set if we want the ERR trap

readonly CURRENT_DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
readonly LOCAL_HELM_REPO_DIR=$( cd "${CURRENT_DIR}/../" && pwd )


chart::lint() {
    echo '- Linting BotKube chart...'
    docker run \
      --workdir=/data/helm \
      --volume "${LOCAL_HELM_REPO_DIR}:/data" \
      quay.io/helmpack/chart-testing:v3.5.0 \
      ct lint --all --chart-dirs . --config ./ct/lint-cfg.yaml --lint-conf ./ct/lint-rules.yaml
}

chart::docs() {
	echo '- Rendering BotKube chart README.md...'
	docker run \
      --volume "${LOCAL_HELM_REPO_DIR}/helm/botkube:/helm-docs" \
      jnorwood/helm-docs:v1.11.0 -l debug -f ./values.yaml -t ./README.tpl.md --sort-values-order file
}

main() {
	chart::lint

	chart::docs
}

main
