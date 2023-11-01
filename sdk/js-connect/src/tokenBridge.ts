import { Connection } from "@solana/web3.js";
import { ethers } from "ethers";
import { CONFIG, TokenBridge } from "@wormhole-foundation/connect-sdk";

import "@wormhole-foundation/connect-sdk-evm-core";
import { EvmTokenBridge } from "@wormhole-foundation/connect-sdk-evm-tokenbridge";

import "@wormhole-foundation/connect-sdk-solana-core";
import { SolanaTokenBridge } from "@wormhole-foundation/connect-sdk-solana-tokenbridge";


export async function getEthTokenBridge(provider: ethers.Provider): Promise<TokenBridge<'Evm'>> {
    return EvmTokenBridge.fromRpc(provider, CONFIG.Devnet.chains)
}

export async function getSolTokenBridge(connection: Connection): Promise<TokenBridge<'Solana'>> {
    return SolanaTokenBridge.fromRpc(connection, CONFIG.Devnet.chains)
}