import {
  ChainsConfig,
  Contracts,
  Network,
  TxHash,
  WormholeCore,
  WormholeMessageId,
  isWormholeMessageId,
  toNative,
} from '@wormhole-foundation/connect-sdk';
import { Provider, TransactionRequest } from 'ethers';
import { Implementation, ImplementationInterface } from './ethers-contracts';
import { ethers_contracts } from '.';

import {
  evmNetworkChainToEvmChainId,
  EvmUnsignedTransaction,
  AnyEvmAddress,
  EvmChainName,
  addChainId,
  addFrom,
  EvmPlatform,
  EvmAddress,
} from '@wormhole-foundation/connect-sdk-evm';

export class EvmWormholeCore implements WormholeCore<'Evm'> {
  readonly chainId: bigint;

  readonly coreAddress: string;

  readonly core: Implementation;
  readonly coreIface: ImplementationInterface;

  private constructor(
    readonly network: Network,
    readonly chain: EvmChainName,
    readonly provider: Provider,
    readonly contracts: Contracts,
  ) {
    this.chainId = evmNetworkChainToEvmChainId.get(network, chain)!;

    this.coreIface = ethers_contracts.Implementation__factory.createInterface();

    const address = this.contracts.coreBridge;
    if (!address) throw new Error('Core bridge address not found');

    this.coreAddress = address;
    this.core = ethers_contracts.Implementation__factory.connect(
      address,
      provider,
    );
  }

  static async fromRpc(
    provider: Provider,
    config: ChainsConfig,
  ): Promise<EvmWormholeCore> {
    const [network, chain] = await EvmPlatform.chainFromRpc(provider);
    return new EvmWormholeCore(
      network,
      chain,
      provider,
      config[chain]!.contracts,
    );
  }

  async *publishMessage(
    sender: AnyEvmAddress,
    message: Uint8Array | string,
  ): AsyncGenerator<EvmUnsignedTransaction> {
    const senderAddr = new EvmAddress(sender).toString();

    const txReq = await this.core.publishMessage.populateTransaction(
      0,
      message,
      200, // TODO: lookup finality
    );

    yield this.createUnsignedTx(
      addFrom(txReq, senderAddr),
      'WormholeCore.publishMessage',
    );
  }

  async parseTransaction(txid: TxHash): Promise<WormholeMessageId[]> {
    const receipt = await this.provider.getTransactionReceipt(txid);
    if (receipt === null) return [];

    return receipt.logs
      .filter((l: any) => {
        return l.address === this.coreAddress;
      })
      .map((log) => {
        const { topics, data } = log;
        const parsed = this.coreIface.parseLog({
          topics: topics.slice(),
          data,
        });
        if (parsed === null) return undefined;

        const emitterAddress = toNative(this.chain, parsed.args.sender);
        return {
          chain: this.chain,
          emitter: emitterAddress.toUniversalAddress(),
          sequence: parsed.args.sequence,
        } as WormholeMessageId;
      })
      .filter(isWormholeMessageId);
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
