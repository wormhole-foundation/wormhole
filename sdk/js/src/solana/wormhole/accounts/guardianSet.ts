import { Connection, PublicKey, Commitment, PublicKeyInitData } from "@solana/web3.js";
import { serializeUint32 } from "byteify";
import { deriveAddress, getAccountData } from "../../utils/account";
import { KEY_LENGTH } from "../../utils/secp256k1";

export function guardianSetKey(wormholeProgramId: PublicKeyInitData, index: number): PublicKey {
  return deriveAddress([Buffer.from("GuardianSet"), serializeUint32(index)], wormholeProgramId);
}

export async function getGuardianSet(
  connection: Connection,
  wormholeProgramId: PublicKeyInitData,
  index: number,
  commitment?: Commitment
): Promise<GuardianSetData> {
  return connection
    .getAccountInfo(guardianSetKey(wormholeProgramId, index), commitment)
    .then((info) => GuardianSetData.deserialize(getAccountData(info)));
}

export class GuardianSetData {
  // pub struct GuardianSetData {
  //     /// Index representing an incrementing version number for this guardian set.
  //     pub index: u32,

  //     /// ETH style public keys
  //     pub keys: Vec<GuardianPublicKey>,

  //     /// Timestamp representing the time this guardian became active.
  //     pub creation_time: u32,

  //     /// Expiration time when VAAs issued by this set are no longer valid.
  //     pub expiration_time: u32,
  // }

  index: number;
  keys: Buffer[];
  creationTime: number;
  expirationTime: number;

  constructor(index: number, keys: Buffer[], creationTime: number, expirationTime: number) {
    this.index = index;
    this.keys = keys;
    this.creationTime = creationTime;
    this.expirationTime = expirationTime;
  }

  static deserialize(data: Buffer): GuardianSetData {
    const index = data.readUInt32LE(0);
    const keysLen = data.readUInt32LE(4);
    const keysEnd = 8 + keysLen * KEY_LENGTH;
    const creationTime = data.readUInt32LE(keysEnd);
    const expirationTime = data.readUInt32LE(4 + keysEnd);

    const keys = [];
    for (let i = 0; i < keysLen; ++i) {
      const start = 8 + i * KEY_LENGTH;
      keys.push(data.subarray(start, start + KEY_LENGTH));
    }
    return new GuardianSetData(index, keys, creationTime, expirationTime);
  }
}
