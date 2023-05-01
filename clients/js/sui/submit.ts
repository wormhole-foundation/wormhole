import {
  CHAIN_ID_SUI,
  CHAIN_ID_TO_NAME,
  CONTRACTS,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";
import { SUI_CLOCK_OBJECT_ID, TransactionBlock } from "@mysten/sui.js";
import { Network } from "../utils";
import { Payload, impossible } from "../vaa";
import {
  executeTransactionBlock,
  getPackageId,
  getProvider,
  getSigner,
  registerChain,
} from "./utils";

export const submit = async (
  payload: Payload,
  vaa: Buffer,
  network: Network,
  rpc?: string,
  privateKey?: string
) => {
  const chain = CHAIN_ID_TO_NAME[CHAIN_ID_SUI];
  const provider = getProvider(network, rpc);
  const signer = getSigner(provider, network, privateKey);

  switch (payload.module) {
    case "Core": {
      const coreObjectId = CONTRACTS[network][chain].core;
      if (!coreObjectId) {
        throw Error("Core bridge object ID is undefined");
      }

      const corePackageId = await getPackageId(provider, coreObjectId);
      switch (payload.type) {
        case "ContractUpgrade":
          throw new Error("ContractUpgrade not supported on Sui");
        case "GuardianSetUpgrade": {
          console.log("Submitting new guardian set");
          const tx = new TransactionBlock();
          setMaxGasBudgetDevnet(network, tx);
          tx.moveCall({
            target: `${corePackageId}::wormhole::update_guardian_set`,
            arguments: [
              tx.object(coreObjectId),
              tx.pure([...vaa]),
              tx.object(SUI_CLOCK_OBJECT_ID),
            ],
          });
          const result = await executeTransactionBlock(signer, tx);
          console.log(JSON.stringify(result));
          break;
        }
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on Sui");
        default:
          impossible(payload);
      }
      break;
    }
    case "NFTBridge": {
      throw new Error("NFT bridge not supported on Sui");
    }
    case "TokenBridge": {
      const coreBridgeStateObjectId = CONTRACTS[network][chain].core;
      if (!coreBridgeStateObjectId) {
        throw Error("Core bridge object ID is undefined");
      }

      const tokenBridgeStateObjectId = CONTRACTS[network][chain].token_bridge;
      if (!tokenBridgeStateObjectId) {
        throw Error("Token bridge object ID is undefined");
      }

      switch (payload.type) {
        case "AttestMeta":
          throw new Error("AttestMeta not supported on Sui");
        case "ContractUpgrade":
          throw new Error("ContractUpgrade not supported on Sui");
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on Sui");
        case "RegisterChain": {
          console.log("Registering chain");
          const tx = await registerChain(
            provider,
            network,
            vaa,
            coreBridgeStateObjectId,
            tokenBridgeStateObjectId
          );
          setMaxGasBudgetDevnet(network, tx);
          const res = await executeTransactionBlock(signer, tx);
          console.log(JSON.stringify(res));
          break;
        }
        case "Transfer":
          throw new Error("Transfer not supported on Sui");
        case "TransferWithPayload":
          throw Error("Can't complete payload 3 transfer from CLI");
        default:
          impossible(payload);
          break;
      }

      break;
    }
    default:
      impossible(payload);
  }
};

/**
 * Currently, (Sui SDK version 0.32.2 and Sui 1.0.0 testnet), there is a
 * mismatch in the max gas budget that causes an error when executing a
 * transaction. Because these values are hardcoded, we set the max gas budget
 * as a temporary workaround.
 * @param network
 * @param tx
 */
const setMaxGasBudgetDevnet = (network: Network, tx: TransactionBlock) => {
  if (network === "DEVNET") {
    // Avoid Error checking transaction input objects: GasBudgetTooHigh { gas_budget: 50000000000, max_budget: 10000000000 }
    tx.setGasBudget(10000000000);
  }
};
