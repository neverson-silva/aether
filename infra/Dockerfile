FROM golang:1.26-alpine AS build
WORKDIR /src/api
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/aether ./cmd/api

FROM alpine:3.20
ARG TARGETARCH
RUN apk add --no-cache ca-certificates tzdata podman podman-compose containers-common curl git \
    && if [ "$TARGETARCH" = "arm64" ]; then PACK_ARCH=linux-arm64; else PACK_ARCH=linux; fi \
    && curl -sSLo /tmp/pack.tgz "https://github.com/buildpacks/pack/releases/download/v0.40.9/pack-v0.40.9-${PACK_ARCH}.tgz" \
    && tar -xzf /tmp/pack.tgz -C /usr/local/bin pack \
    && chmod +x /usr/local/bin/pack \
    && rm -f /tmp/pack.tgz
WORKDIR /app
COPY --from=build /out/aether /usr/local/bin/aether
COPY api/db/migrations /app/api/db/migrations
EXPOSE 8080
ENV AETHER_STATE=/var/lib/aether
ENV AETHER_API_ADDR=0.0.0.0:8080
ENV CONTAINER_HOST=unix:///run/podman/podman.sock
ENV DOCKER_HOST=unix:///run/podman/podman.sock
ENTRYPOINT ["/usr/local/bin/aether"]
