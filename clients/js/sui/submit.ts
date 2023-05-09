import {
  assertChain,
  createWrappedOnSui,
  createWrappedOnSuiPrepare,
  getForeignAssetSui,
  parseAttestMetaVaa,
} from "@certusone/wormhole-sdk";
import { getWrappedCoinType } from "@certusone/wormhole-sdk/lib/cjs/sui";
import {
  CHAIN_ID_SUI,
  CHAIN_ID_TO_NAME,
  CONTRACTS,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";
import { SUI_CLOCK_OBJECT_ID, TransactionBlock } from "@mysten/sui.js";
import { Network } from "../utils";
import { Payload, impossible } from "../vaa";
import {
  assertSuccess,
  executeTransactionBlock,
  getPackageId,
  getProvider,
  getSigner,
  isSuiCreateEvent,
  isSuiPublishEvent,
  registerChain,
} from "./utils";

export const submit = async (
  payload: Payload,
  vaa: Buffer,
  network: Network,
  rpc?: string,
  privateKey?: string
) => {
  const consoleWarnTemp = console.warn;
  console.warn = () => {};

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
        case "AttestMeta": {
          // Test attest VAA: 01000000000100d87023087588d8a482d6082c57f3c93649c9a61a98848fc3a0b271f4041394ff7b28abefc8e5e19b83f45243d073d677e122e41425c2dbae3eb5ae1c7c0ac0ee01000000c056a8000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16000000000000000001020000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a000212544b4e0000000000000000000000000000000000000000000000000000000000457468657265756d205465737420546f6b656e00000000000000000000000000
          const { tokenChain, tokenAddress } = parseAttestMetaVaa(vaa);
          assertChain(tokenChain);
          const coinType = await getForeignAssetSui(
            provider,
            tokenBridgeStateObjectId,
            tokenChain,
            tokenAddress
          );
          if (coinType) {
            // Coin already exists, so we update it
            console.log("Updating wrapped asset...");
            throw new Error("Updating wrapped asset not supported on Sui");
          } else {
            // Coin doesn't exist, so create wrapped asset
            console.log("[1/2] Creating wrapped asset...");
            const prepareTx = await createWrappedOnSuiPrepare(
              provider,
              coreBridgeStateObjectId,
              tokenBridgeStateObjectId,
              parseAttestMetaVaa(vaa).decimals,
              await signer.getAddress()
            );
            setMaxGasBudgetDevnet(network, prepareTx);
            const prepareRes = await executeTransactionBlock(signer, prepareTx);
            assertSuccess(prepareRes, "Prepare registration failed.");
            const coinPackageId =
              prepareRes.objectChanges.find(isSuiPublishEvent).packageId;
            console.log(`  Digest ${prepareRes.digest}`);
            console.log(`  Published to ${coinPackageId}`);
            console.log(`  Type ${getWrappedCoinType(coinPackageId)}`);

            if (!rpc && network !== "DEVNET") {
              // Wait for wrapped asset creation to be propogated to other
              // nodes in case this complete registration call is load balanced
              // to another node.
              await sleep(5000);
            }

            console.log("\n[2/2] Registering asset...");
            const wrappedAssetSetup = prepareRes.objectChanges
              .filter(isSuiCreateEvent)
              .find((e) =>
                /create_wrapped::WrappedAssetSetup/.test(e.objectType)
              );
            const completeTx = await createWrappedOnSui(
              provider,
              coreBridgeStateObjectId,
              tokenBridgeStateObjectId,
              await signer.getAddress(),
              coinPackageId,
              wrappedAssetSetup.objectType,
              vaa
            );
            setMaxGasBudgetDevnet(network, completeTx);
            const completeRes = await executeTransactionBlock(
              signer,
              completeTx
            );
            assertSuccess(completeRes, "Complete registration failed.");
            console.log(`  Digest ${completeRes.digest}`);
            console.log("\nDone!");
          }

          break;
        }
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
    case "CoreRelayer":
        throw Error("Wormhole Relayer not supported on Sui");
    default:
      impossible(payload);
  }

  console.warn = consoleWarnTemp;
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

const sleep = (ms: number): Promise<void> => {
  return new Promise((resolve) => setTimeout(resolve, ms));
};
