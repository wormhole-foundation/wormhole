import { SuiGrpcClient } from "@mysten/sui/grpc";
import { bcs } from "@mysten/sui/bcs";
import { Chain, chainToChainId } from "@wormhole-foundation/sdk";
import { normalizeSuiAddress } from "../chains/sui/utils";

// BCS schema for the `token_registry::CoinTypeKey` dynamic-field key used to look
// up a wrapped/native coin type by (chain, address). Field order must match the
// Move struct: `chain: u16` then `addr: vector<u8>` (a left-padded 32-byte
// external address). Verified against the on-chain table on Sui mainnet.
export const CoinTypeKeyBcs = bcs.struct("CoinTypeKey", {
  chain: bcs.u16(),
  addr: bcs.vector(bcs.u8()),
});

export async function getForeignAssetSui(
  client: SuiGrpcClient,
  tokenBridgeStateObjectId: string,
  originChain: Chain,
  originAddress: Uint8Array
): Promise<string | null> {
  const originChainId = chainToChainId(originChain);
  return getTokenCoinType(
    client,
    tokenBridgeStateObjectId,
    originAddress,
    originChainId
  );
}

/** Marker key value (`struct Key has copy, drop, store { dummy_field: bool }`). */
const RegistryKeyBcs = bcs.struct("Key", { dummy_field: bcs.bool() });

export interface SuiOriginalAssetInfo {
  isWrapped: boolean;
  chainId: number;
  assetAddress: Uint8Array;
}

/**
 * Reverse lookup for a Sui coin type: determine whether it is a Wormhole-wrapped
 * asset and, if so, its origin chain and address. Reads the
 * `token_registry::Key<CoinType>` dynamic field and inspects whether the stored
 * value is a `WrappedAsset` or `NativeAsset`.
 */
export const getOriginalAssetSui = async (
  client: SuiGrpcClient,
  tokenBridgeStateObjectId: string,
  coinType: string
): Promise<SuiOriginalAssetInfo> => {
  if (!isValidSuiType(coinType)) {
    throw new Error(`Invalid Sui type: ${coinType}`);
  }

  const state = await client.getObject({
    objectId: tokenBridgeStateObjectId,
    include: { json: true },
  });
  const registryId = (state.object.json as any)?.token_registry?.id;
  if (!registryId) {
    throw new Error("Unable to fetch token registry object ID");
  }
  const originalPackageId = state.object.type.split("::")[0];

  let fieldId: string;
  try {
    const res = await client.getDynamicField({
      parentId: registryId,
      name: {
        type: `${originalPackageId}::token_registry::Key<${coinType}>`,
        bcs: RegistryKeyBcs.serialize({ dummy_field: false }).toBytes(),
      },
    });
    fieldId = res.dynamicField.fieldId;
  } catch {
    throw new Error(
      `Token of type ${coinType} has not been registered with the token bridge`
    );
  }

  const obj = await client.getObject({
    objectId: fieldId,
    include: { json: true },
  });
  const type = obj.object.type;
  const value = (obj.object.json as any)?.value;

  if (type.includes("wrapped_asset::WrappedAsset<")) {
    return {
      isWrapped: true,
      chainId: Number(value.info.token_chain),
      assetAddress: new Uint8Array(
        Buffer.from(value.info.token_address.value.data, "base64")
      ),
    };
  } else if (type.includes("native_asset::NativeAsset<")) {
    return {
      isWrapped: false,
      chainId: chainToChainId("Sui"),
      assetAddress: new Uint8Array(
        Buffer.from(value.token_address.value.data, "base64")
      ),
    };
  }

  throw new Error(
    `Unrecognized token metadata type ${type} for ${coinType}`
  );
};

export const getTokenCoinType = async (
  client: SuiGrpcClient,
  tokenBridgeStateObjectId: string,
  tokenAddress: Uint8Array,
  tokenChain: number
): Promise<string | null> => {
  const state = await client.getObject({
    objectId: tokenBridgeStateObjectId,
    include: { json: true },
  });

  const fields = state.object.json as any;
  const coinTypesObjectId = fields?.token_registry?.coin_types?.id;
  if (!coinTypesObjectId) {
    throw new Error("Unable to fetch coin types");
  }

  // The dynamic-field key type is declared in the *original* package that
  // defines `token_registry`, which is the package embedded in the state
  // object's type (Move type origins are stable across upgrades), not the
  // upgraded package returned by `getPackageId`.
  const originalPackageId = state.object.type.split("::")[0];
  const keyType = `${originalPackageId}::token_registry::CoinTypeKey`;
  const keyBcs = CoinTypeKeyBcs.serialize({
    chain: tokenChain,
    addr: Array.from(tokenAddress),
  }).toBytes();

  try {
    const res = await client.getDynamicField({
      parentId: coinTypesObjectId,
      name: { type: keyType, bcs: keyBcs },
    });
    // The dynamic-field value is an `ascii::String` holding the coin type.
    const coinType = bcs.string().parse(res.dynamicField.value.bcs);
    return coinType ? trimSuiType(ensureHexPrefix(coinType)) : null;
  } catch {
    // Field not found -> the asset is not registered.
    return null;
  }
};

/**
 * Fetch an object's Move struct fields as JSON. The gRPC `json` representation
 * is flattened relative to the legacy JSON-RPC `content.fields` shape: there are
 * no nested `.fields` wrappers and a UID `id.id` collapses to a plain `id`
 * string. Returns `null` if the object has no Move struct content.
 */
export const getObjectFields = async (
  client: SuiGrpcClient,
  objectId: string
): Promise<Record<string, any> | null> => {
  if (!isValidSuiAddress(objectId)) {
    throw new Error(`Invalid object ID: ${objectId}`);
  }

  const res = await client.getObject({
    objectId,
    include: { json: true },
  });
  return (res.object.json as Record<string, any> | null) ?? null;
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
export const isValidSuiAddress = (address: string): boolean => {
  try {
    const normalized = normalizeSuiAddress(address);
    return /^0x[a-fA-F0-9]{64}$/.test(normalized);
  } catch {
    return false;
  }
};

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
