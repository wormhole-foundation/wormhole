import { ChainAddress, AutomaticTokenBridge, TokenBridge, TokenId, Network, Contracts, ChainsConfig } from '@wormhole-foundation/connect-sdk';
import { AnyEvmAddress, EvmChainName, EvmUnsignedTransaction } from '@wormhole-foundation/connect-sdk-evm';
import { Provider } from 'ethers';
import { ethers_contracts } from '.';
export declare class EvmAutomaticTokenBridge implements AutomaticTokenBridge<'Evm'> {
    readonly network: Network;
    readonly chain: EvmChainName;
    readonly provider: Provider;
    readonly contracts: Contracts;
    readonly tokenBridgeRelayer: ethers_contracts.TokenBridgeRelayer;
    readonly tokenBridge: ethers_contracts.TokenBridgeContract;
    readonly chainId: bigint;
    private constructor();
    redeem(sender: AnyEvmAddress, vaa: TokenBridge.VAA<'TransferWithPayload'>): AsyncGenerator<EvmUnsignedTransaction>;
    static fromProvider(provider: Provider, config: ChainsConfig): Promise<EvmAutomaticTokenBridge>;
    transfer(sender: AnyEvmAddress, recipient: ChainAddress, token: AnyEvmAddress | 'native', amount: bigint, relayerFee: bigint, nativeGas?: bigint): AsyncGenerator<EvmUnsignedTransaction>;
    getRelayerFee(sender: ChainAddress, recipient: ChainAddress, token: TokenId | 'native'): Promise<bigint>;
    private createUnsignedTx;
}
