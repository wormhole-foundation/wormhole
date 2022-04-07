// run with:
// npm run deploy-bridge-implementation-only
// e.g. Ethereum Mainnet
// INFURA_KEY="" MNEMONIC="" npm run deploy-bridge-implementation-only -- --network mainnet
// e.g. BSC
// MNEMONIC="" npm run deploy-bridge-implementation-only -- --network binance
// e.g. Polygon
// MNEMONIC="" npm run deploy-bridge-implementation-only -- --network polygon
const BridgeImplementation = artifacts.require("BridgeImplementation");
module.exports = async function(deployer, network) {
  if (network === "test") return;
  await deployer.deploy(BridgeImplementation);
};
