import { clusterApiUrl } from "@solana/web3.js";

export const MIGRATION_PROGRAM_ADDRESS =
  process.env.REACT_APP_CLUSTER === "mainnet"
    ? "whmRZnmyxdr2TkHXcZoFdtvNYRLQ5Jtbkf6ZbGkJjdk"
    : process.env.REACT_APP_CLUSTER === "testnet"
    ? ""
    : "Ex9bCdVMSfx7EzB3pgSi2R4UHwJAXvTw18rBQm5YQ8gK";

export const SOLANA_URL =
  process.env.REACT_APP_CLUSTER === "mainnet"
    ? clusterApiUrl("mainnet-beta")
    : process.env.REACT_APP_CLUSTER === "testnet"
    ? clusterApiUrl("testnet")
    : "http://localhost:8899";
