const fs = require("fs");

// Loads env variables for test environment (Jest)
const envConfig = fs.readFileSync(".env.test", "utf8");
envConfig.split("\n").forEach((line) => {
  const [key, value] = line.split("=");
  process.env[key] = value;
});
