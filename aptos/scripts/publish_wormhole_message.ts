import { AptosAccount, TxnBuilderTypes, BCS, HexString, AptosClient } from "aptos";
import {aptosAccountObject} from "./constants";
export const NODE_URL = "http://0.0.0.0:8080/v1";

const client = new AptosClient(NODE_URL);

async function publishWormholeMessage(contractAddress: HexString, accountFrom: AptosAccount): Promise<string> {
    const scriptFunctionPayload = new TxnBuilderTypes.TransactionPayloadEntryFunction(
      TxnBuilderTypes.EntryFunction.natural(
        // Send using the sender example program
        `${contractAddress.toString()}::sender`,
        "send_message",
        [],
        [
         BCS.bcsSerializeBytes(Buffer.from("hi my name is bob")), // payload
        ]
      ),
    );
    const [{ sequence_number: sequenceNumber }, chainId] = await Promise.all([
      client.getAccount(accountFrom.address()),
      client.getChainId(),
    ]);
    const rawTxn = new TxnBuilderTypes.RawTransaction(
      TxnBuilderTypes.AccountAddress.fromHex(accountFrom.address()),
      BigInt(sequenceNumber),
      scriptFunctionPayload,
      BigInt(1000), //max gas to be used
      BigInt(1), //price per unit gas
      BigInt(Math.floor(Date.now() / 1000) + 10),
      new TxnBuilderTypes.ChainId(chainId),
    );

    const sim = await client.simulateTransaction(accountFrom, rawTxn);
    sim.forEach((tx) => {
      if (!tx.success) {
        console.error(JSON.stringify(tx, null, 2));
        throw new Error(`Transaction failed: ${tx.vm_status}`);
      }
    });
    const bcsTxn = AptosClient.generateBCSTransaction(accountFrom, rawTxn);
    const transactionRes = await client.submitSignedBCSTransaction(bcsTxn);

    return transactionRes.hash;
  }

  async function main(){
    let accountFrom = AptosAccount.fromAptosAccountObject(aptosAccountObject)
    let hash = await publishWormholeMessage(accountFrom.address(), accountFrom);
    console.log("tx hash: ", hash);
  }

  if (require.main === module) {
    main().then((resp) => console.log(resp));
  }


