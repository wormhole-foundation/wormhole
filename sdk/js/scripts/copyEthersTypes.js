const fs = require("fs");

function copyTypes(srcDir) {
  ["lib/esm", "lib/cjs"].forEach((buildPath) => {
    fs.readdirSync(srcDir).forEach((file) => {
      if (file.endsWith(".d.ts")) {
        fs.copyFileSync(
          `src/ethers-contracts/${file}`,
          `${buildPath}/ethers-contracts/${file}`
        );
      }
    });

    fs.readdirSync(srcDir).forEach((file) => {
      if (file.endsWith(".d.ts")) {
        fs.copyFileSync(
          `src/ethers-contracts/abi/${file}`,
          `${buildPath}/ethers-contracts/abi/${file}`
        );
      }
    });
  });
}

copyTypes("src/ethers-contracts");
copyTypes("src/ethers-relayer-contracts");
