# Knobel Manager Service

[![CI](https://github.com/henok321/knobel-manager-service/actions/workflows/CI.yml/badge.svg)](https://github.com/henok321/knobel-manager-service/actions/workflows/CI.yml)
[![Deploy](https://github.com/henok321/knobel-manager-service/actions/workflows/deploy.yml/badge.svg)](https://github.com/henok321/knobel-manager-service/actions/workflows/deploy.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=henok321_knobel-manager-service&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=henok321_knobel-manager-service)

- [Knobel Manager Service](#knobel-manager-service)
  - [Synopsis](#synopsis)
  - [Frontend](#frontend)
  - [Authentication](#authentication)
  - [CI/CD](#cicd)
  - [Database Migration](#database-migration)
  - [Prerequisites](#prerequisites)
  - [Setup and Development](#setup-and-development)
    - [Obtain Firebase Service Account Credentials](#obtain-firebase-service-account-credentials)
    - [Run Setup](#run-setup)
    - [Start the Application](#start-the-application)
    - [Build and run binary](#build-and-run-binary)
      - [Build](#build)
      - [Run](#run)
    - [Health Check](#health-check)
    - [Makefile targets](#makefile-targets)
  - [License](#license)

## Synopsis

The main goal of this project is to learn the Go programming language and become familiar with its ecosystem.

This service is a small tournament manager for the dice game "Knobeln" or "Schocken." It provides a REST API to manage
players, games, and rounds, backed by a PostgreSQL database.

This project is a work in progress and not yet finished.

## Frontend

The frontend is implemented in React: [knobel-manager-app](https://github.com/henok321/knobel-manager-app).

## Authentication

The service uses JWT for authentication, provided by Firebase Authentication.

## CI/CD

The project uses GitHub Actions for CI/CD. The CI workflow runs on push for the main branch and for pull requests: [CI](.github/workflows/CI.yml).
The CD workflow runs on push to the main branch: [CD](.github/workflows/deploy.yml).

## Database Migration

The project uses `goose` for database migrations: [DB Migration](db/migrations). Use
the [GitHub Action](.github/workflows/db_migration.yml) to run the migrations.

## Prerequisites

Ensure the following dependencies are installed:

- [Go](https://go.dev/doc/install)
- [Docker](https://docs.docker.com/get-docker/)
- [pre-commit](https://pre-commit.com/) (`pip install pre-commit`)
- [Goose](https://github.com/pressly/goose) (`go install github.com/pressly/goose/v3/cmd/goose@latest`)

## Setup and Development

### Obtain Firebase Service Account Credentials

Generate and download a service account config file in
the [Firebase Cloud Console](https://console.firebase.google.com/u/1/project/knobel-manager-webapp/settings/serviceaccounts/adminsdk).

The file should be named `firebase-credentials.json` and placed in the root directory of the project.

### Run Setup

Execute the following command to set up the project:

```sh
make setup
```

This command will:

- Install commit hooks.
- Start the local database.
- Run database migrations.
- Create a `.env` file with necessary environment variables.

Reset database:

```shell
make reset
```

### Start the Application

To run the application locally:

```shell
set -o allexport
source .env
set +o allexport
go run cmd/main.go
```

### Build and run binary

#### Build

```shell
make build
```

#### Run

```shell
set -o allexport
source .env
set +o allexport
./knobel-manager-service
```

### Health Check

Verify the service is running:

```sh
curl http://localhost:8080/health
```

### Makefile targets

For more information on available Makefile targets, run:

```shell
make help
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
