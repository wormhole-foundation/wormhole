import {
  builder,
  getObjectType,
  isValidSuiAddress as isValidFullSuiAddress,
  JsonRpcProvider,
  normalizeSuiAddress,
  PaginatedObjectsResponse,
  RawSigner,
  SuiObjectResponse,
  SuiTransactionBlockResponse,
  TransactionBlock,
} from "@mysten/sui.js";
import { DynamicFieldPage } from "@mysten/sui.js/dist/types/dynamic_fields";
import { ensureHexPrefix } from "../utils";
import { SuiRpcValidationError } from "./error";
import { SuiError } from "./types";

const MAX_PURE_ARGUMENT_SIZE = 16 * 1024;
const UPGRADE_CAP_TYPE = "0x2::package::UpgradeCap";

export const uint8ArrayToBCS = (arr: Uint8Array) =>
  builder.ser("vector<u8>", arr, { maxSize: MAX_PURE_ARGUMENT_SIZE }).toBytes();

export const executeTransactionBlock = async (
  signer: RawSigner,
  transactionBlock: TransactionBlock
): Promise<SuiTransactionBlockResponse> => {
  // Let caller handle parsing and logging info
  transactionBlock.setGasBudget(100000000);
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

// TODO: can we pass in the latest core bridge package Id after an upgrade?
// or do we have to use the first one?
// this is the same type that the guardian will look for
export const getEmitterAddressAndSequenceFromResponseSui = (
  originalCoreBridgePackageId: string,
  response: SuiTransactionBlockResponse
): { emitterAddress: string; sequence: string } => {
  const wormholeMessageEventType = `${originalCoreBridgePackageId}::publish_message::WormholeMessage`;
  const event = response.events?.find((e) =>
    isSameType(e.type, wormholeMessageEventType)
  );
  if (event === undefined) {
    throw new Error(`${wormholeMessageEventType} event type not found`);
  }

  const { sender, sequence } = event.parsedJson || {};
  if (sender === undefined || sequence === undefined) {
    throw new Error("Can't find sender or sequence");
  }

  return { emitterAddress: sender.substring(2), sequence };
};

export const getFieldsFromObjectResponse = (object: SuiObjectResponse) => {
  const content = object.data?.content;
  return content && content.dataType === "moveObject" ? content.fields : null;
};

export const getInnerType = (type: string): string | null => {
  if (!type) return null;
  const match = type.match(/<(.*)>/);
  if (!match || !isValidSuiType(match[1])) {
    return null;
  }

  return match[1];
};

export const getObjectFields = async (
  provider: JsonRpcProvider,
  objectId: string
): Promise<Record<string, any> | null> => {
  if (!isValidSuiAddress(objectId)) {
    throw new Error(`Invalid object ID: ${objectId}`);
  }

  const res = await provider.getObject({
    id: objectId,
    options: {
      showContent: true,
    },
  });
  return getFieldsFromObjectResponse(res);
};

export const getOriginalPackageId = async (
  provider: JsonRpcProvider,
  stateObjectId: string
) => {
  return getObjectType(
    await provider.getObject({
      id: stateObjectId,
      options: { showContent: true },
    })
  )?.split("::")[0];
};

export const getOwnedObjectId = async (
  provider: JsonRpcProvider,
  owner: string,
  type: string
): Promise<string | null> => {
  // Upgrade caps are a special case
  if (isSameType(type, UPGRADE_CAP_TYPE)) {
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
      return getOwnedObjectIdPaginated(provider, owner, type);
    } else {
      throw error;
    }
  }
};

export const getOwnedObjectIdPaginated = async (
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

  const object = res.data.find((d) => isSameType(d.data?.type || "", type));
  if (!object && res.hasNextPage) {
    return getOwnedObjectIdPaginated(
      provider,
      owner,
      type,
      res.nextCursor as string
    );
  } else if (!object && !res.hasNextPage) {
    return null;
  } else {
    return object?.data?.objectId ?? null;
  }
};

/**
 * @param provider
 * @param objectId Core or token bridge state object ID
 * @returns The latest package ID for the provided state object
 */
export async function getPackageId(
  provider: JsonRpcProvider,
  objectId: string
): Promise<string> {
  let currentPackage;
  let nextCursor;
  do {
    const dynamicFields: DynamicFieldPage = await provider.getDynamicFields({
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

  const fields = await getObjectFields(provider, currentPackage.objectId);
  const packageId = fields?.value?.fields?.package;
  if (!packageId) {
    throw new Error("Unable to get current package");
  }

  return packageId;
}

export const getPackageIdFromType = (type: string): string | null => {
  if (!isValidSuiType(type)) return null;
  const packageId = type.split("::")[0];
  if (!isValidSuiAddress(packageId)) return null;
  return packageId;
};

export const getTableKeyType = (tableType: string): string | null => {
  if (!tableType) return null;
  const match = trimSuiType(tableType).match(/0x2::table::Table<(.*)>/);
  if (!match) return null;
  const [keyType] = match[1].split(",");
  if (!isValidSuiType(keyType)) return null;
  return keyType;
};

export const getTokenCoinType = async (
  provider: JsonRpcProvider,
  tokenBridgeStateObjectId: string,
  tokenAddress: Uint8Array,
  tokenChain: number
): Promise<string | null> => {
  const tokenBridgeStateFields = await getObjectFields(
    provider,
    tokenBridgeStateObjectId
  );
  if (!tokenBridgeStateFields) {
    throw new Error("Unable to fetch object fields from token bridge state");
  }

  const coinTypes = tokenBridgeStateFields?.token_registry?.fields?.coin_types;
  const coinTypesObjectId = coinTypes?.fields?.id?.id;
  if (!coinTypesObjectId) {
    throw new Error("Unable to fetch coin types");
  }

  const keyType = getTableKeyType(coinTypes?.type);
  if (!keyType) {
    throw new Error("Unable to get key type");
  }

  const response = await provider.getDynamicFieldObject({
    parentId: coinTypesObjectId,
    name: {
      type: keyType,
      value: {
        addr: [...tokenAddress],
        chain: tokenChain,
      },
    },
  });
  if (response.error) {
    if (response.error.code === "dynamicFieldNotFound") {
      return null;
    }
    throw new Error(
      `Unexpected getDynamicFieldObject response ${response.error}`
    );
  }
  const fields = getFieldsFromObjectResponse(response);
  return fields?.value ? trimSuiType(ensureHexPrefix(fields.value)) : null;
};

export const getTokenFromTokenRegistry = async (
  provider: JsonRpcProvider,
  tokenBridgeStateObjectId: string,
  tokenType: string
): Promise<SuiObjectResponse> => {
  if (!isValidSuiType(tokenType)) {
    throw new Error(`Invalid Sui type: ${tokenType}`);
  }

  const tokenBridgeStateFields = await getObjectFields(
    provider,
    tokenBridgeStateObjectId
  );
  if (!tokenBridgeStateFields) {
    throw new Error(
      `Unable to fetch object fields from token bridge state. Object ID: ${tokenBridgeStateObjectId}`
    );
  }

  const tokenRegistryObjectId =
    tokenBridgeStateFields.token_registry?.fields?.id?.id;
  if (!tokenRegistryObjectId) {
    throw new Error("Unable to fetch token registry object ID");
  }

  const tokenRegistryPackageId = getPackageIdFromType(
    tokenBridgeStateFields.token_registry?.type
  );
  if (!tokenRegistryPackageId) {
    throw new Error("Unable to fetch token registry package ID");
  }

  return provider.getDynamicFieldObject({
    parentId: tokenRegistryObjectId,
    name: {
      type: `${tokenRegistryPackageId}::token_registry::Key<${tokenType}>`,
      value: {
        dummy_field: false,
      },
    },
  });
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
    filter: { StructType: padSuiType(UPGRADE_CAP_TYPE) },
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
      normalizeSuiAddress(o.data?.content?.fields?.package) ===
        normalizeSuiAddress(packageId)
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

/**
 * Get the fully qualified type of a wrapped asset published to the given
 * package ID.
 *
 * All wrapped assets that are registered with the token bridge must satisfy
 * the requirement that module name is `coin` (source: https://github.com/wormhole-foundation/wormhole/blob/a1b3773ee42507122c3c4c3494898fbf515d0712/sui/token_bridge/sources/create_wrapped.move#L88).
 * As a result, all wrapped assets share the same module name and struct name,
 * since the struct name is necessarily `COIN` since it is a OTW.
 * @param coinPackageId packageId of the wrapped asset
 * @returns Fully qualified type of the wrapped asset
 */
export const getWrappedCoinType = (coinPackageId: string): string => {
  if (!isValidSuiAddress(coinPackageId)) {
    throw new Error(`Invalid package ID: ${coinPackageId}`);
  }

  return `${coinPackageId}::coin::COIN`;
};

export const isSameType = (a: string, b: string) => {
  try {
    return trimSuiType(a) === trimSuiType(b);
  } catch (e) {
    return false;
  }
};

export const isSuiError = (error: any): error is SuiError => {
  return (
    error && typeof error === "object" && "code" in error && "message" in error
  );
};

/**
 * This method validates any Sui address, even if it's not 32 bytes long, i.e.
 * "0x2". This differs from Mysten's implementation, which requires that the
 * given address is 32 bytes long.
 * @param address Address to check
 * @returns If given address is a valid Sui address or not
 */
export const isValidSuiAddress = (address: string): boolean =>
  isValidFullSuiAddress(normalizeSuiAddress(address));

export const isValidSuiType = (type: string): boolean => {
  const tokens = type.split("::");
  if (tokens.length !== 3) {
    return false;
  }

  return isValidSuiAddress(tokens[0]) && !!tokens[1] && !!tokens[2];
};

/**
 * Unlike `trimSuiType`, this method does not modify nested types, it just pads
 * the top-level type.
 * @param type
 * @returns
 */
export const padSuiType = (type: string): string => {
  const tokens = type.split("::");
  if (tokens.length < 3 || !isValidSuiAddress(tokens[0])) {
    throw new Error(`Invalid Sui type: ${type}`);
  }

  return [normalizeSuiAddress(tokens[0]), ...tokens.slice(1)].join("::");
};

/**
 * This method removes leading zeroes for types in order to normalize them
 * since some types returned from the RPC have leading zeroes and others don't.
 */
export const trimSuiType = (type: string): string =>
  type.replace(/(0x)(0*)/g, "0x");

/**
 * Create a new EmitterCap object owned by owner.
 * @returns The created EmitterCap object ID
 */
export const newEmitterCap = (
  coreBridgePackageId: string,
  coreBridgeStateObjectId: string,
  owner: string
): TransactionBlock => {
  const tx = new TransactionBlock();
  const [emitterCap] = tx.moveCall({
    target: `${coreBridgePackageId}::emitter::new`,
    arguments: [tx.object(coreBridgeStateObjectId)],
  });
  tx.transferObjects([emitterCap], tx.pure(owner));
  return tx;
};

export const getOldestEmitterCapObjectId = async (
  provider: JsonRpcProvider,
  coreBridgePackageId: string,
  owner: string
): Promise<string | null> => {
  let oldestVersion: string | null = null;
  let oldestObjectId: string | null = null;
  let response: PaginatedObjectsResponse | null = null;
  let nextCursor;
  do {
    response = await provider.getOwnedObjects({
      owner,
      filter: {
        StructType: `${coreBridgePackageId}::emitter::EmitterCap`,
      },
      options: {
        showContent: true,
      },
      cursor: nextCursor,
    });
    if (!response || !response.data) {
      throw new SuiRpcValidationError(response);
    }
    for (const objectResponse of response.data) {
      if (!objectResponse.data) continue;
      const { version, objectId } = objectResponse.data;
      if (oldestVersion === null || version < oldestVersion) {
        oldestVersion = version;
        oldestObjectId = objectId;
      }
    }
    nextCursor = response.hasNextPage ? response.nextCursor : undefined;
  } while (nextCursor);
  return oldestObjectId;
};
