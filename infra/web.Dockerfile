FROM node:20-alpine AS build
WORKDIR /web
COPY frontend/web/package.json frontend/web/package-lock.json ./
RUN npm ci
COPY frontend/web/ ./
RUN npm run build

FROM nginx:alpine
COPY --from=build /web/dist /usr/share/nginx/html
COPY infra/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 4000
