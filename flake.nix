{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    git-hooks.url = "github:cachix/git-hooks.nix";
    git-hooks.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = {
    self,
    nixpkgs,
    git-hooks,
  }: let
    forAllSystems = f:
      nixpkgs.lib.genAttrs
      ["x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin"]
      (system: f nixpkgs.legacyPackages.${system});
  in {
    checks = forAllSystems (pkgs: {
      pre-commit = git-hooks.lib.${pkgs.system}.run {
        src = ./.;
        hooks = {
          gotest.enable = true;
          govet.enable = true;
          alejandra.enable = true;
          golangci-lint = {
            enable = true;
            name = "golangci-lint";
            entry = "${pkgs.golangci-lint}/bin/golangci-lint fmt";
            types = ["go"];
          };
        };
      };
    });

    devShells = forAllSystems (pkgs: {
      default = pkgs.mkShell {
        packages = with pkgs; [
          go

          grpcurl
          buf
          goose
          sqlc
          air
          golangci-lint

          (writeShellScriptBin "run" ''
            go run cmd/main.go
          '')

          (writeShellScriptBin "fmt" ''
            ${golangci-lint}/bin/golangci-lint fmt
          '')

          (writeShellScriptBin "tstv" ''
            CLICOLOR_FORCE=1 go test ./... -v
          '')

          (writeShellScriptBin "tst" ''
            go test ./...
          '')

          (writeShellScriptBin "migrate" ''
            ${goose}/bin/goose -dir internal/db/migrations postgres "$DATABASE_URL" up
          '')

          (writeShellScriptBin "bump-proto" ''
            git -C proto fetch origin
            git -C proto checkout main
            git -C proto pull --ff-only
            git add proto
            git commit -m "chore: bump proto files"
            git push
          '')

          (writeShellScriptBin "regen" ''
            rm -rf internal/db/sqlc/
            ${sqlc}/bin/sqlc generate
            rm -rf internal/gen/
            ${buf}/bin/buf generate
          '')

          (writeShellScriptBin "cover" ''
            go test -coverprofile=coverage.out ./... && \
            go tool cover -html=coverage.out -o coverage.html
          '')
        ];

        shellHook = self.checks.${pkgs.system}.pre-commit.shellHook;
      };
    });

    formatter = forAllSystems (pkgs: pkgs.alejandra);
  };
}
