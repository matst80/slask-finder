# syntax=docker/dockerfile:1

FROM golang:alpine AS build-stage
WORKDIR /app

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY *.go ./
COPY pkg ./pkg
COPY cmd/writer ./cmd/writer

# Build the writer binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /slask-writer ./cmd/writer

# Final stage with distroless image
FROM gcr.io/distroless/base-debian11 
WORKDIR /

# Expose the admin interface port (assuming standard HTTP port)
EXPOSE 8080

# Copy the binary from build stage
COPY --from=build-stage /slask-writer /slask-writer

# Set entrypoint
ENTRYPOINT ["/slask-writer"]