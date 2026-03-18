FROM node:22.16-bullseye-slim@sha256:550b434f7edc3a1875860657a3e306752358029c957280809ae6395ab296faeb AS builder

RUN apt-get update && apt-get install -y curl gettext-base jq

WORKDIR /root
COPY package.json package-lock.json /root/
RUN npm ci

COPY stacking.ts common.ts /root/

CMD ["npx", "tsx", "/root/stacking.ts"]
