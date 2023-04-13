import {
  isValidSuiAddress as isValidFullSuiAddress,
  normalizeSuiAddress,
  RawSigner,
  SuiTransactionBlockResponse,
  TransactionBlock,
} from "@mysten/sui.js";

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

export const getInnerType = (type: string): string | null => {
  const match = type.match(/<(.*)>/);
  if (!match || !isValidSuiType(match[1])) {
    return null;
  }

  return match[1];
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
