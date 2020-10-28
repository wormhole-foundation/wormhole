const Wormhole = artifacts.require("Wormhole");
const WrappedAsset = artifacts.require("WrappedAsset");
const ERC20 = artifacts.require("ERC20PresetMinterPauser");

advanceBlock = () => {
    return new Promise((resolve, reject) => {
        web3.currentProvider.send({
            jsonrpc: "2.0",
            method: "evm_mine",
            id: new Date().getTime()
        }, (err, result) => {
            if (err) {
                return reject(err);
            }
            const newBlockHash = web3.eth.getBlock('latest').hash;

            return resolve(newBlockHash)
        });
    });
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

module.exports = function(callback) {
    const fn = async () => {
        let bridge = await Wormhole.deployed();

        let token = await ERC20.new("Test Token", "TKN");
        await token.mint("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1", "1000000000000000000");
        await token.approve(bridge.address, "1000000000000000000");

        while (true) {
            let ev = await bridge.lockAssets(
                token.address, /* asset address */
                "1000000005",  /* amount */
                "0xe964facf2174f495f35832a29203e421173c8c20db80a0a976f96fce1dc59a1e", /* recipient 36sMzZHzMZRSqSHYCw5KGG6rN5av1C5v11A5KgwPMpas */
                1,     /* target chain: solana */
                Math.floor(Math.random() * 65535),    /* nonce */
                false  /* refund dust? */
            );

            let block = await web3.eth.getBlock('latest');
            console.log("block", block.number, "with txs", block.transactions, "and time", block.timestamp);
            await advanceBlock();
            await sleep(5000);
        }
    }

    fn().catch(reason => console.error(reason))
}

