import { isNativeTerra } from "@certusone/wormhole-sdk";
import { formatUnits } from "@ethersproject/units";
import { LCDClient, Dec, Int } from "@terra-money/terra.js";
import { TxResult } from "@terra-money/wallet-provider";
// import { TerraTokenMetadata } from "../hooks/useTerraTokenMap";
import { TERRA_HOST } from "./consts";

export const NATIVE_TERRA_DECIMALS = 6;

export const getNativeTerraIcon = (symbol = "") =>
  `https://assets.terra.money/icon/60/${symbol}.png`;

// inspired by https://github.com/terra-money/station/blob/dca7de43958ce075c6e46605622203b9859b0e14/src/lib/utils/format.ts#L38
export const formatNativeDenom = (denom = ""): string => {
  const unit = denom.slice(1).toUpperCase();
  const isValidTerra = isNativeTerra(denom);
  return denom === "uluna"
    ? "Luna"
    : isValidTerra
    ? unit.slice(0, 2) + "T"
    : "";
};

export const formatTerraNativeBalance = (balance = ""): string =>
  formatUnits(balance, 6);

export async function waitForTerraExecution(transaction: TxResult) {
  const lcd = new LCDClient(TERRA_HOST);
  let info;
  while (!info) {
    await new Promise((resolve) => setTimeout(resolve, 1000));
    try {
      info = await lcd.tx.txInfo(transaction.result.txhash);
    } catch (e) {
      console.error(e);
    }
  }
  return info;
}

export async function calculateTerraTax(
  amount: string,
  denom: string
): Promise<string> {
  try {
    // Fetch terra tax state from current chain height.
    const lcd = new LCDClient(TERRA_HOST);
    const taxRate = await lcd.treasury.taxRate();
    const taxCap = await lcd.treasury.taxCap(denom);

    // Calculate tax rate for the current denomination.
    const untaxed = new Int(amount);
    const tax = untaxed.toDec().mul(taxRate);
    const cap = (taxCap.amount as Int).toDec();
    const min = new Dec((tax.constructor as any).min(tax, cap));
    return untaxed.sub(min.toInt()).toString();
  } catch(e) {
    return "0";
  }
}
