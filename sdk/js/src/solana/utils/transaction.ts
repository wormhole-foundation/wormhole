import {
  Transaction,
  Keypair,
  Connection,
  PublicKeyInitData,
  PublicKey,
  ConfirmOptions,
  RpcResponseAndContext,
  SignatureResult,
  TransactionSignature,
  Signer,
} from "@solana/web3.js";

/**
 * Object that holds list of unsigned {@link Transaction}s and {@link Keypair}s
 * required to sign for each transaction.
 */
export interface PreparedTransactions {
  unsignedTransactions: Transaction[];
  signers: Signer[];
}

export interface TransactionSignatureAndResponse {
  signature: TransactionSignature;
  response: RpcResponseAndContext<SignatureResult>;
}

/**
 * Resembles WalletContextState and Anchor's NodeWallet's signTransaction function signature
 */
export type SignTransaction = (
  transaction: Transaction
) => Promise<Transaction>;

/**
 *
 * @param payers
 * @returns
 */
export function signTransactionFactory(...payers: Signer[]): SignTransaction {
  return modifySignTransaction(
    (transaction: Transaction) => Promise.resolve(transaction),
    ...payers
  );
}

export function modifySignTransaction(
  signTransaction: SignTransaction,
  ...payers: Signer[]
): SignTransaction {
  return (transaction: Transaction) => {
    for (const payer of payers) {
      transaction.partialSign(payer);
    }
    return signTransaction(transaction);
  };
}

/**
 * Wrapper for {@link Keypair} resembling Solana web3 browser wallet
 */
export class NodeWallet {
  payer: Keypair;
  signTransaction: SignTransaction;

  constructor(payer: Keypair) {
    this.payer = payer;
    this.signTransaction = signTransactionFactory(this.payer);
  }

  static fromSecretKey(
    secretKey: Uint8Array,
    options?:
      | {
          skipValidation?: boolean | undefined;
        }
      | undefined
  ): NodeWallet {
    return new NodeWallet(Keypair.fromSecretKey(secretKey, options));
  }

  publicKey(): PublicKey {
    return this.payer.publicKey;
  }

  pubkey(): PublicKey {
    return this.publicKey();
  }

  key(): PublicKey {
    return this.publicKey();
  }

  toString(): string {
    return this.publicKey().toString();
  }

  keypair(): Keypair {
    return this.payer;
  }

  signer(): Signer {
    return this.keypair();
  }
}

/**
 * The transactions provided to this function should be ready to send.
 * This function will do the following:
 * 1. Add the {@param payer} as the feePayer and latest blockhash to the {@link Transaction}.
 * 2. Sign using {@param signTransaction}.
 * 3. Send raw transaction.
 * 4. Confirm transaction.
 */
export async function signSendAndConfirmTransaction(
  connection: Connection,
  payer: PublicKeyInitData,
  signTransaction: SignTransaction,
  unsignedTransaction: Transaction,
  options?: ConfirmOptions
): Promise<TransactionSignatureAndResponse> {
  const commitment = options?.commitment;
  const { blockhash, lastValidBlockHeight } =
    await connection.getLatestBlockhash(commitment);
  unsignedTransaction.recentBlockhash = blockhash;
  unsignedTransaction.feePayer = new PublicKey(payer);

  // Sign transaction, broadcast, and confirm
  const signed = await signTransaction(unsignedTransaction);
  const signature = await connection.sendRawTransaction(
    signed.serialize(),
    options
  );
  const response = await connection.confirmTransaction(
    {
      blockhash,
      lastValidBlockHeight,
      signature,
    },
    commitment
  );
  return { signature, response };
}

/**
 * @deprecated Please use {@link signSendAndConfirmTransaction} instead, which allows
 * retries to be configured in {@link ConfirmOptions}.
 *
 * The transactions provided to this function should be ready to send.
 * This function will do the following:
 * 1. Add the {@param payer} as the feePayer and latest blockhash to the {@link Transaction}.
 * 2. Sign using {@param signTransaction}.
 * 3. Send raw transaction.
 * 4. Confirm transaction.
 */
export async function sendAndConfirmTransactionsWithRetry(
  connection: Connection,
  signTransaction: SignTransaction,
  payer: string,
  unsignedTransactions: Transaction[],
  maxRetries: number = 0,
  options?: ConfirmOptions
): Promise<TransactionSignatureAndResponse[]> {
  if (unsignedTransactions.length == 0) {
    return Promise.reject("No transactions provided to send.");
  }

  const commitment = options?.commitment;

  let currentRetries = 0;
  const output: TransactionSignatureAndResponse[] = [];
  for (const transaction of unsignedTransactions) {
    while (currentRetries <= maxRetries) {
      try {
        const latest = await connection.getLatestBlockhash(commitment);
        transaction.recentBlockhash = latest.blockhash;
        transaction.feePayer = new PublicKey(payer);

        const signed = await signTransaction(transaction).catch((e) => null);
        if (signed === null) {
          return Promise.reject("Failed to sign transaction.");
        }

        const signature = await connection.sendRawTransaction(
          signed.serialize(),
          options
        );
        const response = await connection.confirmTransaction(
          {
            signature,
            ...latest,
          },
          commitment
        );
        output.push({ signature, response });
        break;
      } catch (e) {
        console.error(e);
        ++currentRetries;
      }
    }
    if (currentRetries > maxRetries) {
      return Promise.reject("Reached the maximum number of retries.");
    }
  }

  return Promise.resolve(output);
}
