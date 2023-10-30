import { ChainsConfig, Contracts, Network, TxHash, WormholeCore, WormholeMessageId } from '@wormhole-foundation/connect-sdk';
import { Provider } from 'ethers';
import { Implementation, ImplementationInterface } from './ethers-contracts';
import { EvmUnsignedTransaction, AnyEvmAddress, EvmChainName } from '@wormhole-foundation/connect-sdk-evm';
export declare class EvmWormholeCore implements WormholeCore<'Evm'> {
    readonly network: Network;
    readonly chain: EvmChainName;
    readonly provider: Provider;
    readonly contracts: Contracts;
    readonly chainId: bigint;
    readonly coreAddress: string;
    readonly core: Implementation;
    readonly coreIface: ImplementationInterface;
    private constructor();
    static fromProvider(provider: Provider, config: ChainsConfig): Promise<EvmWormholeCore>;
    publishMessage(sender: AnyEvmAddress, message: Uint8Array | string): AsyncGenerator<EvmUnsignedTransaction>;
    parseTransaction(txid: TxHash): Promise<WormholeMessageId[]>;
    private createUnsignedTx;
}
