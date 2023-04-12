import {
  isValidSuiAddress,
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

export const getInnerType = (type: string): string | null => {
  const match = type.match(/<(.*)>/);
  if (!match || !isValidSuiType(match[1])) {
    return null;
  }

  return match[1];
};

export const isValidSuiType = (type: string): boolean => {
  const tokens = type.split("::");
  if (tokens.length !== 3) {
    return false;
  }

  return isValidSuiAddress(tokens[0]) && !!tokens[1] && !!tokens[2];
};
