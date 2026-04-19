{
  config,
  lib,
  pkgs,
  ...
}: let
  cfg = config.services.null;
  svcCfg = cfg.storage;
  garage = "${svcCfg.package}/bin/garage";
  stateDir = "/var/lib/null-garage";
  garageCfg = pkgs.writeText "null-garage.toml" ''
    metadata_dir = "${stateDir}/meta"
    data_dir = "${stateDir}/data"
    db_engine = "sqlite"
    replication_factor = 1
    rpc_bind_addr = "127.0.0.1:${toString svcCfg.rpcPort}"
    rpc_public_addr = "127.0.0.1:${toString svcCfg.rpcPort}"

    [s3_api]
    api_bind_addr = "127.0.0.1:${toString svcCfg.s3Port}"
    s3_region = "${svcCfg.region}"
    root_domain = ".s3.local"

    [admin]
    api_bind_addr = "127.0.0.1:${toString svcCfg.adminPort}"
  '';
  inherit (lib) types mkIf mkOption mkEnableOption;
in {
  options.services.null.storage = {
    enable =
      mkEnableOption "garage S3 storage for null"
      // {default = true;};

    package = mkOption {
      type = types.package;
      default = pkgs.garage;
      description = "garage package to use";
    };

    s3Port = mkOption {
      type = types.port;
      default = 55580;
      description = "S3 API port";
    };

    rpcPort = mkOption {
      type = types.port;
      default = 55581;
      description = "internal RPC port";
    };

    adminPort = mkOption {
      type = types.port;
      default = 55582;
      description = "admin API port (used for readiness check)";
    };

    region = mkOption {
      type = types.str;
      default = "garage";
      description = "S3 region name";
    };

    bucket = mkOption {
      type = types.str;
      default = "null-core";
      description = "bucket used by null-core";
    };

    keyName = mkOption {
      type = types.str;
      default = "null-core";
      description = "garage key name (matched on bucket allow)";
    };

    secretsFile = mkOption {
      type = types.nullOr types.str;
      default = null;
      example = "/run/secrets/env/null/storage";
      description = ''
        env file providing S3_ACCESS_KEY, S3_SECRET_KEY, and GARAGE_RPC_SECRET.
        Falls back to services.null.secretsFile if unset.
      '';
    };
  };

  config = mkIf (cfg.enable && svcCfg.enable) {
    assertions = [
      {
        assertion = (svcCfg.secretsFile != null) || (cfg.secretsFile != null);
        message = "services.null.storage requires secretsFile (or services.null.secretsFile) providing S3_ACCESS_KEY, S3_SECRET_KEY, GARAGE_RPC_SECRET";
      }
    ];

    users.users.null-garage = {
      isSystemUser = true;
      group = "null-garage";
      home = stateDir;
    };
    users.groups.null-garage = {};

    systemd.tmpfiles.settings.null-storage = {
      "${stateDir}" = {
        d = {
          user = "null-garage";
          group = "null-garage";
          mode = "0700";
        };
      };
      "${stateDir}/meta" = {
        d = {
          user = "null-garage";
          group = "null-garage";
          mode = "0700";
        };
      };
      "${stateDir}/data" = {
        d = {
          user = "null-garage";
          group = "null-garage";
          mode = "0700";
        };
      };
    };

    systemd.services.null-garage = {
      description = "null: garage object store";
      wantedBy = ["multi-user.target"];
      after = ["network.target"];
      serviceConfig = {
        Type = "simple";
        User = "null-garage";
        Group = "null-garage";
        StateDirectory = "null-garage";
        WorkingDirectory = stateDir;
        EnvironmentFile = [(svcCfg.secretsFile or cfg.secretsFile)];
        ExecStart = "${garage} -c ${garageCfg} server";
        Restart = "on-failure";
        RestartSec = 3;
      };
    };

    systemd.services.null-storage-setup = {
      description = "null: garage layout, key, and bucket setup";
      wantedBy = ["multi-user.target"];
      after = ["null-garage.service"];
      requires = ["null-garage.service"];
      serviceConfig = {
        Type = "oneshot";
        RemainAfterExit = true;
        User = "null-garage";
        Group = "null-garage";
        EnvironmentFile = [(svcCfg.secretsFile or cfg.secretsFile)];
      };
      script = ''
        set -eu
        G="${garage} -c ${garageCfg}"

        # wait for daemon
        for _ in $(seq 1 60); do
          $G status >/dev/null 2>&1 && break
          sleep 1
        done

        NODE_ID=$($G node id -q | cut -d@ -f1)
        if ! $G layout show 2>&1 | grep -q "$NODE_ID"; then
          $G layout assign -z dc1 -c 1G "$NODE_ID"
          VER=$($G layout show 2>&1 | sed -n 's/.*layout version: //p' | head -1)
          $G layout apply --version $((VER + 1))
        fi

        $G key import --yes -n ${svcCfg.keyName} "$S3_ACCESS_KEY" "$S3_SECRET_KEY" 2>&1 | grep -v "already exists" || true
        $G bucket create ${svcCfg.bucket} 2>&1 | grep -v "already exists" || true
        $G bucket allow --read --write --owner --key ${svcCfg.keyName} ${svcCfg.bucket}
      '';
    };
  };
}
