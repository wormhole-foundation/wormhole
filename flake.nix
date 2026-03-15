{
  description = "Wormhole cross-chain bridge — Rust components";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    fenix = {
      url = "github:nix-community/fenix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    naersk = {
      url = "github:nix-community/naersk";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, fenix, naersk }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      stableToolchain = fenix.packages.${system}.stable.toolchain;
      stableNaersk = naersk.lib.${system}.override {
        cargo = stableToolchain;
        rustc = stableToolchain;
      };
      buildInputs = with pkgs; [ pkg-config openssl protobuf ];
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
          src = ./terra;
          nativeBuildInputs = buildInputs;
          doCheck = false;
        };

        solana = stableNaersk.buildPackage {
          pname = "wormhole-solana";
          version = "0.1.0";
          src = ./solana;
          nativeBuildInputs = buildInputs;
          BRIDGE_ADDRESS = "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth";
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
