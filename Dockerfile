FROM golang:1.24.4-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./api ./api
COPY ./cmd ./cmd
COPY ./internal ./internal
COPY ./pkg ./pkg

RUN CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
    go build -o knobel-manager-service \
    -a -ldflags="-s -w -extldflags '-static'" ./cmd/

FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl && \
    apt-get clean && rm -rf /var/lib/apt/lists/* &&  \
    groupadd --gid 1001 appgroup && \
    useradd --uid 1001 --gid appgroup --create-home appuser

WORKDIR /home/appuser

COPY --from=builder /app/knobel-manager-service /home/appuser/knobel-manager-service
RUN chown appuser:appgroup /home/appuser/knobel-manager-service

EXPOSE 8080

ENV ENVIRONMENT=production

USER appuser

CMD ["/home/appuser/knobel-manager-service"]
