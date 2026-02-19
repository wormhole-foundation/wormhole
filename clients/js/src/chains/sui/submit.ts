import {
  getWrappedCoinType,
  uint8ArrayToBCS,
} from "@certusone/wormhole-sdk/lib/esm/sui";
import {
  createWrappedOnSui,
  createWrappedOnSuiPrepare,
} from "@certusone/wormhole-sdk/lib/esm/token_bridge/createWrapped";
import { Transaction } from "@mysten/sui/transactions";
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
  SUI_CLOCK_OBJECT_ID,
} from "./utils";
import {
  Chain,
  Network,
  VAA,
  assertChain,
  contracts,
  deserialize,
} from "@wormhole-foundation/sdk";
import { getForeignAssetSui } from "../../sdk/sui";

export const submit = async (
  payload: Payload,
  vaa: Buffer,
  network: Network,
  rpc?: string,
  privateKey?: string
) => {
  const chain: Chain = "Sui";
  const client = getProvider(network, rpc);
  const signer = getSigner(client, network, privateKey);

  switch (payload.module) {
    case "Core": {
      const coreObjectId = contracts.coreBridge.get(network, chain);
      if (!coreObjectId) {
        throw Error("Core bridge object ID is undefined");
      }

      const corePackageId = await getPackageId(client, coreObjectId);
      switch (payload.type) {
        case "ContractUpgrade":
          throw new Error("ContractUpgrade not supported on Sui");
        case "GuardianSetUpgrade": {
          const tx = new Transaction();
          const [verifiedVaa] = tx.moveCall({
            target: `${corePackageId}::vaa::parse_and_verify`,
            arguments: [
              tx.object(coreObjectId),
              tx.pure("vector<u8>", [...uint8ArrayToBCS(vaa)]),
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
      const coreBridgeStateObjectId = contracts.coreBridge.get(network, chain);
      if (!coreBridgeStateObjectId) {
        throw Error("Core bridge object ID is undefined");
      }

      const tokenBridgeStateObjectId = contracts.tokenBridge.get(
        network,
        chain
      );
      if (!tokenBridgeStateObjectId) {
        throw Error("Token bridge object ID is undefined");
      }

      switch (payload.type) {
        case "AttestMeta": {
          // Test attest VAA: 01000000000100d87023087588d8a482d6082c57f3c93649c9a61a98848fc3a0b271f4041394ff7b28abefc8e5e19b83f45243d073d677e122e41425c2dbae3eb5ae1c7c0ac0ee01000000c056a8000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16000000000000000001020000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a000212544b4e0000000000000000000000000000000000000000000000000000000000457468657265756d205465737420546f6b656e00000000000000000000000000
          const parsedAttest: VAA<"TokenBridge:AttestMeta"> = deserialize(
            "TokenBridge:AttestMeta",
            vaa
          );
          const tokenChain = parsedAttest.payload.token.chain;
          assertChain(tokenChain);
          const tokenAddress = parsedAttest.payload.token.address;
          const decimals = parsedAttest.payload.decimals;
          const coinType = await getForeignAssetSui(
            client,
            tokenBridgeStateObjectId,
            tokenChain,
            tokenAddress.toUint8Array()
          );
          if (coinType) {
            // Coin already exists, so we update it
            console.log("Updating wrapped asset...");
            throw new Error("Updating wrapped asset not supported on Sui");
          } else {
            // Coin doesn't exist, so create wrapped asset
            console.log("[1/2] Creating wrapped asset...");
            const prepareTx = await createWrappedOnSuiPrepare(
              client as any,
              coreBridgeStateObjectId,
              tokenBridgeStateObjectId,
              decimals,
              signer.keypair.getPublicKey().toSuiAddress()
            );
            setMaxGasBudgetDevnet(network, prepareTx as any);
            const prepareRes = await executeTransactionBlock(
              signer,
              prepareTx as any
            );
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

            if (!rpc && network !== "Devnet") {
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
              client as any,
              coreBridgeStateObjectId,
              tokenBridgeStateObjectId,
              signer.keypair.getPublicKey().toSuiAddress(),
              coinPackageId,
              wrappedAssetSetup.objectType,
              vaa
            );
            setMaxGasBudgetDevnet(network, completeTx as any);
            const completeRes = await executeTransactionBlock(
              signer,
              completeTx as any
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
            client,
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
};

const sleep = (ms: number): Promise<void> => {
  return new Promise((resolve) => setTimeout(resolve, ms));
};
