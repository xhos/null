{
  config,
  lib,
  pkgs,
  ...
}: let
  cfg = config.services.null;
  svcCfg = cfg.emailParser;

  mkEnvFiles = svcSecretsFile:
    builtins.filter (f: f != null) [cfg.secretsFile svcSecretsFile];

  inherit (lib) types mkIf mkOption optionalAttrs;
in {
  options.services.null.emailParser = {
    enable = mkOption {
      type = types.bool;
      default = false;
      description = "enable email parser SMTP ingest service";
    };

    package = mkOption {
      type = types.package;
      description = "the null-email-parser package to use";
    };

    smtpPort = mkOption {
      type = types.port;
      default = 2525;
      description = "SMTP listen port";
    };

    grpcPort = mkOption {
      type = types.port;
      default = 55557;
      description = "gRPC health check port";
    };

    hostname = mkOption {
      type = types.str;
      default = "127.0.0.1";
      description = "bind address";
    };

    domain = mkOption {
      type = types.str;
      example = "mail.finances.example.com";
      description = "email domain for receiving transaction emails";
    };

    tls = {
      certFile = mkOption {
        type = types.nullOr types.path;
        default = null;
        description = "path to TLS certificate (fullchain.pem)";
      };

      keyFile = mkOption {
        type = types.nullOr types.path;
        default = null;
        description = "path to TLS private key (privkey.pem)";
      };

      disableRequired = mkOption {
        type = types.bool;
        default = false;
        description = "allow connections without TLS, unsafe — only for development";
      };
    };

    environment = mkOption {
      type = types.submodule {freeformType = types.attrsOf types.str;};
      default = {};
      description = "extra environment variables for null-email-parser";
    };

    secretsFile = mkOption {
      type = types.nullOr types.str;
      default = null;
      description = "email parser secrets file";
    };
  };

  config = mkIf (cfg.enable && svcCfg.enable) {
    services.null.emailParser.environment =
      {
        NULL_CORE_URL = "${cfg.core.hostname}:${toString cfg.core.port}";
        DOMAIN = svcCfg.domain;
        SMTP_PORT = "${svcCfg.hostname}:${toString svcCfg.smtpPort}";
        GRPC_PORT = "${svcCfg.hostname}:${toString svcCfg.grpcPort}";
        LOG_LEVEL = cfg.logLevel;
        LOG_FORMAT = cfg.logFormat;
      }
      // optionalAttrs svcCfg.tls.disableRequired {
        UNSAFE_DISABLE_TLS_REQUIRED = "true";
      }
      // optionalAttrs (svcCfg.tls.certFile != null) {
        TLS_CERT = toString svcCfg.tls.certFile;
        TLS_KEY = toString svcCfg.tls.keyFile;
      };

    systemd.services.null-email-parser = {
      description = "null: email parser";
      wantedBy = ["multi-user.target"];
      after = ["network.target" "null-core.service"];
      requires = ["null-core.service"];
      inherit (svcCfg) environment;
      serviceConfig =
        (import ./hardening.nix)
        // {
          ExecStartPre = "${pkgs.bash}/bin/sh -c 'until ${pkgs.netcat}/bin/nc -z ${cfg.core.hostname} ${toString cfg.core.port}; do sleep 1; done'";
          ExecStart = "${svcCfg.package}/bin/server";
          EnvironmentFile = mkEnvFiles svcCfg.secretsFile;
          Slice = "system-null.slice";
          User = cfg.user;
          Group = cfg.group;
          StateDirectory = "null-email-parser";
          WorkingDirectory = "/var/lib/null-email-parser";
          ReadWritePaths = ["/var/lib/null-email-parser"];
        };
    };
  };
}
