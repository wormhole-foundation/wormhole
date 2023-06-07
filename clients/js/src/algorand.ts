import {
  _submitVAAAlgorand,
  signSendAndConfirmAlgorand,
} from "@certusone/wormhole-sdk/lib/esm/algorand";
import { CONTRACTS } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { Account, Algodv2, mnemonicToSecretKey } from "algosdk";
import { NETWORKS } from "./consts";
import { Network } from "./utils";
import { Payload, impossible } from "./vaa";

export async function execute_algorand(
  payload: Payload,
  vaa: Uint8Array,
  network: Network
) {
  const chainName = "algorand";
  const { key, rpc } = NETWORKS[network][chainName];
  if (!key) {
    throw Error(`No ${network} key defined for Algorand`);
  }

  if (!rpc) {
    throw Error(`No ${network} rpc defined for Algorand`);
  }

  const contracts = CONTRACTS[network][chainName];
  console.log("contracts", contracts);
  const ALGORAND_HOST = {
    algodToken: "",
    algodServer: rpc,
    algodPort: "",
  };
  if (network === "DEVNET") {
    ALGORAND_HOST.algodToken =
      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
    ALGORAND_HOST.algodPort = "4001";
  }

  let target_contract: string;
  switch (payload.module) {
    case "Core": {
      if (!contracts.core) {
        throw new Error(
          `Core bridge address not defined for Algorand ${network}`
        );
      }

      target_contract = contracts.core;
      switch (payload.type) {
        case "GuardianSetUpgrade":
          console.log("Submitting new guardian set");
          break;
        case "ContractUpgrade":
          console.log("Upgrading core contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on algorand");
        default:
          impossible(payload);
      }

      break;
    }
    case "NFTBridge": {
      if (!contracts.nft_bridge) {
        // NOTE: this code can safely be removed once the algorand NFT bridge is
        // released, but it's fine for it to stay, as the condition will just be
        // skipped once 'contracts.nft_bridge' is defined
        throw new Error("NFT bridge not supported yet for Algorand");
      }

      target_contract = contracts.nft_bridge;
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on algorand");
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
        throw new Error(
          `Token bridge address not defined for Algorand ${network}`
        );
      }

      target_contract = contracts.token_bridge;
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on algorand");
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
    default:
      target_contract = impossible(payload);
  }

  const target = BigInt(parseInt(target_contract));
  const CORE_ID = BigInt(parseInt(contracts.core));
  const algodClient = new Algodv2(
    ALGORAND_HOST.algodToken,
    ALGORAND_HOST.algodServer,
    ALGORAND_HOST.algodPort
  );
  const algoWallet: Account = mnemonicToSecretKey(key);

  // Create transaction
  const txs = await _submitVAAAlgorand(
    algodClient,
    target,
    CORE_ID,
    vaa,
    algoWallet.addr
  );

  // Sign and send transaction
  const result = await signSendAndConfirmAlgorand(algodClient, txs, algoWallet);
  console.log("Confirmed in round:", result["confirmed-round"]);
}
