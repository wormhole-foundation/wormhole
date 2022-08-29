import { AptosAccount, TxnBuilderTypes, BCS, HexString, MaybeHexString, AptosClient, FaucetClient, AptosAccountObject } from "aptos";
import {aptosAccountObject} from "./constants";
export const NODE_URL = "http://0.0.0.0:8080";
export const FAUCET_URL = "http://localhost:8081";

const client = new AptosClient(NODE_URL);

async function initWormhole(contractAddress: HexString, accountFrom: AptosAccount): Promise<string> {
    const scriptFunctionPayload = new TxnBuilderTypes.TransactionPayloadEntryFunction(
      TxnBuilderTypes.EntryFunction.natural(
        `${contractAddress.toString()}::wormhole`,
        "init",
        [],
        [
         BCS.bcsSerializeUint64(101), //chain_id
         BCS.bcsSerializeUint64(202), //governance_chain_id
         BCS.bcsSerializeBytes(Buffer.from("0x12323aaa11111aaaaaaa2")), //governance contract address
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

    const bcsTxn = AptosClient.generateBCSTransaction(accountFrom, rawTxn);
    const transactionRes = await client.submitSignedBCSTransaction(bcsTxn);

    return transactionRes.hash;
  }

  async function main(){
    let accountFrom = AptosAccount.fromAptosAccountObject(aptosAccountObject)
    let accountAddress = accountFrom.address();//new HexString("277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b");
    console.log("account address: ", accountAddress);
    // TODO(csongor): the module is now published under the derived resource account
    let hash = await initWormhole(accountAddress, accountFrom);
    console.log("tx hash: ", hash);
  }

  if (require.main === module) {
    main().then((resp) => console.log(resp));
  }


