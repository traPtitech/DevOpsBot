FROM --platform=$BUILDPLATFORM golang:1-alpine AS builder

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

FROM alpine:3

WORKDIR /work

COPY --from=builder /dev-ops-bot ./

ENTRYPOINT ["/work/dev-ops-bot"]
