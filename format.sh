#!/usr/bin/env bash
DOCKERFILE="Dockerfile.format"

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

GOIMPORTS_ARGS=""
DOCKER=""
SELF_ARGS_WITHOUT_DOCKER="${*/c/}"

while getopts 'cwdlh' opt; do
    case "$opt" in
    c)
        DOCKER="true"
        ;;
    w)
        GOIMPORTS_ARGS+="-w "
        ;;
    d)
        GOIMPORTS_ARGS+="-d "
        ;;

    l)
        GOIMPORTS_ARGS+="-l "
        ;;

    h)
        print_help
        exit 0
        ;;

    ?)
        printf "Invalid command option.\n"
        print_help
        exit 1
        ;;
    esac
done
shift "$(($OPTIND - 1))"

if [ "$GOIMPORTS_ARGS" == "" ]; then
    printf "Need to specify at least one argument.\n"
    print_help
    exit 1
fi

# run this script recursively inside docker, if requested
if [ "$DOCKER" == "true" ]; then
    # The easy thing to do here would be to use a bind mount to share the code with the container. 
    # But this doesn't work in scenarios where we are in a container already. 
    # But it's easy so we just won't support that case for now.
    # If we wanted to support it, my idea would be to `docker run`, `docker cp`, `docker exec`, `docker rm`.

    if grep -sq 'docker\|lxc\|kubepods' /proc/1/cgroup; then
        printf "Already running inside a container. This situation isn't supported (yet).\n"
        exit 1
    fi

    DOCKER_IMAGE="$(docker build -q -f "$DOCKERFILE" .)"
    COMMAND="./$(basename "$0") $SELF_ARGS_WITHOUT_DOCKER"
    MOUNT="--workdir /app --mount=type=bind,target=/app,source=$PWD"

    # for safety, mount as readonly unless -w flag was given
    if ! [[ "$GOIMPORTS_ARGS" =~ "w" ]]; then
        MOUNT+=",readonly"
    fi
    docker run "$MOUNT" "$DOCKER_IMAGE" "$COMMAND"
    exit "$?"
fi

# Check for dependencies
if ! command -v goimports >/dev/null 2>&1; then
    printf "%s\n" "Require goimports. You can run this command in a docker container instead with '-c' and not worry about it or install it: go install golang.org/x/tools/cmd/goimports@latest"
fi

# The actual formatting is done here!

# see http://mywiki.wooledge.org/BashPitfalls, pitfall #1
GOFMT_OUTPUT="$(find ./node ./event_database -type f -name '*.go' -not -path './node/pkg/proto/*' -exec goimports $GOIMPORTS_ARGS {} + 2>&1)"

if [ -n "$GOFMT_OUTPUT" ]; then
    printf "%s\n" "$GOFMT_OUTPUT"
    #printf "All the following files are not correctly formatted\n${GOFMT_OUTPUT}\n"
    exit 1
fi
