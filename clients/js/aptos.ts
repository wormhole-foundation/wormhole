import { AptosAccount, TxnBuilderTypes, AptosClient, BCS } from "aptos";
import { NETWORKS } from "./networks";
import { impossible, Payload } from "./vaa";
import { CONTRACTS } from "@certusone/wormhole-sdk";
import { Bytes, Seq } from "aptos/dist/transaction_builder/bcs/types";

export async function execute_aptos(
  payload: Payload,
  vaa: Buffer,
  network: "MAINNET" | "TESTNET" | "DEVNET",
  contract: string | undefined,
) {
  const chain = "aptos";

  // turn VAA bytes into BCS format. That is, add a length prefix
  const serializer = new BCS.Serializer();
  serializer.serializeBytes(vaa);
  const bcsVAA = serializer.getBytes();

  switch (payload.module) {
    case "Core":
      contract = contract ?? CONTRACTS[network][chain]["core"];
      if (contract === undefined) {
        throw Error("core bridge contract is undefined")
      }
      switch (payload.type) {
        case "GuardianSetUpgrade":
          console.log("Submitting new guardian set")
          // TODO(csongor): implement this and test
          break
        case "ContractUpgrade":
          console.log("Upgrading core contract")
          callEntryFunc(network, `${contract}::contract_upgrade`, "submit_vaa", [bcsVAA]);
          break
        default:
          impossible(payload)
      }
      break
    case "NFTBridge":
      contract = contract ?? CONTRACTS[network][chain]["nft_bridge"];
      if (contract === undefined) {
        throw Error("nft bridge contract is undefined")
      }
      break
    case "TokenBridge":
      contract = contract ?? CONTRACTS[network][chain]["token_bridge"];
      if (contract === undefined) {
        throw Error("token bridge contract is undefined")
      }
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract")
          callEntryFunc(network, `${contract}::contract_upgrade`, "submit_vaa", [bcsVAA]);
          break
        case "RegisterChain":
          console.log("Registering chain")
          callEntryFunc(network, `${contract}::register_chain`, "submit_vaa", [bcsVAA]);
          break
        case "Transfer":
          console.log("Completing transfer")
          break
        case "AttestMeta":
          console.log("Creating wrapped token")
          break
        case "TransferWithPayload":
          throw Error("Can't complete payload 3 transfer from CLI")
        default:
          impossible(payload)
          break
      }
      break
    default:
      impossible(payload)
  }

}

export async function callEntryFunc(
  network: "MAINNET" | "TESTNET" | "DEVNET",
  module: string,
  func: string,
  args: Seq<Bytes>,
) {
  let key: string | undefined = NETWORKS[network]["aptos"].key;
  if (key === undefined) {
    throw new Error("No key for aptos");
  }
  const accountFrom = new AptosAccount(new Uint8Array(Buffer.from(key, "hex")));

  const client = new AptosClient(NETWORKS[network]["aptos"].rpc);
  const [{ sequence_number: sequenceNumber }, chainId] = await Promise.all([
    client.getAccount(accountFrom.address()),
    client.getChainId(),
  ]);

  const txPayload = new TxnBuilderTypes.TransactionPayloadEntryFunction(
    TxnBuilderTypes.EntryFunction.natural(
      module,
      func,
      [],
      args
    )
  );

  const rawTxn = new TxnBuilderTypes.RawTransaction(
    TxnBuilderTypes.AccountAddress.fromHex(accountFrom.address()),
    BigInt(sequenceNumber),
    txPayload,
    BigInt(10000), //max gas to be used. TODO(csongor): we could compute this from the simulation below...
    BigInt(1), //price per unit gas
    BigInt(Math.floor(Date.now() / 1000) + 10),
    new TxnBuilderTypes.ChainId(chainId),
  );

  // simulate transaction before submitting
  const sim = await client.simulateTransaction(accountFrom, rawTxn);
  sim.forEach((tx) => {
    if (!tx.success) {
      console.error(JSON.stringify(tx, null, 2));
      throw new Error(`Transaction failed: ${tx.vm_status}`);
    }
  });
  // simulation successful... let's do it
  const bcsTxn = AptosClient.generateBCSTransaction(accountFrom, rawTxn);
  const transactionRes = await client.submitSignedBCSTransaction(bcsTxn);

  return transactionRes.hash;
}
