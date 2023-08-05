const fs = require("fs");
const path = require("path");

try {
  const envValuesFilePath = path.resolve(__dirname, "./worm-submit-env-values");
  const envConfig = fs.readFileSync(envValuesFilePath, "utf8");

  // We store these env variables into a specific object to isolate them from global 'process.env' values
  // This is needed to simulate `worm submit` API calls
  global.wormSubmitCliEnv = {};

  // Loads env variables for worm CLI test environment (Jest)
  envConfig.split("\n").forEach((line) => {
    const [key, value] = line.split("=");
    global.wormSubmitCliEnv[key] = value;
  });
} catch (err) {
  console.error(
    `Error reading Worm CLI environtment values file at ${envValuesFilePath}. Error: ${err}`
  );
}
