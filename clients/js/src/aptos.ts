import {
  CONTRACTS,
  ChainId,
  ChainName,
  assertChain,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { transferFromAptos } from "@certusone/wormhole-sdk/lib/esm/token_bridge/transfer";
import { AptosAccount, AptosClient, BCS, TxnBuilderTypes, Types } from "aptos";
import { ethers } from "ethers";
import { sha3_256 } from "js-sha3";
import { NETWORKS } from "./consts";
import { Network } from "./utils";
import { Payload, impossible } from "./vaa";
import { CHAINS, ensureHexPrefix } from "@certusone/wormhole-sdk";
import { TokenBridgeState } from "@certusone/wormhole-sdk/lib/esm/aptos/types";
import {
  generateSignAndSubmitEntryFunction,
  tryNativeToUint8Array,
} from "@certusone/wormhole-sdk/lib/esm/utils";

export async function execute_aptos(
  payload: Payload,
  vaa: Buffer,
  network: Network,
  contract: string | undefined,
  rpc: string | undefined
) {
  const chain = "aptos";

  // turn VAA bytes into BCS format. That is, add a length prefix
  const serializer = new BCS.Serializer();
  serializer.serializeBytes(vaa);
  const bcsVAA = serializer.getBytes();

  switch (payload.module) {
    case "Core": {
      contract = contract ?? CONTRACTS[network][chain]["core"];
      if (contract === undefined) {
        throw Error("core bridge contract is undefined");
      }

      switch (payload.type) {
        case "GuardianSetUpgrade":
          console.log("Submitting new guardian set");
          await callEntryFunc(
            network,
            rpc,
            `${contract}::guardian_set_upgrade`,
            "submit_vaa_entry",
            [],
            [bcsVAA]
          );
          break;
        case "ContractUpgrade":
          console.log("Upgrading core contract");
          await callEntryFunc(
            network,
            rpc,
            `${contract}::contract_upgrade`,
            "submit_vaa_entry",
            [],
            [bcsVAA]
          );
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on aptos");
        default:
          impossible(payload);
      }

      break;
    }
    case "NFTBridge": {
      contract = contract ?? CONTRACTS[network][chain]["nft_bridge"];
      if (contract === undefined) {
        throw Error("nft bridge contract is undefined");
      }

      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          await callEntryFunc(
            network,
            rpc,
            `${contract}::contract_upgrade`,
            "submit_vaa_entry",
            [],
            [bcsVAA]
          );
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on aptos");
        case "RegisterChain":
          console.log("Registering chain");
          await callEntryFunc(
            network,
            rpc,
            `${contract}::register_chain`,
            "submit_vaa_entry",
            [],
            [bcsVAA]
          );
          break;
        case "Transfer": {
          console.log("Completing transfer");
          await callEntryFunc(
            network,
            rpc,
            `${contract}::complete_transfer`,
            "submit_vaa_entry",
            [],
            [bcsVAA]
          );
          break;
        }
        default:
          impossible(payload);
      }

      break;
    }
    case "TokenBridge": {
      contract = contract ?? CONTRACTS[network][chain]["token_bridge"];
      if (contract === undefined) {
        throw Error("token bridge contract is undefined");
      }

      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          await callEntryFunc(
            network,
            rpc,
            `${contract}::contract_upgrade`,
            "submit_vaa_entry",
            [],
            [bcsVAA]
          );
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on aptos");
        case "RegisterChain":
          console.log("Registering chain");
          await callEntryFunc(
            network,
            rpc,
            `${contract}::register_chain`,
            "submit_vaa_entry",
            [],
            [bcsVAA]
          );
          break;
        case "AttestMeta": {
          console.log("Creating wrapped token");
          // Deploying a wrapped asset requires two transactions:
          // 1. Publish a new module under a resource account that defines a type T
          // 2. Initialise a new coin with that type T
          // These need to be done in separate transactions, because a
          // transaction that deploys a module cannot use that module
          //
          // Tx 1.
          try {
            await callEntryFunc(
              network,
              rpc,
              `${contract}::wrapped`,
              "create_wrapped_coin_type",
              [],
              [bcsVAA]
            );
          } catch (e) {
            console.log("this one already happened (probably)");
          }

          // We just deployed the module (notice the "wait" argument which makes
          // the previous step block until finality).
          // Now we're ready to do Tx 2. The module above got deployed to a new
          // resource account, which is seeded by the token bridge's address and
          // the origin information of the token. We can recompute this address
          // offline:
          const tokenAddress = payload.tokenAddress;
          const tokenChain = payload.tokenChain;
          assertChain(tokenChain);
          let wrappedContract = deriveWrappedAssetAddress(
            hex(contract),
            tokenChain,
            hex(tokenAddress)
          );

          // Tx 2.
          console.log(`Deploying resource account ${wrappedContract}`);
          let token = new TxnBuilderTypes.TypeTagStruct(
            TxnBuilderTypes.StructTag.fromString(`${wrappedContract}::coin::T`)
          );
          await callEntryFunc(
            network,
            rpc,
            `${contract}::wrapped`,
            "create_wrapped_coin",
            [token],
            [bcsVAA]
          );

          break;
        }
        case "Transfer": {
          console.log("Completing transfer");
          // TODO: only handles wrapped assets for now
          const tokenAddress = payload.tokenAddress;
          const tokenChain = payload.tokenChain;
          assertChain(tokenChain);
          let wrappedContract = deriveWrappedAssetAddress(
            hex(contract),
            tokenChain,
            hex(tokenAddress)
          );
          const token = new TxnBuilderTypes.TypeTagStruct(
            TxnBuilderTypes.StructTag.fromString(`${wrappedContract}::coin::T`)
          );
          await callEntryFunc(
            network,
            rpc,
            `${contract}::complete_transfer`,
            "submit_vaa_and_register_entry",
            [token],
            [bcsVAA]
          );
          break;
        }
        case "TransferWithPayload":
          throw Error("Can't complete payload 3 transfer from CLI");
        default:
          impossible(payload);
      }

      break;
    }
    case "WormholeRelayer":
      throw Error("Wormhole Relayer not supported on Aptos");
    default:
      impossible(payload);
  }
}

export async function transferAptos(
  dstChain: ChainName,
  dstAddress: string,
  tokenAddress: string,
  amount: string,
  network: Network,
  rpc: string
) {
  const { key } = NETWORKS[network].aptos;
  if (!key) {
    throw new Error("No key for aptos");
  }
  rpc = rpc ?? NETWORKS[network].aptos.rpc;
  if (!rpc) {
    throw new Error("No rpc for aptos");
  }
  const { token_bridge } = CONTRACTS[network].aptos;
  if (!token_bridge) {
    throw new Error("token bridge contract is undefined");
  }
  const account = new AptosAccount(new Uint8Array(Buffer.from(key, "hex")));
  const client = new AptosClient(rpc);
  const transferPayload = transferFromAptos(
    token_bridge,
    tokenAddress === "native" ? "0x1::aptos_coin::AptosCoin" : tokenAddress,
    amount,
    dstChain,
    tryNativeToUint8Array(dstAddress, dstChain)
  );
  const tx = (await generateSignAndSubmitEntryFunction(
    client,
    account,
    transferPayload
  )) as Types.UserTransaction;
  await client.waitForTransaction(tx.hash);
  console.log(`hash: ${tx.hash}`);
}

export function deriveWrappedAssetAddress(
  token_bridge_address: Uint8Array, // 32 bytes
  origin_chain: ChainId,
  origin_address: Uint8Array // 32 bytes
): string {
  let chain: Buffer = Buffer.alloc(2);
  chain.writeUInt16BE(origin_chain);
  if (origin_address.length != 32) {
    throw new Error(`${origin_address}`);
  }
  // from https://github.com/aptos-labs/aptos-core/blob/25696fd266498d81d346fe86e01c330705a71465/aptos-move/framework/aptos-framework/sources/account.move#L90-L95
  let DERIVE_RESOURCE_ACCOUNT_SCHEME = Buffer.alloc(1);
  DERIVE_RESOURCE_ACCOUNT_SCHEME.writeUInt8(255);
  return sha3_256(
    Buffer.concat([
      token_bridge_address,
      chain,
      Buffer.from("::", "ascii"),
      origin_address,
      DERIVE_RESOURCE_ACCOUNT_SCHEME,
    ])
  );
}

export function deriveResourceAccount(
  deployer: Uint8Array, // 32 bytes
  seed: string
): string {
  // from https://github.com/aptos-labs/aptos-core/blob/25696fd266498d81d346fe86e01c330705a71465/aptos-move/framework/aptos-framework/sources/account.move#L90-L95
  let DERIVE_RESOURCE_ACCOUNT_SCHEME = Buffer.alloc(1);
  DERIVE_RESOURCE_ACCOUNT_SCHEME.writeUInt8(255);
  return sha3_256(
    Buffer.concat([
      deployer,
      Buffer.from(seed, "ascii"),
      DERIVE_RESOURCE_ACCOUNT_SCHEME,
    ])
  );
}

export async function callEntryFunc(
  network: "MAINNET" | "TESTNET" | "DEVNET",
  rpc: string | undefined,
  module: string,
  func: string,
  ty_args: BCS.Seq<TxnBuilderTypes.TypeTag>,
  args: BCS.Seq<BCS.Bytes>
): Promise<string> {
  let key: string | undefined = NETWORKS[network]["aptos"].key;
  if (key === undefined) {
    throw new Error("No key for aptos");
  }
  const accountFrom = new AptosAccount(new Uint8Array(Buffer.from(key, "hex")));
  let client: AptosClient;
  // if rpc arg is passed in, then override default rpc value for that network
  if (typeof rpc != "undefined") {
    client = new AptosClient(rpc);
  } else {
    client = new AptosClient(NETWORKS[network]["aptos"].rpc);
  }
  const [{ sequence_number: sequenceNumber }, chainId] = await Promise.all([
    client.getAccount(accountFrom.address()),
    client.getChainId(),
  ]);

  const txPayload = new TxnBuilderTypes.TransactionPayloadEntryFunction(
    TxnBuilderTypes.EntryFunction.natural(module, func, ty_args, args)
  );

  const rawTxn = new TxnBuilderTypes.RawTransaction(
    TxnBuilderTypes.AccountAddress.fromHex(accountFrom.address()),
    BigInt(sequenceNumber),
    txPayload,
    BigInt(100000), //max gas to be used. TODO(csongor): we could compute this from the simulation below...
    BigInt(100), //price per unit gas TODO(csongor): we should get this dynamically
    BigInt(Math.floor(Date.now() / 1000) + 10),
    new TxnBuilderTypes.ChainId(chainId)
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
  return Buffer.from(
    ethers.utils.hexlify(x, { allowMissingPrefix: true }).substring(2),
    "hex"
  );
}

export async function queryRegistrationsAptos(
  network: Network,
  module: "Core" | "NFTBridge" | "TokenBridge"
): Promise<Object> {
  const n = NETWORKS[network]["aptos"];
  const client = new AptosClient(n.rpc);
  const contracts = CONTRACTS[network]["aptos"];
  let stateObjectId: string | undefined;

  switch (module) {
    case "TokenBridge":
      stateObjectId = contracts.token_bridge;
      if (stateObjectId === undefined) {
        throw Error(`Unknown token bridge contract on ${network} for Aptos`);
      }
      break;
    default:
      throw new Error(`Invalid module: ${module}`);
  }

  stateObjectId = ensureHexPrefix(stateObjectId);
  const state = (
    await client.getAccountResource(
      stateObjectId,
      `${stateObjectId}::state::State`
    )
  ).data as TokenBridgeState;

  const handle = state.registered_emitters.handle;

  // Query the bridge registration for all the chains in parallel.
  const registrations: string[][] = await Promise.all(
    Object.entries(CHAINS)
      .filter(([cname, _]) => cname !== "aptos" && cname !== "unset")
      .map(async ([cname, cid]) => [
        cname,
        await (async () => {
          let result = null;
          try {
            result = await client.getTableItem(handle, {
              key_type: "u64",
              value_type: "vector<u8>",
              key: cid.toString(),
            });
          } catch {
            // Not logging anything because a chain not registered returns an error.
          }

          return result;
        })(),
      ])
  );

  const results: { [key: string]: string } = {};
  for (let [cname, queryResponse] of registrations) {
    if (queryResponse) {
      results[cname] = queryResponse;
    }
  }
  return results;
}
