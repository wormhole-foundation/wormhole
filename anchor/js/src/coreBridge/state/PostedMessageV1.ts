import { BN } from "@coral-xyz/anchor";
import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
} from "@solana/web3.js";

export enum MessageStatus {
  Unset = 0,
  Writing = 1,
}

export class PostedMessageV1 {
  finality: number;
  emitterAuthority: PublicKey;
  status: MessageStatus;
  _gap0: Buffer;
  postedTimestamp: number;
  nonce: number;
  sequence: BN;
  solanaChainId: number;
  emitter: PublicKey;
  payload: Buffer;

  protected constructor(
    finality: number,
    emitterAuthority: PublicKey,
    status: MessageStatus,
    _gap0: Buffer,
    postedTimestamp: number,
    nonce: number,
    sequence: BN,
    solanaChainId: number,
    emitter: PublicKey,
    payload: Buffer
  ) {
    this.finality = finality;
    this.emitterAuthority = emitterAuthority;
    this.status = status;
    this._gap0 = _gap0;
    this.postedTimestamp = postedTimestamp;
    this.nonce = nonce;
    this.sequence = sequence;
    this.solanaChainId = solanaChainId;
    this.emitter = emitter;
    this.payload = payload;
  }

  static fromAccountInfo(info: AccountInfo<Buffer>): PostedMessageV1 {
    const data = info.data;
    const discriminator = data.subarray(0, 4);
    if (!discriminator.equals(Buffer.from([109, 115, 103, 0]))) {
      throw new Error(`Invalid discriminator: ${discriminator}`);
    }
    return PostedMessageV1.deserialize(data.subarray(4));
  }

  static async fromAccountAddress(
    connection: Connection,
    address: PublicKey,
    commitmentOrConfig?: Commitment | GetAccountInfoConfig
  ): Promise<PostedMessageV1> {
    const accountInfo = await connection.getAccountInfo(
      address,
      commitmentOrConfig
    );
    if (accountInfo == null) {
      throw new Error(`Unable to find PostedMessageV1 account at ${address}`);
    }
    return PostedMessageV1.fromAccountInfo(accountInfo);
  }

  static deserialize(data: Buffer): PostedMessageV1 {
    const finality = data.readUInt8(0);
    const emitterAuthority = new PublicKey(data.subarray(1, 33));
    const status = (() => {
      switch (data.readUInt8(33)) {
        case 0: {
          return MessageStatus.Unset;
        }
        case 1: {
          return MessageStatus.Writing;
        }
        default: {
          throw new Error("Invalid MessageStatus");
        }
      }
    })();
    const _gap0 = data.subarray(34, 37);
    const postedTimestamp = data.readUInt32LE(37);
    const nonce = data.readUInt32LE(41);
    const sequence = new BN(data.subarray(45, 53), undefined, "le");
    const solanaChainId = data.readUInt16LE(53);
    const emitter = new PublicKey(data.subarray(55, 87));
    const payloadLen = data.readUInt32LE(87);
    const payload = data.subarray(91, 91 + payloadLen);

    return new PostedMessageV1(
      finality,
      emitterAuthority,
      status,
      _gap0,
      postedTimestamp,
      nonce,
      sequence,
      solanaChainId,
      emitter,
      payload
    );
  }
}
