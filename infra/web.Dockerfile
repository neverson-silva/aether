FROM docker.io/library/node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32 AS build
WORKDIR /web
COPY frontend/elisyum_ds/package.json frontend/elisyum_ds/package-lock.json /elisyum_ds/
RUN npm --prefix /elisyum_ds ci --include=dev --ignore-scripts --no-audit --no-fund
COPY frontend/elisyum_ds /elisyum_ds
COPY frontend/web/package.json frontend/web/package-lock.json ./
RUN npm ci --legacy-peer-deps --include=dev --no-audit --no-fund
COPY frontend/web/ ./
RUN npm run build

FROM docker.io/library/nginx:alpine@sha256:db35bfc6b2951e7f8a72db5db120288c127ffaeeb4a6d4b95a26fead017d5913
COPY --from=build /web/dist /usr/share/nginx/html
COPY infra/ingress-error.html /usr/share/nginx/html/ingress-error.html
COPY infra/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 4000
