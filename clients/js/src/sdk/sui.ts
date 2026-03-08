import {
  JsonRpcProvider,
  SuiObjectResponse,
  isValidSuiAddress as isValidFullSuiAddress,
  normalizeSuiAddress,
} from "@mysten/sui.js";
import { Chain, chainToChainId } from "@wormhole-foundation/sdk-base";

export async function getForeignAssetSui(
  provider: JsonRpcProvider,
  tokenBridgeStateObjectId: string,
  originChain: Chain,
  originAddress: Uint8Array
): Promise<string | null> {
  const originChainId = chainToChainId(originChain);
  return getTokenCoinType(
    provider,
    tokenBridgeStateObjectId,
    originAddress,
    originChainId
  );
}

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

export const getFieldsFromObjectResponse = (object: SuiObjectResponse) => {
  const content = object.data?.content;
  return content && content.dataType === "moveObject" ? content.fields : null;
};

export function ensureHexPrefix(x: string): string {
  return x.substring(0, 2) !== "0x" ? `0x${x}` : x;
}

/**
 * This method validates any Sui address, even if it's not 32 bytes long, i.e.
 * "0x2". This differs from Mysten's implementation, which requires that the
 * given address is 32 bytes long.
 * @param address Address to check
 * @returns If given address is a valid Sui address or not
 */
export const isValidSuiAddress = (address: string): boolean =>
  isValidFullSuiAddress(normalizeSuiAddress(address));

export const getTableKeyType = (tableType: string): string | null => {
  if (!tableType) return null;
  const match = trimSuiType(tableType).match(/0x2::table::Table<(.*)>/);
  if (!match) return null;
  const [keyType] = match[1].split(",");
  if (!isValidSuiType(keyType)) return null;
  return keyType;
};

/**
 * This method removes leading zeroes for types in order to normalize them
 * since some types returned from the RPC have leading zeroes and others don't.
 */
export const trimSuiType = (type: string): string =>
  type.replace(/(0x)(0*)/g, "0x");

export const isValidSuiType = (type: string): boolean => {
  const tokens = type.split("::");
  if (tokens.length !== 3) {
    return false;
  }

  return isValidSuiAddress(tokens[0]) && !!tokens[1] && !!tokens[2];
};
