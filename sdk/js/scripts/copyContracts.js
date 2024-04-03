const copydir = require("copy-dir");
console.log("Copying from ../../ethereum/ethers-contracts");
copydir.sync("../../ethereum/ethers-contracts", "src/ethers-contracts");
copydir.sync(
  "../../relayer/ethereum/ethers-contracts",
  "src/ethers-relayer-contracts"
);
