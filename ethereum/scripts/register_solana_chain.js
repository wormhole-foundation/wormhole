// run this script with truffle exec

const jsonfile = require("jsonfile");
const TokenBridge = artifacts.require("TokenBridge");
const NFTBridge = artifacts.require("NFTBridgeEntrypoint");
const TokenImplementation = artifacts.require("TokenImplementation");
const BridgeImplementationFullABI = jsonfile.readFileSync("../build/contracts/BridgeImplementation.json").abi

module.exports = async function (callback) {
    try {
        const accounts = await web3.eth.getAccounts();
        const tokenBridge = new web3.eth.Contract(BridgeImplementationFullABI, TokenBridge.address);
        const nftBridge = new web3.eth.Contract(BridgeImplementationFullABI, NFTBridge.address);

        // Register the Solana token bridge endpoint
        await tokenBridge.methods.registerChain("0x01000000000100c9f4230109e378f7efc0605fb40f0e1869f2d82fda5b1dfad8a5a2dafee85e033d155c18641165a77a2db6a7afbf2745b458616cb59347e89ae0c7aa3e7cc2d400000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000546f6b656e4272696467650100000001c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f").send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        await nftBridge.methods.registerChain("0x010000000001007985ba742002ae745c19722fea4d82102e68526c7c9d94d0e5d0a809071c98451c9693b230b3390f4ca9555a3ba9a9abbe87cf6f9e400682213e4fbbe1dabb9e0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000004e4654427269646765010000000196ee982293251b48729804c8e8b24b553eb6b887867024948d2236fd37a577ab").send({
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