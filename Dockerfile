# Use a base image with the appropriate glibc version
FROM golang:bookworm AS builder

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Build the Go binary
RUN go build -o knobel-manager-service .

# Start a new smaller base image with a compatible glibc version
FROM debian:bookworm-slim

# Set the working directory
WORKDIR /

# Copy the binary from the builder stage
COPY --from=builder /app/knobel-manager-service /knobel-manager-service

EXPOSE 8080

# Command to run the service
CMD ["/knobel-manager-service"]
