import { Commitment, Connection, PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { serializeUint16 } from "byteify";
import { deriveAddress, getAccountData } from "../../utils/account";

export function claimKey(
  programId: PublicKeyInitData,
  emitterAddress: Buffer | Uint8Array | string,
  emitterChain: number,
  sequence: bigint | number
): PublicKey {
  const address = typeof emitterAddress == "string" ? Buffer.from(emitterAddress, "hex") : Buffer.from(emitterAddress);
  if (address.length != 32) {
    throw Error("address.length != 32");
  }
  const sequenceSerialized = Buffer.alloc(8);
  sequenceSerialized.writeBigInt64BE(typeof sequence == "number" ? BigInt(sequence) : sequence);
  return deriveAddress([address, serializeUint16(emitterChain), sequenceSerialized], programId);
}

export async function getClaim(
  connection: Connection,
  programId: PublicKeyInitData,
  emitterAddress: Buffer | Uint8Array | string,
  emitterChain: number,
  sequence: bigint | number,
  commitment?: Commitment
): Promise<boolean> {
  return connection
    .getAccountInfo(claimKey(programId, emitterAddress, emitterChain, sequence), commitment)
    .then((info) => !!getAccountData(info)[0]);
}
