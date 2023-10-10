/*
    This script advances Anvil network state. It runs as a sidecar pod alongside the devnet and
    ensures that manual token transfers triggered through the web UI will be able to be confirmed.
 */

import fetch from "node-fetch"

const RPC_URL = 'http://localhost:8545'

const advanceBlock = () => {
    return new Promise((resolve, reject) => {
        fetch(RPC_URL, {
            method: 'POST',
            headers: {'content-type': 'application/json'},
            body: JSON.stringify({
                jsonrpc: '2.0',
                id: new Date().getTime(),
                method: "evm_mine",
            })
        }).then(() => {
            resolve(0);
        })
    });
}

function sleep(ms: number) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

const fn = async () => {
    while (true) {
        await advanceBlock();
        await sleep(1000);
    }
}

fn().catch(reason => console.error(reason))