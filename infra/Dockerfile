FROM docker.io/library/golang:1.26-alpine AS build
WORKDIR /src/api
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/aether-api ./cmd/api \
    && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/aether-worker ./cmd/worker \
    && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/aether-monitoring ./cmd/monitoring

FROM docker.io/library/alpine:3.20
ARG TARGETARCH
RUN apk add --no-cache ca-certificates tzdata curl git docker-cli docker-cli-compose \
    && if [ "$TARGETARCH" = "arm64" ]; then PACK_ARCH=linux-arm64; else PACK_ARCH=linux; fi \
    && curl -sSLo /tmp/pack.tgz "https://github.com/buildpacks/pack/releases/download/v0.40.9/pack-v0.40.9-${PACK_ARCH}.tgz" \
    && tar -xzf /tmp/pack.tgz -C /usr/local/bin pack \
    && chmod +x /usr/local/bin/pack \
    && rm -f /tmp/pack.tgz \
    && mkdir -p /var/run
WORKDIR /app
COPY --from=build /out/aether-api /usr/local/bin/aether-api
COPY --from=build /out/aether-worker /usr/local/bin/aether-worker
COPY --from=build /out/aether-monitoring /usr/local/bin/aether-monitoring
COPY api/db/migrations /app/api/db/migrations
EXPOSE 8080 8081 8082
ENV AETHER_STATE=/var/lib/aether
ENV AETHER_API_ADDR=0.0.0.0:8080
ENV DOCKER_HOST=unix:///var/run/docker.sock
ENV DOCKER_API_VERSION=1.40
ENTRYPOINT ["/usr/local/bin/aether-api"]
