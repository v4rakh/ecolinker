{
  description = "ecolinker flake";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-25.05";
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
        version = "0.1.1";
      in {
        packages = {
          default = pkgs.buildGoModule {
            pname = "ecolinker";
            version = version;
            pwd = ./.;
            src = ./.;
            tags = [ "prod" ];
            env.CGO_ENABLED = 0;
            vendorHash = "sha256-vwxu/Y0uRcy13hAL6y5NAffItWX3hDSjPPmFJZU3bM0=";
          };
        };
        devShells.default =
          pkgs.mkShell { packages = with pkgs; [ gnumake go git-cliff ]; };
      });
}
