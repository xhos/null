# ariand

the main backend service for the arian project.

## development

> behold, this codebase is the pinnacle of DDD (dilly-dally driven development)

### environment

i use devenv to manage my development environment. that includes installing the dependencies and setting up the necessary tools and scripts. you can find the configuration in the [devenv.nix](./devenv.nix) file. it's not required, but highly recommended for a consistent development experience.

### sql migrations

i use [goose 🪿](https://github.com/pressly/goose) to manage sql migrations. migration files are located in [migrations](./internal/db/migrations). you can run them using the `migrate` script defined in [devenv.nix](./devenv.nix). it will automatically apply the migrations to the database.

### notes

#### dev postgres

```shell
docker run -d \
  --name arian-postgres \
  -p 5432:5432 \
  -e POSTGRES_USER=arian \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=arian \
  -v postgres_data:/var/lib/postgresql/data \
  postgres:17-alpine
```

```shell
migrate
```

```shell
docker run -p 6969:6969 -d gusaul/grpcox
```
