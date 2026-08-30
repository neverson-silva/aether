# Build Engine — Cloud Native Buildpacks (CNB)

The Aether build engine is **Cloud Native Buildpacks**. The worker runs `pack`
with the `127.0.0.1:1500/builder:node-spa` builder from the local registry:

```
source → detection → build → launch → OCI image (ready for Docker Engine)
```

## Aether buildpacks

O builder contém **apenas buildpacks da Aether** (nenhum Paketo de terceiros):

| Buildpack | Detecta | Faz |
|---|---|---|
| `aether/spa-static` | React (CRA), Vue, Angular, Vite, Gatsby, Docusaurus, Eleventy, Next `output: export`, Svelte, Astro, **Lume e SSGs de Deno** | runtime automático (Node/Bun/Deno), install + build, resolve output (Vite `outDir`, Angular `outputPath`, Lume `_site`/`--dest`), serve com static-web-server + SPA fallback + `$PORT` |
| `aether/node-server` | apps com servidor: Node (Express, NestJS, Next, Nuxt, Remix, SvelteKit, Astro SSR), **Bun** (Hono, Elysia, Bun.serve), **Deno** (Fresh, Oak, Hono/Deno) | runtime **Deno > Bun > Node** (arch-aware), install + build, `start:prod` > `start` (exec direto ou `bun`/`deno run`) + `$PORT` |
| `aether/php-server` | composer.json (Laravel, Symfony, Slim, CakePHP, Yii, Laminas), WordPress (`wp-config.php`), `artisan`, `index.php` | PHP **8.x estático** (static-php.dev, 8.0–8.5) ou **7.4 via PPA ondrej**; composer install; `php -S` com router (`public/index.php` p/ Laravel/Symfony) + `$PORT` |
| `aether/ruby-server` | Gemfile (Rails, Sinatra, Rack), `config.ru`, `*.gemspec`, Rakefile | Ruby (apt LTS) + bundler; rails assets precompile; **puma** > rackup; `$PORT` |
| `aether/go-server` | `go.mod` | Go (versão do go.mod, arch-aware), `CGO_ENABLED=0 go build` (root ou `./cmd/*`), binário estático + `$PORT` |
| `aether/rust-server` | `Cargo.toml` | toolchain stable (static.rust-lang.org), `cargo build --release`, binário de `target/release` + `$PORT` |
| `aether/dotnet-server` | `*.csproj`/`*.sln`/`global.json` | SDK via dotnet-install.sh (channel do global.json, x64/arm64), `dotnet publish`, exec `dotnet <assembly>.dll` + `ASPNETCORE_URLS` |
| `aether/jvm-server` | `pom.xml` (Maven), `build.gradle(.kts)`/`settings.gradle`/`gradlew` (Gradle — Java/Kotlin/Groovy) | JDK **Temurin** (release do pom/gradle) ou **GraalVM** quando detecta native (Quarkus/Spring/Micronaut/`native-image`); Maven 3.9/Gradle (wrapper ou dist); `mvn package`/`gradle build`, native via `-Pnative`/`nativeCompile` com fallback jar; exec binário nativo ou `java -jar`/`-cp` |

Todos seguem a especificação CNB (buildpack.toml `api=0.10`, `bin/detect`, `bin/build`, layers, `launch.toml` com processo `web`).

**Ordem recomendada no builder**: `php-server` → `ruby-server` → `dotnet-server` → `go-server` → `rust-server` → `jvm-server` → `node-server` → `spa-static` (buildpacks com marcadores fortes primeiro; `node-server` rejeita apps com `package.json`+`start` dev, e `spa-static` pega o resto estático).

Ambos seguem a especificação CNB (buildpack.toml `api=0.10`, `bin/detect`,
`bin/build`, layers, `launch.toml` com processo `web`).

## Builder lifecycle

`builders/build-builder.sh` builds the builder with Docker Engine and publishes
it to the local registry at `127.0.0.1:1500/builder:node-spa`.

O script:
1. garante o registry local (`aether-registry`, `--network host` — visível aos
   lifecycle containers via `--docker-host=inherit`);
2. baixa o **lifecycle CNB** (padrão `0.21.15`, arch-aware);
3. copia **todos os buildpacks** (`aether/*`) para `/cnb/buildpacks` com as
   versões dos `buildpack.toml`;
4. gera `order.toml` (ordem de detecção: php → ruby → dotnet → go → rust →
   jvm → node → spa) e `stack.toml` (run/build image `ubuntu:24.04`);
5. `docker build` + `docker push`.

```bash
./builders/build-builder.sh          # usa o uname -m do host
pack build my-app -B 127.0.0.1:1500/builder:node-spa --docker-host=inherit
```

The development and installation flows prepare the registry and builder before
starting the deployment worker.

## Zero configuration

```bash
# SPA (React/Vue/Angular/Vite...)
pack build my-spa -p ./frontend -B 127.0.0.1:1500/builder:node-spa --docker-host=inherit
docker run -p 8080:8080 -e PORT=8080 my-spa

# App Node com servidor (NestJS, Next, Express...)
pack build my-api -p ./api -B 127.0.0.1:1500/builder:node-spa --docker-host=inherit
docker run -p 8080:8080 -e PORT=8080 my-api

# App Bun (Hono/Elysia/Bun.serve — bun.lockb, bun.lock ou bunfig.toml)
pack build my-bun-api -p ./api -B 127.0.0.1:1500/builder:node-spa --docker-host=inherit
docker run -p 8080:8080 -e PORT=8080 my-bun-api

# App Deno (Fresh/Oak/Hono-Deno — deno.json/deno.lock/import_map.json)
pack build my-deno-api -p ./api -B 127.0.0.1:1500/builder:node-spa --docker-host=inherit
docker run -p 8080:8080 -e PORT=8080 my-deno-api

# Site estático em Deno (Lume — _config.ts)
pack build my-site -p ./site -B 127.0.0.1:1500/builder:node-spa --docker-host=inherit
docker run -p 8080:8080 -e PORT=8080 my-site
```

No Dockerfile, start command, manual HTTP server, or hardcoded port is needed.
The `web` process respects `$PORT`. Runtime selection is automatic: **Deno**
(`deno.json`/`deno.lock`/`import_map.json`) > **Bun**
(`bun.lockb`/`bun.lock`/`bunfig.toml`/`packageManager bun@`/`engines.bun`) >
**Node** (`engines.node`). Binaries are architecture-aware.

## Undetected applications

If no buildpack detects the application, deployment fails with a clear log
diagnostic. Provide a Dockerfile or use `build_type: custom` with
install/build/start.

## Tests

Validation uses `bin/detect` and `bin/build` in an Ubuntu 24.04 glibc CNB
environment. Heavy builds should be validated with
`./builders/build-builder.sh` and `pack build`:

| Cenário | Buildpack |
|---|---|
| Express (start `node …`) | node-server |
| Bun server (`bun.lockb`/`bunfig.toml`/`packageManager: bun@`) | node-server |
| Deno server (`deno.json` task `start`/`deno run`) | node-server |
| Vite/Angular/Next-export (node ou bun) | spa-static |
| Lume/SSG Deno (`_config.ts`) | spa-static |
| Laravel/Symfony/Slim (composer.json) · WordPress · `index.php` | php-server |
| Rails/Sinatra (Gemfile) · config.ru | ruby-server |
| go.mod | go-server |
| Cargo.toml | rust-server |
| .csproj/.sln/global.json | dotnet-server |
| pom.xml (Maven) · build.gradle(.kts) (Java/Kotlin/Groovy) | jvm-server |
| Quarkus/Spring/Micronaut + native | jvm-server (GraalVM) |

Runtimes escolhidos automaticamente (arch-aware x64/arm64): node-server **Deno > Bun > Node**;
spa-static idem; php **8.x estático / 7.4 PPA**; jvm **GraalVM (native) > Temurin**.

## Determinism and performance

**Determinismo**
- Installs **travados**: `npm ci`/`pnpm --frozen-lockfile`/`bun --frozen-lockfile`/`bundle --deployment`/`cargo --locked` (com fallback documentado quando não há lockfile).
- Versões resolvidas de fontes canônicas no build (Go por `go.mod`, Rust do `[pkg.rust]` stable, .NET pelo `global.json`, PHP pela major do `require.php`, Java pelo `<release>`/toolchain).
- Flags de build fixas e reprodutíveis (abaixo).

**Perfis de framework (build + start específicos de produção)**
- **Node**: Next.js (standalone → `node .next/standalone/server.js`; senão `next start`), Nuxt (`.output/server/index.mjs`), SvelteKit (`node build/index.js`), Astro SSR (`dist/server/entry.mjs`), Remix (`remix-serve`) — todos com `NODE_ENV=production` + `PORT`.
- **Bun/Deno**: **compile nativo** (`bun build --compile` / `deno compile`) → binário único nativo (startup instantâneo); fallback para `bun run`/`deno task start`.
- **PHP**: composer `--optimize-autoloader --classmap-authoritative`; **Laravel** `config/route/view/event:cache`; **Symfony** `cache:warmup`; **OPcache + JIT** (`opcache.jit=tracing`, `validate_timestamps=0`) via `php.ini` da layer.
- **Ruby**: `BUNDLE_WITHOUT=development:test`; Rails `assets:precompile`; **puma `-e production`** com `RAILS_ENV`/`SECRET_KEY_BASE` gerado se ausente.
- **Go**: `CGO_ENABLED=0`, `-trimpath -buildvcs=false`, `-ldflags "-s -w"` (binário estático mínimo).
- **Rust**: `cargo build --release` com **LTO thin + codegen-units=1 + strip + panic=abort** (~332KB em app hello-world).
- **.NET**: publish **self-contained + ReadyToRun + Deterministic** (warm start, sem dependência de runtime no launch).
- **JVM**: `JAVA_TOOL_OPTIONS="-XX:+UseContainerSupport -XX:MaxRAMPercentage=75 -XX:+ExitOnOutOfMemoryError"`; GraalVM native via `-Pnative`/`nativeCompile` com fallback jar.

**Binários arch-aware (x64/arm64) em todos os runtimes** — o build roda nativo na arquitetura do host.
