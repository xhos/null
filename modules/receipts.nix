{
  config,
  lib,
  ...
}: let
  cfg = config.services.null;
  svcCfg = cfg.receipts;

  mkEnvFiles = svcSecretsFile:
    builtins.filter (f: f != null) [cfg.secretsFile svcSecretsFile];

  inherit (lib) types mkIf mkOption optionalAttrs;
in {
  options.services.null.receipts = {
    enable = mkOption {
      type = types.bool;
      default = true;
      description = "enable receipt OCR service";
    };

    package = mkOption {
      type = types.package;
      description = "the null-receipts package to use";
    };

    port = mkOption {
      type = types.port;
      default = 55556;
      description = "listen port";
    };

    hostname = mkOption {
      type = types.str;
      default = "127.0.0.1";
      description = "bind address";
    };

    provider = mkOption {
      type = types.enum ["ollama" "gemini"];
      default = "ollama";
      description = "vision model provider for receipt OCR";
    };

    ollama = {
      host = mkOption {
        type = types.str;
        default = "http://localhost:11434";
        description = "ollama API endpoint";
      };
      model = mkOption {
        type = types.str;
        default = "qwen2.5vl:3b";
        description = "ollama model name";
      };
    };

    gemini = {
      model = mkOption {
        type = types.str;
        default = "gemini-2.0-flash";
        description = "gemini model name";
      };
    };

    environment = mkOption {
      type = types.submodule {freeformType = types.attrsOf types.str;};
      default = {};
      description = "extra environment variables for null-receipts";
    };

    secretsFile = mkOption {
      type = types.nullOr types.str;
      default = null;
      description = "receipts-specific secrets file (e.g. GOOGLE_API_KEY for gemini)";
    };
  };

  config = mkIf (cfg.enable && svcCfg.enable) {
    assertions = [
      {
        assertion = !(svcCfg.provider == "gemini") || svcCfg.secretsFile != null;
        message = "services.null.receipts.secretsFile is required when using the gemini provider (must contain GOOGLE_API_KEY)";
      }
    ];

    services.null.receipts.environment =
      {
        LISTEN_ADDRESS = "${svcCfg.hostname}:${toString svcCfg.port}";
        LOG_LEVEL = cfg.logLevel;
        LOG_FORMAT = cfg.logFormat;
        PROVIDER = svcCfg.provider;
      }
      // optionalAttrs (svcCfg.provider == "ollama") {
        OLLAMA_HOST = svcCfg.ollama.host;
        OLLAMA_MODEL = svcCfg.ollama.model;
      }
      // optionalAttrs (svcCfg.provider == "gemini") {
        GEMINI_MODEL = svcCfg.gemini.model;
      };

    systemd.services.null-receipts = {
      description = "null: receipt OCR";
      wantedBy = ["multi-user.target"];
      after = ["network.target"];
      inherit (svcCfg) environment;
      serviceConfig =
        (import ./hardening.nix)
        // {
          ExecStart = "${svcCfg.package}/bin/server";
          EnvironmentFile = mkEnvFiles svcCfg.secretsFile;
          Slice = "system-null.slice";
          User = cfg.user;
          Group = cfg.group;
        };
    };
  };
}
