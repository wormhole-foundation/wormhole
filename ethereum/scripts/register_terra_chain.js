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

        // Register the terra token bridge endpoint
        await tokenBridge.methods.registerChain("0x010000000001009a895e8b42444fdf60a71666121d7e84b3a49579ba3b84fff1dbdf9ec93360390c05a88f66c90df2034cb38427ba9b01632e780ce7b84df559a1bf44c316447d01000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000546f6b656e4272696467650100000003000000000000000000000000784999135aaa8a3ca5914468852fdddbddd8789d").send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        // Register the terra NFT bridge endpoint
        await nftBridge.methods.registerChain("0x01000000000100880ab34efd145fa1c9c4f2f7b233f3bc99cb730ab2d2138babf989a2555e2e8f737b759f91970d300c4364249b0740b134dbefb943fb55fd76483261ce68c17b0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000023c2cdd0000000000000000000000000000000000000000000000004e465442726964676501000000030000000000000000000000000fe5c51f539a651152ae461086d733777a54a134").send({
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

