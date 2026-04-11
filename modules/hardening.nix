{
  Type = "simple";
  Restart = "on-failure";
  RestartSec = 3;

  CapabilityBoundingSet = "";
  NoNewPrivileges = true;
  PrivateUsers = true;
  PrivateTmp = true;
  PrivateDevices = true;
  PrivateMounts = true;
  ProtectClock = true;
  ProtectControlGroups = true;
  ProtectHome = true;
  ProtectHostname = true;
  ProtectKernelLogs = true;
  ProtectKernelModules = true;
  ProtectKernelTunables = true;
  RestrictAddressFamilies = ["AF_INET" "AF_INET6" "AF_UNIX"];
  RestrictNamespaces = true;
  RestrictRealtime = true;
  RestrictSUIDSGID = true;
  UMask = "0077";
}
