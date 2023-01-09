import {
  Commitment,
  Connection,
  PublicKey,
  PublicKeyInitData,
} from "@solana/web3.js";
import { deriveAddress, getAccountData } from "../../utils";

export function deriveClaimKey(
  programId: PublicKeyInitData,
  emitterAddress: Buffer | Uint8Array | string,
  emitterChain: number,
  sequence: bigint | number
): PublicKey {
  const address =
    typeof emitterAddress == "string"
      ? Buffer.from(emitterAddress, "hex")
      : Buffer.from(emitterAddress);
  if (address.length != 32) {
    throw Error("address.length != 32");
  }
  const sequenceSerialized = Buffer.alloc(8);
  sequenceSerialized.writeBigUInt64BE(
    typeof sequence == "number" ? BigInt(sequence) : sequence
  );
  return deriveAddress(
    [
      address,
      (() => {
        const buf = Buffer.alloc(2);
        buf.writeUInt16BE(emitterChain as number);
        return buf;
      })(),
      sequenceSerialized,
    ],
    programId
  );
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
    .getAccountInfo(
      deriveClaimKey(programId, emitterAddress, emitterChain, sequence),
      commitment
    )
    .then((info) => !!getAccountData(info)[0]);
}
