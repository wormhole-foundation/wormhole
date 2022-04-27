import { chunks, importCoreWasm, ixFromRust } from "@certusone/wormhole-sdk";
import { sendAndConfirmTransactionsWithRetry } from "@certusone/wormhole-sdk/lib/esm/utils/solana";
import {
  Connection,
  Keypair,
  PublicKey,
  Transaction,
  TransactionInstruction,
} from "@solana/web3.js";

export async function postVaaWithRetry(
  connection: Connection,
  signTransaction: (transaction: Transaction) => Promise<Transaction>,
  bridge_id: string,
  payer: string,
  vaa: Buffer,
  maxRetries: number
) {
  const unsignedTransactions: Transaction[] = [];
  const signature_set = Keypair.generate();
  const instructions = await createVerifySignaturesInstructions(
    connection,
    bridge_id,
    payer,
    vaa,
    signature_set
  );
  const finalInstruction = await createPostVaaInstruction(
    bridge_id,
    payer,
    vaa,
    signature_set
  );
  if (!finalInstruction) {
    return Promise.reject("Failed to construct the transaction.");
  }

  //The verify signatures instructions can be batched into groups of 2 safely,
  //reducing the total number of transactions.
  const batchableChunks = chunks(instructions, 2);
  batchableChunks.forEach((chunk) => {
    let transaction;
    if (chunk.length === 1) {
      transaction = new Transaction().add(chunk[0]);
    } else {
      transaction = new Transaction().add(chunk[0], chunk[1]);
    }
    unsignedTransactions.push(transaction);
  });

  //the postVaa instruction can only execute after the verifySignature transactions have
  //successfully completed.
  const finalTransaction = new Transaction().add(finalInstruction);

  //The signature_set keypair also needs to sign the verifySignature transactions, thus a wrapper is needed.
  const partialSignWrapper = (transaction: Transaction) => {
    transaction.partialSign(signature_set);
    return signTransaction(transaction);
  };

  await sendAndConfirmTransactionsWithRetry(
    connection,
    partialSignWrapper,
    payer,
    unsignedTransactions,
    maxRetries
  );
  //While the signature_set is used to create the final instruction, it doesn't need to sign it.
  await sendAndConfirmTransactionsWithRetry(
    connection,
    signTransaction,
    payer,
    [finalTransaction],
    maxRetries
  );

  return Promise.resolve();
}

/*
  This returns an array of instructions required to verify the signatures of a VAA, and upload it to the blockchain.
  signature_set should be a new keypair, and also needs to partial sign the transaction when these instructions are submitted.
  */
export async function createVerifySignaturesInstructions(
  connection: Connection,
  bridge_id: string,
  payer: string,
  vaa: Buffer,
  signature_set: Keypair
): Promise<TransactionInstruction[]> {
  const output: TransactionInstruction[] = [];
  const {
    guardian_set_address,
    parse_guardian_set,
    parse_vaa,
    verify_signatures_ix,
  } = await importCoreWasm();
  const { guardian_set_index } = parse_vaa(new Uint8Array(vaa));
  let guardian_addr = new PublicKey(
    guardian_set_address(bridge_id, guardian_set_index)
  );
  let acc = await connection.getAccountInfo(guardian_addr);
  if (acc?.data === undefined) {
    return output;
  }
  let guardian_data = parse_guardian_set(new Uint8Array(acc?.data));

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
    output.push(ixs[0], ixs[1]);
  }
  return output;
}

/*
  This will return the postVaaInstruction. This should only be executed after the verifySignaturesInstructions have been executed.
  signatureSetKeypair should be the same keypair used for verifySignaturesInstructions, but does not need to partialSign the transaction
  when this instruction is submitted.
  */
export async function createPostVaaInstruction(
  bridge_id: string,
  payer: string,
  vaa: Buffer,
  signatureSetKeypair: Keypair
): Promise<TransactionInstruction> {
  const { post_vaa_ix } = await importCoreWasm();
  return ixFromRust(
    post_vaa_ix(bridge_id, payer, signatureSetKeypair.publicKey.toString(), vaa)
  );
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
