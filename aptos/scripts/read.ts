import { AptosAccount, TxnBuilderTypes, BCS, HexString, MaybeHexString, AptosClient, FaucetClient, AptosAccountObject} from "aptos";
import {aptosAccountObject} from "./constants";

export const NODE_URL = "http://localhost:8080/v1";
export const FAUCET_URL = "http://localhost:8081";

//<:!:section_2
//:!:>section_3
const client = new AptosClient(NODE_URL);

async function getWormholeState(contractAddress: HexString, accountAddress: MaybeHexString): Promise<any> {
    try {
      const resource = await client.getAccountResource(
        accountAddress,
        `${contractAddress.toString()}::State::WormholeState`,
      );
      return resource;
    } catch (_) {
      return "";
    }
}

async function getResources(accountAddress: MaybeHexString): Promise<any>{
    try {
      const resources = await client.getAccountResources(
        accountAddress
      );
      return resources;
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

  async function main(){
    let accountFrom = AptosAccount.fromAptosAccountObject(aptosAccountObject)
    let accountAddress = accountFrom.address();

    //resources 
    let resources = await getResources(accountAddress);
    console.log("resources: ", resources);
    
    //get specific transaction
    let tx = await getTransaction("0xfbc4f4e03408d311c4d64e0e6447131c982947fdff46af5ed1850a74273e9e8a");
    console.log("my tx is:", tx)

    //@ts-ignore
    //console.log("my tx changes: ", tx.changes[0].data, tx.changes[1].data)

    let wormholeState = await getWormholeState(accountAddress, accountAddress);
    console.log("==========================< Wormhole State >==========================\n", wormholeState);
}
  
  if (require.main === module) {
    main().then((resp) => console.log(resp));
  }

  //<:!:section_7


