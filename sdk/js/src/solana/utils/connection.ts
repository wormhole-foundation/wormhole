import { Provider } from "@project-serum/anchor";
import { Connection } from "@solana/web3.js";

export function createReadOnlyProvider(
  connection?: Connection
): Provider | undefined {
  if (connection === undefined) {
    return undefined;
  }

  return { connection };
}
