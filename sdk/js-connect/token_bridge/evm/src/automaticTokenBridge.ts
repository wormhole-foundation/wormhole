import {
  ChainAddress,
  AutomaticTokenBridge,
  TokenBridge,
  serialize,
  TokenId,
  chainToChainId,
  toChainId,
  Network,
  Contracts,
  ChainsConfig,
  nativeChainAddress,
} from '@wormhole-foundation/connect-sdk';
import {
  evmNetworkChainToEvmChainId,
  AnyEvmAddress,
  EvmChainName,
  addChainId,
  addFrom,
  EvmUnsignedTransaction,
  EvmPlatform,
  EvmAddress,
} from '@wormhole-foundation/connect-sdk-evm';
import { Provider, TransactionRequest } from 'ethers';

import { ethers_contracts } from '.';

export class EvmAutomaticTokenBridge implements AutomaticTokenBridge<'Evm'> {
  readonly tokenBridgeRelayer: ethers_contracts.TokenBridgeRelayer;
  readonly tokenBridge: ethers_contracts.TokenBridgeContract;
  readonly chainId: bigint;

  private constructor(
    readonly network: Network,
    readonly chain: EvmChainName,
    readonly provider: Provider,
    readonly contracts: Contracts,
  ) {
    if (network === 'Devnet')
      throw new Error('AutomaticTokenBridge not supported on Devnet');

    this.chainId = evmNetworkChainToEvmChainId(network, chain);

    const tokenBridgeAddress = this.contracts.tokenBridge!;
    if (!tokenBridgeAddress)
      throw new Error(
        `Wormhole Token Bridge contract for domain ${chain} not found`,
      );

    this.tokenBridge = ethers_contracts.Bridge__factory.connect(
      tokenBridgeAddress,
      provider,
    );

    const relayerAddress = this.contracts.relayer;
    if (!relayerAddress)
      throw new Error(
        `Wormhole Token Bridge Relayer contract for domain ${chain} not found`,
      );

    this.tokenBridgeRelayer =
      ethers_contracts.TokenBridgeRelayer__factory.connect(
        relayerAddress,
        provider,
      );
  }
  async *redeem(
    sender: AnyEvmAddress,
    vaa: TokenBridge.VAA<'TransferWithPayload'>,
  ): AsyncGenerator<EvmUnsignedTransaction> {
    const senderAddr = new EvmAddress(sender).toString();
    const txReq =
      await this.tokenBridgeRelayer.completeTransferWithRelay.populateTransaction(
        serialize(vaa),
      );

    return this.createUnsignedTx(
      addFrom(txReq, senderAddr),
      'TokenBridgeRelayer.completeTransferWithRelay',
    );
  }

  static async fromProvider(
    provider: Provider,
    config: ChainsConfig,
  ): Promise<EvmAutomaticTokenBridge> {
    const [network, chain] = await EvmPlatform.chainFromRpc(provider);
    return new EvmAutomaticTokenBridge(
      network,
      chain,
      provider,
      config[chain]!.contracts!,
    );
  }

  //alternative naming: initiateTransfer
  async *transfer(
    sender: AnyEvmAddress,
    recipient: ChainAddress,
    token: AnyEvmAddress | 'native',
    amount: bigint,
    relayerFee: bigint,
    nativeGas?: bigint,
  ): AsyncGenerator<EvmUnsignedTransaction> {
    const senderAddr = new EvmAddress(sender).toString();
    const recipientChainId = chainToChainId(recipient.chain);
    const recipientAddress = recipient.address
      .toUniversalAddress()
      .toUint8Array();
    const nativeTokenGas = nativeGas ? nativeGas : 0n;

    if (token === 'native') {
      const txReq =
        await this.tokenBridgeRelayer.wrapAndTransferEthWithRelay.populateTransaction(
          nativeTokenGas,
          recipientChainId,
          recipientAddress,
          0, // skip batching
          { value: relayerFee + amount + nativeTokenGas },
        );
      yield this.createUnsignedTx(
        addFrom(txReq, senderAddr),
        'TokenBridgeRelayer.wrapAndTransferETHWithRelay',
      );
    } else {
      //TODO check for ERC-2612 (permit) support on token?
      const tokenAddr = new EvmAddress(token).toString();
      // TODO: allowance?

      const txReq =
        await this.tokenBridgeRelayer.transferTokensWithRelay.populateTransaction(
          tokenAddr,
          amount,
          nativeTokenGas,
          recipientChainId,
          recipientAddress,
          0,
        );

      yield this.createUnsignedTx(
        addFrom(txReq, senderAddr),
        'TokenBridgeRelayer.transferTokensWithRelay',
      );
    }
  }

  async getRelayerFee(
    sender: ChainAddress,
    recipient: ChainAddress,
    token: TokenId | 'native',
  ): Promise<bigint> {
    const tokenId: TokenId =
      token === 'native'
        ? nativeChainAddress([this.chain, await this.tokenBridge.WETH()])
        : token;

    const destChainId = toChainId(recipient.chain);
    const destTokenAddress = new EvmAddress(
      tokenId.address.toString(),
    ).toString();

    const tokenContract = EvmPlatform.getTokenImplementation(
      this.provider,
      destTokenAddress,
    );

    const decimals = await tokenContract.decimals();

    return await this.tokenBridgeRelayer.calculateRelayerFee(
      destChainId,
      destTokenAddress,
      decimals,
    );
  }

  private createUnsignedTx(
    txReq: TransactionRequest,
    description: string,
    parallelizable: boolean = false,
  ): EvmUnsignedTransaction {
    return new EvmUnsignedTransaction(
      addChainId(txReq, this.chainId),
      this.network,
      this.chain,
      description,
      parallelizable,
    );
  }
}
