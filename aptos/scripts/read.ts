import { AptosAccount, TxnBuilderTypes, BCS, HexString, MaybeHexString, AptosClient, FaucetClient, AptosAccountObject} from "aptos";
import {aptosAccountObject} from "./constants";

export const NODE_URL = "http://localhost:8080/v1";
export const FAUCET_URL = "http://localhost:8081";

//<:!:section_2
//:!:>section_3
const client = new AptosClient(NODE_URL);

async function getWormholeState(contractAddress: HexString, accountAddress: MaybeHexString): Promise<string> {
    try {
      const resource = await client.getAccountResource(
        accountAddress,
        `${contractAddress.toString()}::wormhole::WormholeState`,
      );
      let x = (resource as any).data;
      console.log("x: ", x);
      return (resource as any).data;//["message"];
    } catch (_) {
      return "";
    }
}

async function getResources(accountAddress: MaybeHexString): Promise<string> {
    try {
      const resources = await client.getAccountResources(
        accountAddress
      );
      console.log("getResources:resources: ", resources)
      let x = (resources as any).data;
      //console.log("x: ", x);
      return (resources as any).data;//["message"];
    } catch (_) {
      return "";
    }
}

async function getTransaction(hash: string) {
    try {
      const txs = await client.getTransactionByHash(hash);
      console.log("getTransactions:transactions: ", txs)
      return txs;
    } catch (_) {
      return "";
    }
}

async function getTransactions() {
    try {
      const txs = await client.getTransactions({ limit: 1000});
      console.log("getTransactions:transactions: ", txs)
      return txs;
    } catch (_) {
      return "";
    }
}

async function getAllTransactions() {
    let all_txs: any[] = new Array(0);
    let start = BigInt(0);
    let cur_txs = [];
    while (true){
        try {
            const txs = await client.getTransactions({ limit: 1000, start: start});
            if (txs.length==0){
                return all_txs;
            }
            console.log("getTransactions:transactions: ", txs)
            all_txs = all_txs.concat(txs as any[]);
            start = start + BigInt(1000);
        } catch (_) {
            return all_txs
        }
    }
    
}

  async function main(){
    let accountFrom = AptosAccount.fromAptosAccountObject(aptosAccountObject)
    let accountAddress = accountFrom.address();
    await getResources(accountAddress);
    //let txs = await getTransactions();
    //console.log("num txs: ", txs.length);
    //let state = await getWormholeState(accountAddress, accountAddress);
    //console.log(state)
    let tx = await getTransaction("0x520b91f14e8dd9a1c02b972e2bb4f6b7f6152b734b8d7114b9fa90a92707e62d");
    console.log("my tx is:", tx)
    //@ts-ignore
    //console.log("my tx changes: ", tx.changes[0].data, tx.changes[1].data)
    //@ts-ignore
    //console.log("data piece 0: ", tx.changes[0].data)
    //@ts-ignore
    //console.log("data piece 1: ", tx.changes[1].data)
}
  
  if (require.main === module) {
    main().then((resp) => console.log(resp));
  }

  //<:!:section_7


