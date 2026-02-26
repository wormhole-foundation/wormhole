import { SuiClient, SuiTransactionBlockResponse } from "@mysten/sui/client";
import { Ed25519Keypair } from "@mysten/sui/keypairs/ed25519";
import { Transaction } from "@mysten/sui/transactions";
import { fromB64, toB64 } from "@mysten/bcs";
import { NETWORKS } from "../../consts";
import { Payload, VAA, parse, serialiseVAA } from "../../vaa";
import { SuiRpcValidationError } from "./error";
import { Network } from "@wormhole-foundation/sdk";
import { isValidSuiAddress } from "../../sdk/sui";

const SUI_CLOCK_OBJECT_ID =
  "0x0000000000000000000000000000000000000000000000000000000000000006";
const UPGRADE_CAP_TYPE = "0x2::package::UpgradeCap";

// Re-export for other modules
export { SUI_CLOCK_OBJECT_ID };

// Type for signer - combines keypair and client
export interface SuiSigner {
  keypair: Ed25519Keypair;
  client: SuiClient;
}

export const assertSuccess = (
  res: SuiTransactionBlockResponse,
  error: string
): void => {
  if (res?.effects?.status?.status !== "success") {
    throw new Error(`${error} Response: ${JSON.stringify(res)}`);
  }
};

export const executeTransactionBlock = async (
  signer: SuiSigner,
  transaction: Transaction
): Promise<SuiTransactionBlockResponse> => {
  const res = await signer.client.signAndExecuteTransaction({
    signer: signer.keypair,
    transaction,
    options: {
      showInput: true,
      showEffects: true,
      showEvents: true,
      showObjectChanges: true,
    },
  });

  return res;
};

export const findOwnedObjectByType = async (
  client: SuiClient,
  owner: string,
  type: string,
  cursor?: string
): Promise<string | null> => {
  const res = await client.getOwnedObjects({
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

  const object = res.data.find((d) => d.data?.type === type);

  if (!object && res.hasNextPage) {
    return findOwnedObjectByType(client, owner, type, res.nextCursor as string);
  } else if (!object && !res.hasNextPage) {
    return null;
  } else {
    return object?.data?.objectId ?? null;
  }
};

export const getCreatedObjects = (
  res: SuiTransactionBlockResponse
): { type: string; objectId: string; owner: string }[] =>
  res.objectChanges?.filter(isSuiCreateEvent).map((e) => {
    let owner: string;
    if (typeof e.owner === "string") {
      owner = e.owner;
    } else if ("AddressOwner" in e.owner) {
      owner = e.owner.AddressOwner;
    } else if ("ObjectOwner" in e.owner) {
      owner = e.owner.ObjectOwner;
    } else {
      owner = "Shared";
    }

    return {
      owner,
      type: e.objectType,
      objectId: e.objectId,
    };
  }) ?? [];

export const getOwnedObjectId = async (
  client: SuiClient,
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
    const res = await client.getOwnedObjects({
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
      return objects[0].data?.objectId ?? null;
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
      return findOwnedObjectByType(client, owner, type);
    } else {
      throw error;
    }
  }
};

// TODO(kp): remove this once it's in the sdk
export const getPackageId = async (
  client: SuiClient,
  objectId: string
): Promise<string> => {
  let currentPackage;
  let nextCursor: string | null | undefined;
  do {
    const dynamicFields = await client.getDynamicFields({
      parentId: objectId,
      cursor: nextCursor,
    });
    currentPackage = dynamicFields.data.find((field) =>
      field.name.type.endsWith("CurrentPackage")
    );
    nextCursor = dynamicFields.hasNextPage ? dynamicFields.nextCursor : null;
  } while (nextCursor && !currentPackage);
  if (!currentPackage) {
    throw new Error("CurrentPackage not found");
  }

  const obj = await client.getObject({
    id: currentPackage.objectId,
    options: {
      showContent: true,
    },
  });
  const content = obj.data?.content;
  const packageId =
    content && "fields" in content
      ? (content.fields as any).value?.fields?.package
      : null;
  if (!packageId) {
    throw new Error("Unable to get current package");
  }

  return packageId;
};

export const getProvider = (network?: Network, rpc?: string): SuiClient => {
  if (!network && !rpc) {
    throw new Error("Must provide network or RPC to initialize provider");
  }

  rpc = rpc || NETWORKS[network!].Sui.rpc;
  if (!rpc) {
    throw new Error(`No default RPC found for Sui ${network}`);
  }

  return new SuiClient({ url: rpc });
};

export const getPublishedPackageId = (
  res: SuiTransactionBlockResponse
): string => {
  const publishEvents = res.objectChanges?.filter(isSuiPublishEvent) ?? [];
  if (publishEvents.length !== 1) {
    throw new Error(
      "Unexpected number of publish events found:" +
        JSON.stringify(publishEvents, null, 2)
    );
  }

  return publishEvents[0].packageId;
};

export const getSigner = (
  client: SuiClient,
  network: Network,
  customPrivateKey?: string
): SuiSigner => {
  const privateKey: string | undefined =
    customPrivateKey || NETWORKS[network].Sui.key;
  if (!privateKey) {
    throw new Error(`No private key found for Sui ${network}`);
  }

  let bytes = privateKey.startsWith("0x")
    ? Buffer.from(privateKey.slice(2), "hex")
    : fromB64(privateKey);
  if (bytes.length === 33) {
    // remove the first flag byte after checking it is indeed the Ed25519 scheme flag 0x00
    if (bytes[0] !== 0) {
      throw new Error("Only the Ed25519 scheme flag is supported");
    }
    bytes = bytes.subarray(1);
  }
  const keypair = Ed25519Keypair.fromSecretKey(bytes);
  return { keypair, client };
};

/**
 * This function returns the object ID of the `UpgradeCap` that belongs to the
 * given package and owner if it exists.
 *
 * Structs created by the Sui framework such as `UpgradeCap`s all have the same
 * type (e.g. `0x2::package::UpgradeCap`) and have a special field, `package`,
 * we can use to differentiate them.
 * @param client Sui RPC client
 * @param owner Address of the current owner of the `UpgradeCap`
 * @param packageId ID of the package that the `UpgradeCap` was created for
 * @returns The object ID of the `UpgradeCap` if it exists, otherwise `null`
 */
export const getUpgradeCapObjectId = async (
  client: SuiClient,
  owner: string,
  packageId: string
): Promise<string | null> => {
  const res = await client.getOwnedObjects({
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
      (o.data?.content?.fields as any)?.package === packageId
  );
  if (objects.length === 1) {
    // We've found the object we're looking for
    return objects[0].data?.objectId ?? null;
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

export const isSuiCreateEvent = <
  T extends NonNullable<SuiTransactionBlockResponse["objectChanges"]>[number],
  K extends Extract<T, { type: "created" }>
>(
  event: T
): event is K => event?.type === "created";

export const isSuiPublishEvent = <
  T extends NonNullable<SuiTransactionBlockResponse["objectChanges"]>[number],
  K extends Extract<T, { type: "published" }>
>(
  event: T
): event is K => event?.type === "published";

// Normalize Sui address
export const normalizeSuiAddress = (address: string): string => {
  // Remove 0x prefix, pad to 64 chars, add 0x back
  const hex = address.replace(/^0x/, "").padStart(64, "0");
  return `0x${hex}`;
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

export const registerChain = async (
  client: SuiClient,
  network: Network,
  vaa: Buffer,
  coreBridgeStateObjectId: string,
  tokenBridgeStateObjectId: string,
  transaction?: Transaction
): Promise<Transaction> => {
  if (network === "Devnet") {
    // Modify the VAA to only have 1 guardian signature
    // TODO: remove this when we can deploy the devnet core contract
    // deterministically with multiple guardians in the initial guardian set
    // Currently the core contract is setup with only 1 guardian in the set
    const parsedVaa = parse(vaa);
    parsedVaa.signatures = [parsedVaa.signatures[0]];
    vaa = Buffer.from(serialiseVAA(parsedVaa as VAA<Payload>), "hex");
  }

  // Get package IDs
  const coreBridgePackageId = await getPackageId(
    client,
    coreBridgeStateObjectId
  );
  const tokenBridgePackageId = await getPackageId(
    client,
    tokenBridgeStateObjectId
  );

  // Register chain
  let tx = transaction;
  if (!tx) {
    tx = new Transaction();
    tx.setGasBudget(1000000);
  }

  // Get VAA
  const [verifiedVaa] = tx.moveCall({
    target: `${coreBridgePackageId}::vaa::parse_and_verify`,
    arguments: [
      tx.object(coreBridgeStateObjectId),
      tx.pure("vector<u8>", [...vaa]),
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
    arguments: [tx.object(coreBridgeStateObjectId), verifiedVaa, decreeTicket],
    typeArguments: [
      `${tokenBridgePackageId}::register_chain::GovernanceWitness`,
    ],
  });

  // Register chain
  tx.moveCall({
    target: `${tokenBridgePackageId}::register_chain::register_chain`,
    arguments: [tx.object(tokenBridgeStateObjectId), decreeReceipt],
  });

  return tx;
};

/**
 * Currently, (Sui SDK version 0.32.2 and Sui 1.0.0 testnet), there is a
 * mismatch in the max gas budget that causes an error when executing a
 * transaction. Because these values are hardcoded, we set the max gas budget
 * as a temporary workaround.
 * @param network
 * @param tx
 */
export const setMaxGasBudgetDevnet = (network: Network, tx: Transaction) => {
  if (network === "Devnet" || network === "Testnet") {
    // Avoid Error checking transaction input objects: GasBudgetTooHigh { gas_budget: 50000000000, max_budget: 10000000000 }
    tx.setGasBudget(10000000000);
  }
};
