import { Transaction } from "@mysten/sui/transactions";
import { fromBase64 } from "@mysten/sui/utils";
import { Payload, impossible } from "../../vaa";
import {
  assertSuccess,
  executeTransactionBlock,
  getOriginalPackageId,
  getPackageId,
  getProvider,
  getPublishedPackageId,
  getSigner,
  getUpgradeCapObjectId,
  normalizeSuiAddress,
  registerChain,
  setMaxGasBudgetDevnet,
  SUI_CLOCK_OBJECT_ID,
} from "./utils";
import { SuiGrpcClient } from "@mysten/sui/grpc";
import {
  Chain,
  Network,
  VAA,
  assertChain,
  contracts,
  deserialize,
} from "@wormhole-foundation/sdk";
import { getForeignAssetSui } from "../../sdk/sui";
import { buildWrappedCoinBytecode } from "./wrappedCoinBytecode";

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
              tx.pure("vector<u8>", [...vaa]),
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
        case "TransferFees":
          throw new Error("TransferFees not supported on Sui");
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
            const signerAddress = signer.keypair.getPublicKey().toSuiAddress();

            console.log("[1/2] Creating wrapped asset...");
            const prepareTx = await createWrappedOnSuiPrepare(
              client,
              coreBridgeStateObjectId,
              tokenBridgeStateObjectId,
              decimals,
              signerAddress
            );
            setMaxGasBudgetDevnet(network, prepareTx);
            const prepareRes = await executeTransactionBlock(signer, prepareTx);
            console.log(`  Digest ${prepareRes.digest}`);
            assertSuccess(prepareRes, "Prepare registration failed.");

            // Get the coin package ID from the published package.
            const coinPackageId = getPublishedPackageId(prepareRes);

            console.log(`  Published to ${coinPackageId}`);
            console.log(`  Type ${getWrappedCoinType(coinPackageId)}`);

            if (!rpc && network !== "Devnet") {
              // Wait for wrapped asset creation to be propagated to other
              // nodes in case this complete registration call is load balanced
              // to another node.
              await sleep(5000);
            }

            console.log("\n[2/2] Registering asset...");
            const wrappedAssetSetup = prepareRes.changedObjects.find(
              (o) =>
                o.created &&
                o.type !== undefined &&
                /create_wrapped::WrappedAssetSetup/.test(o.type)
            );
            if (!wrappedAssetSetup || !wrappedAssetSetup.type) {
              throw new Error(
                "Wrapped asset setup not found. Changed objects: " +
                  JSON.stringify(prepareRes.changedObjects)
              );
            }

            const completeTx = await createWrappedOnSui(
              client,
              coreBridgeStateObjectId,
              tokenBridgeStateObjectId,
              signerAddress,
              coinPackageId,
              wrappedAssetSetup.objectId,
              wrappedAssetSetup.type,
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

const getWrappedCoinType = (coinPackageId: string): string =>
  `${coinPackageId}::coin::COIN`;

/**
 * Build the publish payload for a Wormhole wrapped-coin module. The module
 * bytecode is a fixed template (see {@link buildWrappedCoinBytecode}) parametrized
 * only by the original token bridge package ID and the (capped) decimals.
 */
const getCoinBuildOutput = async (
  client: SuiGrpcClient,
  coreBridgePackageId: string,
  tokenBridgePackageId: string,
  tokenBridgeStateObjectId: string,
  decimals: number
): Promise<{ modules: string[]; dependencies: string[] }> => {
  const originalTokenBridgePackageId = await getOriginalPackageId(
    client,
    tokenBridgeStateObjectId
  );
  return {
    modules: [
      buildWrappedCoinBytecode(originalTokenBridgePackageId, decimals),
    ],
    dependencies: ["0x1", "0x2", tokenBridgePackageId, coreBridgePackageId].map(
      (d) => normalizeSuiAddress(d)
    ),
  };
};

/**
 * Step 1 of wrapped asset creation: publish a coin package whose decimals match
 * the attested token. The resulting `WrappedAssetSetup` and coin `UpgradeCap`
 * are transferred to the signer for use in `createWrappedOnSui`.
 */
const createWrappedOnSuiPrepare = async (
  client: SuiGrpcClient,
  coreBridgeStateObjectId: string,
  tokenBridgeStateObjectId: string,
  decimals: number,
  signerAddress: string
): Promise<Transaction> => {
  const [coreBridgePackageId, tokenBridgePackageId] = await Promise.all([
    getPackageId(client, coreBridgeStateObjectId),
    getPackageId(client, tokenBridgeStateObjectId),
  ]);
  const build = await getCoinBuildOutput(
    client,
    coreBridgePackageId,
    tokenBridgePackageId,
    tokenBridgeStateObjectId,
    decimals
  );

  const tx = new Transaction();
  const [upgradeCap] = tx.publish({
    modules: build.modules.map((m) => Array.from(fromBase64(m))),
    dependencies: build.dependencies,
  });
  tx.transferObjects([upgradeCap], signerAddress);
  return tx;
};

/**
 * Step 2 of wrapped asset creation: verify the attestation VAA and complete the
 * registration using the `WrappedAssetSetup` and coin `UpgradeCap` produced by
 * `createWrappedOnSuiPrepare`.
 */
const createWrappedOnSui = async (
  client: SuiGrpcClient,
  coreBridgeStateObjectId: string,
  tokenBridgeStateObjectId: string,
  signerAddress: string,
  coinPackageId: string,
  wrappedAssetSetupObjectId: string,
  wrappedAssetSetupType: string,
  attestVAA: Buffer
): Promise<Transaction> => {
  const [coreBridgePackageId, tokenBridgePackageId] = await Promise.all([
    getPackageId(client, coreBridgeStateObjectId),
    getPackageId(client, tokenBridgeStateObjectId),
  ]);

  const coinType = getWrappedCoinType(coinPackageId);
  const coinMetadataObjectId = (
    await client.getCoinMetadata({ coinType })
  )?.coinMetadata?.id;
  if (!coinMetadataObjectId) {
    throw new Error(`Coin metadata object not found for coin type ${coinType}.`);
  }

  const coinUpgradeCapObjectId = await getUpgradeCapObjectId(
    client,
    signerAddress,
    coinPackageId
  );
  if (!coinUpgradeCapObjectId) {
    throw new Error(
      `Coin upgrade cap not found for ${coinType} under owner ${signerAddress}. You must call 'createWrappedOnSuiPrepare' first.`
    );
  }

  const tx = new Transaction();
  const [vaa] = tx.moveCall({
    target: `${coreBridgePackageId}::vaa::parse_and_verify`,
    arguments: [
      tx.object(coreBridgeStateObjectId),
      tx.pure("vector<u8>", [...attestVAA]),
      tx.object(SUI_CLOCK_OBJECT_ID),
    ],
  });
  const [message] = tx.moveCall({
    target: `${tokenBridgePackageId}::vaa::verify_only_once`,
    arguments: [tx.object(tokenBridgeStateObjectId), vaa],
  });

  // WrappedAssetSetup is parametrized by <CoinType, VersionType>; the version
  // type is the second type argument.
  const versionType = wrappedAssetSetupType.split(", ")[1].replace(">", "");
  tx.moveCall({
    target: `${tokenBridgePackageId}::create_wrapped::complete_registration`,
    arguments: [
      tx.object(tokenBridgeStateObjectId),
      tx.object(coinMetadataObjectId),
      tx.object(wrappedAssetSetupObjectId),
      tx.object(coinUpgradeCapObjectId),
      message,
    ],
    typeArguments: [coinType, versionType],
  });
  return tx;
};
