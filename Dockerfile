FROM --platform=$BUILDPLATFORM golang:1.25-trixie AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./

ENV GO111MODULE=on \
    GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY './gen' './gen'
COPY './api' './api'
COPY './cmd' './cmd'
COPY './pkg' './pkg'
COPY './openapi' './openapi'

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod tidy

RUN --mount=type=cache,target=/root/.cache/go-build \
     GOOS="$TARGETOS" GOARCH="$TARGETARCH" make build

FROM debian:trixie-slim

ENV DEBIAN_FRONTEND=noninteractive

# hadolint ignore=DL3008
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl && \
    apt-get clean && rm -rf /var/lib/apt/lists/* &&  \
    groupadd --gid 1001 appgroup && \
    useradd --uid 1001 --gid appgroup --create-home appuser

WORKDIR /home/appuser

COPY --from=builder /app/knobel-manager-service /home/appuser/knobel-manager-service
COPY --from=builder /app/openapi /home/appuser/openapi
COPY ./db_migration /home/appuser/db_migration

RUN chown -R appuser:appgroup /home/appuser/knobel-manager-service /home/appuser/openapi /home/appuser/db_migration

EXPOSE 8080

ENV DB_MIGRATION_DIR=/home/appuser/db_migration

USER appuser

CMD ["/home/appuser/knobel-manager-service"]
