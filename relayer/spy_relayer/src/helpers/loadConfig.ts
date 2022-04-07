import { config } from "dotenv";
const configFile: string = process.env.SPY_RELAY_CONFIG
  ? process.env.SPY_RELAY_CONFIG
  : ".env.sample";
console.log("loading config file [%s]", configFile);
config({ path: configFile });
export {};
