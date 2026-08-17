# Engine de Build — Cloud Native Buildpacks (CNB)

A engine de build oficial do Aether é **Cloud Native Buildpacks**. O worker
roda `pack` com o builder `127.0.0.1:5000/builder:node-spa` (publicado no
registry local), que executa o lifecycle CNB completo:

```
source → detection → build → launch → OCI image (pronta para podman run)
```

## Buildpacks próprios

O builder contém **apenas buildpacks da Aether** (nenhum Paketo de terceiros):

| Buildpack | Detecta | Faz |
|---|---|---|
| `aether/spa-static` | React (CRA), Vue, Angular, Vite, Gatsby, Docusaurus, Eleventy, Next `output: export`, Svelte | instala Node via `engines.node`, `npm/pnpm/yarn install` + build, resolve o diretório de output (Angular `outputPath`/`dist/<projeto>`, Vite `outDir`), serve estático com static-web-server + SPA history fallback + cache headers + `$PORT` |
| `aether/node-server` | apps Node com servidor próprio (Express, NestJS, Next, Nuxt, Remix, SvelteKit, Astro SSR) | instala Node via `engines.node`, `npm/pnpm/yarn install` + build, processo web `start:prod` > `start` (exec direto de `node dist/...` quando possível, senão `npm run`) + `$PORT` |

Ambos seguem a especificação CNB (buildpack.toml `api=0.10`, `bin/detect`,
`bin/build`, layers, `launch.toml` com processo `web`).

## Builder

`builders/build-builder.sh` monta o builder via `podman build` (o
`pack builder create` não exporta corretamente para o podman — "duplicate
paths" no unpack) e publica no registry local `aether-registry:5000`.
`dev.sh`/`install.sh` garantem registry + builder automaticamente.

## Zero-config

```bash
# SPA (React/Vue/Angular/Vite...)
pack build my-spa -p ./frontend -B 127.0.0.1:5000/builder:node-spa --docker-host=inherit
podman run -p 8080:8080 -e PORT=8080 my-spa

# App Node com servidor (NestJS, Next, Express...)
pack build my-api -p ./api -B 127.0.0.1:5000/builder:node-spa --docker-host=inherit
podman run -p 8080:8080 -e PORT=8080 my-api
```

Sem Dockerfile, sem comando de start, sem servidor HTTP manual, sem porta
hardcoded — o processo `web` respeita `$PORT`.

## Não detectado?

Se nenhum buildpack detectar a aplicação, o deploy falha com diagnóstico
claro no log. Nesse caso a configuração manual é necessária: forneça um
Dockerfile na fonte ou use `build_type: custom` com install/build/start.

## Testes

`scripts/cnb-e2e.sh` valida 6 aplicações reais (source → CNB build → podman
run → HTTP 200): node-simple, nestjs, react-vite, vue-vite, angular e
pnpm-node.
