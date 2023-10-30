import {
  Connection,
  PublicKey,
  Commitment,
  PublicKeyInitData,
} from '@solana/web3.js';
import { utils } from '@wormhole-foundation/connect-sdk-solana';

export function deriveGuardianSetKey(
  wormholeProgramId: PublicKeyInitData,
  index: number,
): PublicKey {
  return utils.deriveAddress(
    [
      Buffer.from('GuardianSet'),
      (() => {
        const buf = Buffer.alloc(4);
        buf.writeUInt32BE(index);
        return buf;
      })(),
    ],
    wormholeProgramId,
  );
}

export async function getGuardianSet(
  connection: Connection,
  wormholeProgramId: PublicKeyInitData,
  index: number,
  commitment?: Commitment,
): Promise<GuardianSetData> {
  return connection
    .getAccountInfo(deriveGuardianSetKey(wormholeProgramId, index), commitment)
    .then((info) => GuardianSetData.deserialize(utils.getAccountData(info)));
}

export class GuardianSetData {
  index: number;
  keys: Buffer[];
  creationTime: number;
  expirationTime: number;

  constructor(
    index: number,
    keys: Buffer[],
    creationTime: number,
    expirationTime: number,
  ) {
    this.index = index;
    this.keys = keys;
    this.creationTime = creationTime;
    this.expirationTime = expirationTime;
  }

  static deserialize(data: Buffer): GuardianSetData {
    const index = data.readUInt32LE(0);
    const keysLen = data.readUInt32LE(4);
    const keysEnd = 8 + keysLen * utils.ETHEREUM_KEY_LENGTH;
    const creationTime = data.readUInt32LE(keysEnd);
    const expirationTime = data.readUInt32LE(4 + keysEnd);

    const keys = [];
    for (let i = 0; i < keysLen; ++i) {
      const start = 8 + i * utils.ETHEREUM_KEY_LENGTH;
      keys.push(data.subarray(start, start + utils.ETHEREUM_KEY_LENGTH));
    }
    return new GuardianSetData(index, keys, creationTime, expirationTime);
  }
}
