#!/usr/bin/env bash
set -euo pipefail
#
# This script provisions a working Wormhole dev environment on a minimally provisioned VM.
# It expects to run as a user without root permissions.
#
# Can safely run multiple times to update to the latest versions.
#

# Make sure this is a supported OS.
DISTRO="$(lsb_release --id --short)-$(lsb_release --release --short)"
case "$DISTRO" in
    Debian-10 | Debian-11 | RedHatEnterprise-8.* | Ubuntu-22.*) true ;;  # okay (no operation)
    *)
        echo "This script is only for Debian 10 or 11, RHEL 8, or Ubuntu 22"
        exit 1
    ;;
esac

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

# Make sure OS-provided Docker package isn't installed
case "$DISTRO" in
    Debian-*|Ubuntu-*)
        if dpkg -s docker.io &>/dev/null; then
            echo "Docker is already installed from repository. Please uninstall it first."
            exit 1
        fi
    ;;
    RedHatEnterprise-8.*)
        if rpm -q podman-docker &>/dev/null; then
            echo "podman-docker is installed. Please uninstall it first."
            exit 1
        fi
    ;;
    *) echo "Internal error: $DISTRO not matched in case block." && exit 1 ;;
esac

# Upgrade everything
# (this ensures that an existing Docker CE installation is up to date before continuing)
case "$DISTRO" in
    Debian-*|Ubuntu-*)    sudo apt-get update && sudo apt-get upgrade -y ;;
    RedHatEnterprise-8.*) sudo dnf upgrade -y ;;
    *) echo "Internal error: $DISTRO not matched in case block." && exit 1 ;;
esac

# Install dependencies
case "$DISTRO" in
    Debian-*|Ubuntu-*)    sudo apt-get -y install bash-completion git git-review vim  ;;
    RedHatEnterprise-8.*) sudo dnf -y install curl bash-completion git git-review vim ;;
    *) echo "Internal error: $DISTRO not matched in case block." && exit 1 ;;
esac

# Install Go
ARCH=amd64
GO=1.20.10

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
    case "$DISTRO" in
      Debian-*|Ubuntu-*)
        curl -fsSL https://get.docker.com -o get-docker.sh
        sudo sh get-docker.sh
      ;;
      RedHatEnterprise-8.*)
        # get-docker.sh doesn't support RHEL x86_64.
        curl -fsSL https://download.docker.com/linux/centos/docker-ce.repo -o docker-ce.repo
        sudo cp docker-ce.repo /etc/yum.repos.d/docker-ce.repo
        # This is a no-op if the packages are already installed, but the "upgrade everything" step above will keep them up to date.
        sudo dnf -y install docker-ce docker-ce-cli containerd.io
      ;;
    esac
  )
  rm -rf "$TMP"
  sudo gpasswd -a $USER docker
fi

sudo systemctl enable --now docker

# Install Minikube
# Use 1.24 until this regression is resolved:
#    https://github.com/kubernetes/minikube/issues/13542
case "$DISTRO" in
  Debian-*|Ubuntu-*)
    TMP=$(mktemp -d)
    (
      cd "$TMP"
      curl -LO https://github.com/kubernetes/minikube/releases/download/v1.24.0/minikube_1.24.0-0_amd64.deb
      sudo dpkg -i minikube_1.24.0-0_amd64.deb
    )
    rm -rf "$TMP"
  ;;
  RedHatEnterprise-*)
    sudo dnf -y install https://github.com/kubernetes/minikube/releases/download/v1.24.0/minikube-1.24.0-0.x86_64.rpm
  ;;
  *) echo "Internal error: $DISTRO not matched in case block." && exit 1 ;;
esac

# Install tilt.
# This script places the binary at /usr/local/bin/tilt and then self-tests by trying to execute it from PATH.
# So we need to ensure that PATH contains /usr/local/bin, which is not the case in the environment created by sudo by default on RHEL.
case "$DISTRO" in
  Debian-*|Ubuntu-*) true ;;
  RedHatEnterprise-*)
    echo -e "Defaults\tsecure_path=/sbin:/bin:/usr/sbin:/usr/bin:/usr/local/sbin:/usr/local/bin" | sudo tee /etc/sudoers.d/tilt_installer_path
  ;;
  *) echo "Internal error: $DISTRO not matched in case block." && exit 1 ;;
esac
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
