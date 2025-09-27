# syntax=docker/dockerfile:1

FROM golang:alpine AS build-stage
WORKDIR /app

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY *.go ./
COPY pkg ./pkg
COPY cmd ./cmd

# Build both binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o /slask-reader ./cmd/reader
RUN CGO_ENABLED=0 GOOS=linux go build -o /slask-writer ./cmd/writer
RUN CGO_ENABLED=0 GOOS=linux go build -o /price-watcher ./cmd/price-watcher
RUN CGO_ENABLED=0 GOOS=linux go build -o /embeddings ./cmd/embeddings

# Final stage with distroless image
FROM gcr.io/distroless/base-debian11 
WORKDIR /

# Expose port 8080 (both services use this port)
EXPOSE 8080

# Copy both binaries from build stage
COPY --from=build-stage /slask-reader /slask-reader
COPY --from=build-stage /slask-writer /slask-writer
COPY --from=build-stage /price-watcher /price-watcher
COPY --from=build-stage /embeddings /embeddings

# Default entrypoint (can be overridden during deployment)
ENTRYPOINT ["/slask-reader"]