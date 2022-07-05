#!/bin/bash

set -e

version=$(cut -d'=' -f2- .release)
if [[ -z ${version} ]]; then
    echo "Invalid version set in .release"
    exit 1
fi

if [[ -z ${GITHUB_TOKEN} ]]; then
    echo "GITHUB_TOKEN not set. Usage: GITHUB_TOKEN=<TOKEN> ./hack/release.sh"
    exit 1
fi

echo "Publishing release ${version}"

fetch_previous_version() {
    git fetch origin --tags
    previous_version=$(git tag --sort=-creatordate | head -1 | tr -d '\r')
}

generate_changelog() {
    local version=$1

    # generate changelog from github
    github_changelog_generator --user kubeshop --project botkube -t ${GITHUB_TOKEN} --future-release ${version} -o CHANGELOG.md
    sed -i.bak '$d' CHANGELOG.md
}

update_chart_yamls() {
    local version_to_replace=$1
    local version=$2
    echo "Updating release version $version_to_replace-> $version"
    dir=(./helm)
    for d in ${dir[@]}
    do
        find $d -type f -name "*.yaml" -exec sed -i.bak "s/$version_to_replace/$version/g" {} \;
    done
}

fetch_previous_version
update_chart_yamls $previous_version $version
update_chart_yamls "v9.99.9-dev" $version
generate_changelog $version
make release

echo "=========================== Done ============================="
echo "Congratulations!! Release ${version} published."
echo "Don't forget to add changelog in the release description."
echo "=============================================================="
