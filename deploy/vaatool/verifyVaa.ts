import { createPublicClient, http, Address, Hex, PublicClient } from "viem";
import { Chain, isChainId, toChain } from "@wormhole-foundation/sdk"; 

const VERIFICATION_FAILED_ERROR_SIGNATURE = "0x32629d58";

const WORMHOLE_VERIFIER_ABI = [
  {
    inputs: [{ internalType: "bytes", name: "data", type: "bytes" }],
    name: "verify",
    outputs: [
      { internalType: "uint16", name: "emitterChainId", type: "uint16" },
      { internalType: "bytes32", name: "emitterAddress", type: "bytes32" },
      { internalType: "uint64", name: "sequence", type: "uint64" },
      { internalType: "uint16", name: "payloadOffset", type: "uint16" },
    ],
    stateMutability: "view",
    type: "function",
  },
] as const;

export type VerificationResult = {
  verified: true;
  emitterChainId: number;
  emitterAddress: Address;
  sequence: bigint;
  payloadOffset: number;
} | {
  verified: false;
  error: string;
}

async function verifyVaa(
  client: PublicClient,
  verifierAddress: Address,
  vaa: Hex,
): Promise<VerificationResult> {
  try {
    const result = await client.readContract({
      address: verifierAddress,
      abi: WORMHOLE_VERIFIER_ABI,
      functionName: "verify",
      args: [vaa],
    });
    return {
      verified: true,
      emitterChainId: result[0],
      emitterAddress: result[1],
      sequence: result[2],
      payloadOffset: result[3],
    };
  } catch (error: any) {
    if (!("cause" in error && "raw" in error.cause)) {
      return { verified: false, error: error instanceof Error ? error.message : "Unknown error" };
    }
    const hexData = error.cause.raw as Hex;
    if (hexData.startsWith(VERIFICATION_FAILED_ERROR_SIGNATURE)) {
      const flags = hexData.slice(VERIFICATION_FAILED_ERROR_SIGNATURE.length);
      return { verified: false, error: `Verification failed with flags: 0x${flags}` };
    }
    return { verified: false, error: `Contract call failed with unknown error data: ${hexData}` };
  }
}

function toMaybeUnknownChain(chainId: number): Chain | "Unknown" {
  if (isChainId(chainId)) {
    return toChain(chainId);
  }
  return "Unknown";
}

function getVaaType(vaa: Hex): "Multisig" | "Schnorr" | undefined {
  if (vaa.startsWith("0x01")) {
    return "Multisig";
  }
  if (vaa.startsWith("0x02")) {
    return "Schnorr";
  }
  return undefined;
}

async function main() {
  const rpcUrl = process.argv[2];
  const verifierAddress = process.argv[3] as Address;
  const vaaBase64 = process.argv[4];
  if (!vaaBase64) {
    console.error("Usage: tsx verifyV2Vaa.ts <rpc_url> <verifier_address> <base64_v2_vaa>");
    process.exit(1);
  }
  const client = createPublicClient({ transport: http(rpcUrl),});
  const vaaHex = ("0x" + Buffer.from(vaaBase64, "base64").toString("hex")) as Hex;
  const vaaType = getVaaType(vaaHex);
  if (vaaType === undefined) {
    console.error(`Invalid VAA type`);
    process.exit(1);
  }
  console.log(`Verifying ${vaaType} VAA...`);
  const result = await verifyVaa(client, verifierAddress, vaaHex);
  if (!result.verified) {
    console.error("VAA verification failed:");
    console.error(result.error);
    process.exit(1);
  }
  const emitterChain = toMaybeUnknownChain(result.emitterChainId);
  console.log("VAA verified successfully");
  console.log("================================================");
  console.log(`Emitter Chain: ${emitterChain} (${result.emitterChainId})`);
  console.log("Emitter Address:", result.emitterAddress);
  console.log("Sequence:", result.sequence.toString());
  console.log("Payload Offset:", result.payloadOffset);
  console.log("================================================");
}

await main().catch((error) => console.error(error));
