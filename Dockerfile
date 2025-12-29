FROM --platform=$BUILDPLATFORM golang:1.25-trixie AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./

ENV GO111MODULE=on \
    GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org \
    CGO_ENABLED=0

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY openapi ./openapi
COPY ./Makefile ./Makefile
RUN --mount=type=cache,target=/go/pkg/mod \
    make openapi

COPY './api' './api'
COPY './cmd' './cmd'
COPY './pkg' './pkg'

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod tidy

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -o knobel-manager-service \
    -a -ldflags="-s -w -extldflags '-static'" ./cmd/

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
