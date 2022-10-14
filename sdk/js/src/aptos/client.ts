import { AptosAccount, AptosClient, Types, TxnBuilderTypes } from "aptos";

export class AptosClientWrapper {
  client: AptosClient;

  constructor(client: AptosClient) {
    this.client = client;
  }

  executeEntryFunction = async (
    senderAddress: string,
    payload: Types.EntryFunctionPayload,
    opts?: Partial<Types.SubmitTransactionRequest>,
  ): Promise<TxnBuilderTypes.RawTransaction> => {
    // overwriting `max_gas_amount` default
    // rest of defaults are defined here: https://aptos-labs.github.io/ts-sdk-doc/classes/AptosClient.html#generateTransaction
    const customOpts = Object.assign(
      {
        max_gas_amount: "10000",
      },
      opts,
    );

    return this.client.generateTransaction(senderAddress, payload, customOpts);
  };
}

export const signAndSubmitTransaction = (
  client: AptosClient,
  sender: AptosAccount,
  rawTx: TxnBuilderTypes.RawTransaction,
): Promise<Types.UserTransaction> => {
  // simulate transaction
  return (
    client
      .simulateTransaction(sender, rawTx)
      .then((sims) =>
        sims.forEach((tx) => {
          if (!tx.success) {
            console.error(JSON.stringify(tx, null, 2));
            throw new Error(`Transaction failed: ${tx.vm_status}`);
          }
        }),
      )
      // sign & submit transaction if simulation is successful
      .then((_) => client.signTransaction(sender, rawTx))
      .then((signedTx) => client.submitTransaction(signedTx))
      .then(
        (pendingTx) =>
          client.waitForTransactionWithResult(pendingTx.hash) as Promise<Types.UserTransaction>,
      )
  );
};
