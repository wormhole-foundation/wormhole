import { AptosAccount, AptosClient, Types } from "aptos";

export class AptosClientWrapper {
  client: AptosClient;

  constructor(client: AptosClient) {
    this.client = client;
  }

  executeEntryFunction = async (
    sender: AptosAccount,
    payload: Types.EntryFunctionPayload,
    opts?: Partial<Types.SubmitTransactionRequest>,
  ): Promise<string> => {
    // overwriting `max_gas_amount` default
    // rest of defaults are defined here: https://aptos-labs.github.io/ts-sdk-doc/classes/AptosClient.html#generateTransaction
    const customOpts = Object.assign(
      {
        max_gas_amount: "10000",
      },
      opts,
    );

    return (
      this.client
        // create raw transaction
        // TODO: compare `generateTransaction` flow with `generateBCSTransaction`
        .generateTransaction(sender.address(), payload, customOpts)
        // simulate transaction
        .then((rawTx) =>
          this.client
            .simulateTransaction(sender, rawTx)
            .then((sims) =>
              sims.forEach((tx) => {
                if (!tx.success) {
                  console.error(JSON.stringify(tx, null, 2));
                  throw new Error(`Transaction failed: ${tx.vm_status}`);
                }
              }),
            )
            .then((_) => rawTx),
        )
        // sign & submit transaction if simulation is successful
        .then((rawTx) => this.client.signTransaction(sender, rawTx))
        .then((signedTx) => this.client.submitTransaction(signedTx))
        .then(async (pendingTx) => {
          await this.client.waitForTransaction(pendingTx.hash);
          return pendingTx.hash;
        })
    );
  };
}
