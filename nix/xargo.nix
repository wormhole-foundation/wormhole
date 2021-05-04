{ rust ? import ./rust.nix }: (
  self: super: {
    xargo = self.rustPlatform.buildRustPackage rec {
      pname = "xargo";
      src = self.fetchFromGitHub {
        owner = "japaric";
        repo = pname;
      };
      cargoSha256 = "unset";
      meta = with self.lib; {
        description = "The rust cross-compilation tool";
        homepage = "https://github.com/japaric/xargo";
        maintainers = maintainers.tailhook;
      };
    };
  }
)
