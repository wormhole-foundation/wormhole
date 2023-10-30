import { Program } from '@project-serum/anchor';
import { Connection } from '@solana/web3.js';
import { AnyAddress, ChainId, ChainsConfig, Contracts, Network, RpcConnection, UnsignedTransaction, WormholeCore, WormholeMessageId } from '@wormhole-foundation/connect-sdk';
import { SolanaChainName } from '@wormhole-foundation/connect-sdk-solana';
import { Wormhole as WormholeCoreContract } from './types';
export declare class SolanaWormholeCore implements WormholeCore<'Solana'> {
    readonly network: Network;
    readonly chain: SolanaChainName;
    readonly connection: Connection;
    readonly contracts: Contracts;
    readonly chainId: ChainId;
    readonly coreBridge: Program<WormholeCoreContract>;
    private constructor();
    static fromProvider(connection: RpcConnection<'Solana'>, config: ChainsConfig): Promise<SolanaWormholeCore>;
    publishMessage(sender: AnyAddress, message: string | Uint8Array): AsyncGenerator<UnsignedTransaction, any, unknown>;
    parseTransaction(txid: string): Promise<WormholeMessageId[]>;
}
