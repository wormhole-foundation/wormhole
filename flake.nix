{
  description = "Wormhole cross-chain bridge — Rust components";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    fenix = { url = "github:nix-community/fenix"; inputs.nixpkgs.follows = "nixpkgs"; };
    naersk = { url = "github:nix-community/naersk"; inputs.nixpkgs.follows = "nixpkgs"; };
    
    # Dep sources
    spl-token-src = { url = "github:meta-introspector/spl-token/7ae1b553adb5889e57e7afb054e557ab7dd0d873"; flake = false; };
    metaplex-src = { url = "github:meta-introspector/metaplex-program-library/a7ab32ab0defd89c98f205c80ebdaf77ed60152d"; flake = false; };
    cw20-legacy-src = { url = "github:meta-introspector/cw20-legacy/d12724701ffb9920cb1fdf1eee49efdcc20048e5"; flake = false; };
  };

  outputs = { self, nixpkgs, fenix, naersk, spl-token-src, metaplex-src, cw20-legacy-src }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};

      # Stable toolchain for sdk/cosmwasm/terra
      stableToolchain = fenix.packages.${system}.stable.toolchain;
      stableNaersk = naersk.lib.${system}.override {
        cargo = stableToolchain;
        rustc = stableToolchain;
      };

      # Nightly 2022-02-24 for solana (matches rust-toolchain.toml)
      solanaNightly = fenix.packages.${system}.toolchainOf {
        channel = "nightly";
        date = "2022-02-24";
        sha256 = "sha256-TpJKRroEs7V2BTo2GFPJlEScYVArFY2MnGpYTxbnSo8=";
      };
      solanaToolchain = solanaNightly.minimalToolchain;
      solanaNaersk = naersk.lib.${system}.override {
        cargo = solanaToolchain;
        rustc = solanaToolchain;
      };

      buildInputs = with pkgs; [ pkg-config openssl protobuf ];

      # Assemble terra source with cw20-legacy dep
      terraSrc = pkgs.runCommand "wormhole-terra-src" {} ''
        cp -r ${./terra} $out
        chmod -R u+w $out
        mkdir -p $out/deps
        cp -r ${cw20-legacy-src} $out/deps/cw20-legacy
      '';

      solanaSrc = pkgs.runCommand "wormhole-solana-src" {} ''
        cp -r ${./solana} $out
        chmod -R u+w $out
        mkdir -p $out/deps/metaplex-program-library/token-metadata
        cp -r ${spl-token-src} $out/deps/spl-token
        cp -r ${metaplex-src}/token-metadata/program $out/deps/metaplex-program-library/token-metadata/program
        chmod -R u+w $out/deps
        sed -i 's/#!\[deny(missing_docs)\]/#![allow(missing_docs)]/' $out/deps/spl-token/src/lib.rs
      '';
    in {
      packages.${system} = {
        sdk = stableNaersk.buildPackage {
          pname = "wormhole-sdk-rust";
          version = "0.1.0";
          src = ./sdk/rust;
          nativeBuildInputs = buildInputs;
          doCheck = false;
        };

        cosmwasm = stableNaersk.buildPackage {
          pname = "wormhole-cosmwasm";
          version = "0.1.0";
          src = ./cosmwasm;
          nativeBuildInputs = buildInputs;
          doCheck = false;
        };

        terra = stableNaersk.buildPackage {
          pname = "wormhole-terra";
          version = "0.1.0";
          src = terraSrc;
          nativeBuildInputs = buildInputs;
          doCheck = false;
        };

        solana = solanaNaersk.buildPackage {
          pname = "wormhole-solana";
          version = "0.1.0";
          src = solanaSrc;
          nativeBuildInputs = buildInputs;
          BRIDGE_ADDRESS = "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth";
          CHAIN_ID = "1";
          EMITTER_ADDRESS = "0000000000000000000000000000000000000000000000000000000000000000";
          PROTOC = "${pkgs.protobuf}/bin/protoc";
          doCheck = false;
        };
      };

      devShells.${system}.default = pkgs.mkShell {
        nativeBuildInputs = buildInputs ++ [ stableToolchain ];
        PROTOC = "${pkgs.protobuf}/bin/protoc";
      };
    };
}
