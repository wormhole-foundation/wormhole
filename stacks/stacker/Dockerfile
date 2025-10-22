FROM node:20-bookworm AS builder

RUN apt-get update && apt-get install -y curl gettext-base jq

WORKDIR /root
COPY package.json /root/
RUN npm i

COPY stacking.ts common.ts /root/

CMD ["npx", "tsx", "/root/stacking.ts"]
