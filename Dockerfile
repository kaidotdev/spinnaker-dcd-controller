# syntax=docker/dockerfile:experimental

FROM golang:1.22-bullseye AS builder

ENV CGO_ENABLED 0

WORKDIR /build/

COPY go.mod go.sum /build/
RUN --mount=type=cache,target=/root/go/pkg/mod go mod download

COPY main.go /build/main.go
COPY api /build/api
COPY controllers /build/controllers

RUN --mount=type=cache,target=/root/.cache/go-build go build -trimpath -o /usr/local/bin/main -ldflags="-s -w" /build/main.go

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /usr/local/bin/main /usr/local/bin/main

USER 65532

ENTRYPOINT ["/usr/local/bin/main"]
