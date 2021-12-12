#!/usr/bin/env bash
set -euo pipefail
#
# This script provisions a working Wormhole dev environment on a blank Debian VM.
# It expects to run as a user without root permissions.
#
# Can safely run multiple times to update to the latest versions.
#

# Make sure this is Debian 10 or 11
if [ "$(lsb_release -rs)" != "10" ] && [ "$(lsb_release -rs)" != "11" ]; then
  echo "This script is only for Debian 10 or 11"
  exit 1
fi

# Refuse to run as root
if [[ $EUID -eq 0 ]]; then
    echo "This script must not be run as root" 1>&2
    exit 1
fi

# Check if we can use sudo to get root
if ! sudo -n true; then
    echo "This script requires sudo to run."
    exit 1
fi

# Make sure Docker Debian package isn't installed
if dpkg -s docker.io &>/dev/null; then
    echo "Docker is already installed from Debian's repository. Please uninstall it first."
    exit 1
fi

# Upgrade everything
# (this ensures that an existing Docker CE installation is up to date before continuing)
sudo apt-get update && sudo apt-get upgrade -y

# Install dependencies
sudo apt-get -y install bash-completion git git-review vim

# Install Go
ARCH=amd64
GO=1.17.5

(
  if [[ -d /usr/local/go ]]; then
    sudo rm -rf /usr/local/go
  fi

  TMP=$(mktemp -d)

  (
    cd "$TMP"
    curl -OJ "https://dl.google.com/go/go${GO}.linux-${ARCH}.tar.gz"
    sudo tar -C /usr/local -xzf "go${GO}.linux-${ARCH}.tar.gz"

    echo 'PATH=/usr/local/go/bin:$PATH' | sudo tee /etc/profile.d/local_go.sh
  )

  rm -rf "$TMP"
)

. /etc/profile.d/local_go.sh

# Install Docker and add ourselves to Docker group
if [[ ! -f /usr/bin/docker ]]; then
  TMP=$(mktemp -d)
  (
    cd "$TMP"
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
  )
  rm -rf "$TMP"
  sudo gpasswd -a $USER docker
fi

sudo systemctl enable --now docker

# Install Minikube
TMP=$(mktemp -d)
(
  cd "$TMP"
  curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube_latest_amd64.deb
  sudo dpkg -i minikube_latest_amd64.deb
)
rm -rf "$TMP"

# Install tilt
curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | sudo bash

# Shell aliases
cat <<'EOF' | sudo tee /etc/profile.d/wormhole_aliases.sh
alias kubectl="minikube kubectl --"
alias vi=vim
alias kc=kubectl

. <(kubectl completion bash)
. <(minikube completion bash)
complete -F __start_kubectl kc

function use-namespace {
  kubectl config set-context --current --namespace=$1
}

export DOCKER_BUILDKIT=1

alias start-recommended-minikube="minikube start --driver=docker --kubernetes-version=v1.22.3 --cpus=$(nproc) --memory=16G --disk-size=120g --namespace=wormhole"
EOF

cat <<EOF

┍━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┑
│                                                                 │
│                            SUCCESS                              │
│                                                                 │
│           Re-log into your session to apply changes.            │
│                                                                 │
└━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┘
EOF
