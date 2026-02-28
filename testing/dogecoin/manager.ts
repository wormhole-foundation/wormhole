// https://sepolia.etherscan.io/address/0x086a699900262d829512299abe07648870000dd1#readContract
export const M_THRESHOLD = 5;
export const N_TOTAL = 7;
const currentManagerSet =
  "0x01050702349de56ca5dd06db8660419d6f150662e0f04febdbf6512d7cfe78c23b51491c035163bfd9518b0a536a17f330a1589fe21d7404b51f525a0a990a65a701952ebb036d40b0b85bca49e41f05a26950578bb13a424507ce34a80f83d3cf601e25818b0307681002ae28b9399e828d0f46d54c31d5d6ff187b3bdddc6615987a466455f50375abc8955c8a8c875ee1febd157132adcc1b992d69a946e83485b8360e23a277030212d206546216917a75533ed6c975f8f794ba0d8a7fb84dedf65ebb20e64841037ff483369b52bd87a73f23413dd8fcace71de7f7823c5c9120f1e9cfe5733a88";
const managerPubkeys = currentManagerSet
  .substring(8)
  .match(/.{66}/g)!
  .map((x) => Buffer.from(x, "hex"));
// TODO: read this dynamically and correctly from the contract
export async function loadManagerKeys(chainId: number) {
  return { mThreshold: M_THRESHOLD, nTotal: N_TOTAL, pubkeys: managerPubkeys };
}

// Manager signatures response type
export interface ManagerSignaturesResponse {
  vaaHash: string;
  vaaId: string;
  destinationChain: number;
  managerSetIndex: number;
  required: number;
  total: number;
  isComplete: boolean;
  signatures: {
    signerIndex: number;
    signatures: string[]; // base64-encoded DER signatures
  }[];
}

// Fetch signatures from Guardian Manager RPC
export async function fetchManagerSignatures(
  guardianRpc: string,
  emitterChain: number,
  emitterAddress: string,
  sequence: bigint,
  maxRetries: number = 60,
  retryDelayMs: number = 2000,
): Promise<ManagerSignaturesResponse> {
  const url = `${guardianRpc}/v1/manager/signed_vaa/${emitterChain}/${emitterAddress}/${sequence}`;

  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    console.log(
      `Fetching manager signatures (attempt ${attempt}/${maxRetries}): ${url}`,
    );

    try {
      const response = await fetch(url);
      if (response.ok) {
        const data = (await response.json()) as ManagerSignaturesResponse;
        if (data.isComplete) {
          console.log("Signatures complete!");
          console.log(`  Required: ${data.required}/${data.total}`);
          console.log(`  Signatures collected: ${data.signatures.length}`);
          return data;
        } else {
          console.log(
            `Signatures incomplete: ${data.signatures.length}/${data.required} required`,
          );
        }
      } else {
        const text = await response.text();
        console.log(`Response ${response.status}: ${text}`);
      }
    } catch (error) {
      console.log(`Fetch error: ${error}`);
    }

    if (attempt < maxRetries) {
      console.log(`Signatures not ready, waiting ${retryDelayMs / 1000}s...`);
      await new Promise((resolve) => setTimeout(resolve, retryDelayMs));
    }
  }

  throw new Error("Failed to fetch manager signatures after max retries");
}
