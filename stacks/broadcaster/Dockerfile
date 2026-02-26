FROM node:22.16-bullseye-slim@sha256:550b434f7edc3a1875860657a3e306752358029c957280809ae6395ab296faeb

WORKDIR /app

# Copy package.json and install dependencies
COPY package.json package-lock.json ./
RUN npm ci

# Copy application source files
COPY tx-broadcaster.ts common.ts ./

# Default command - runs the transaction broadcaster
CMD ["npx", "tsx", "tx-broadcaster.ts"]
