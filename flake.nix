{
  description = "ecolinker flake";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs =
    inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];

      perSystem =
        { pkgs, ... }:
        let
          version = "0.4.1";
        in
        {
          packages = {
            default = pkgs.buildGoModule {
              pname = "ecolinker";
              version = version;
              pwd = ./.;
              src = ./.;
              env.CGO_ENABLED = 0;
              vendorHash = "sha256-ndJMpi9hX+Tq7cfrqlJUJ0mJxv6ACIrBXCUDfYcolcg=";
            };
          };
        };
    };
}
