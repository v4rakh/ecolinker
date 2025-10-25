{
  description = "ecolinker flake";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default";
    flake-utils = {
      url = "github:numtide/flake-utils";
      inputs.systems.follows = "systems";
    };
  };

  outputs = { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = "0.2.2";
      in {
        packages = {
          default = pkgs.buildGoModule {
            pname = "ecolinker";
            version = version;
            pwd = ./.;
            src = ./.;
            tags = [ "prod" ];
            env.CGO_ENABLED = 0;
            vendorHash = "sha256-8rNMMy1f64rur6vU4TzUMkiF/qvocdkM5ZrMZ5DTjtM=";
          };
        };
        devShells.default =
          pkgs.mkShell { packages = with pkgs; [ gnumake go git-cliff ]; };
      });
}
