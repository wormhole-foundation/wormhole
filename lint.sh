#!/usr/bin/env bash
DOCKERFILE="Dockerfile.lint"

# fail if any command fails
set -e
set -o pipefail

print_help() {
    printf "Usage: $(basename $0) [-h] [-c] [-w] [-d] [-l].\n"
    printf "-h\tPrint this help.\n"
    printf "-c\tRun in docker and don't worry about dependencies\n"
    printf "-w\tAutomatically fix all formatting issues\n"
    printf "-d\tPrint diff for all formatting issues\n"
    printf "-l\tList files that have formatting issues\n"
}

DOCKER=""
GOLANGCI_LINT_ARGS=""
SELF_ARGS_WITHOUT_DOCKER="${*/c/}"

while getopts 'hcg' opt; do
    case "$opt" in
    c)
        DOCKER="true"
        ;;
    g)
        GOLANGCI_LINT_ARGS+="--out-format=github-actions "
        ;;

    h)
        print_help
        exit 0
        ;;

    ?)
        printf "Invalid command option."
        print_help
        exit 1
        ;;
    esac
done
shift "$(($OPTIND - 1))"

# run this script recursively inside docker, if requested
if [ "$DOCKER" == "true" ]; then

    if grep -sq 'docker\|lxc' /proc/1/cgroup; then
        printf "Already running inside a container. This situation isn't supported (yet).\n"
        exit 1
    fi

    DOCKER_IMAGE="$(docker build -q -f "$DOCKERFILE" .)"
    COMMAND="./$(basename "$0") $SELF_ARGS_WITHOUT_DOCKER"
    MOUNT="--workdir /app --mount=type=bind,target=/app,source=$PWD"

    docker run "$MOUNT" "$DOCKER_IMAGE" "$COMMAND"
    exit "$?"
fi

# Check for dependencies
if ! command -v golangci-lint >/dev/null 2>&1; then
    printf "%s\n" "Require golangci-lint. You can run this command in a docker container instead with '-c' and not worry about it or install it: https://golangci-lint.run/usage/install/"
fi

# Do the actual linting!
cd node/
golangci-lint run --skip-dirs pkg/supervisor --timeout=10m "$GOLANGCI_LINT_ARGS" ./...
