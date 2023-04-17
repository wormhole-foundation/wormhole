import {
  isValidSuiAddress as isValidFullSuiAddress,
  JsonRpcProvider,
  normalizeSuiAddress,
  RawSigner,
  SuiObjectResponse,
  SuiTransactionBlockResponse,
  TransactionBlock,
} from "@mysten/sui.js";
import { SuiError } from "./types";
import { ChainId, ChainName, coalesceChainId } from "../utils";

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

export const getFieldsFromObjectResponse = (object: SuiObjectResponse) => {
  const content = object.data?.content;
  return content && content.dataType === "moveObject" ? content.fields : null;
};

export const getInnerType = (type: string): string | null => {
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

export const getTokenFromTokenRegistry = async (
  provider: JsonRpcProvider,
  tokenBridgeAddress: string,
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

  return provider.getDynamicFieldObject({
    parentId: tokenRegistryObjectId,
    name: {
      type: `${tokenBridgeAddress}::token_registry::Key<${tokenType}>`,
      value: {
        dummy_field: false,
      },
    },
  });
};

export const getTokenCoinType = async (
  provider: JsonRpcProvider,
  tokenBridgeAddress: string,
  tokenBridgeStateObjectId: string,
  tokenAddress: Uint8Array,
  tokenChain: number
): Promise<string | null> => {
  const tokenBridgeStateFields = await getObjectFields(
    provider,
    tokenBridgeStateObjectId
  );
  if (!tokenBridgeStateFields) {
    throw new Error(
      `Unable to fetch object fields from token bridge state. Object ID: ${tokenBridgeStateObjectId}`
    );
  }
  const coinTypesObjectId =
    tokenBridgeStateFields?.token_registry?.fields?.coin_types?.fields?.id?.id;
  if (!coinTypesObjectId) {
    throw new Error("Unable to fetch coin types object ID");
  }
  try {
    // This call errors if the key doesn't exist in the coin_types table
    const coinTypeValue = await provider.getDynamicFieldObject({
      parentId: coinTypesObjectId,
      name: {
        type: `${tokenBridgeAddress}::token_registry::CoinTypeKey`,
        value: {
          addr: [...tokenAddress],
          chain: tokenChain,
        },
      },
    });
    const fields = getFieldsFromObjectResponse(coinTypeValue);
    return fields?.value || null;
  } catch {
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
