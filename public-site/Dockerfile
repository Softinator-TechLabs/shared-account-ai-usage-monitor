FROM node:22-bookworm-slim AS build
WORKDIR /src
COPY package*.json ./
RUN npm ci
COPY public-site ./public-site
COPY internal/web/assets/GEIST-LICENSE.txt ./internal/web/assets/GEIST-LICENSE.txt
RUN npm run build:public-site

FROM nginxinc/nginx-unprivileged:1.28-alpine
COPY public-site/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/public-site/dist/website /usr/share/nginx/html
COPY LICENSE THIRD_PARTY_NOTICES.md /usr/share/licenses/shared-ai-usage/
COPY internal/web/assets/GEIST-LICENSE.txt /usr/share/licenses/shared-ai-usage/
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
