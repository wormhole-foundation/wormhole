#!/bin/bash
# Make updating to a new version of go a bit easier.
#
# Usage:
#     scripts/update-go-version.sh 1.21.8
#
# Any actual go package dependency updates should be manually done for
# correctness and safety. Always verify any major dependency updates.

DOCKER=${DOCKER:-podman}

# Update the github actions to use the updated version of go
update_github_actions() {
    local version=$1
    local directory=.github/workflows
    # Don't cd in and then cd out
    (
        cd "$directory" || return 1
        echo "Updating github actions under $directory"
        git grep -l go-version | xargs sed -r -i -e '/go-version/s/:.*$/: "'"${version}"'"/g'
        return "${PIPESTATUS[1]}"
    )
    return $?
}

# Update the documentation on versions of go to use
update_developer_docs() {
    local version=$1
    local documents="DEVELOP.md docs/operations.md"
    echo "Updating developer docs: $documents"
    # shellcheck disable=SC2086
    sed -i -e '/golang.org\/dl/s/>= 1\.[0-9]*\.[0-9x]*/>= '"${version}"'/' $documents
    return $?
}

get_docker_image_digest() {
    local version="$1"
    local image="${2:-docker.io/golang}"

    echo "Attempting to pull ${image}:${version} to retrieve the image digest" >&2
    # shellcheck disable=SC2155
    local digest=$($DOCKER pull "${image}:${version}" | awk '/^Digest:/{print $NF}')

    if [[ ${PIPESTATUS[0]} -ne 0 || -z "$digest" ]]; then
        echo "ERROR: could not determine digest for ${image}:${version} container image" >&2
        return 1
    fi

    echo "$digest"
}

# Keep go in Dockerfiles for wormhole specific stuff up to date with the latest go
# It is often impossible to update third party Dockerfiles due to the necessity of
# actual code changes to build with newer versions of go or go.mod dependency sad.
update_our_dockerfiles() {
    local version=$1
    local image=docker.io/golang

    # shellcheck disable=SC2207
    local wormhole_dockerfiles=($(git grep -lEi 'FROM.*go(lang)' | grep -Ev '^(wormchain/D|third_party|algorand|terra)'))

    # shellcheck disable=SC2155
    local digest=$(get_docker_image_digest "$version" "docker.io/golang")

    for dockerfile in "${wormhole_dockerfiles[@]}"; do
        if grep -qEi 'FROM.*go.*alpine' "$dockerfile"; then
            echo "WARNING: '$dockerfile' uses alpine and not debian. Please update manually" >&2
            continue
        fi

        # Flag ordering here is important to work correctly on macOS with crappy bsd sed and on Linux with more sensible gnu sed
        sed -E -i -e '/docker\.io\/golang:/s/(:)[0-9]*\.[0-9]*\.([0-9]|[0-9a-zA-Z-])*(@)sha256:[0-9a-zA-Z-]*( (AS|as)*.*$)?/\1'"$version"'\3'"$digest"'\4/g' "$dockerfile"
        # shellcheck disable=SC2181
        if [[ $? -ne 0 ]]; then
            echo "ERROR: problem updating $dockerfile to ${version}@${digest}" >&2
            return 1
        fi

        if ! grep -q "${image}:${version}@${digest}" "$dockerfile"; then
            echo "ERROR: Problem updating $dockerfile to ${version}@${digest}, please manually verify" >&2
            return 1
        fi
        printf "Successfully updated %-38s to %s\n" "$dockerfile" "${image}:${version}@${digest}"
    done
}

function main() {
    local version=$1
    if [ -z "$version" ]; then
        echo -e "ERROR: Missing go version\nUsage:\n\t$0 <GO VERSION>" >&2
        exit 1
    elif echo "$version" | grep -q ^v; then
        echo "ERROR: use explicit semver versions, not a git tag for this script" >&2
        exit 1
    fi

    if ! update_github_actions "$version"; then
        echo "ERROR: Problem updating github actions" >&2
        exit 1
    fi
    if ! update_developer_docs "$version"; then
        echo "ERROR: Problem updating developer docs" >&2
            exit 1
    fi
    if ! update_our_dockerfiles "${version}-bullseye"; then
        echo "ERROR: Problem updating dockerfiles" >&2
        exit 1
    fi
}

main "$@"
