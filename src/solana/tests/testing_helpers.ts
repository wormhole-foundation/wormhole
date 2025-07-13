import {
  Connection,
  Finality,
  LAMPORTS_PER_SOL,
  PublicKey,
  Signer,
  Transaction,
  TransactionInstruction,
  sendAndConfirmTransaction,
  Keypair,
  TransactionSignature,
  VersionedTransactionResponse
} from "@solana/web3.js"
import { mocks } from "@wormhole-foundation/sdk-definitions/testing"
import { Contracts } from '@wormhole-foundation/sdk-definitions'
import fs from "fs/promises"
import fsSync from "fs"
import * as toml from 'toml'

// Copied from @wormhole-foundation/wormhole-scaffolding/solana/ts/helpers/utils.ts
// TODO: There's probably some way to import this?

export const MOCK_GUARDIANS = new mocks.MockGuardians(0, ["cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0"])

type Tuple<T, N extends number, R extends T[] = []> = R['length'] extends N
  ? R
  : Tuple<T, N, [T, ...R]>;

export class TestsHelper {
  static readonly LOCALHOST = 'http://localhost:8899';
  readonly connection: Connection;
  readonly finality: Finality;

  /** Connections cache. */
  private static readonly connections: Partial<Record<Finality, Connection>> = {};

  constructor(finality: Finality = 'confirmed') {
    if (TestsHelper.connections[finality] === undefined) {
      TestsHelper.connections[finality] = new Connection(TestsHelper.LOCALHOST, finality);
    }
    this.connection = TestsHelper.connections[finality];
    this.finality = finality;
  }

  keypair = {
    generate: (): Keypair => Keypair.generate(),
    read: async (path: string): Promise<Keypair> =>
      this.keypair.from(JSON.parse(await fs.readFile(path, { encoding: 'utf8' }))),
    from: (bytes: number[]): Keypair => Keypair.fromSecretKey(Uint8Array.from(bytes)),
    several: <N extends number>(amount: N): Tuple<Keypair, N> =>
      Array.from({ length: amount }).map(Keypair.generate) as Tuple<Keypair, N>,
  };

  /** Waits that a transaction is confirmed. */
  async confirm(signature: TransactionSignature) {
    const latestBlockHash = await this.connection.getLatestBlockhash();

    return this.connection.confirmTransaction({
      signature,
      blockhash: latestBlockHash.blockhash,
      lastValidBlockHeight: latestBlockHash.lastValidBlockHeight,
    });
  }

  async sendAndConfirm(
    ixs: TransactionInstruction | Transaction | TransactionInstruction[],
    payer: Signer,
    ...signers: Signer[]
  ): Promise<TransactionSignature> {
    return sendAndConfirm(this.connection, ixs, payer, ...signers);
  }

  async getTransaction(
    signature: TransactionSignature | Promise<TransactionSignature>,
  ): Promise<VersionedTransactionResponse | null> {
    return this.connection.getTransaction(await signature, {
      commitment: this.finality,
      maxSupportedTransactionVersion: 1,
    });
  }

  /** Requests airdrop to an account or several ones. */
  async airdrop(to: PublicKey[]): Promise<void> {
    await Promise.all(to.map(async (account) =>
      this.confirm(await this.connection.requestAirdrop(account, 50 * LAMPORTS_PER_SOL))
    ));
  }
}

export function findPda(programId: PublicKey, seeds: Array<string | Uint8Array>) {
  const [address, bump] = PublicKey.findProgramAddressSync(
    seeds.map((seed) => {
      if (typeof seed === 'string') {
        return Buffer.from(seed);
      } else {
        return seed;
      }
    }),
    programId,
  );
  return {
    address,
    bump,
  };
}

export async function sendAndConfirm(
  connection: Connection,
  ixs: TransactionInstruction | Transaction | Array<TransactionInstruction>,
  payer: Signer,
  ...signers: Signer[]
): Promise<TransactionSignature> {
  const { value } = await connection.getLatestBlockhashAndContext();
  const tx = new Transaction({
    ...value,
    feePayer: payer.publicKey,
  }).add(...(Array.isArray(ixs) ? ixs : [ixs]));

  return sendAndConfirmTransaction(connection, tx, [payer, ...signers], {});
}

/** Helper allowing to abstract over the Wormhole configuration (network and addresses) */
export class WormholeContracts {
  readonly network = "Devnet";

  private core: PublicKey;

  constructor(name = "wormhole-core-v1") {
    const anchorCfg = toml.parse(fsSync.readFileSync("./Anchor.toml", "utf-8"));

    this.core = new PublicKey(
      anchorCfg.test.genesis.find((cfg: any) => cfg.name == name).address,
    );
  }

  get coreBridge(): PublicKey {
    return this.core;
  }

  get addresses(): Contracts {
    return {
      coreBridge: this.core.toString(),
    };
  }
}

export async function expectFailure(
  f: () => Promise<unknown>,
  handle: (error: Error) => void | Promise<void>
): Promise<void> {
  let result;
  try {
    result = await f();
  } catch (error) {
    if (error instanceof Error) {
      // In most cases the handler won't return a promise, but just in case.
      try {
        await handle(error);
      } catch (assertion) {
        throw new Error(`${assertion}
Original error: ${error.stack || error}`);
      }
      return;
    }

    throw error;
  }

  throw new Error(`Did not fail. Result: ${result}`);
}