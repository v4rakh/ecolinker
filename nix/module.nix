{ self, lib }:
{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.services.ecolinker;
in
{
  options.services.ecolinker = {
    enable = lib.mkEnableOption "[ecolinker](https://git.myservermanager.com/varakh/ecolinker) - ecolinker";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
      defaultText = lib.literalExpression "self.packages.\${system}.default";
      description = "The ecolinker package to use.";
    };

    environment = lib.mkOption {
      type = lib.types.attrsOf lib.types.str;
      default = { };
      example = {
        SERVER_LISTEN = "127.0.0.1";
        SERVER_PORT = "8080";
      };
      description = ''
        Environment variables for ecolinker. Non-sensitive values go here.
        Secrets (ECOFLOW_ACCESS_KEY, etc.) must be
        set via {option}`environmentFiles` so they are not stored in the nix store.
        See [configuration reference](https://git.myservermanager.com/varakh/ecolinker) for all options.
      '';
    };

    environmentFiles = lib.mkOption {
      type = lib.types.listOf lib.types.path;
      default = [ ];
      example = [ "/run/secrets/ecolinker.env" ];
      description = ''
        Files containing additional environment variables for ecolinker.
        Secrets such as ECOFLOW_ACCESS_KEY, ECOFLOW_SECRET_KEY, DB_POSTGRES_PASSWORD, and PROMETHEUS_SECURE_TOKEN must be provided here
        rather than in {option}`environment` to avoid storing them in the nix store.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.ecolinker = {
      description = "ecolinker - retrieve information from EcoFlow";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];

      environment = cfg.environment;

      serviceConfig = {
        ExecStart = "${cfg.package}/bin/ecolinker server serve";
        EnvironmentFile = cfg.environmentFiles;
        DynamicUser = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        RestrictNamespaces = true;
        PrivateDevices = true;
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectControlGroups = true;
        LockPersonality = true;
        MemoryDenyWriteExecute = true;
        RestrictRealtime = true;
      };
    };
  };
}
