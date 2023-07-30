const fs = require("fs");
const path = require("path");

try {
  const envValuesFilePath = path.resolve(__dirname, "./cli-test-env-values");
  const envConfig = fs.readFileSync(envValuesFilePath, "utf8");

  // Loads env variables for worm CLI test environment (Jest)
  // This is needed to simulate `worm submit` API calls
  envConfig.split("\n").forEach((line) => {
    const [key, value] = line.split("=");
    process.env[key] = value;
  });
} catch (err) {
  console.error(
    `Error reading Worm CLI environtment values file at ${envValuesFilePath}. Error: ${err}`
  );
}
