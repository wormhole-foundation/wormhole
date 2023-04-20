import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { calculateFee } from "@cosmjs/stargate";
import { getSigningCosmWasmClient } from "@sei-js/core";

import { impossible, Payload } from "./vaa";
import { NETWORKS } from "./networks";
import { CONTRACTS } from "@certusone/wormhole-sdk/lib/cjs/utils/consts";

export async function execute_sei(
  payload: Payload,
  vaa: Buffer,
  network: "MAINNET" | "TESTNET" | "DEVNET",
) {
  let chain = "sei";
  let n = NETWORKS[network][chain];
  let contracts = CONTRACTS[network][chain];

  let target_contract: string;
  let execute_msg: object;

  switch (payload.module) {
    case "Core":
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
          throw new Error("RecoverChainId not supported on sei")
        default:
          impossible(payload);
      }
      break;
    case "NFTBridge":
      if (contracts.nft_bridge === undefined) {
        // NOTE: this code can safely be removed once the sei NFT bridge is
        // released, but it's fine for it to stay, as the condition will just be
        // skipped once 'contracts.nft_bridge' is defined
        throw new Error("NFT bridge not supported yet for sei");
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
          throw new Error("RecoverChainId not supported on sei")
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
          throw new Error("RecoverChainId not supported on sei")
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
      target_contract = impossible(payload);
      execute_msg = impossible(payload);
  }
    
  const wallet = await DirectSecp256k1HdWallet.fromMnemonic(n.key, { prefix: "sei" });
  const [ account ] = await wallet.getAccounts();
  const client = await getSigningCosmWasmClient(n.rpc, wallet);
  const fee = calculateFee(300000, "0.1usei");
  const result = await client.execute(
    account.address,
    target_contract,
    execute_msg,
    fee
  );

  console.log(`TX hash: ${result.transactionHash}`);
}
