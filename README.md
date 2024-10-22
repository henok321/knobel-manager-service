# Knobel Manager Service

## Synopsis

The main goal of this project is to learn the programming language Go and to get familiar with the Go ecosystem.

This service is a small tournament manager for the dice game "Knobeln" or "Schocken". It is implemented in Java and uses
the Spring
Boot framework. The service provides a REST API to manage players, games and rounds. The service is backed by a Postgres
database.

This project is WIP and not yet finished.

## Frontend

The frontend is implemented in React and can be found [here](https://github.com/henok321/knobel-manager-app).

## Authentication

The service uses JWT for authentication that is provided by Firebase Authentication.

## Build and run

### Local

Start the Postgres database:

```bash
docker-compose up -d
```

Start the application:

```bash
go run cmd/main.go
```

### Health check

```bash
curl http://localhost:8080/health
```

## CI/CD

The project uses GitHub Actions for CI/CD. The CI workflow runs on push for main branch and for pull request and can be
found [here](.github/workflows/ci.yml). The CD workflow runs on push to the main branch and can be
found [here](.github/workflows/deploy.yml).

## Persistence

The service uses a Postgres database and `goose` for database migrations. There is a Github Action that runs the
database migrations on every push to the `main` branch. The migrations can be
found [here](.github/workflows/db-migration.yml).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

```
