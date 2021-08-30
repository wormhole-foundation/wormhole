import {
  TxResult,
  ConnectedWallet as TerraConnectedWallet,
} from "@terra-money/wallet-provider";
import { LCDClient } from "@terra-money/terra.js";
import bech32 from "bech32";

// TODO: Loop txInfo for timed out transactions.
// lcd.tx.txInfo(transaction.result.txhash);
export async function waitForTerraExecution(
  wallet: TerraConnectedWallet,
  transaction: TxResult
) {
  new LCDClient({
    URL: wallet.network.lcd,
    chainID: "columbus-4",
  });
  return transaction;
}

export function canonicalAddress(humanAddress: string) {
  return new Uint8Array(bech32.fromWords(bech32.decode(humanAddress).words));
}
export function humanAddress(canonicalAddress: Uint8Array) {
  return bech32.encode("terra", bech32.toWords(canonicalAddress));
}
