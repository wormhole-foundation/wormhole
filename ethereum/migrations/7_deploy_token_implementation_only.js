// run with:
// npm run deploy-token-implementation-only
// e.g. Ethereum Mainnet
// INFURA_KEY="" MNEMONIC="" npm run deploy-token-implementation-only -- --network mainnet
// e.g. BSC
// MNEMONIC="" npm run deploy-token-implementation-only -- --network binance
// e.g. Polygon
// MNEMONIC="" npm run deploy-token-implementation-only -- --network polygon
const TokenImplementation = artifacts.require("TokenImplementation");
module.exports = async function(deployer, network) {
  if (network === "test") return;
  await deployer.deploy(TokenImplementation);
};
