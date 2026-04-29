import {
  _submitVAAAlgorand,
  signSendAndConfirmAlgorand,
} from "@certusone/wormhole-sdk/lib/esm/algorand";
import { Account, Algodv2, mnemonicToSecretKey } from "algosdk";
import { NETWORKS } from "./consts";
import { Payload, impossible } from "./vaa";
import { transferFromAlgorand } from "@certusone/wormhole-sdk/lib/esm/token_bridge/transfer";
import { tryNativeToHexString } from "./sdk/array";
import {
  Chain,
  chainToChainId,
  contracts,
  Network,
  toChainId,
} from "@wormhole-foundation/sdk-base";

export async function execute_algorand(
  payload: Payload,
  vaa: Uint8Array,
  network: Network
) {
  const chain: Chain = "Algorand";
  const { key, rpc } = NETWORKS[network][chain];
  if (!key) {
    throw Error(`No ${network} key defined for Algorand`);
  }

  if (!rpc) {
    throw Error(`No ${network} rpc defined for Algorand`);
  }

  const coreContract = contracts.coreBridge.get(network, chain);
  if (!coreContract) {
    throw new Error(`Core bridge address not defined for Algorand ${network}`);
  }

  let target_contract: string;
  switch (payload.module) {
    case "Core": {
      target_contract = coreContract;
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
      const nftContract = contracts.nftBridge.get(network, chain);
      if (!nftContract) {
        throw new Error("NFT bridge not supported yet for Algorand");
      }

      target_contract = nftContract;
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
      const tbContract = contracts.tokenBridge.get(network, chain);
      if (!tbContract) {
        throw new Error(
          `Token bridge address not defined for Algorand ${network}`
        );
      }

      target_contract = tbContract;
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
  const CORE_ID = BigInt(parseInt(coreContract));
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
  dstChain: Chain,
  dstAddress: string,
  tokenAddress: string,
  amount: string,
  network: Network,
  rpc: string
) {
  const { key } = NETWORKS[network].Algorand;
  if (!key) {
    throw Error(`No ${network} key defined for Algorand`);
  }
  const client = getClient(network, rpc);
  const wallet: Account = mnemonicToSecretKey(key);
  const CORE_ID = BigInt(parseInt(contracts.coreBridge(network, "Algorand")));
  const TOKEN_BRIDGE_ID = BigInt(
    parseInt(contracts.tokenBridge(network, "Algorand"))
  );
  const recipient = tryNativeToHexString(dstAddress, chainToChainId(dstChain));
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
    // @ts-ignore: legacy chain ids
    toChainId(dstChain),
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
  if (network === "Devnet") {
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
