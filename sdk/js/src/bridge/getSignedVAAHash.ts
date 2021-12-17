import { importCoreWasm } from "../solana/wasm";
import { ethers } from "ethers";
import { uint8ArrayToHex } from "..";

export async function getSignedVAAHash(signedVAA: Uint8Array) {
  const { parse_vaa } = await importCoreWasm();
  const parsedVAA = parse_vaa(signedVAA);
  const body = [
    ethers.utils.defaultAbiCoder.encode(["uint32"], [parsedVAA.timestamp]).substring(2 + (64 - 8)),
    ethers.utils.defaultAbiCoder.encode(["uint32"], [parsedVAA.nonce]).substring(2 + (64 - 8)),
    ethers.utils.defaultAbiCoder.encode(["uint16"], [parsedVAA.emitter_chain]).substring(2 + (64 - 4)),
    ethers.utils.defaultAbiCoder.encode(["bytes32"], [parsedVAA.emitter_address]).substring(2),
    ethers.utils.defaultAbiCoder.encode(["uint64"], [parsedVAA.sequence]).substring(2 + (64 - 16)),
    ethers.utils.defaultAbiCoder.encode(["uint8"], [parsedVAA.consistency_level]).substring(2 + (64 - 2)),
    uint8ArrayToHex(parsedVAA.payload),
  ];
  return ethers.utils.solidityKeccak256(["bytes"], [ethers.utils.solidityKeccak256(["bytes"], ["0x" + body.join("")])]);
}
