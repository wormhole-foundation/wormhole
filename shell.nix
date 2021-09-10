{ sources ? import nix/sources.nix # Please specify inputs obtained from `sources` as separate params like below
, cargo2nix ? sources.cargo2nix
, nixpkgs ? sources.nixpkgs
, rust-olay ? import sources.rust-overlay
}:
let
  scripts-olay = import ./nix/scripts.nix;
  cargo2nix-olay = import "${cargo2nix}/overlay";
  tilt-olay = final: prev: {
    tilt = prev.callPackage ./nix/tilt.nix {};
  };
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
      niv
      nodejs-16_x
      openssl
      pkgconfig
      protobuf
      python3
      python3Packages.autopep8
      whcluster
      whinotify
      whkube
      whtilt
      whremote
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
      tilt
      # xargo
    ]
  );
  DOCKER_BUILDKIT = 1;
  PROTOC = "${pkgs.protobuf}/bin/protoc";
  EMITTER_ADDRESS = "changeme";
}
