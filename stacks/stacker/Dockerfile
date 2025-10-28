FROM node@sha256:d72f64f16a391f3a7ed6b8dfd450da7da2c493fb5a5b668aa026968d252a67af AS builder

RUN apt-get update && apt-get install -y curl gettext-base jq

WORKDIR /root
COPY package.json /root/
RUN npm i

COPY stacking.ts common.ts /root/

CMD ["npx", "tsx", "/root/stacking.ts"]
