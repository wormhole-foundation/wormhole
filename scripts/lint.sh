#!/usr/bin/env bash

# fail if any command fails
set -eo pipefail -o nounset

ROOT="$(dirname "$(dirname "$(realpath "$0")")")"
DOCKERFILE="$ROOT/scripts/Dockerfile.lint"

# The custom golangci-lint (with our linters/ module plugins baked in) is built
# from source under linters/ via `make -C linters build-golangci-lint`. The
# golangci-lint version is owned by linters/ (GOLANGCI_LINT_VERSION in
# linters/Makefile, which must match `version:` in linters/.custom-gcl.yml).
#
# Built binaries are cached, content-addressed by a hash of the linters/ source
# tree (which includes .custom-gcl.yml and the Makefile, so a version bump flips
# the hash). We build once and only rebuild when the linter source changes.
LINTERS_DIR="$ROOT/linters"
WORMHOLE_GOLANGCI_LINT_CACHE="$ROOT/.wormhole-lint-cache"

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
    GOFMT_OUTPUT="$(find "./sdk" "./node" "./wormchain" "./linters" -type f -name '*.go' -not -path '*.pb.go' -print0 | xargs -r -0 goimports $GOIMPORTS_ARGS 2>&1)"

    if [ -n "$GOFMT_OUTPUT" ]; then
        if [ "$GITHUB_ACTION" == "true" ]; then
            GOFMT_OUTPUT="$(echo "$GOFMT_OUTPUT" | awk '{print "::error file="$0"::Formatting error. Please format using ./scripts/lint.sh -d format."}')"
        fi
        echo "$GOFMT_OUTPUT" >&2
        exit 1
    fi
}

# Content hash of the linters/ source, used to decide whether a rebuild is needed.
# Hashes every file under linters/ (regardless of git tracking) except the
# bin/ and build/ build outputs, so any source edit triggers exactly one rebuild.
# Runs find from inside LINTERS_DIR so paths are relative: sha256sum embeds the
# path, so an absolute one would make the hash checkout-path dependent.
linters_source_hash() {
    ( cd "$LINTERS_DIR" && find . -type f \
        -not -path '*/bin/*' -not -path '*/build/*' \
        -print0 | sort -z | xargs -0 sha256sum | sha256sum | cut -d' ' -f1 )
}

ensure_wormhole_golangci_lint() {
    local hash dir bin tmp

    # In the -c docker image the custom linter is already built from source and
    # placed on PATH; run that instead of rebuilding (the repo mount is read-only).
    if command -v wormhole-golangci-lint >/dev/null 2>&1; then
        command -v wormhole-golangci-lint
        return
    fi

    hash="$(linters_source_hash)"
    dir="$WORMHOLE_GOLANGCI_LINT_CACHE/$hash"
    bin="$dir/wormhole-golangci-lint"

    if [ ! -x "$bin" ]; then
        echo "Building wormhole-golangci-lint (source ${hash:0:12})..." >&2
        make -C "$LINTERS_DIR" build-golangci-lint >&2
        mkdir -p "$dir"
        tmp="$(mktemp "$dir/.build.XXXXXX")"
        cp "$LINTERS_DIR/bin/wormhole-golangci-lint" "$tmp"
        chmod +x "$tmp"
        mv "$tmp" "$bin"  # atomic publish; safe under concurrent runs
    fi

    # Delete old cache entries. (stderr only — stdout is the binary path.)
    for old in "$WORMHOLE_GOLANGCI_LINT_CACHE"/*/; do
        [ -d "$old" ] || continue  # no subdirs yet: glob stayed literal
        if [ "$(basename "$old")" != "$hash" ]; then
            echo "Removing stale wormhole-golangci-lint ($(basename "$old" | cut -c1-12))..." >&2
            rm -rf "$old"
        fi
    done

    echo "$bin"
}

lint(){
    # === Spell check
    if ! command -v cspell >/dev/null 2>&1; then
        printf "%s\n" "cspell is not installed. Skipping spellcheck"
    else
        cspell "*/**.*md"
    fi

    # === Go linting (custom wormhole-golangci-lint)
    local LINT_BIN
    LINT_BIN="$(ensure_wormhole_golangci_lint)"

    # Lint each Go module root. linters/ is the custom linter's own code; its
    # rules/<name>/ are separate modules (mirroring `make -C linters test`), so
    # lint each one too. Run in a subshell so a failure still halts under
    # `set -e` without leaking the working directory between modules.
    lint_module() {
        ( cd "$1" && "$LINT_BIN" run --timeout=10m $GOLANGCI_LINT_ARGS ./... )
    }

    lint_module "$ROOT/node"
    lint_module "$ROOT/sdk"
    lint_module "$ROOT/linters"
    while IFS= read -r gomod; do
        lint_module "$(dirname "$gomod")"
    done < <(find "$ROOT/linters/rules" -name go.mod)
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
