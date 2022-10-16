import { AptosAccount, TxnBuilderTypes, AptosClient, BCS } from "aptos";
import { NETWORKS } from "./networks";
import { impossible, Payload } from "./vaa";
import { assertChain, ChainId, CONTRACTS } from "@certusone/wormhole-sdk";
import { Bytes, Seq } from "aptos/dist/transaction_builder/bcs/types";
import { TypeTag } from "aptos/dist/transaction_builder/aptos_types";
import { sha3_256 } from "js-sha3";
import { ethers } from "ethers";

export async function execute_aptos(
  payload: Payload,
  vaa: Buffer,
  network: "MAINNET" | "TESTNET" | "DEVNET",
  contract: string | undefined,
  rpc: string | undefined
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
          await callEntryFunc(network, rpc, `${contract}::guardian_set_upgrade`, "submit_vaa_entry", [], [bcsVAA]);
          break
        case "ContractUpgrade":
          console.log("Upgrading core contract")
          await callEntryFunc(network, rpc, `${contract}::contract_upgrade`, "submit_vaa_entry", [], [bcsVAA]);
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
          await callEntryFunc(network, rpc, `${contract}::contract_upgrade`, "submit_vaa_entry", [], [bcsVAA]);
          break
        case "RegisterChain":
          console.log("Registering chain")
          await callEntryFunc(network, rpc, `${contract}::register_chain`, "submit_vaa_entry", [], [bcsVAA]);
          break
        case "AttestMeta":
          console.log("Creating wrapped token")
          // Deploying a wrapped asset requires two transactions:
          // 1. Publish a new module under a resource account that defines a type T
          // 2. Initialise a new coin with that type T
          // These need to be done in separate transactions, becasue a
          // transaction that deploys a module cannot use that module
          //
          // Tx 1.
          await callEntryFunc(network, rpc, `${contract}::wrapped`, "create_wrapped_coin_type", [], [bcsVAA]);

          // We just deployed the module (notice the "wait" argument which makes
          // the previous step block until finality).
          // Now we're ready to do Tx 2. The module above got deployed to a new
          // resource account, which is seeded by the token bridge's address and
          // the origin information of the token. We can recompute this address
          // offline:
          let tokenAddress = payload.tokenAddress;
          let tokenChain = payload.tokenChain;
          assertChain(tokenChain);
          let wrappedContract = deriveWrappedAssetAddress(hex(contract), tokenChain, hex(tokenAddress));

          // Tx 2.
          console.log(`Deploying resource account ${wrappedContract}`);
          const token = new TxnBuilderTypes.TypeTagStruct(TxnBuilderTypes.StructTag.fromString(`${wrappedContract}::coin::T`));
          await callEntryFunc(network, rpc, `${contract}::wrapped`, "create_wrapped_coin", [token], [bcsVAA]);

          break
        case "Transfer":
          console.log("Completing transfer")
          await callEntryFunc(network, rpc, `${contract}::complete_transfer`, "submit_vaa_entry", [], [bcsVAA]);
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

export function deriveWrappedAssetAddress(
  token_bridge_address: Uint8Array, // 32 bytes
  origin_chain: ChainId,
  origin_address: Uint8Array, // 32 bytes
): string {
  let chain: Buffer = Buffer.alloc(2);
  chain.writeUInt16BE(origin_chain);
  return sha3_256(Buffer.concat([token_bridge_address, chain, Buffer.from("::", "ascii"), origin_address]));
}

export function deriveResourceAccount(
  deployer: Uint8Array, // 32 bytes
  seed: string,
) {
  // from https://github.com/aptos-labs/aptos-core/blob/25696fd266498d81d346fe86e01c330705a71465/aptos-move/framework/aptos-framework/sources/account.move#L90-L95
  let DERIVE_RESOURCE_ACCOUNT_SCHEME = Buffer.alloc(1);
  DERIVE_RESOURCE_ACCOUNT_SCHEME.writeUInt8(255);
  return sha3_256(Buffer.concat([deployer, Buffer.from(seed, "ascii"), DERIVE_RESOURCE_ACCOUNT_SCHEME]))
}

export async function callEntryFunc(
  network: "MAINNET" | "TESTNET" | "DEVNET",
  rpc: string | undefined,
  module: string,
  func: string,
  ty_args: Seq<TypeTag>,
  args: Seq<Bytes>,
) {
  let key: string | undefined = NETWORKS[network]["aptos"].key;
  if (key === undefined) {
    throw new Error("No key for aptos");
  }
  const accountFrom = new AptosAccount(new Uint8Array(Buffer.from(key, "hex")));
  let client: AptosClient;
  // if rpc arg is passed in, then override default rpc value for that network
  if (typeof rpc != 'undefined'){
    client = new AptosClient(rpc);
  } else {
    client = new AptosClient(NETWORKS[network]["aptos"].rpc);
  }
  const [{ sequence_number: sequenceNumber }, chainId] = await Promise.all([
    client.getAccount(accountFrom.address()),
    client.getChainId(),
  ]);

  const txPayload = new TxnBuilderTypes.TransactionPayloadEntryFunction(
    TxnBuilderTypes.EntryFunction.natural(
      module,
      func,
      ty_args,
      args
    )
  );

  const rawTxn = new TxnBuilderTypes.RawTransaction(
    TxnBuilderTypes.AccountAddress.fromHex(accountFrom.address()),
    BigInt(sequenceNumber),
    txPayload,
    BigInt(30000), //max gas to be used. TODO(csongor): we could compute this from the simulation below...
    BigInt(100), //price per unit gas TODO(csongor): we should get this dynamically
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

  await client.waitForTransaction(transactionRes.hash);
  return transactionRes.hash;
}

// strip the 0x prefix from a hex string
function hex(x: string): Buffer {
  return Buffer.from(ethers.utils.hexlify(x, { allowMissingPrefix: true }).substring(2), "hex");
}
