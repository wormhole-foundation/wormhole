import { BlockTxBroadcastResult, Coin, Int } from "@terra-money/terra.js";

import { GAS_PRICE } from "./client";

export function parseEventsFromLog(receipt: BlockTxBroadcastResult): any[] {
  return JSON.parse(receipt.raw_log)[0].events;
}

export function computeGasPaid(receipt: BlockTxBroadcastResult): Int {
  const gasPrice = new Coin("uluna", GAS_PRICE).amount;
  // LocalTerra seems to spend all the gas_wanted
  // instead of spending gas_used...
  return new Int(gasPrice.mul(receipt.gas_wanted).ceil());
}
