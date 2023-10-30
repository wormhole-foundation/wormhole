import { ChainAddress, ChainsConfig, Contracts, NativeAddress, Network, TokenBridge, TokenId } from '@wormhole-foundation/connect-sdk';
import { Provider } from 'ethers';
import { TokenBridgeContract } from './ethers-contracts';
import { AnyEvmAddress, EvmChainName, EvmUnsignedTransaction } from '@wormhole-foundation/connect-sdk-evm';
export declare class EvmTokenBridge implements TokenBridge<'Evm'> {
    readonly network: Network;
    readonly chain: EvmChainName;
    readonly provider: Provider;
    readonly contracts: Contracts;
    readonly tokenBridge: TokenBridgeContract;
    readonly tokenBridgeAddress: string;
    readonly chainId: bigint;
    private constructor();
    static fromProvider(provider: Provider, config: ChainsConfig): Promise<EvmTokenBridge>;
    isWrappedAsset(token: AnyEvmAddress): Promise<boolean>;
    getOriginalAsset(token: AnyEvmAddress): Promise<TokenId>;
    hasWrappedAsset(token: TokenId): Promise<boolean>;
    getWrappedAsset(token: TokenId): Promise<NativeAddress<'Evm'>>;
    isTransferCompleted(vaa: TokenBridge.VAA<'Transfer' | 'TransferWithPayload'>): Promise<boolean>;
    createAttestation(token: AnyEvmAddress): AsyncGenerator<EvmUnsignedTransaction>;
    submitAttestation(vaa: TokenBridge.VAA<'AttestMeta'>): AsyncGenerator<EvmUnsignedTransaction>;
    transfer(sender: AnyEvmAddress, recipient: ChainAddress, token: AnyEvmAddress | 'native', amount: bigint, payload?: Uint8Array): AsyncGenerator<EvmUnsignedTransaction>;
    redeem(sender: AnyEvmAddress, vaa: TokenBridge.VAA<'Transfer' | 'TransferWithPayload'>, unwrapNative?: boolean): AsyncGenerator<EvmUnsignedTransaction>;
    getWrappedNative(): Promise<NativeAddress<'Evm'>>;
    private createUnsignedTx;
}
