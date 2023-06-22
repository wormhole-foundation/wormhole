// run this script with truffle exec

const jsonfile = require("jsonfile");
const TokenBridge = artifacts.require("TokenBridge");
const NFTBridge = artifacts.require("NFTBridgeEntrypoint");
const BridgeImplementationFullABI = jsonfile.readFileSync(
  "../build/contracts/BridgeImplementation.json"
).abi;

// The input parameter is a RegExp
// It returns an array of process.env variables satisfying the input RegExp
function getFilteredEnvs(regexp) {
  const filteredEnvs = [];
  for (const [key, value] of Object.entries(process.env)) {
    if (regexp.test(key) && value) {
      console.log("getFilteredEnvs: pushing " + key);
      filteredEnvs.push(value);
    }
  }
  return filteredEnvs;
}

module.exports = async function(callback) {
  try {
    const accounts = await web3.eth.getAccounts();
    const tokenBridge = new web3.eth.Contract(
      BridgeImplementationFullABI,
      TokenBridge.address
    );
    const nftBridge = new web3.eth.Contract(
      BridgeImplementationFullABI,
      NFTBridge.address
    );

    const TokenBridgeRegExp = new RegExp("REGISTER_.*_TOKEN_BRIDGE_VAA");
    const NFTBridgeRegExp = new RegExp("REGISTER_.*_NFT_BRIDGE_VAA");

    const TokenBridgeVAAs = getFilteredEnvs(TokenBridgeRegExp);
    const NFTBridgeVAAs = getFilteredEnvs(NFTBridgeRegExp);

    // Register the token bridge endpoints
    console.log("Registering " + TokenBridgeVAAs.length + " Token Bridges...");
    for (const vaa of TokenBridgeVAAs) {
      await tokenBridge.methods.registerChain("0x" + vaa).send({
        value: 0,
        from: accounts[0],
        gasLimit: 2000000,
      });
    }

    // Register the NFT bridge endpoints
    console.log("Registering " + NFTBridgeVAAs.length + " NFT Bridges...");
    for (const vaa of NFTBridgeVAAs) {
      await nftBridge.methods.registerChain("0x" + vaa).send({
        value: 0,
        from: accounts[0],
        gasLimit: 2000000,
      });
    }
    console.log("Finished registering all Bridges...");

    callback();
  } catch (e) {
    callback(e);
  }
};
