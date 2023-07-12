import {
  _submitVAAAlgorand,
  signSendAndConfirmAlgorand,
} from "@certusone/wormhole-sdk/lib/esm/algorand";
import {
  CONTRACTS,
  ChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { Account, Algodv2, mnemonicToSecretKey } from "algosdk";
import { NETWORKS } from "./consts";
import { Network } from "./utils";
import { Payload, impossible } from "./vaa";
import { transferFromAlgorand } from "@certusone/wormhole-sdk/lib/esm/token_bridge/transfer";
import { tryNativeToHexString } from "@certusone/wormhole-sdk/lib/esm/utils";

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
    case "WormholeRelayer":
      throw Error("Wormhole Relayer not supported on Algorand");
    default:
      target_contract = impossible(payload);
  }

  const target = BigInt(parseInt(target_contract));
  const CORE_ID = BigInt(parseInt(contracts.core));
  const algodClient = getClient(network, rpc);
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

export async function transferAlgorand(
  dstChain: ChainName,
  dstAddress: string,
  tokenAddress: string,
  amount: string,
  network: Network,
  rpc: string
) {
  const { key } = NETWORKS[network].algorand;
  if (!key) {
    throw Error(`No ${network} key defined for Algorand`);
  }
  const contracts = CONTRACTS[network].algorand;
  const client = getClient(network, rpc);
  const wallet: Account = mnemonicToSecretKey(key);
  const CORE_ID = BigInt(parseInt(contracts.core));
  const TOKEN_BRIDGE_ID = BigInt(parseInt(contracts.token_bridge));
  const recipient = tryNativeToHexString(dstAddress, dstChain);
  if (!recipient) {
    throw new Error("Failed to convert recipient address");
  }
  const assetId = tokenAddress === "native" ? BigInt(0) : BigInt(tokenAddress);
  const txs = await transferFromAlgorand(
    client,
    TOKEN_BRIDGE_ID,
    CORE_ID,
    wallet.addr,
    assetId,
    BigInt(amount),
    recipient,
    dstChain,
    BigInt(0)
  );
  const result = await signSendAndConfirmAlgorand(client, txs, wallet);
  console.log("Confirmed in round:", result["confirmed-round"]);
}

function getClient(network: Network, rpc: string) {
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
  const client = new Algodv2(
    ALGORAND_HOST.algodToken,
    ALGORAND_HOST.algodServer,
    ALGORAND_HOST.algodPort
  );
  return client;
}
