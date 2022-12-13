import { AccountsCoder, Idl } from "@project-serum/anchor";
import { accountSize, IdlTypeDef } from "../../anchor";

export class WormholeAccountsCoder<A extends string = string>
  implements AccountsCoder
{
  constructor(private idl: Idl) {}

  public async encode<T = any>(accountName: A, account: T): Promise<Buffer> {
    switch (accountName) {
      default: {
        throw new Error(`Invalid account name: ${accountName}`);
      }
    }
  }

  public decode<T = any>(accountName: A, ix: Buffer): T {
    return this.decodeUnchecked(accountName, ix);
  }

  public decodeUnchecked<T = any>(accountName: A, ix: Buffer): T {
    switch (accountName) {
      default: {
        throw new Error(`Invalid account name: ${accountName}`);
      }
    }
  }

  public memcmp(accountName: A, _appendData?: Buffer): any {
    switch (accountName) {
      case "postVaa": {
        return {
          dataSize: 56, // + 4 + payload.length
        };
      }
      default: {
        throw new Error(`Invalid account name: ${accountName}`);
      }
    }
  }

  public size(idlAccount: IdlTypeDef): number {
    return accountSize(this.idl, idlAccount) ?? 0;
  }
}

export interface PostVAAData {
  version: number;
  guardianSetIndex: number;
  timestamp: number;
  nonce: number;
  emitterChain: number;
  emitterAddress: Buffer;
  sequence: bigint;
  consistencyLevel: number;
  payload: Buffer;
}

function encodePostVaaData(account: PostVAAData): Buffer {
  const payload = account.payload;
  const serialized = Buffer.alloc(60 + payload.length);
  serialized.writeUInt8(account.version, 0);
  serialized.writeUInt32LE(account.guardianSetIndex, 1);
  serialized.writeUInt32LE(account.timestamp, 5);
  serialized.writeUInt32LE(account.nonce, 9);
  serialized.writeUInt16LE(account.emitterChain, 13);
  serialized.write(account.emitterAddress.toString("hex"), 15, "hex");
  serialized.writeBigUInt64LE(account.sequence, 47);
  serialized.writeUInt8(account.consistencyLevel, 55);
  serialized.writeUInt32LE(payload.length, 56);
  serialized.write(payload.toString("hex"), 60, "hex");

  return serialized;
}

function decodePostVaaAccount<T = any>(buf: Buffer): T {
  return {} as T;
}
