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
        await nftBridge.methods.registerChain("0x010000000001008ebe6a1971ae336bb7817aa7e8ffc13e1582ebbc00dc85e33b592dfea998787a1a9ccece2efce7fcdf228153baafb6f1232b320805c82fac90b5e49c3b5ad4fd0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000004e46544272696467650100000003000000000000000000000000288246bebae560e006d01c675ae332ac8e146bb7").send({
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

