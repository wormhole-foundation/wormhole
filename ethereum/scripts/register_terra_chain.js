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
        await nftBridge.methods.registerChain("0x01000000000100d2db61323a0f9aea98391658d2766c78520bb0ec74e5ef0f861be9df530279d058602c3121d67e24aae374f25438ada7d334ea3e6a4c49214aff562e94a62f760000000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000004e46544272696467650100000003000000000000000000000000b13e74a8d66ea38474f0122d99348c3749ede3c0").send({
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

