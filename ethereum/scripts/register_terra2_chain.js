// run this script with truffle exec

const jsonfile = require("jsonfile");
const TokenBridge = artifacts.require("TokenBridge");
const BridgeImplementationFullABI = jsonfile.readFileSync(
  "../build/contracts/BridgeImplementation.json"
).abi;
const terra2TokenBridgeVAA = process.env.REGISTER_TERRA2_TOKEN_BRIDGE_VAA;

module.exports = async function (callback) {
  try {
    const accounts = await web3.eth.getAccounts();
    const tokenBridge = new web3.eth.Contract(
      BridgeImplementationFullABI,
      TokenBridge.address
    );

    // Register the terra2 token bridge endpoint
    await tokenBridge.methods.registerChain("0x" + terra2TokenBridgeVAA).send({
      value: 0,
      from: accounts[0],
      gasLimit: 2000000,
    });

    callback();
  } catch (e) {
    callback(e);
  }
};
