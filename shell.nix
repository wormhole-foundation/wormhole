{ sources ? import nix/sources.nix # Please specify inputs obtained from `sources` as separate params like below
, cargo2nix ? sources.cargo2nix
, nixpkgs ? sources.nixpkgs
, rust-olay ? import sources.rust-overlay
}:
let
  tilt-olay = self: super: {
    tilt184 = self.tilt.overrideAttrs (
      oldAttrs: {
        version = "0.18.4";
        src = self.fetchFromGitHub {
          owner = "tilt-dev";
          repo = oldAttrs.pname;
          rev = "v0.18.4";
          sha256 = "sha256-xqBgbsrVSAOqtfHbEF07i6XIdiBXMYoR7H4Kc4xK7x0=";
        };
        buildFlagsArray = [ "-ldflags=-X main.version=0.18.4" ];
      }
    );
  };
  scripts-olay = self: super: {
    whcluster = self.writeShellScriptBin "whcluster" ''
      set -e
      default_minikube_args="--cpus=10 --memory=10gb --disk-size=200gb"
      export MINIKUBE_ARGS=''${MINIKUBE_ARGS:-$default_minikube_args}
      ${self.minikube}/bin/minikube start $MINIKUBE_ARGS
      ${self.minikube}/bin/minikube ssh 'echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf && sudo sysctl -p'
      ${self.whkube}/bin/whkube
    '';
    whkube = self.writeShellScriptBin "whkube" ''${self.kubectl}/bin/kubectl config set-context --current --namespace=wormhole'';
    whtilt = self.writeShellScriptBin "whtilt" ''
      echo "Starting Tilt with ''${1:=5} guardians"
      ${self.tilt184}/bin/tilt up --update-mode exec -- --num=$1
    '';
    whinotify = self.writeShellScriptBin "whinotify" ''
      ${self.minikube}/bin/minikube ssh 'echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf && sudo sysctl -p'
    '';
    # stan-work config
    whsw = self.writeShellScriptBin "whsw" ''
      ${pkgs.mutagen}/bin/mutagen sync terminate whsw-sync || true
      ${pkgs.mutagen}/bin/mutagen sync create -n whsw-sync . stan-work:~/wormhole
      ${pkgs.mutagen}/bin/mutagen sync flush whsw-sync
      export MINIKUBE_ARGS='--cpus=30 --memory=110g --disk-size=1000gb'
      ssh stan-work \
        ". ~/.zprofile && \
            cd wormhole && \
            nix-shell --command ' MINIKUBE_ARGS=\"$MINIKUBE_ARGS\" whcluster && \
            killall tilt || true'"
      ssh -L 10350:127.0.0.1:10350 stan-work \
        "cd wormhole && \
            source ~/.zprofile && \
            nix-shell shell.nix --command 'whtilt $WH_GUARDIAN_COUNT'"
    '';
  };
  cargo2nix-olay = import "${cargo2nix}/overlay";
  pkgs = import nixpkgs {
    overlays = [
      # cargo2nix-olay
      rust-olay
      tilt-olay
      scripts-olay
    ];
  };
  cargo2nix-drv = import cargo2nix {
    inherit nixpkgs;
  };
in
pkgs.mkShell {
  nativeBuildInputs = (
    with pkgs; [
      go
      gopls
      hidapi
      libudev
      openssl
      pkgconfig
      protobuf
      whcluster
      whinotify
      whkube
      whtilt
      whsw
      # (
      #   rust-bin.stable."1.51.0".default.override {
      #     extensions = [
      #       "rust-src"
      #       "rust-analysis"
      #     ];
      #   }
      # )
      # Provided on Fedora:
      kubectl
      minikube
      tilt184
      # xargo
    ]
  );
  DOCKER_BUILDKIT = 1;
  PROTOC = "${pkgs.protobuf}/bin/protoc";
}
