const fs = require("fs");
fs.readdirSync("src/ethers-contracts").forEach((file) => {
  if (file.endsWith(".d.ts")) {
    fs.copyFileSync(
      `src/ethers-contracts/${file}`,
      `lib/ethers-contracts/${file}`
    );
  }
});
