import {
  connectToTerra,
  queryBalanceOnTerra,
  queryTerra,
  relayTerra,
  setAccountNumOnTerra,
  setSeqNumOnTerra,
  TerraConnectionData,
} from "./terra";

export type ConnectionData = {
  terraData: TerraConnectionData;
};

import { logger } from "../helpers";

export function connectRelayer(): ConnectionData {
  let td = connectToTerra();
  return { terraData: td };
}

export async function setAccountNum(connectionData: ConnectionData) {
  try {
    await setAccountNumOnTerra(connectionData.terraData);
  } catch (e) {
    logger.error("setAccountNum: query failed: %o", e);
  }
}

export async function setSeqNum(connectionData: ConnectionData) {
  try {
    await setSeqNumOnTerra(connectionData.terraData);
  } catch (e) {
    logger.error("setSeqNum: query failed: %o", e);
  }
}

// Exceptions from this method are caught at the higher level.
export async function relay(
  signedVAAs: Array<string>,
  connectionData: ConnectionData
): Promise<any> {
  return await relayTerra(connectionData.terraData, signedVAAs);
}

export async function query(
  productIdStr: string,
  priceIdStr: string
): Promise<any> {
  let result: any;
  try {
    let terraData = connectToTerra();
    result = await queryTerra(terraData, productIdStr, priceIdStr);
  } catch (e) {
    logger.error("query failed: %o", e);
    result = "Error: unhandled exception";
  }

  return result;
}

export async function queryBalance(
  connectionData: ConnectionData
): Promise<number> {
  let balance: number = NaN;
  try {
    balance = await queryBalanceOnTerra(connectionData.terraData);
  } catch (e) {
    logger.error("balance query failed: %o", e);
  }

  return balance;
}
