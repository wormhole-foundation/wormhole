import { getSignedVAA, CHAIN_ID_SOLANA} from "@certusone/wormhole-sdk";
import { zeroPad } from "ethers/lib/utils";
import { PublicKey} from "@solana/web3.js";

var P2W_INSTANCE: any = undefined;

// Import p2w wasm bindings; be smart about it
export async function p2w_core(): Promise<any> {
    // Only import once if P2W wasm is needed
    if (!P2W_INSTANCE) {
	    P2W_INSTANCE = await import("../../rust/pkg");
    }
    return P2W_INSTANCE;
}

export function sol_addr2buf(addr: string): Buffer {
    return Buffer.from(zeroPad(new PublicKey(addr).toBytes(), 32));
}

export async function parseAttestation(vaa_payload: Uint8Array): Promise<any> {
    const p2w = await p2w_core();

    return await p2w.parse_attestation(vaa_payload);
}

export async function parseBatchAttestation(vaa_payload: Uint8Array): Promise<any> {
    const p2w = await p2w_core();

    console.log("p2w.parse_batch_attestaion is", p2w.parse_batch_attestation);

    return await p2w.parse_batch_attestation(vaa_payload);
}
