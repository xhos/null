{pkgs, ...}: {
  packages = with pkgs; [
    grpcurl
    buf

    goose
    sqlc

    air
    golangci-lint
  ];

  languages.go.enable = true;

  scripts.run.exec = ''
    go run cmd/main.go
  '';

  scripts.fmt.exec = "golangci-lint fmt";

  scripts.tstv.exec = ''
    CLICOLOR_FORCE=1 go test ./... -v
  '';

  scripts.tst.exec = ''
    go test ./...
  '';

  scripts.migrate.exec = ''
    goose -dir internal/db/migrations postgres "$DATABASE_URL" up
  '';

  scripts.bump-proto.exec = ''
    git -C proto fetch origin
    git -C proto checkout main
    git -C proto pull --ff-only
    git add proto
    git commit -m "chore: bump proto files"
    git push
  '';

  scripts.regen.exec = ''
    rm -rf internal/db/sqlc/; sqlc generate; rm -rf internal/gen/; buf generate
  '';

  scripts.cover.exec = "go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html";

  git-hooks.hooks = {
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

  dotenv.enable = true;
}
