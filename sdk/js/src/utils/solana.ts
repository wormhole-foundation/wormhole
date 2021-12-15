import { Connection, PublicKey, Transaction } from "@solana/web3.js";

/*
    The transactions provided to this function should be ready to be sent.
    This function will only add the feePayer and blockhash, and then sign, send, and confirm the transaction.
*/
export async function sendAndConfirmTransactionsWithRetry(
  connection: Connection,
  signTransaction: (transaction: Transaction) => Promise<Transaction>,
  payer: string,
  unsignedTransactions: Transaction[],
  maxRetries: number = 0
) {
  if (!(unsignedTransactions && unsignedTransactions.length)) {
    return Promise.reject("No transactions provided to send.");
  }
  let currentRetries = 0;
  let currentIndex = 0;
  const transactionReceipts = [];
  while (
    !(currentIndex >= unsignedTransactions.length) &&
    !(currentRetries > maxRetries)
  ) {
    let transaction = unsignedTransactions[currentIndex];
    let signed = null;
    try {
      const { blockhash } = await connection.getRecentBlockhash();
      transaction.recentBlockhash = blockhash;
      transaction.feePayer = new PublicKey(payer);
    } catch (e) {
      console.error(e);
      currentRetries++;
      //Behavior after this is undefined, so best just to restart and try again.
      continue;
    }
    try {
      signed = await signTransaction(transaction);
    } catch (e) {
      //Eject here because this is most likely an intentional rejection from the user, or a genuine unrecoverable failure.
      return Promise.reject("Failed to sign transaction.");
    }
    try {
      const txid = await connection.sendRawTransaction(signed.serialize());
      const receipt = await connection.confirmTransaction(txid);
      transactionReceipts.push(receipt);
      currentIndex++;
    } catch (e) {
      console.error(e);
      currentRetries++;
    }
  }

  if (currentRetries > maxRetries) {
    return Promise.reject("Reached the maximum number of retries.");
  } else {
    return Promise.resolve(transactionReceipts);
  }
}
