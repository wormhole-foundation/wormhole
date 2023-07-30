const fs = require("fs");

// Loads env variables for worm CLI test environment (Jest)
// This is needed to simulate `worm submit` API calls
const envConfig = fs.readFileSync("./cli-test-env-values", "utf8");
envConfig.split("\n").forEach((line) => {
  const [key, value] = line.split("=");
  process.env[key] = value;
});
