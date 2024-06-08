# syntax=docker/dockerfile:1

FROM golang:alpine AS build-stage
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /go-crapmail

FROM gcr.io/distroless/base-debian11 
WORKDIR /

EXPOSE 25
EXPOSE 8080

COPY *.html /
COPY --from=build-stage /go-crapmail /go-crapmail
ENTRYPOINT ["/go-crapmail"]