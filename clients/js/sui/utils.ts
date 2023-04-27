import { CHAIN_ID_SUI } from "@certusone/wormhole-sdk";
import { CHAIN_ID_TO_NAME } from "@certusone/wormhole-sdk/lib/cjs/utils/consts";
import {
  Connection,
  Ed25519Keypair,
  JsonRpcProvider,
  PaginatedObjectsResponse,
  RawSigner,
  SUI_CLOCK_OBJECT_ID,
  SuiTransactionBlockResponse,
  TransactionBlock,
  fromB64,
  getPublishedObjectChanges,
  normalizeSuiAddress,
} from "@mysten/sui.js";
import { CONTRACTS } from "../consts";
import { NETWORKS } from "../networks";
import { Network } from "../utils";
import { Payload, VAA, impossible, parse, serialiseVAA } from "../vaa";
import { SuiRpcValidationError } from "./error";
import { SuiCreateEvent, SuiPublishEvent } from "./types";

const UPGRADE_CAP_TYPE = "0x2::package::UpgradeCap";

// TODO(kp): remove this once it's in the sdk
export async function getPackageId(
  provider: JsonRpcProvider,
  stateObjectId: string
): Promise<string> {
  const fields = await provider
    .getObject({
      id: stateObjectId,
      options: {
        showContent: true,
      },
    })
    .then((result) => {
      if (result.data?.content?.dataType === "moveObject") {
        return result.data.content.fields;
      }
      throw new Error("Not a moveObject");
    });
  if ("upgrade_cap" in fields) {
    return fields.upgrade_cap.fields.package;
  }
  throw new Error("upgrade_cap not found");
}

export const execute_sui = async (
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
        case "GuardianSetUpgrade": {
          console.log("Submitting new guardian set");
          const tx = new TransactionBlock();
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
        case "ContractUpgrade":
          throw new Error("ContractUpgrade not supported on Sui");
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

      const coreBridgePackageId = await getPackageId(
        provider,
        coreBridgeStateObjectId
      );
      const tokenBridgePackageId = await getPackageId(
        provider,
        tokenBridgeStateObjectId
      );

      switch (payload.type) {
        case "ContractUpgrade":
          throw new Error("ContractUpgrade not supported on Sui");
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on Sui");
        case "RegisterChain": {
          console.log("Registering chain");
          if (network === "DEVNET") {
            // Modify the VAA to only have 1 guardian signature
            // TODO: remove this when we can deploy the devnet core contract
            // deterministically with multiple guardians in the initial guardian set
            // Currently the core contract is setup with only 1 guardian in the set
            const parsedVaa = parse(vaa);
            parsedVaa.signatures = [parsedVaa.signatures[0]];
            vaa = Buffer.from(serialiseVAA(parsedVaa as VAA<Payload>), "hex");
          }

          // Get VAA
          const tx = new TransactionBlock();
          tx.setGasBudget(1000000);
          const [verifiedVaa] = tx.moveCall({
            target: `${coreBridgePackageId}::vaa::parse_and_verify`,
            arguments: [
              tx.object(coreBridgeStateObjectId),
              tx.pure([...vaa]),
              tx.object(SUI_CLOCK_OBJECT_ID),
            ],
          });

          // Get decree ticket
          const [decreeTicket] = tx.moveCall({
            target: `${tokenBridgePackageId}::register_chain::authorize_governance`,
            arguments: [tx.object(tokenBridgeStateObjectId)],
          });

          // Get decree receipt
          const [decreeReceipt] = tx.moveCall({
            target: `${coreBridgePackageId}::governance_message::verify_vaa`,
            arguments: [
              tx.object(coreBridgeStateObjectId),
              verifiedVaa,
              decreeTicket,
            ],
            typeArguments: [
              `${tokenBridgePackageId}::register_chain::GovernanceWitness`,
            ],
          });

          // Register chain
          tx.moveCall({
            target: `${tokenBridgePackageId}::register_chain::register_chain`,
            arguments: [tx.object(tokenBridgeStateObjectId), decreeReceipt],
          });

          await executeTransactionBlock(signer, tx);
          break;
        }
        case "AttestMeta":
          throw new Error("AttestMeta not supported on Sui");
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

export const executeTransactionBlock = async (
  signer: RawSigner,
  transactionBlock: TransactionBlock
): Promise<SuiTransactionBlockResponse> => {
  // Let caller handle parsing and logging info
  return signer.signAndExecuteTransactionBlock({
    transactionBlock,
    options: {
      showInput: true,
      showEffects: true,
      showEvents: true,
      showObjectChanges: true,
    },
  });
};

export const getCreatedObjects = (
  res: SuiTransactionBlockResponse
): { type: string; objectId: string; owner: string }[] => {
  return res.objectChanges.filter(isSuiCreateEvent).map((e) => ({
    type: e.objectType,
    objectId: e.objectId,
    owner: e.owner["AddressOwner"] || e.owner["ObjectOwner"] || e.owner,
  }));
};

export const findOwnedObjectByType = async (
  provider: JsonRpcProvider,
  owner: string,
  type: string,
  cursor?: string
): Promise<string | null> => {
  const res: PaginatedObjectsResponse = await provider.getOwnedObjects({
    owner,
    filter: undefined, // Filter must be undefined to avoid 504 responses
    cursor: cursor || undefined,
    options: {
      showType: true,
    },
  });

  if (!res || !res.data) {
    throw new SuiRpcValidationError(res);
  }

  const object = res.data.find((d) => d.data.type === type);

  if (!object && res.hasNextPage) {
    return findOwnedObjectByType(
      provider,
      owner,
      type,
      res.nextCursor as string
    );
  } else if (!object && !res.hasNextPage) {
    return null;
  } else {
    return object.data.objectId;
  }
};

export const getOwnedObjectId = async (
  provider: JsonRpcProvider,
  owner: string,
  packageId: string,
  moduleName: string,
  structName: string
): Promise<string | null> => {
  const type = `${packageId}::${moduleName}::${structName}`;

  // Upgrade caps are a special case
  if (normalizeSuiType(type) === normalizeSuiType(UPGRADE_CAP_TYPE)) {
    throw new Error(
      "`getOwnedObjectId` should not be used to get the object ID of an `UpgradeCap`. Use `getUpgradeCapObjectId` instead."
    );
  }

  try {
    const res = await provider.getOwnedObjects({
      owner,
      filter: { StructType: type },
      options: {
        showContent: true,
      },
    });
    if (!res || !res.data) {
      throw new SuiRpcValidationError(res);
    }

    const objects = res.data.filter((o) => o.data?.objectId);
    if (objects.length === 1) {
      return objects[0].data?.objectId;
    } else if (objects.length > 1) {
      const objectsStr = JSON.stringify(objects, null, 2);
      throw new Error(
        `Found multiple objects owned by ${owner} of type ${type}. This may mean that we've received an unexpected response from the Sui RPC and \`worm\` logic needs to be updated to handle this. Objects: ${objectsStr}`
      );
    } else {
      return null;
    }
  } catch (error) {
    // Handle 504 error by using findOwnedObjectByType method
    const is504HttpError = `${error}`.includes("504 Gateway Time-out");
    if (error && is504HttpError) {
      return findOwnedObjectByType(provider, owner, type);
    } else {
      throw error;
    }
  }
};

export const getProvider = (
  network?: Network,
  rpc?: string
): JsonRpcProvider => {
  if (!network && !rpc) {
    throw new Error("Must provide network or RPC to initialize provider");
  }

  rpc = rpc || NETWORKS[network]["sui"].rpc;
  if (!rpc) {
    throw new Error(`No default RPC found for Sui ${network}`);
  }

  return new JsonRpcProvider(new Connection({ fullnode: rpc }));
};

export const getPublishedPackageId = (
  res: SuiTransactionBlockResponse
): string => {
  const publishEvents = getPublishedObjectChanges(res);
  if (publishEvents.length !== 1) {
    throw new Error(
      "Unexpected number of publish events found:" +
        JSON.stringify(publishEvents, null, 2)
    );
  }

  return publishEvents[0].packageId;
};

export const getSigner = (
  provider: JsonRpcProvider,
  network: Network,
  customPrivateKey?: string
): RawSigner => {
  const privateKey: string | undefined =
    customPrivateKey || NETWORKS[network]["sui"].key;
  if (!privateKey) {
    throw new Error(`No private key found for Sui ${network}`);
  }

  const bytes = fromB64(privateKey);
  const keypair = Ed25519Keypair.fromSecretKey(bytes.slice(1));
  return new RawSigner(keypair, provider);
};

/**
 * This function returns the object ID of the `UpgradeCap` that belongs to the
 * given package and owner if it exists.
 *
 * Structs created by the Sui framework such as `UpgradeCap`s all have the same
 * type (e.g. `0x2::package::UpgradeCap`) and have a special field, `package`,
 * we can use to differentiate them.
 * @param provider Sui RPC provider
 * @param owner Address of the current owner of the `UpgradeCap`
 * @param packageId ID of the package that the `UpgradeCap` was created for
 * @returns The object ID of the `UpgradeCap` if it exists, otherwise `null`
 */
export const getUpgradeCapObjectId = async (
  provider: JsonRpcProvider,
  owner: string,
  packageId: string
): Promise<string | null> => {
  const res = await provider.getOwnedObjects({
    owner,
    filter: { StructType: UPGRADE_CAP_TYPE },
    options: {
      showContent: true,
    },
  });
  if (!res || !res.data) {
    throw new SuiRpcValidationError(res);
  }

  const objects = res.data.filter(
    (o) =>
      o.data?.objectId &&
      o.data?.content?.dataType === "moveObject" &&
      o.data?.content?.fields?.package === packageId
  );
  if (objects.length === 1) {
    // We've found the object we're looking for
    return objects[0].data?.objectId;
  } else if (objects.length > 1) {
    const objectsStr = JSON.stringify(objects, null, 2);
    throw new Error(
      `Found multiple upgrade capabilities owned by ${owner} from package ${packageId}. Objects: ${objectsStr}`
    );
  } else {
    return null;
  }
};

export const isSameType = (a: string, b: string) => {
  try {
    return normalizeSuiType(a) === normalizeSuiType(b);
  } catch (e) {
    return false;
  }
};

export const isSuiPublishEvent = (event: any): event is SuiPublishEvent => {
  return event.type === "published";
};

export const isSuiCreateEvent = (event: any): event is SuiCreateEvent => {
  return event.type === "created";
};

export const isValidSuiAddress = (objectId: string): boolean => {
  return /^(0x)?[0-9a-f]{1,64}$/.test(objectId);
};

// todo(aki): this needs to correctly handle types such as
// 0x2::dynamic_field::Field<0x3c6d386861470e6f9cb35f3c91f69e6c1f1737bd5d217ca06a15f582e1dc1ce3::state::MigrationControl, bool>
export const normalizeSuiType = (type: string): string => {
  const tokens = type.split("::");
  if (tokens.length !== 3 || !isValidSuiAddress(tokens[0])) {
    throw new Error(`Invalid Sui type: ${type}`);
  }

  return [normalizeSuiAddress(tokens[0]), tokens[1], tokens[2]].join("::");
};
