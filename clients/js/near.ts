import { impossible, Payload } from "./vaa";
import { NETWORKS } from "./networks";
import { CONTRACTS } from "@certusone/wormhole-sdk";

const BN = require("bn.js");
const nearAPI = require("near-api-js");

export async function execute_near(
  payload: Payload,
  vaa: string,
  network: "MAINNET" | "TESTNET" | "DEVNET"
) {
  let n = NETWORKS[network]["near"];
  let contracts = CONTRACTS[network]["near"];

  let account: string;

  switch (payload.module) {
    case "Core":
      if (contracts.core === undefined) {
        throw new Error("Core bridge not supported yet for near");
      }
      account = "wormhole." + n.baseAccount;
      switch (payload.type) {
        case "GuardianSetUpgrade":
          console.log("Submitting new guardian set");
          break;
        case "ContractUpgrade":
          console.log("Upgrading core contract");
          break;
        default:
          impossible(payload);
      }
      break;
    case "NFTBridge":
      if (contracts.nft_bridge === undefined) {
        throw new Error("NFT bridge not supported yet for near");
      }
      account = "nft." + n.baseAccount;
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RegisterChain":
          console.log("Registering chain");
          break;
        case "Transfer":
          console.log("Completing transfer");
          break;
        default:
          impossible(payload);
      }
      break;
    case "TokenBridge":
      account = "token." + n.baseAccount;
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RegisterChain":
          console.log("Registering chain");
          break;
        case "Transfer":
          console.log("Completing transfer");
          break;
        case "AttestMeta":
          console.log("Creating wrapped token");
          break;
        case "TransferWithPayload":
          throw Error("Can't complete payload 3 transfer from CLI");
        default:
          impossible(payload);
          break;
      }
      break;
    default:
      impossible(payload);
  }

  let target_contract = account;

  let key = nearAPI.utils.KeyPair.fromString(n.key);

  let keyStore = new nearAPI.keyStores.InMemoryKeyStore();
  keyStore.setKey(n.networkId, account, key);

  let near = await nearAPI.connect({
    deps: {
      keyStore,
    },
    networkId: n.networkId,
    nodeUrl: n.rpc,
  });

  let nearAccount = new nearAPI.Account(near.connection, account);

  console.log("submitting vaa the first time");
  let result1 = await nearAccount.functionCall({
    contractId: target_contract,
    methodName: "submit_vaa",
    args: {
      vaa: vaa,
    },
    attachedDeposit: new BN("100000000000000000000000"),
    gas: new BN("300000000000000"),
  });

  // You have to feed a vaa twice into the contract (two submits),
  // The first time, it checks if it has been seen at all.
  // The second time, it executes.
  console.log("submitting vaa the second time");
  let result2 = await nearAccount.functionCall({
    contractId: target_contract,
    methodName: "submit_vaa",
    args: {
      vaa: vaa,
    },
    attachedDeposit: new BN("100000000000000000000000"),
    gas: new BN("300000000000000"),
  });

  let txHash = result1.transaction.hash + ":" + result2.transaction.hash;
  console.log("Hash: " + txHash);
}
