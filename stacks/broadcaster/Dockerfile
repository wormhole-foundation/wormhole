FROM node@sha256:101246ec9f5f803f58d6c1eb1d40477c7a99049718a5849ae78229cc6f9198b2

WORKDIR /app

# Copy package.json and install dependencies
COPY package.json package-lock.json ./
RUN npm ci

# Copy application source files
COPY tx-broadcaster.ts common.ts ./

# Default command - runs the transaction broadcaster
CMD ["npx", "tsx", "tx-broadcaster.ts"]
