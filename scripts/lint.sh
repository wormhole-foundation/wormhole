#!/usr/bin/env bash

# fail if any command fails
set -eo pipefail -o nounset

ROOT="$(dirname "$(dirname "$(realpath "$0")")")"
DOCKERFILE="$ROOT/scripts/Dockerfile.lint"

VALID_COMMANDS=("lint" "format")

SELF_ARGS_WITHOUT_DOCKER=""
GOIMPORTS_ARGS=""
GOLANGCI_LINT_ARGS=""

print_help() {
    cat <<-EOF >&2
	Usage: $(basename "$0") [-h] [-c] [-w] [-d] [-l] COMMAND
        COMMAND can be one of: "${VALID_COMMANDS[*]}"
	    -h  Print this help.
	    -c  Run in docker and don't worry about dependencies
	    -w  Automatically fix all formatting issues
	    -d  Print diff for all formatting issues
	    -l  List files that have formatting issues
	    -g  Format output to be parsed by github actions
	EOF
}

format(){

    if [ "$GOIMPORTS_ARGS" == "" ]; then
        GOIMPORTS_ARGS="-l"
    fi

    # only -l supports output as github action
    if [ "$GITHUB_ACTION" == "true" ]; then
        GOIMPORTS_ARGS="-l"
    fi

    # Check for dependencies
    if ! command -v goimports >/dev/null 2>&1; then
        printf "%s\n" "Require goimports. You can run this command in a docker container instead with '-c' and not worry about it or install it: \n\tgo install golang.org/x/tools/cmd/goimports@latest" >&2
        exit 1
    fi

    # Use -exec because of pitfall #1 in http://mywiki.wooledge.org/BashPitfalls
    GOFMT_OUTPUT="$(find "./sdk" "./node" "./wormchain" -type f -name '*.go' -not -path '*.pb.go' -print0 | xargs -r -0 goimports $GOIMPORTS_ARGS 2>&1)"

    if [ -n "$GOFMT_OUTPUT" ]; then
        if [ "$GITHUB_ACTION" == "true" ]; then
            GOFMT_OUTPUT="$(echo "$GOFMT_OUTPUT" | awk '{print "::error file="$0"::Formatting error. Please format using ./scripts/lint.sh -d format."}')"
        fi
        echo "$GOFMT_OUTPUT" >&2
        exit 1
    fi
}

lint(){
    # Check for dependencies
    if ! command -v golangci-lint >/dev/null 2>&1; then
        printf "%s\n" "Require golangci-lint. You can run this command in a docker container instead with '-c' and not worry about it or install it: https://golangci-lint.run/usage/install/"
    fi

    # Do the actual linting!
    cd "$ROOT"/node
    golangci-lint run --timeout=10m --path-prefix=node $GOLANGCI_LINT_ARGS ./...

    cd "${ROOT}/sdk"
    golangci-lint run --timeout=10m $GOLANGCI_LINT_ARGS ./...
}

DOCKER="false"
GITHUB_ACTION="false"

while getopts 'cwdlgh' opt; do
    case "$opt" in
    c)
        DOCKER="true"
        ;;
    w)
        GOIMPORTS_ARGS+="-w "
        SELF_ARGS_WITHOUT_DOCKER+="-w "
        ;;
    d)
        GOIMPORTS_ARGS+="-d "
        SELF_ARGS_WITHOUT_DOCKER+="-d "
        ;;
    l)
        GOIMPORTS_ARGS+="-l "
        SELF_ARGS_WITHOUT_DOCKER+="-l "
        ;;
    g)
        GITHUB_ACTION="true"
        SELF_ARGS_WITHOUT_DOCKER+="-g "
        ;;
    h)
        print_help
        exit 0
        ;;
    ?)
        echo "Invalid command option." >&2
        print_help
        exit 1
        ;;
    esac
done
shift $((OPTIND - 1))

if [ "$#" -ne "1" ]; then
    echo "Need to specify COMMAND." >&2
    print_help
    exit 1
fi

COMMAND="$1"

if [[ ! " ${VALID_COMMANDS[*]} " == *" $COMMAND "* ]]; then
    echo "Invalid command $COMMAND." >&2
    print_help
    exit 1
fi

# run this script recursively inside docker, if requested
if [ "$DOCKER" == "true" ]; then
    # The easy thing to do here would be to use a bind mount to share the code with the container.
    # But this doesn't work in scenarios where we are in a container already.
    # But it's easy so we just won't support that case for now.
    # If we wanted to support it, my idea would be to `docker run`, `docker cp`, `docker exec`, `docker rm`.

    if grep -Esq 'docker|lxc|kubepods' /proc/1/cgroup; then
        echo "Already running inside a container. This situation isn't supported (yet)." >&2
        exit 1
    fi

    DOCKER_IMAGE="$(docker build -q -f "$DOCKERFILE" .)"
    DOCKER_EXEC="./scripts/$(basename "$0")"
    MOUNT="--mount=type=bind,target=/app,source=$PWD"

    # for safety, mount as readonly unless -w flag was given
    if ! [[ "$GOIMPORTS_ARGS" =~ "w" ]]; then
        MOUNT+=",readonly"
    fi
    docker run --workdir /app "$MOUNT" "$DOCKER_IMAGE" "$DOCKER_EXEC" $SELF_ARGS_WITHOUT_DOCKER "$COMMAND"
    exit "$?"
fi

case $COMMAND in
  "lint")
    lint
    ;;

  "format")
    format
    ;;
esac
