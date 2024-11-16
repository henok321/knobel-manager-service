# Use a base image with the appropriate glibc version
FROM golang:bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o knobel-manager-service -a -ldflags="-s -w" -installsuffix cgo ./cmd/

FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /

COPY --from=builder /app/knobel-manager-service /knobel-manager-service

EXPOSE 8080

CMD ["/knobel-manager-service"]
