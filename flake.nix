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
        version = "0.4.1";
      in {
        packages = {
          default = pkgs.buildGoModule {
            pname = "ecolinker";
            version = version;
            pwd = ./.;
            src = ./.;
            tags = [ "prod" ];
            env.CGO_ENABLED = 0;
            vendorHash = "sha256-pZVNlZqzQU5/eu9bN5tQO45IMnmd9kRPM9I0YAo5quA=";
          };
        };
        devShells.default =
          pkgs.mkShell { packages = with pkgs; [ gnumake go git-cliff ]; };
      });
}
