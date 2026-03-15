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

      commonBuildInputs = with pkgs; [ pkg-config openssl protobuf ];

      # Local deps (cloned into deps/)
      depsDir = ./deps;

    in {
      packages.${system} = {
        sdk = stableNaersk.buildPackage {
          pname = "wormhole-sdk-rust";
          version = "0.1.0";
          src = ./sdk/rust;
          nativeBuildInputs = commonBuildInputs;
          doCheck = false;
        };

        solana = stableNaersk.buildPackage {
          pname = "wormhole-solana";
          version = "0.1.0";
          src = ./solana;
          nativeBuildInputs = commonBuildInputs;
          BRIDGE_ADDRESS = "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth";
          PROTOC = "${pkgs.protobuf}/bin/protoc";
          postPatch = ''
            # Redirect git deps to local submodules
            mkdir -p .cargo
            cat > .cargo/config.toml <<'CFG'
            [source.spl-token-git]
            git = "https://github.com/wormhole-foundation/spl-token.git"
            rev = "7ae1b55"
            replace-with = "vendored-sources"

            [source.metaplex-git]
            git = "https://github.com/wormhole-foundation/metaplex-program-library"
            rev = "a7ab32ab0defd89c98f205c80ebdaf77ed60152d"
            replace-with = "vendored-sources"

            [source.vendored-sources]
            directory = "vendor"
            CFG

            # Vendor the git deps
            mkdir -p vendor
            cp -r ${depsDir}/spl-token vendor/spl-token-0.0.0
            cp -r ${depsDir}/metaplex-program-library vendor/metaplex-0.0.0
          '';
          doCheck = false;
        };

        cosmwasm = stableNaersk.buildPackage {
          pname = "wormhole-cosmwasm";
          version = "0.1.0";
          src = ./cosmwasm;
          nativeBuildInputs = commonBuildInputs;
          postPatch = ''
            mkdir -p .cargo
            cat > .cargo/config.toml <<'CFG'
            [source.ntt-git]
            git = "https://github.com/wormhole-foundation/example-native-token-transfers.git"
            rev = "cd1f8fe13b9aba3bf1a38938d952841c98cb7288"
            replace-with = "vendored-sources"

            [source.vendored-sources]
            directory = "vendor"
            CFG

            mkdir -p vendor
            cp -r ${depsDir}/example-native-token-transfers vendor/ntt-0.0.0
          '';
          doCheck = false;
        };

        # Terra: cw20-legacy repo deleted, use crates.io version instead
        terra = stableNaersk.buildPackage {
          pname = "wormhole-terra";
          version = "0.1.0";
          src = ./terra;
          nativeBuildInputs = commonBuildInputs;
          postPatch = ''
            # Replace git dep with crates.io version
            sed -i 's|cw20-legacy = { git.*}|cw20-legacy = "0.2.0"|' Cargo.toml
            # Regenerate lockfile to pick up crates.io source
            cargo generate-lockfile 2>/dev/null || true
          '';
          doCheck = false;
        };
      };

      devShells.${system}.default = pkgs.mkShell {
        nativeBuildInputs = commonBuildInputs ++ [ stableToolchain ];
        PROTOC = "${pkgs.protobuf}/bin/protoc";
      };
    };
}
