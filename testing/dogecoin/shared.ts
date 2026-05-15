export const DOGECOIN_CHAIN_ID = 65;

// API endpoints
export const ELECTRS_API = "https://doge-electrs-testnet-demo.qed.me";
export const EXPLORER_URL = "https://doge-testnet-explorer.qed.me";

// Transaction constants
export const FEE = 1_000_000; // 0.01 DOGE fee
export const KOINU_PER_DOGE = 100_000_000;

// Fetch UTXOs for an address
export async function fetchUtxos(address: string): Promise<any[]> {
  const response = await fetch(`${ELECTRS_API}/address/${address}/utxo`);
  console.log(response);
  if (!response.ok) {
    throw new Error(`Failed to fetch UTXOs: ${response.statusText}`);
  }
  return response.json();
}

// Fetch raw transaction hex
export async function fetchRawTx(txid: string): Promise<string> {
  const response = await fetch(`${ELECTRS_API}/tx/${txid}/hex`);
  if (!response.ok) {
    throw new Error(`Failed to fetch raw tx: ${response.statusText}`);
  }
  return response.text();
}

// Broadcast transaction
export async function broadcastTx(txHex: string): Promise<string> {
  const response = await fetch(`${ELECTRS_API}/tx`, {
    method: "POST",
    headers: { "Content-Type": "text/plain" },
    body: txHex,
  });
  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to broadcast: ${error}`);
  }
  return response.text();
}

// Get explorer URL for a transaction
export function explorerTxUrl(txid: string): string {
  return `${EXPLORER_URL}/tx/${txid}`;
}
