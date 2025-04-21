import type { Config } from "@jest/types";

const config: Config.InitialOptions = {
  preset: "ts-jest",
  testEnvironment: "node",
  // Disable worker threads to avoid BigInt serialization issues
  maxWorkers: 1,
};
export default config;
