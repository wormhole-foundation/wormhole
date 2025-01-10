import * as anchor from "@coral-xyz/anchor";

export async function getTransactionDetails(
  tx: string
): Promise<anchor.web3.VersionedTransactionResponse> {
  let txDetails: anchor.web3.VersionedTransactionResponse | null = null;
  while (!txDetails) {
    txDetails = await anchor.getProvider().connection.getTransaction(tx, {
      maxSupportedTransactionVersion: 0,
      commitment: "confirmed",
    });
  }
  return txDetails;
}

export async function logCostAndCompute(method: string, tx: string) {
  const SOL_PRICE = 217.54; // 2025-01-03
  const txDetails = await getTransactionDetails(tx);
  const lamports =
    txDetails.meta.preBalances[0] - txDetails.meta.postBalances[0];
  const sol = lamports / 1_000_000_000;
  console.log(
    `${method}: lamports ${lamports} SOL ${sol}, $${sol * SOL_PRICE}, CU ${
      txDetails.meta.computeUnitsConsumed
    }, tx https://explorer.solana.com/tx/${tx}?cluster=custom&customUrl=http%3A%2F%2Flocalhost%3A8899`
  );
}
