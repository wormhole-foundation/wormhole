import {
  getWrappedCoinType,
  uint8ArrayToBCS,
} from "@certusone/wormhole-sdk/lib/esm/sui";
import {
  createWrappedOnSui,
  createWrappedOnSuiPrepare,
} from "@certusone/wormhole-sdk/lib/esm/token_bridge/createWrapped";
import { getForeignAssetSui } from "@certusone/wormhole-sdk/lib/esm/token_bridge/getForeignAsset";
import {
  CHAIN_ID_SUI,
  CHAIN_ID_TO_NAME,
  CONTRACTS,
  assertChain,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { parseAttestMetaVaa } from "@certusone/wormhole-sdk/lib/esm/vaa/tokenBridge";
import { SUI_CLOCK_OBJECT_ID, TransactionBlock } from "@mysten/sui.js";
import { Network } from "../../utils";
import { Payload, impossible } from "../../vaa";
import {
  assertSuccess,
  executeTransactionBlock,
  getPackageId,
  getProvider,
  getSigner,
  isSuiCreateEvent,
  isSuiPublishEvent,
  registerChain,
  setMaxGasBudgetDevnet,
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
          const tx = new TransactionBlock();
          const [verifiedVaa] = tx.moveCall({
            target: `${corePackageId}::vaa::parse_and_verify`,
            arguments: [
              tx.object(coreObjectId),
              tx.pure(uint8ArrayToBCS(vaa)),
              tx.object(SUI_CLOCK_OBJECT_ID),
            ],
          });

          const [decreeTicket] = tx.moveCall({
            target: `${corePackageId}::update_guardian_set::authorize_governance`,
            arguments: [tx.object(coreObjectId)],
          });

          const [decreeReceipt] = tx.moveCall({
            target: `${corePackageId}::governance_message::verify_vaa`,
            arguments: [tx.object(coreObjectId), verifiedVaa, decreeTicket],
            typeArguments: [
              `${corePackageId}::update_guardian_set::GovernanceWitness`,
            ],
          });

          console.log("Submitting new guardian set");
          setMaxGasBudgetDevnet(network, tx);
          tx.moveCall({
            target: `${corePackageId}::update_guardian_set::update_guardian_set`,
            arguments: [
              tx.object(coreObjectId),
              decreeReceipt,
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
            console.log(`  Digest ${prepareRes.digest}`);
            assertSuccess(prepareRes, "Prepare registration failed.");

            // Get the coin package ID from the publish event
            const coinPackageId =
              prepareRes.objectChanges?.find(isSuiPublishEvent)?.packageId;
            if (!coinPackageId) {
              throw new Error("Publish coin failed.");
            }

            console.log(`  Published to ${coinPackageId}`);
            console.log(`  Type ${getWrappedCoinType(coinPackageId)}`);

            if (!rpc && network !== "DEVNET") {
              // Wait for wrapped asset creation to be propagated to other
              // nodes in case this complete registration call is load balanced
              // to another node.
              await sleep(5000);
            }

            console.log("\n[2/2] Registering asset...");
            const wrappedAssetSetup = prepareRes.objectChanges
              ?.filter(isSuiCreateEvent)
              .find((e) =>
                /create_wrapped::WrappedAssetSetup/.test(e.objectType)
              );
            if (!wrappedAssetSetup) {
              throw new Error(
                "Wrapped asset setup not found. Object changes: " +
                  JSON.stringify(prepareRes.objectChanges)
              );
            }

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
    case "WormholeRelayer":
      throw Error("Wormhole Relayer not supported on Sui");
    default:
      impossible(payload);
  }

  console.warn = consoleWarnTemp;
};

const sleep = (ms: number): Promise<void> => {
  return new Promise((resolve) => setTimeout(resolve, ms));
};
