{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";

    null-core.url = "github:xhos/null-core";
    null-core.inputs.nixpkgs.follows = "nixpkgs";

    null-gateway.url = "github:xhos/null-gateway";
    null-gateway.inputs.nixpkgs.follows = "nixpkgs";

    null-web.url = "github:xhos/arian-web";
    null-web.inputs.nixpkgs.follows = "nixpkgs";

    null-receipts.url = "github:xhos/arian-receipts";
    null-receipts.inputs.nixpkgs.follows = "nixpkgs";

    null-email-parser.url = "github:xhos/arian-email-parser";
    null-email-parser.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = {nixpkgs, ...} @ inputs: {
    nixosModules.default = {pkgs, ...}: {
      imports = [
        ./modules/shared.nix
        ./modules/core.nix
        ./modules/gateway.nix
        ./modules/web.nix
        ./modules/receipts.nix
        ./modules/email-parser.nix
        ./modules/storage.nix
      ];

      services.null.core.package = inputs.null-core.packages.${pkgs.system}.default;
      services.null.gateway.package = inputs.null-gateway.packages.${pkgs.system}.default;
      services.null.web.package = inputs.null-web.packages.${pkgs.system}.default;
      services.null.receipts.package = inputs.null-receipts.packages.${pkgs.system}.default;
      services.null.emailParser.package = inputs.null-email-parser.packages.${pkgs.system}.default;
    };
  };
}
