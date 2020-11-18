/*
    This script advances Ganache network state. It runs as a sidecar pod alongside the devnet and
    ensures that manual token transfers triggered through the web UI will be able to be confirmed.
 */

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
        while (true) {
            console.log(await advanceBlock());
            await sleep(1000);
        }
    }

    fn().catch(reason => console.error(reason))
}
