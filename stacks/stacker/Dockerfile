FROM node@sha256:101246ec9f5f803f58d6c1eb1d40477c7a99049718a5849ae78229cc6f9198b2 AS builder

RUN apt-get update && apt-get install -y curl gettext-base jq

WORKDIR /root
COPY package.json package-lock.json /root/
RUN npm i

COPY stacking.ts common.ts /root/

CMD ["npx", "tsx", "/root/stacking.ts"]
