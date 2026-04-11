{
  config,
  lib,
  ...
}: let
  cfg = config.services.null;
  svcCfg = cfg.core;
  isUnixSocket = lib.hasPrefix "/" cfg.database.host;

  mkDatabaseUrl = dbName:
    if isUnixSocket
    then "postgresql:///${dbName}?host=${cfg.database.host}"
    else "postgresql://${cfg.database.user}@${cfg.database.host}:${toString cfg.database.port}/${dbName}";

  mkEnvFiles = svcSecretsFile:
    builtins.filter (f: f != null) [cfg.secretsFile svcSecretsFile];

  inherit (lib) types mkIf mkOption optionals optionalAttrs;
in {
  options.services.null.core = {
    enable = mkOption {
      type = types.bool;
      default = true;
      description = "enable core backend service";
    };

    package = mkOption {
      type = types.package;
      description = "the null-core package to use";
    };

    port = mkOption {
      type = types.port;
      default = 55555;
      description = "listen port";
    };

    hostname = mkOption {
      type = types.str;
      default = "127.0.0.1";
      description = "bind address";
    };

    exchangeApiUrl = mkOption {
      type = types.str;
      default = "https://api.frankfurter.dev/v1";
      description = "currency exchange rate API URL";
    };

    environment = mkOption {
      type = types.submodule {freeformType = types.attrsOf types.str;};
      default = {};
      description = "extra environment variables for null-core";
    };

    secretsFile = mkOption {
      type = types.nullOr types.str;
      default = null;
      description = "core-specific secrets file";
    };
  };

  config = mkIf (cfg.enable && svcCfg.enable) {
    services.null.core.environment =
      {
        LISTEN_ADDRESS = "${svcCfg.hostname}:${toString svcCfg.port}";
        LOG_LEVEL = cfg.logLevel;
        LOG_FORMAT = cfg.logFormat;
        EXCHANGE_API_URL = svcCfg.exchangeApiUrl;
        NULL_GATEWAY_URL = "http://${cfg.gateway.hostname}:${toString cfg.gateway.port}";
        NULL_RECEIPTS_URL = "${cfg.receipts.hostname}:${toString cfg.receipts.port}";
      }
      // optionalAttrs isUnixSocket {
        DATABASE_URL = mkDatabaseUrl cfg.database.name;
      };

    systemd.services.null-core = {
      description = "null: core backend";
      wantedBy = ["multi-user.target"];
      after = ["network.target"] ++ optionals cfg.database.enable ["null-db-setup.service"];
      requires = optionals cfg.database.enable ["null-db-setup.service"];
      inherit (svcCfg) environment;
      serviceConfig =
        (import ./hardening.nix)
        // {
          ExecStart = "${svcCfg.package}/bin/null";
          EnvironmentFile = mkEnvFiles svcCfg.secretsFile;
          Slice = "system-null.slice";
          StateDirectory = "null";
          User = cfg.user;
          Group = cfg.group;
          ReadWritePaths = [cfg.dataDir];
        };
    };
  };
}
