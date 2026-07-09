A lightweight, reliable and private file-sharing application for musicians and music producers.

## Requirements

- Go 1.22+
- Docker (for local Postgres and MinIO via docker-compose)

## Environment variables

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | Postgres connection string, e.g. `postgres://user:pass@host/db?sslmode=disable` |
| `JWT_SECRET` | Yes | Secret key used to sign session tokens |
| `STORAGE_DIR` | No | Local directory for uploaded files when R2 isn't configured. Defaults to `./data/uploads` |
| `R2_ACCOUNT_ID` | No | Cloudflare account ID; enables R2 storage when set together with the other `R2_*` vars |
| `R2_ACCESS_KEY_ID` | No | R2 API access key ID |
| `R2_SECRET_ACCESS_KEY` | No | R2 API secret access key |
| `R2_BUCKET` | No | R2 bucket name |

## Running locally

Start the database and local object storage:

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
| `make compose-up` | Start local Postgres and MinIO containers and wait until ready |
| `make compose-down` | Stop local Postgres and MinIO containers |
| `make build` | Build a production Docker image |
