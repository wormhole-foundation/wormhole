import { AptosAccount, TxnBuilderTypes, BCS, AptosClient, FaucetClient } from "aptos";

// generate new account and print private key
const new_account = new AptosAccount();
let p = new_account.toPrivateKeyObject();
console.log("new account object: ", p);
