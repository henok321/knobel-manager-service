FROM golang:1.25.4-trixie AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go env -w GOPROXY=direct && go env -w GOSUMDB=off && go mod download

COPY ./spec ./spec
COPY ./Makefile ./Makefile

RUN make openapi

COPY ./api ./api
COPY ./cmd ./cmd
COPY ./internal ./internal
COPY ./pkg ./pkg
COPY ./spec ./spec

RUN CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
    go build -o knobel-manager-service \
    -a -ldflags="-s -w -extldflags '-static'" ./cmd/

FROM debian:trixie-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl && \
    apt-get clean && rm -rf /var/lib/apt/lists/* &&  \
    groupadd --gid 1001 appgroup && \
    useradd --uid 1001 --gid appgroup --create-home appuser

WORKDIR /home/appuser

COPY --from=builder /app/knobel-manager-service /home/appuser/knobel-manager-service
COPY --from=builder /app/spec /home/appuser/spec
COPY ./db_migration /home/appuser/db_migration
RUN chown -R appuser:appgroup /home/appuser/knobel-manager-service /home/appuser/spec /home/appuser/db_migration

EXPOSE 8080

ENV ENVIRONMENT=production
ENV DB_MIGRATION_DIR=/home/appuser/db_migration

USER appuser

CMD ["/home/appuser/knobel-manager-service"]
