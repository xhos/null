{pkgs, ...}: {
  packages = with pkgs; [
    # grpc stuff
    grpcurl
    buf

    # sql stuff
    goose
    sqlc

    # dev
    air
  ];

  languages.go.enable = true;

  scripts.run.exec = ''
    go run cmd/main.go
  '';

  scripts.build.exec = ''
    BUILD_TIME="$(date -u '+%Y-%m-%d %H:%M:%S UTC')"
    GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")"
    GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")"
    
    echo "Building with version info:"
    echo "  Build Time: $BUILD_TIME"
    echo "  Git Commit: $GIT_COMMIT" 
    echo "  Git Branch: $GIT_BRANCH"
    
    go build -ldflags "-X 'ariand/internal/version.BuildTime=$BUILD_TIME' -X 'ariand/internal/version.GitCommit=$GIT_COMMIT' -X 'ariand/internal/version.GitBranch=$GIT_BRANCH'" -o ariand cmd/main.go
  '';

  scripts.fmt.exec = ''
    go fmt ./...
  '';

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
    git commit -m "⬆️ bump proto files"
    git push
  '';

  scripts.sqlc-regen.exec = ''
    rm -rf internal/db/sqlc/; sqlc generate
  '';

  scripts.regen.exec = ''
    rm -rf internal/db/sqlc/; sqlc generate; rm -rf internal/gen/; buf generate
  '';

  scripts.cover.exec = "go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html";

  git-hooks.hooks = {
    gotest.enable = true;
    gofmt.enable = true;
    govet.enable = true;
  };

  dotenv.enable = true;
}
