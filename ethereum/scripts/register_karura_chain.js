// run this script with truffle exec

const jsonfile = require("jsonfile");
const TokenBridge = artifacts.require("TokenBridge");
const TokenImplementation = artifacts.require("TokenImplementation");
const BridgeImplementationFullABI = jsonfile.readFileSync("../build/contracts/BridgeImplementation.json").abi

module.exports = async function (callback) {
    try {
        const accounts = await web3.eth.getAccounts();
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, TokenBridge.address);

        // Register the Karura endpoint
      await initialized.methods.registerChain("0x01000000000100117681cb5efce1ad4126c7e23f529112e99bc28bb21461ec896555395449e8b3065aa8f80a9d5aa7f417cf8194336594201ec3b3a54bb72a91648c8b8829dbb10000000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000037912b700000000000000000000000000000000000000000000546f6b656e42726964676501000000090000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16").send({
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

