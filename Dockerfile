# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26.5
ARG ALPINE_VERSION=3.23

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

RUN apk add --no-cache gcc musl-dev
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/smq ./cmd/smq

FROM alpine:${ALPINE_VERSION}

RUN apk add --no-cache ca-certificates \
    && addgroup -S smq \
    && adduser -S -G smq -h /data smq \
    && install -d -o smq -g smq /data

COPY --from=builder /out/smq /usr/local/bin/smq

USER smq
WORKDIR /data
VOLUME ["/data"]
EXPOSE 8097

ENTRYPOINT ["/usr/local/bin/smq"]
CMD ["--dbpath=/data/smq.db"]
