FROM --platform=$BUILDPLATFORM golang:1-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS builder

ENV CGO_ENABLED 0

WORKDIR /work

COPY go.* .
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG VERSION=snapshot
ARG TARGETOS
ARG TARGETARCH
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go build -o /dev-ops-bot -ldflags="-s -w -X github.com/traPtitech/DevOpsBot/pkg/utils.version=$VERSION" .

FROM alpine:3@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

WORKDIR /work

COPY --from=builder /dev-ops-bot ./

ENTRYPOINT ["/work/dev-ops-bot"]
