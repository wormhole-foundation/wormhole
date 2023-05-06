import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { calculateFee } from "@cosmjs/stargate";
import { getSigningCosmWasmClient } from "@sei-js/core";

import { CONTRACTS } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { NETWORKS } from "./networks";
import { Network } from "./utils";
import { impossible, Payload } from "./vaa";

export async function execute_sei(
  payload: Payload,
  vaa: Buffer,
  network: Network
) {
  const contracts = CONTRACTS[network].sei;
  const { rpc, key } = NETWORKS[network].sei;
  if (!key) {
    throw Error(`No ${network} key defined for NEAR`);
  }

  if (!rpc) {
    throw Error(`No ${network} rpc defined for NEAR`);
  }

  let target_contract: string;
  let execute_msg: object;
  switch (payload.module) {
    case "Core": {
      if (!contracts.core) {
        throw new Error(`Core bridge address not defined for Sei ${network}`);
      }

      target_contract = contracts.core;
      // sigh...
      execute_msg = {
        submit_v_a_a: {
          vaa: vaa.toString("base64"),
        },
      };
      switch (payload.type) {
        case "GuardianSetUpgrade":
          console.log("Submitting new guardian set");
          break;
        case "ContractUpgrade":
          console.log("Upgrading core contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on sei");
        default:
          impossible(payload);
      }

      break;
    }
    case "NFTBridge": {
      if (!contracts.nft_bridge) {
        // NOTE: this code can safely be removed once the sei NFT bridge is
        // released, but it's fine for it to stay, as the condition will just be
        // skipped once 'contracts.nft_bridge' is defined
        throw new Error("NFT bridge not supported yet for Sei");
      }

      target_contract = contracts.nft_bridge;
      execute_msg = {
        submit_vaa: {
          data: vaa.toString("base64"),
        },
      };
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on sei");
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
      if (!contracts.token_bridge) {
        throw new Error(`Token bridge address not defined for Sei ${network}`);
      }

      target_contract = contracts.token_bridge;
      execute_msg = {
        submit_vaa: {
          data: vaa.toString("base64"),
        },
      };
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on sei");
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
    }
    default:
      target_contract = impossible(payload);
      execute_msg = impossible(payload);
  }

  const wallet = await DirectSecp256k1HdWallet.fromMnemonic(key, {
    prefix: "sei",
  });
  const [account] = await wallet.getAccounts();
  const client = await getSigningCosmWasmClient(rpc, wallet);
  const fee = calculateFee(300000, "0.1usei");
  const result = await client.execute(
    account.address,
    target_contract,
    execute_msg,
    fee
  );

  console.log(`TX hash: ${result.transactionHash}`);
}
