# An overlay for custom scripts
final: prev: {
  # Run a local minikube cluster. minikube config controlled with $MINIKUBE_ARGS
  # Minimum requirements:
  # * User can use Docker
  # * Docker supports BuildKit
  whcluster = final.writeShellScriptBin "whcluster" ''
    set -e
    default_minikube_args="--cpus=10 --memory=10gb --disk-size=200gb --driver=kvm2 -p minikube-$USER"
    export MINIKUBE_ARGS=''${MINIKUBE_ARGS:-$default_minikube_args}
    ${final.minikube}/bin/minikube start $MINIKUBE_ARGS
    ${final.whinotify}/bin/whinotify
    ${final.whkube}/bin/whkube
  '';

  # Change current kubectl context to the wormhole namespace
  whkube = final.writeShellScriptBin "whkube" ''${final.kubectl}/bin/kubectl config set-context --current --namespace=wormhole'';

  # Run tilt on the local cluster. Takes guardian count as argument.
  whtilt = final.writeShellScriptBin "whtilt" ''
    tilt_port=''${1:-10350}
    n_guardians=''${2:-5}
    echo "Starting Tilt with $n_guardians guardians"
    ${final.killall}/bin/killall tilt
    ${final.tilt}/bin/tilt up --update-mode exec --port $tilt_port -- --num=$n_guardians
  '';

  # increase sysctl value for inotify watch count to sufficient level
  whinotify = final.writeShellScriptBin "whinotify" ''
    ${final.minikube}/bin/minikube ssh -p minikube-$USER 'echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf && sudo sysctl -p'
  '';

  # one-stop-shop for setting up a cluster on a remote machine and
  # running tilt on it. MINIKUBE_ARGS defaults expect the remote to be
  # beefy.
  # 
  # Usage: whremote <remote-machine>
  #
  # Minimum remote machine requirements:
  # * Can run whcluster (see above)
  # * Remote user has a working nix installation (single/multi user)
  #
  # Remote machine nice-to-haves:
  # * You have passwordless/cached session access to the remote - 
  # * The remote is more powerful than your local machine. If not, use whcluster + whtilt locally instead. 
  whremote = final.writeShellScriptBin "whremote" ''
    set -x
    set -e
    remote_machine=$1
    tilt_port=''${2:-10350}

    explorer_port=''${EXPLORER_PORT:-8001}
    guardian_grpc_port=''${GUARDIAN_GRPC_PORT:-8080}

    # Use Mutagen to watch local repo and sync it with remote_machine's ~/wormhole
    ${final.mutagen}/bin/mutagen sync terminate whremote-sync || true
    ${final.mutagen}/bin/mutagen sync create -n whremote-sync . $remote_machine:~/wormhole
    ${final.mutagen}/bin/mutagen sync flush whremote-sync

    # Use larger cpu-count and memory values on the remote
    export MINIKUBE_ARGS=''${MINIKUBE_ARGS:='--cpus=15 --memory=55g --disk-size=250gb --driver=kvm2 -p minikube-$USER'}

    # Set up/update the remote minikube cluster with whcluster
    ssh $remote_machine \
      ". ~/.bash_profile; . ~/.zprofile; . ~/.profile; \
       cd wormhole && \
       nix-shell --option sandbox false --command ' MINIKUBE_ARGS=\"$MINIKUBE_ARGS\" whcluster'"

    # Run tilt using whtilt on the remote and forward its default port to localhost
    ssh \
      -L $tilt_port:127.0.0.1:$tilt_port \
      -L $explorer_port:127.0.0.1:$explorer_port \
      -L $guardian_grpc_port:127.0.0.1:$guardian_grpc_port \
      $remote_machine \
      ". ~/.bash_profile; . ~/.zprofile; . ~/.profile; \
        cd wormhole && \
        nix-shell --option sandbox false --command 'whtilt $tilt_port $n_guardians'"
  '';
}
