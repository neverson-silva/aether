FROM node:22-alpine AS build
WORKDIR /web
COPY frontend/aether_ds/package.json frontend/aether_ds/package-lock.json /aether_ds/
RUN npm --prefix /aether_ds ci --ignore-scripts --no-audit --no-fund
COPY frontend/web/package.json frontend/web/package-lock.json ./
RUN npm ci
COPY frontend/aether_ds /aether_ds
COPY frontend/web/ ./
RUN npm run build

FROM nginx:alpine
COPY --from=build /web/dist /usr/share/nginx/html
COPY infra/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 4000
