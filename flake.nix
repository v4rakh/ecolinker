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
              doCheck = false;
              vendorHash = "sha256-Z5qoRFnIzkhlCucceXYpJYMFtjqkDFKigGAwn9yehkc=";
              ldflags = [
                "-s"
                "-w"
              ];
            };
          };

          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              git-cliff
              gnumake
              go
              golangci-lint
              grype
            ];
          };
        };
    };
}
