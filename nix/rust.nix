{ sources ? import ./sources.nix }:

let
  pkgs =
  import sources.nixpkgs {overlays = [ (import sources.nixpkgs-mozilla) ]; };
  channel = "nightly";
  date = "2020-11-19";
  targets = [];
  chan = pkgs.rustChannelOfTargets channel date targets;
in chan
