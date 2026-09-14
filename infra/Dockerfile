FROM docker.io/library/golang:1.26-alpine@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468 AS build
WORKDIR /src/api
COPY api/go.mod api/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download \
    && mkdir -p /out/bin \
    && GOBIN=/out/bin go install github.com/sigstore/cosign/v2/cmd/cosign@v2.4.1
COPY api/ ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/aether-api ./cmd/api \
    && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/aether-worker ./cmd/worker \
    && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/aether-monitoring ./cmd/monitoring

FROM docker.io/library/alpine:3.20@sha256:d9e853e87e55526f6b2917df91a2115c36dd7c696a35be12163d44e6e2a4b6bc
ARG TARGETARCH
RUN apk add --no-cache ca-certificates tzdata curl git docker-cli docker-cli-compose \
    && if [ "$TARGETARCH" = "arm64" ]; then PACK_ARCH=linux-arm64; else PACK_ARCH=linux; fi \
    && curl -sSLo /tmp/pack.tgz "https://github.com/buildpacks/pack/releases/download/v0.40.9/pack-v0.40.9-${PACK_ARCH}.tgz" \
    && curl -sSLo /tmp/pack.tgz.sha256 "https://github.com/buildpacks/pack/releases/download/v0.40.9/pack-v0.40.9-${PACK_ARCH}.tgz.sha256" \
    && cd /tmp \
    && sed -i "s#pack-v0.40.9-${PACK_ARCH}.tgz#pack.tgz#" pack.tgz.sha256 \
    && sha256sum -c pack.tgz.sha256 \
    && cd / \
    && tar -xzf /tmp/pack.tgz -C /usr/local/bin pack \
    && chmod +x /usr/local/bin/pack \
    && rm -f /tmp/pack.tgz /tmp/pack.tgz.sha256 \
    && addgroup -S aether \
    && adduser -S -G aether -h /home/aether aether \
    && mkdir -p /var/run /home/aether/.cache/pack /var/lib/aether \
    && chown -R aether:aether /home/aether /var/lib/aether
WORKDIR /app
COPY --from=build /out/aether-api /usr/local/bin/aether-api
COPY --from=build /out/aether-worker /usr/local/bin/aether-worker
COPY --from=build /out/aether-monitoring /usr/local/bin/aether-monitoring
COPY --from=build /out/bin/cosign /usr/local/bin/cosign
COPY api/db/migrations /app/api/db/migrations
EXPOSE 8080 8081 8082
ENV AETHER_STATE=/var/lib/aether
ENV AETHER_API_ADDR=0.0.0.0:8080
ENV DOCKER_HOST=unix:///var/run/docker.sock
ENV DOCKER_API_VERSION=1.40
ENV HOME=/home/aether
USER aether
ENTRYPOINT ["/usr/local/bin/aether-api"]
