import { TxResult, ConnectedWallet as TerraConnectedWallet } from "@terra-money/wallet-provider";
import { TxInfo, LCDClient } from "@terra-money/terra.js";

// TODO: Loop txInfo for timed out transactions.
// lcd.tx.txInfo(transaction.result.txhash);
export async function waitForTerraExecution(
  wallet: TerraConnectedWallet,
  transaction: TxResult
) {
  const lcd = new LCDClient({
    URL: wallet.network.lcd,
    chainID: "columbus-4", 
  });
  return transaction;
}
