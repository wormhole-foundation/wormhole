import { Program } from '@project-serum/anchor';
import { Connection } from '@solana/web3.js';
import {
  AnyAddress,
  ChainId,
  ChainsConfig,
  Contracts,
  Network,
  RpcConnection,
  UnsignedTransaction,
  WormholeCore,
  WormholeMessageId,
  toChainId,
  toNative,
} from '@wormhole-foundation/connect-sdk';
import {
  SolanaChainName,
  SolanaPlatform,
} from '@wormhole-foundation/connect-sdk-solana';
import { Wormhole as WormholeCoreContract } from './types';
import { createReadOnlyWormholeProgramInterface } from './utils';

const SOLANA_SEQ_LOG = 'Program log: Sequence: ';

export class SolanaWormholeCore implements WormholeCore<'Solana'> {
  readonly chainId: ChainId;
  readonly coreBridge: Program<WormholeCoreContract>;

  private constructor(
    readonly network: Network,
    readonly chain: SolanaChainName,
    readonly connection: Connection,
    readonly contracts: Contracts,
  ) {
    this.chainId = toChainId(chain);

    const coreBridgeAddress = contracts.coreBridge;
    if (!coreBridgeAddress)
      throw new Error(
        `CoreBridge contract Address for chain ${chain} not found`,
      );

    this.coreBridge = createReadOnlyWormholeProgramInterface(
      coreBridgeAddress,
      connection,
    );
  }

  static async fromRpc(
    connection: RpcConnection<'Solana'>,
    config: ChainsConfig,
  ): Promise<SolanaWormholeCore> {
    const [network, chain] = await SolanaPlatform.chainFromRpc(connection);
    return new SolanaWormholeCore(
      network,
      chain,
      connection,
      config[chain]!.contracts,
    );
  }

  publishMessage(
    sender: AnyAddress,
    message: string | Uint8Array,
  ): AsyncGenerator<UnsignedTransaction, any, unknown> {
    throw new Error('Method not implemented.');
  }

  async parseTransaction(txid: string): Promise<WormholeMessageId[]> {
    const response = await this.connection.getTransaction(txid);
    if (!response || !response.meta?.innerInstructions![0].instructions)
      throw new Error('transaction not found');

    const instructions = response.meta?.innerInstructions![0].instructions;
    const accounts = response.transaction.message.accountKeys;

    // find the instruction where the programId equals the Wormhole ProgramId and the emitter equals the Token Bridge
    const bridgeInstructions = instructions.filter((i) => {
      const programId = accounts[i.programIdIndex].toString();
      const wormholeCore = this.coreBridge.programId.toString();
      return programId === wormholeCore;
    });

    if (bridgeInstructions.length === 0)
      throw new Error('no bridge messages found');

    // TODO: unsure about the single bridge instruction and the [2] index, will this always be the case?
    const [logmsg] = bridgeInstructions;
    const emitterAcct = accounts[logmsg.accounts[2]];
    const emitter = toNative(this.chain, emitterAcct.toString());

    const sequence = response.meta?.logMessages
      ?.filter((msg) => msg.startsWith(SOLANA_SEQ_LOG))?.[0]
      ?.replace(SOLANA_SEQ_LOG, '');

    if (!sequence) {
      throw new Error('sequence not found');
    }

    return [
      {
        chain: this.chain,
        emitter: emitter.toUniversalAddress(),
        sequence: BigInt(sequence),
      },
    ];
  }
}
