#!/usr/bin/env bash
set -euo pipefail
#
# This script downloads and installs development dependencies required for
# bridge development on Linux.
#
# Tested on amd64 with CentOS 8, Debian 10 and Ubuntu 20.04. Likely works on other distros as well.
#
# We use this to set up our CI, but you might find it useful for your own development needs.
#
# This installer pulls in binaries from, and therefore trusts, a number of third-party sources:
#
#  - k3s.io by Rancher Labs.
#  - Go binary distribution by Google.
#  - Tilt binary distribution by Tilt.
#
# Idempotent and safe to run multiple times for upgrades (but restarts your cluster).
# Goes without saying, but this is NOT for production, or anywhere close to it.
#
# If you want to allow other users on the host access to the k3s cluster, look into
# run the installer script with an appropriate K3S_KUBECONFIG_MODE.

if [[ $EUID -ne 0 ]]; then
  echo "This script must be run as root"
  exit 1
fi

# TODO: https://docs.docker.com/engine/security/userns-remap/

if ! docker info; then
  echo "Please install and configure Docker first"
  exit 1
fi

# On Ubuntu/Debian, switch to iptables-legacy
# https://github.com/rancher/k3s/issues/1114
if update-alternatives --set iptables /usr/sbin/iptables-legacy; then
  systemctl restart docker
fi

# Ensure that our binaries are not shadowed by the distribution.
export PATH=/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/sbin:/bin

# Install Go binaries.
ARCH=amd64
GO=1.17.0
# TODO(leo): verify checksum

(
  if [[ -d /usr/local/go ]]; then
    rm -rf /usr/local/go
  fi

  TMP=$(mktemp -d)

  (
    cd "$TMP"
    curl -OJ "https://dl.google.com/go/go${GO}.linux-${ARCH}.tar.gz"
    tar -C /usr/local -xzf "go${GO}.linux-${ARCH}.tar.gz"

    echo 'PATH=/usr/local/go/bin:$PATH' >/etc/profile.d/local_go.sh
  )

  rm -rf "$TMP"
)

. /etc/profile.d/local_go.sh

# Install Tilt (latest stable release by Tilt). Install script looks fine.
curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash

# Install k3s with sane defaults and make it use the local Docker daemon.
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="v1.19.3+k3s2" INSTALL_K3S_EXEC="server
  --disable-cloud-controller
  --kube-scheduler-arg=address=127.0.0.1
  --kube-controller-manager-arg=address=127.0.0.1
  --disable traefik
  --docker
" sh -s -

cat <<'EOF' > /etc/profile.d/k3s.sh
alias kc=kubectl

source <(kubectl completion bash)
complete -F __start_kubectl kc

function use-namespace {
  kubectl config set-context --current --namespace=$1
}

# Required for tilt to find the local cluster
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
EOF

cat <<'EOF' > /etc/profile.d/buildkit.sh
# Enable buildkit support in Docker for incremental builds
export DOCKER_BUILDKIT=1
EOF

. /etc/profile.d/k3s.sh

# Set default namespace to wormhole to make it easier to reset without deleting the cluster.
! kubectl create namespace wormhole
use-namespace wormhole

# Trick tilt into not pushing images by pretending to be docker-desktop
# FIXME: https://github.com/tilt-dev/tilt/issues/3654
sed -i 's/  name: default/  name: docker-desktop/g' $KUBECONFIG
sed -i 's/current-context: default/current-context: docker-desktop/g' $KUBECONFIG
sed -i 's/cluster: default/cluster: docker-desktop/g' $KUBECONFIG

while ! k3s kubectl get all; do
  echo "Waiting for k3s..."
  systemctl status k3s.service
  sleep 5
done

echo "Done! You have to reopen your shell or source the new profile scripts:"
echo "  source /etc/profile.d/k3s.sh"
echo "  source /etc/profile.d/buildkit.sh"
echo "  source /etc/profile.d/local_go.sh"
