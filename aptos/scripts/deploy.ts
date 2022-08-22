import { AptosAccount, TxnBuilderTypes, BCS, HexString, AptosClient, FaucetClient } from "aptos";

export const NODE_URL = "http://127.0.0.1:8080";
export const FAUCET_URL = "http://127.0.0.1:8000";

const client = new AptosClient(NODE_URL);
const faucetClient = new FaucetClient(NODE_URL, FAUCET_URL);

/** Publish a new module to the blockchain within the specified account */
export async function publishModule(accountFrom: AptosAccount, moduleHex: string): Promise<string> {
  const moduleBundlePayload = new TxnBuilderTypes.TransactionPayloadModuleBundle(
    new TxnBuilderTypes.ModuleBundle([new TxnBuilderTypes.Module(new HexString(moduleHex).toUint8Array())]),
  );

  const [{ sequence_number: sequenceNumber }, chainId] = await Promise.all([
    client.getAccount(accountFrom.address()),
    client.getChainId(),
  ]);

  const rawTxn = new TxnBuilderTypes.RawTransaction(
    TxnBuilderTypes.AccountAddress.fromHex(accountFrom.address()),
    BigInt(sequenceNumber),
    moduleBundlePayload,
    BigInt(1000),
    BigInt(1),
    BigInt(Math.floor(Date.now() / 1000) + 10),
    new TxnBuilderTypes.ChainId(chainId),
  );

  const bcsTxn = AptosClient.generateBCSTransaction(accountFrom, rawTxn);
  const transactionRes = await client.submitSignedBCSTransaction(bcsTxn);

  return transactionRes.hash;
}

// export async function main(){
//     const alice = new AptosAccount();
//     publishModule(alice, )
// }

// if (require.main === module) {
//     main().then((resp) => console.log(resp));
//   }

