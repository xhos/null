{
  config,
  lib,
  ...
}: let
  cfg = config.services.null;
  svcCfg = cfg.gateway;
  isUnixSocket = lib.hasPrefix "/" cfg.database.host;

  mkDatabaseUrl = dbName:
    if isUnixSocket
    then "postgresql:///${dbName}?host=${cfg.database.host}"
    else "postgresql://${cfg.database.user}@${cfg.database.host}:${toString cfg.database.port}/${dbName}";

  mkEnvFiles = svcSecretsFile:
    builtins.filter (f: f != null) [cfg.secretsFile svcSecretsFile];

  inherit (lib) types mkIf mkOption optionals optionalAttrs concatStringsSep;
in {
  options.services.null.gateway = {
    enable = mkOption {
      type = types.bool;
      default = true;
      description = "enable authentication gateway";
    };

    package = mkOption {
      type = types.package;
      description = "the null-gateway package to use";
    };

    port = mkOption {
      type = types.port;
      default = 55550;
      description = "listen port";
    };

    hostname = mkOption {
      type = types.str;
      default = "127.0.0.1";
      description = "bind address";
    };

    url = mkOption {
      type = types.str;
      example = "https://api.finances.example.com";
      description = "public-facing URL of the gateway (used for BETTER_AUTH_URL and NEXT_PUBLIC_GATEWAY_URL)";
    };

    trustedOrigins = mkOption {
      type = types.listOf types.str;
      default = [];
      example = ["https://finances.example.com"];
      description = "CORS allowed origins";
    };

    cookieDomain = mkOption {
      type = types.nullOr types.str;
      default = null;
      example = ".example.com";
      description = "cookie domain for cross-subdomain sessions";
    };

    environment = mkOption {
      type = types.submodule {freeformType = types.attrsOf types.str;};
      default = {};
      description = "extra environment variables for null-gateway";
    };

    secretsFile = mkOption {
      type = types.nullOr types.str;
      default = null;
      example = "/run/secrets/null-gateway";
      description = "gateway-specific secrets file, must contain BETTER_AUTH_SECRET";
    };
  };

  config = mkIf (cfg.enable && svcCfg.enable) {
    assertions = [
      {
        assertion = svcCfg.secretsFile != null;
        message = "services.null.gateway.secretsFile is required (must contain BETTER_AUTH_SECRET)";
      }
    ];

    services.null.gateway.environment =
      {
        BETTER_AUTH_URL = svcCfg.url;
        NULL_CORE_URL = "http://${cfg.core.hostname}:${toString cfg.core.port}";
        TRUSTED_ORIGINS = concatStringsSep "," svcCfg.trustedOrigins;
        HOSTNAME = svcCfg.hostname;
        PORT = toString svcCfg.port;
        LOG_LEVEL = cfg.logLevel;
        LOG_FORMAT = cfg.logFormat;
      }
      // optionalAttrs isUnixSocket {
        AUTH_DATABASE_URL = mkDatabaseUrl cfg.database.authName;
      }
      // optionalAttrs (svcCfg.cookieDomain != null) {
        COOKIE_DOMAIN = svcCfg.cookieDomain;
      };

    systemd.services.null-gateway = {
      description = "null: auth gateway";
      wantedBy = ["multi-user.target"];
      after =
        ["network.target" "null-core.service"]
        ++ optionals cfg.database.enable ["null-db-setup.service"];
      requires =
        ["null-core.service"]
        ++ optionals cfg.database.enable ["null-db-setup.service"];
      inherit (svcCfg) environment;
      serviceConfig =
        (import ./hardening.nix)
        // {
          ExecStart = "${svcCfg.package}/bin/null-gateway";
          EnvironmentFile = mkEnvFiles svcCfg.secretsFile;
          Slice = "system-null.slice";
          User = cfg.user;
          Group = cfg.group;
          StateDirectory = "null-gateway";
          WorkingDirectory = "/var/lib/null-gateway";
          ReadWritePaths = ["/var/lib/null-gateway"];
        };
    };
  };
}
