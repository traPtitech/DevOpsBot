FROM --platform=$BUILDPLATFORM golang:1.20-alpine AS builder

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
    go build -o /dev-ops-bot -ldflags="-s -w -X main.version=$VERSION" .

FROM --platform=$BUILDPLATFORM golang:1.20-alpine AS installer

ENV CGO_ENABLED 0
ARG TARGETOS
ARG TARGETARCH
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

RUN apk add --no-cache wget

RUN wget https://github.com/mikefarah/yq/releases/latest/download/yq_"$TARGETOS"_"$TARGETARCH" -O /yq && \
    chmod +x /yq

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go install sigs.k8s.io/kustomize/kustomize/v5@latest
# keep output directory the same between platforms; workaround for https://github.com/golang/go/issues/57485
RUN cp /go/bin/kustomize /kustomize || cp /go/bin/"$GOOS"_"$GOARCH"/kustomize /kustomize

FROM alpine:3

WORKDIR /work

# Install commands for deploy scripts
RUN apk add --no-cache git openssh curl
RUN mkdir -p /root/.ssh && ssh-keyscan github.com >> /root/.ssh/known_hosts

COPY --from=installer /yq /usr/local/bin/
COPY --from=installer /kustomize /usr/local/bin/

COPY --from=builder /dev-ops-bot ./

ENTRYPOINT ["/work/dev-ops-bot"]
