FROM node@sha256:d72f64f16a391f3a7ed6b8dfd450da7da2c493fb5a5b668aa026968d252a67af

WORKDIR /app

# Copy package.json and install dependencies
COPY package.json ./
RUN npm ci

# Copy application source files
COPY tx-broadcaster.ts common.ts ./

# Default command - runs the transaction broadcaster
CMD ["npx", "tsx", "tx-broadcaster.ts"]
