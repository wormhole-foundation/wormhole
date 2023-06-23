import {relayer} from "@certusone/wormhole-sdk"
import {ChainName, Network} from "@certusone/wormhole-sdk"
import {ethers} from "ethers";

export async function getStatus() {
    // ts-node status.ts --tx TXHASH --chain CHAINNAME --env MAINNET/DEVNET/TESTNET
    const txHash = process.argv[process.argv.findIndex(x => x === '--tx') + 1];
    const sourceChain: ChainName = process.argv[process.argv.findIndex(x => x === '--chain') + 1] as ChainName
    const envIndex = process.argv.findIndex(x => x === '--env') + 1;
    const environment: Network = envIndex == 0 ? 'MAINNET' : process.argv[envIndex] as Network;

    console.log(relayer.stringifyWormholeRelayerInfo(await relayer.getWormholeRelayerInfo(sourceChain, txHash, {environment: environment})));

}

getStatus();