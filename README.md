[![CI](https://github.com/henok321/knobel-manager-service/actions/workflows/CI.yml/badge.svg)](https://github.com/henok321/knobel-manager-service/actions/workflows/CI.yml)
[![Deploy](https://github.com/henok321/knobel-manager-service/actions/workflows/deploy.yml/badge.svg)](https://github.com/henok321/knobel-manager-service/actions/workflows/deploy.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=henok321_knobel-manager-service&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=henok321_knobel-manager-service)

# Knobel Manager Service

## Synopsis

The main goal of this project is to learn the programming language Go and to get familiar with the Go ecosystem.

This service is a small tournament manager for the dice game "Knobeln" or "Schocken". The service provides a REST API
to manage players, games and rounds. The service is backed by a Postgres database.

This project is WIP and not yet finished.

## Frontend

The frontend is implemented in React and can be found [here](https://github.com/henok321/knobel-manager-app).

## Authentication

The service uses JWT for authentication that is provided by Firebase Authentication.

## Build and run

### Prerequisites

#### Linting

Install [golangci-lint](https://golangci-lint.run/welcome/install/#local-installation) and start linting:

```shell
golangci-lint run --fix --verbose 
```

To verify the schema of the `.golangci.yml` config file run:

```shell
golangci-lint config verify --verbose --config .golangci.yml
```

#### Commit hooks

To ensure a consistent code style and apply the linting rules to new code, we use [pre-commit](https://pre-commit.com/).
To install the commit hooks, run:

```shell
pre-commit install --hook-type pre-commit --hook-type pre-push
```

### Local

#### Firebase config

Generate and download a service account config file in
the [Firebase Cloud Console](https://console.firebase.google.com/u/1/project/knobel-manager-webapp/settings/serviceaccounts/adminsdk).

```shell
export FIREBASE_SECRET=$(jq -c . ./firebaseServiceAccount.json)
```

#### Start database

```shell
docker-compose up -d
export DATABASE_URL="postgres://postgres:secret@localhost:5432/postgres?sslmode=disable"
```

#### Start the application

```shell
go run cmd/main.go
```

### Health check

```shell
curl http://localhost:8080/health
```

## CI/CD

The project uses GitHub Actions for CI/CD. The CI workflow runs on push for main branch and for pull request and can be
found [here](.github/workflows/CI.yml). The CD workflow runs on push to the main branch and can be
found [here](.github/workflows/deploy.yml).

## Database migration

The project uses `goose` for database migrations. The migrations can be found [here](db/migrations). Use
the [GitHub Action](.github/workflows/db_migration.yml) to run the migrations.

## Persistence

The service uses a Postgres database and `goose` for database migrations. There is
a [GitHub Action](.github/workflows/db_migration.yml) that runs the database migrations on every push to the `main`
branch. The migrations can be found [here](.github/workflows/db_migration.yml).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

