FROM node@sha256:d72f64f16a391f3a7ed6b8dfd450da7da2c493fb5a5b668aa026968d252a67af

WORKDIR /app
COPY . .

WORKDIR /app/test
RUN npm ci
