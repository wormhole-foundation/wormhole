// run this script with truffle exec

const jsonfile = require("jsonfile");
const TokenBridge = artifacts.require("TokenBridge");
const NFTBridge = artifacts.require("NFTBridgeEntrypoint");
const TokenImplementation = artifacts.require("TokenImplementation");
const BridgeImplementationFullABI = jsonfile.readFileSync("../build/contracts/BridgeImplementation.json").abi
const solTokenBridgeVAA = process.env.REGISTER_SOL_TOKEN_BRIDGE_VAA
const solNFTBridgeVAA = process.env.REGISTER_SOL_NFT_BRIDGE_VAA

module.exports = async function (callback) {
    try {
        const accounts = await web3.eth.getAccounts();
        const tokenBridge = new web3.eth.Contract(BridgeImplementationFullABI, TokenBridge.address);
        const nftBridge = new web3.eth.Contract(BridgeImplementationFullABI, NFTBridge.address);

        // Register the Solana token bridge endpoint
        await tokenBridge.methods.registerChain("0x" + solTokenBridgeVAA).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        // Register the Solana NFT bridge endpoint
        await nftBridge.methods.registerChain("0x" + solNFTBridgeVAA).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        callback();
    }
    catch (e) {
        callback(e);
    }
}
