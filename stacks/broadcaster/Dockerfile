FROM node:20-bookworm

WORKDIR /app

# Copy package.json and install dependencies
COPY package.json ./
RUN npm ci

# Copy application source files
COPY tx-broadcaster.ts common.ts ./

# Default command - runs the transaction broadcaster
CMD ["npx", "tsx", "tx-broadcaster.ts"]
