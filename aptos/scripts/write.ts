import { AptosAccount, TxnBuilderTypes, BCS, HexString, MaybeHexString, AptosClient, FaucetClient, AptosAccountObject } from "aptos";
import {aptosAccountObject} from "./constants";
export const NODE_URL = "http://localhost:8080/v1";
export const FAUCET_URL = "http://localhost:8081";

//<:!:section_2
//:!:>section_3
const client = new AptosClient(NODE_URL);


async function testInitWormholeState(contractAddress: HexString, accountFrom: AptosAccount): Promise<string> {
  const scriptFunctionPayload = new TxnBuilderTypes.TransactionPayloadEntryFunction(
    TxnBuilderTypes.EntryFunction.natural(
      `${contractAddress.toString()}::Wormhole`,
      "testInitWormholeState",
      [],
      []
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
    BigInt(1000),
    BigInt(1),
    BigInt(Math.floor(Date.now() / 1000) + 10),
    new TxnBuilderTypes.ChainId(chainId),
  );
  const bcsTxn = AptosClient.generateBCSTransaction(accountFrom, rawTxn);
  const transactionRes = await client.submitSignedBCSTransaction(bcsTxn);
  return transactionRes.hash;
}

async function initWormhole(contractAddress: HexString, accountFrom: AptosAccount): Promise<string> {
    const scriptFunctionPayload = new TxnBuilderTypes.TransactionPayloadEntryFunction(
      TxnBuilderTypes.EntryFunction.natural(
        `${contractAddress.toString()}::Wormhole`,
        "init",
        [],
        [
         BCS.bcsSerializeUint64(101),
         BCS.bcsSerializeUint64(202), 
         BCS.bcsSerializeBytes(Buffer.from("0x12323aaa11111aaaaaaa2")),
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
      BigInt(1000),
      BigInt(1),
      BigInt(Math.floor(Date.now() / 1000) + 10),
      new TxnBuilderTypes.ChainId(chainId),
    );
    
    const bcsTxn = AptosClient.generateBCSTransaction(accountFrom, rawTxn);
    const transactionRes = await client.submitSignedBCSTransaction(bcsTxn);
    
    return transactionRes.hash;
    //return new Promise((resolve, reject)=>resolve("foo"));
  }

async function testInit(contractAddress: HexString, accountFrom: AptosAccount){
    const scriptFunctionPayload = new TxnBuilderTypes.TransactionPayloadEntryFunction(
      TxnBuilderTypes.EntryFunction.natural(
        `${contractAddress.toString()}::Wormhole`,
        "testInit",
        [],
        [],
      ),
    );
    console.log("here1")
    const [{ sequence_number: sequenceNumber }, chainId] = await Promise.all([
      client.getAccount(accountFrom.address()),
      client.getChainId(),
    ]);
    console.log("here2")
    const rawTxn = new TxnBuilderTypes.RawTransaction(
      TxnBuilderTypes.AccountAddress.fromHex(accountFrom.address()),
      BigInt(sequenceNumber),
      scriptFunctionPayload,
      BigInt(1000),
      BigInt(1),
      BigInt(Math.floor(Date.now() / 1000) + 10),
      new TxnBuilderTypes.ChainId(chainId),
    );
    const bcsTxn = AptosClient.generateBCSTransaction(accountFrom, rawTxn);
    const transactionRes = await client.submitSignedBCSTransaction(bcsTxn);
    console.log(transactionRes);
    return transactionRes.hash;
  }


  //testSetChainId
  async function testSetChainId(contractAddress: HexString, accountFrom: AptosAccount){
    const scriptFunctionPayload = new TxnBuilderTypes.TransactionPayloadEntryFunction(
      TxnBuilderTypes.EntryFunction.natural(
        `${contractAddress.toString()}::Wormhole`,
        "testSetChainId",
        [],
        [],
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
      BigInt(1000),
      BigInt(1),
      BigInt(Math.floor(Date.now() / 1000) + 10),
      new TxnBuilderTypes.ChainId(chainId),
    );
    const bcsTxn = AptosClient.generateBCSTransaction(accountFrom, rawTxn);
    const transactionRes = await client.submitSignedBCSTransaction(bcsTxn);
    console.log(transactionRes);
    return transactionRes.hash;
  }

  async function testInitMessageHandles(contractAddress: HexString, accountFrom: AptosAccount){
    const scriptFunctionPayload = new TxnBuilderTypes.TransactionPayloadEntryFunction(
      TxnBuilderTypes.EntryFunction.natural(
        `${contractAddress.toString()}::Wormhole`,
        "testInitMessageHandles",
        [],
        [],
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
      BigInt(1000),
      BigInt(1),
      BigInt(Math.floor(Date.now() / 1000) + 10),
      new TxnBuilderTypes.ChainId(chainId),
    );
    const bcsTxn = AptosClient.generateBCSTransaction(accountFrom, rawTxn);
    const transactionRes = await client.submitSignedBCSTransaction(bcsTxn);
    console.log(transactionRes);
    return transactionRes.hash;
  }

  async function testDoNothing(contractAddress: HexString, accountFrom: AptosAccount){
    const scriptFunctionPayload = new TxnBuilderTypes.TransactionPayloadEntryFunction(
      TxnBuilderTypes.EntryFunction.natural(
        `${contractAddress.toString()}::Wormhole`,
        "doNothing",
        [],
        [],
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
      BigInt(1000),
      BigInt(1),
      BigInt(Math.floor(Date.now() / 1000) + 10),
      new TxnBuilderTypes.ChainId(chainId),
    );
    const bcsTxn = AptosClient.generateBCSTransaction(accountFrom, rawTxn);
    const transactionRes = await client.submitSignedBCSTransaction(bcsTxn);
    console.log(transactionRes);
    return transactionRes.hash;
  }
  
  async function main(){
    let accountFrom = AptosAccount.fromAptosAccountObject(aptosAccountObject)
    let accountAddress = accountFrom.address();//new HexString("277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b");
    console.log("account address: ", accountAddress);
    let hash = await initWormhole(accountAddress, accountFrom);
    //let hash = await testInit(accountAddress, accountFrom);
    //let hash = await testSetChainId(accountAddress, accountFrom);
    //let hash = await testInitMessageHandles(accountAddress, accountFrom);
    //let hash = await testInitWormholeState(accountAddress, accountFrom);
    //let hash = await testDoNothing(accountAddress, accountFrom);
    console.log("tx hash: ", hash);
  }

  if (require.main === module) {
    main().then((resp) => console.log(resp));
  }

  //<:!:section_7
