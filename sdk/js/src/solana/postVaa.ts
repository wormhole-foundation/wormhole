import {
  Connection,
  Keypair,
  PublicKey,
  Transaction,
  TransactionInstruction,
} from "@solana/web3.js";
import { chunks } from "..";
import { sendAndConfirmTransactionsWithRetry } from "../utils/solana";
import { ixFromRust } from "./rust";
import { importCoreWasm } from "./wasm";

/**
 *
 * Post a VAA, retrying transactions `maxRetries` number of times in case they
 * fail.
 *
 * The `signTransaction` argument is a function that (partially) signs a
 * transaction.
 *
 * In a web3 context when you have some `wallet` [[WalletContextState]]
 * available, this argument is typically just
 *
 * ```typescript
 * wallet.signTransaction
 * ```
 *
 * If instead you have some private key `signer`, you can just construct a
 * function like so:
 *
 * ```typescript
 * async (tx) => {
 *   tx.partialSign(signer)
 *   return tx
 * }
 * ```
 */
export async function postVaaWithRetry(
  connection: Connection,
  signTransaction: (transaction: Transaction) => Promise<Transaction>,
  bridge_id: PublicKey | string,
  payer: PublicKey | string,
  vaa: Buffer,
  maxRetries: number
) {
  // The signature set account is an ordinary account (as opposed to a PDA which
  // is owned by a program), so it can only be initialised when the transaction
  // is signed with the corresponding private key. We create a new keypair here,
  // and sign the transactions with it so the contract can allocate the account
  // to store the signatures.
  const signature_set = Keypair.generate();
  const verify_signatures_instructions = await createVerifySignaturesInstructions(
    connection,
    bridge_id,
    payer,
    vaa,
    signature_set.publicKey
  );
  const post_vaa_instruction = await createPostVaaInstruction(
    bridge_id,
    payer,
    vaa,
    signature_set.publicKey
  );
  if (!post_vaa_instruction) {
    return Promise.reject("Failed to construct the post VAA instruction.");
  }

  //The verify signatures instructions can be batched into groups of 2 safely,
  //reducing the total number of transactions.
  const verify_signatures_txs: Transaction[] =
    chunks(verify_signatures_instructions, 2).flatMap((chunk) => new Transaction().add(...chunk));

  //the postVaa instruction can only execute after the verifySignature transactions have
  //successfully completed.
  const post_vaa_tx = new Transaction().add(post_vaa_instruction);

  //The signature_set keypair also needs to sign the verifySignature transactions, thus a wrapper is needed.
  const partialSignWrapper = (transaction: Transaction) => {
    transaction.partialSign(signature_set);
    return signTransaction(transaction);
  };

  await sendAndConfirmTransactionsWithRetry(
    connection,
    partialSignWrapper,
    payer,
    verify_signatures_txs,
    maxRetries
  );
  //While the signature_set is used to create the final instruction, it doesn't need to sign it.
  await sendAndConfirmTransactionsWithRetry(
    connection,
    signTransaction,
    payer,
    [post_vaa_tx],
    maxRetries
  );

  return Promise.resolve();
}

/**
 *
 * Returns an array of instructions required to verify the signatures of a
 * VAA, and upload it to the blockchain.  signature_set should be a new public
 * key, and also needs to partial sign the transaction when these instructions
 * are submitted.
 *
 * @throws if guardian set account doesn't exist
 */
export async function createVerifySignaturesInstructions(
  connection: Connection,
  bridge_id: PublicKey | string,
  payer: PublicKey | string,
  vaa: Buffer,
  signature_set: PublicKey
): Promise<TransactionInstruction[]> {
  const {
    guardian_set_address,
    parse_guardian_set,
    parse_vaa,
    verify_signatures_ix,
  } = await importCoreWasm();
  const { guardian_set_index } = parse_vaa(new Uint8Array(vaa));
  let guardian_addr = new PublicKey(
    guardian_set_address(coalescePubkeyString(bridge_id), guardian_set_index)
  );
  let acc = await connection.getAccountInfo(guardian_addr);
  if (acc === null) {
    throw Error("Could not fetch guardian set account")
  }
  let guardian_data = parse_guardian_set(new Uint8Array(acc.data));

  let txs: any[][] = verify_signatures_ix(
    coalescePubkeyString(bridge_id),
    coalescePubkeyString(payer),
    guardian_set_index,
    guardian_data,
    signature_set.toString(),
    vaa
  );
  return txs.flatMap(tx => tx.map(ixFromRust));
}

/*
This will return the postVaaInstruction. This should only be executed after the verifySignaturesInstructions have been executed.
signature_set should be the same pbukey used for verifySignaturesInstructions, but does not need to sign the transaction
when this instruction is submitted.
*/
export async function createPostVaaInstruction(
  bridge_id: PublicKey | string,
  payer: PublicKey | string,
  vaa: Buffer,
  signature_set: PublicKey
): Promise<TransactionInstruction> {
  const { post_vaa_ix } = await importCoreWasm();
  return ixFromRust(
    post_vaa_ix(coalescePubkeyString(bridge_id), coalescePubkeyString(payer), signature_set.toString(), vaa)
  );
}

function coalescePubkeyString(k: PublicKey | string): string {
  return k instanceof PublicKey ? k.toString() : k
}

/*
  @deprecated
  Instead, either use postVaaWithRetry or create, sign, and send the verifySignaturesInstructions & postVaaInstruction yourself.

  This function is equivalent to a postVaaWithRetry with a maxRetries of 0.
*/
export async function postVaa(
  connection: Connection,
  signTransaction: (transaction: Transaction) => Promise<Transaction>,
  bridge_id: string,
  payer: string,
  vaa: Buffer
) {
  const {
    guardian_set_address,
    parse_guardian_set,
    parse_vaa,
    post_vaa_ix,
    verify_signatures_ix,
  } = await importCoreWasm();
  const { guardian_set_index } = parse_vaa(new Uint8Array(vaa));
  let guardian_addr = new PublicKey(
    guardian_set_address(bridge_id, guardian_set_index)
  );
  let acc = await connection.getAccountInfo(guardian_addr);
  if (acc?.data === undefined) {
    return;
  }
  let guardian_data = parse_guardian_set(new Uint8Array(acc?.data));

  let signature_set = Keypair.generate();
  let txs = verify_signatures_ix(
    bridge_id,
    payer,
    guardian_set_index,
    guardian_data,
    signature_set.publicKey.toString(),
    vaa
  );
  // Add transfer instruction to transaction
  for (let tx of txs) {
    let ixs: Array<TransactionInstruction> = tx.map((v: any) => {
      return ixFromRust(v);
    });
    let transaction = new Transaction().add(...ixs);
    const { blockhash } = await connection.getRecentBlockhash();
    transaction.recentBlockhash = blockhash;
    transaction.feePayer = new PublicKey(payer);
    transaction.partialSign(signature_set);

    // Sign transaction, broadcast, and confirm
    const signed = await signTransaction(transaction);
    const txid = await connection.sendRawTransaction(signed.serialize());
    await connection.confirmTransaction(txid);
  }

  let ix = ixFromRust(
    post_vaa_ix(bridge_id, payer, signature_set.publicKey.toString(), vaa)
  );
  let transaction = new Transaction().add(ix);
  const { blockhash } = await connection.getRecentBlockhash();
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payer);

  const signed = await signTransaction(transaction);
  const txid = await connection.sendRawTransaction(signed.serialize());
  await connection.confirmTransaction(txid);
}
