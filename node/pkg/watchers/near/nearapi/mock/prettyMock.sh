# This script requires `sponge`.
# To install sponge on RHEL 8:
# subscription-manager repos --enable codeready-builder-for-rhel-8-x86_64-rpms
# dnf install moreutils

find "$(dirname "$(realpath "$0")")" -type f -name '*.json' -exec sh -c "jq . {} | sponge {}" \;
