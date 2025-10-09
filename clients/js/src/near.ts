import BN from "bn.js";
import { Account, connect, KeyPair } from "near-api-js";
import { InMemoryKeyStore } from "near-api-js/lib/key_stores";
import { NETWORKS } from "./consts";
import { impossible, Payload } from "./vaa";
import {
  transferNearFromNear,
  transferTokenFromNear,
} from "@certusone/wormhole-sdk/lib/esm/token_bridge/transfer";
import {
  Chain,
  chainToChainId,
  contracts,
  Network,
} from "@wormhole-foundation/sdk-base";
import { tryNativeToUint8Array } from "./sdk/array";
import { castChainIdToOldSdk } from "./utils";

export function keyPairToImplicitAccount(keyPair: KeyPair): string {
  return Buffer.from(keyPair.getPublicKey().data).toString("hex");
}

export const execute_near = async (
  payload: Payload,
  vaa: string,
  network: Network
): Promise<void> => {
  const { rpc, key, networkId } = NETWORKS[network].Near;
  if (!key) {
    throw Error(`No ${network} key defined for NEAR`);
  }

  if (!rpc) {
    throw Error(`No ${network} rpc defined for NEAR`);
  }

  let target_contract: string;
  let numSubmits = 1;
  switch (payload.module) {
    case "Core": {
      const coreContract = contracts.coreBridge(network, "Near");
      if (!coreContract) {
        throw new Error(`Core bridge address not defined for NEAR ${network}`);
      }
      target_contract = coreContract;
      switch (payload.type) {
        case "GuardianSetUpgrade":
          console.log("Submitting new guardian set");
          break;
        case "ContractUpgrade":
          console.log("Upgrading core contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on near");
        default:
          impossible(payload);
      }
      break;
    }
    case "NFTBridge": {
      const nftContract = contracts.nftBridge.get(network, "Near");
      if (!nftContract) {
        throw new Error(`NFT bridge address not defined for NEAR ${network}`);
      }

      numSubmits = 2;
      target_contract = nftContract;
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on near");
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
    }
    case "TokenBridge": {
      const tbContract = contracts.tokenBridge(network, "Near");
      if (!tbContract) {
        throw new Error(`Token bridge address not defined for NEAR ${network}`);
      }

      numSubmits = 2;
      target_contract = tbContract;
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on near");
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
      }

      break;
    }
    case "WormholeRelayer":
      throw Error("Wormhole Relayer not supported on Near");
    default:
      impossible(payload);
  }

  const keyPair = KeyPair.fromString(key);
  const deployerAccount = keyPairToImplicitAccount(keyPair);
  const keyStore = new InMemoryKeyStore();
  keyStore.setKey(networkId, deployerAccount, keyPair);
  const near = await connect({
    keyStore,
    networkId,
    nodeUrl: rpc,
    headers: {},
  });
  const nearAccount = new Account(near.connection, deployerAccount);

  console.log("submitting vaa the first time");
  const result1 = await nearAccount.functionCall({
    contractId: target_contract,
    methodName: "submit_vaa",
    args: { vaa },
    attachedDeposit: new BN("100000000000000000000000"),
    gas: new BN("300000000000000"),
  });
  if (numSubmits <= 1) {
    console.log("Hash: " + result1.transaction.hash);
    return;
  }

  // You have to feed a vaa twice into the contract (two submits),
  // The first time, it checks if it has been seen at all.
  // The second time, it executes.
  console.log("submitting vaa the second time");
  const result2 = await nearAccount.functionCall({
    contractId: target_contract,
    methodName: "submit_vaa",
    args: { vaa },
    attachedDeposit: new BN("100000000000000000000000"),
    gas: new BN("300000000000000"),
  });
  const txHash = result1.transaction.hash + ":" + result2.transaction.hash;
  console.log("Hash: " + txHash);
};

export async function transferNear(
  dstChain: Chain,
  dstAddress: string,
  tokenAddress: string,
  amount: string,
  network: Network,
  rpc: string
) {
  const { key, networkId } = NETWORKS[network].Near;
  if (!key) {
    throw Error(`No ${network} key defined for NEAR`);
  }
  const core = contracts.coreBridge(network, "Near");
  const token_bridge = contracts.tokenBridge(network, "Near");
  if (core === undefined) {
    throw Error(`Unknown core contract on ${network} for NEAR`);
  }
  if (token_bridge === undefined) {
    throw Error(`Unknown token bridge contract on ${network} for NEAR`);
  }
  const keyPair = KeyPair.fromString(key);
  const deployerAccount = keyPairToImplicitAccount(keyPair);
  const keyStore = new InMemoryKeyStore();
  keyStore.setKey(networkId, deployerAccount, keyPair);
  const near = await connect({
    keyStore,
    networkId,
    nodeUrl: rpc,
    headers: {},
  });
  const nearAccount = new Account(near.connection, deployerAccount);
  if (tokenAddress === "native") {
    const msg = await transferNearFromNear(
      near.connection.provider,
      core,
      token_bridge,
      BigInt(amount),
      tryNativeToUint8Array(dstAddress, chainToChainId(dstChain)),
      castChainIdToOldSdk(chainToChainId(dstChain)),
      BigInt(0)
    );
    const result = await nearAccount.functionCall(msg);
    console.log(result.transaction.hash);
  } else {
    const msgs = await transferTokenFromNear(
      near.connection.provider,
      nearAccount.accountId,
      core,
      token_bridge,
      tokenAddress,
      BigInt(amount),
      tryNativeToUint8Array(dstAddress, chainToChainId(dstChain)),
      castChainIdToOldSdk(chainToChainId(dstChain)),
      BigInt(0)
    );
    for (const msg of msgs) {
      const result = await nearAccount.functionCall(msg);
      console.log(result.transaction.hash);
    }
  }
}
