// run this script with truffle exec

const jsonfile = require("jsonfile");
const TokenBridge = artifacts.require("TokenBridge");
const TokenImplementation = artifacts.require("TokenImplementation");
const BridgeImplementationFullABI = jsonfile.readFileSync("../build/contracts/BridgeImplementation.json").abi

module.exports = async function (callback) {
    try {
        const accounts = await web3.eth.getAccounts();
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, TokenBridge.address);

        // Register the Karura(11) endpoint
        await initialized.methods.registerChain("0x010000000001009e3ca8d814d6accfdb76a9f8766d5c5eaa8beac35993a093e383c395501956041fbe6facf4fa54662f924f2ad9f5f03fc6dc51d5fd3dbbc8f164162b820befc500000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000155388100000000000000000000000000000000000000000000546f6b656e427269646765010000000b0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16").send({
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

