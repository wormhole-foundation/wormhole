const fs = require("fs");
["lib/esm", "lib/cjs"].forEach((buildPath) => {
  fs.readdirSync("src/ethers-contracts").forEach((file) => {
    if (file.endsWith(".d.ts")) {
      fs.copyFileSync(
        `src/ethers-contracts/${file}`,
        `${buildPath}/ethers-contracts/${file}`
      );
    }
  });

  fs.readdirSync("src/ethers-contracts/abi").forEach((file) => {
    if (file.endsWith(".d.ts")) {
      fs.copyFileSync(
        `src/ethers-contracts/abi/${file}`,
        `${buildPath}/ethers-contracts/abi/${file}`
      );
    }
  });
});
