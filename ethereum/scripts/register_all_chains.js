// run this script with truffle exec

const jsonfile = require("jsonfile");
const TokenBridge = artifacts.require("TokenBridge");
const NFTBridge = artifacts.require("NFTBridgeEntrypoint");
const BridgeImplementationFullABI = jsonfile.readFileSync(
  "../build/contracts/BridgeImplementation.json"
).abi;
const solTokenBridgeVAA = process.env.REGISTER_SOL_TOKEN_BRIDGE_VAA;
const solNFTBridgeVAA = process.env.REGISTER_SOL_NFT_BRIDGE_VAA;
const terraTokenBridgeVAA = process.env.REGISTER_TERRA_TOKEN_BRIDGE_VAA;
const terraNFTBridgeVAA = process.env.REGISTER_TERRA_NFT_BRIDGE_VAA;
const terra2TokenBridgeVAA = process.env.REGISTER_TERRA2_TOKEN_BRIDGE_VAA;
const bscTokenBridgeVAA = process.env.REGISTER_BSC_TOKEN_BRIDGE_VAA;
const algoTokenBridgeVAA = process.env.REGISTER_ALGO_TOKEN_BRIDGE_VAA;
const nearTokenBridgeVAA = process.env.REGISTER_NEAR_TOKEN_BRIDGE_VAA;
const wormchainTokenBridgeVAA = process.env.REGISTER_WORMCHAIN_TOKEN_BRIDGE_VAA;
const aptosTokenBridgeVAA = process.env.REGISTER_APTOS_TOKEN_BRIDGE_VAA;

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

    // Register the Solana token bridge endpoint
    console.log("Registering solana...");
    await tokenBridge.methods.registerChain("0x" + solTokenBridgeVAA).send({
      value: 0,
      from: accounts[0],
      gasLimit: 2000000,
    });

    // Register the Solana NFT bridge endpoint
    await nftBridge.methods.registerChain("0x" + solNFTBridgeVAA).send({
      value: 0,
      from: accounts[0],
      gasLimit: 2000000,
    });

    // Register the terra token bridge endpoint
    console.log("Registering Terra...");
    await tokenBridge.methods.registerChain("0x" + terraTokenBridgeVAA).send({
      value: 0,
      from: accounts[0],
      gasLimit: 2000000,
    });

    // Register the terra NFT bridge endpoint
    await nftBridge.methods.registerChain("0x" + terraNFTBridgeVAA).send({
      value: 0,
      from: accounts[0],
      gasLimit: 2000000,
    });

    // Register the terra2 token bridge endpoint
    console.log("Registering Terra2...");
    await tokenBridge.methods.registerChain("0x" + terra2TokenBridgeVAA).send({
      value: 0,
      from: accounts[0],
      gasLimit: 2000000,
    });

    // Register the BSC endpoint
    console.log("Registering BSC...");
    await tokenBridge.methods.registerChain("0x" + bscTokenBridgeVAA).send({
      value: 0,
      from: accounts[0],
      gasLimit: 2000000,
    });

    // Register the ALGO endpoint
    console.log("Registering Algo...");
    await tokenBridge.methods.registerChain("0x" + algoTokenBridgeVAA).send({
      value: 0,
      from: accounts[0],
      gasLimit: 2000000,
    });

    // Register the near token bridge endpoint
    console.log("Registering Near...");
    await tokenBridge.methods.registerChain("0x" + nearTokenBridgeVAA).send({
      value: 0,
      from: accounts[0],
      gasLimit: 2000000,
    });

    // Register the wormhole token bridge endpoint
    console.log("Registering Wormchain...");
    await tokenBridge.methods
      .registerChain("0x" + wormchainTokenBridgeVAA)
      .send({
        value: 0,
        from: accounts[0],
        gasLimit: 2000000,
      });

    // Register the APTOS endpoint
    console.log("Registering Aptos...");
    await tokenBridge.methods.registerChain("0x" + aptosTokenBridgeVAA).send({
      value: 0,
      from: accounts[0],
      gasLimit: 2000000,
    });

    callback();
  } catch (e) {
    callback(e);
  }
};
