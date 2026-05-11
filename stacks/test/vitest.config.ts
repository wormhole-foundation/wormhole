import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    testTimeout: 3_600_000, // 1 hour for integration tests
    hookTimeout: 3_600_000, // 1 hour for setup/teardown
    globals: true,
    environment: "node",
    watch: false,
    pool: "forks",
    poolOptions: {
      forks: {
        singleFork: true,
      },
    },
  },
});
