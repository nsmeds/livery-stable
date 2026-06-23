A lightweight, reliable and private file-sharing application for musicians and music producers.

## Requirements

- Go 1.22+
- Docker (for local Postgres via docker-compose)

## Environment variables

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | Postgres connection string, e.g. `postgres://user:pass@host/db?sslmode=disable` |
| `JWT_SECRET` | Yes | Secret key used to sign session tokens |

## Running locally

Start the database:

```sh
make compose-up
```

Run the server:

```sh
DATABASE_URL=postgres://postgres:postgres@localhost:5432/livery_stable?sslmode=disable \
JWT_SECRET=your-secret-here \
go run ./cmd
```

Migrations run automatically on startup.

## Make recipes

| Recipe | Description |
|---|---|
| `make test` | Start the database (if needed) and run the test suite |
| `make lint` | Run static analysis |
| `make compose-up` | Start local Postgres containers and wait until ready |
| `make compose-down` | Stop local Postgres containers |
| `make build` | Build a production Docker image |
