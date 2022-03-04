//code examples

//https://morioh.com/p/195b602b4350

const cosmosjs = require("@cosmostation/cosmosjs");

const chainId = "cosmoshub-2";
const cosmos = cosmosjs.network(lcdUrl, chainId);

const mnemonic = "...";
cosmos.setPath("m/44'/118'/0'/0/0");
const address = cosmos.getAddress(mnemonic);
const ecpairPriv = cosmos.getECPairPriv(mnemonic);

cosmos.getAccounts(address).then((data) => {
  let stdSignMsg = cosmos.NewStdMsg({
    type: "cosmos-sdk/MsgSend",
    from_address: address,
    to_address: "cosmos18vhdczjut44gpsy804crfhnd5nq003nz0nf20v",
    amountDenom: "uatom",
    amount: 100000,
    feeDenom: "uatom",
    fee: 5000,
    gas: 200000,
    memo: "",
    account_number: data.value.account_number,
    sequence: data.value.sequence,
  });
});

const signedTx = cosmos.sign(stdSignMsg, ecpairPriv);
cosmos.broadcast(signedTx).then((response) => console.log(response));

let stdSignMsg = cosmos.NewStdMsg({
  type: "cosmos-sdk/MsgSend",
  from_address: address,
  to_address: "cosmos18vhdczjut44gpsy804crfhnd5nq003nz0nf20v",
  amountDenom: "uatom",
  amount: 1000000,
  feeDenom: "uatom",
  fee: 5000,
  gas: 200000,
  memo: "",
  account_number: data.value.account_number,
  sequence: data.value.sequence,
});
