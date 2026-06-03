{ self, lib }:
{
  config,
  pkgs,
  ...
}:
let
  cfg = config.programs.ecolinker;
  tomlFormat = pkgs.formats.toml { };
in
{
  options.programs.ecolinker = {
    enable = lib.mkEnableOption "[ecolinker](https://git.myservermanager.com/varakh/ecolinker) - ecolinker";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
      defaultText = lib.literalExpression "self.packages.\${system}.default";
      description = "The ecolinker package to use.";
    };

    settings = lib.mkOption {
      type = tomlFormat.type;
      default = { };
      example = lib.literalExpression ''
        {
          server.url = "http://192.168.1.2:8181";
          auth = {
            user = "administrator";
            passwordFile = config.sops.secrets.ecolinker-password.path;
          };
          device.serialNumber = "...";
          parsing.raw = false;
        }
      '';
      description = ''
        Configuration for ecolinker, written to {file}`$XDG_CONFIG_HOME/ecolinker.toml`.
        See the [ecolinker documentation](https://git.myservermanager.com/varakh/ecolinker) for available options.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages = [ cfg.package ];

    home.file.".config/ecolinker.toml".source = tomlFormat.generate "ecolinker.toml" cfg.settings;
  };
}
