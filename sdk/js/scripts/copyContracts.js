const copydir = require("copy-dir");
const fs = require("fs");
fs.rmSync("src/ethers-contracts", { recursive: true, force: true });
copydir.sync("../../ethereum/ethers-contracts", "src/ethers-contracts");
