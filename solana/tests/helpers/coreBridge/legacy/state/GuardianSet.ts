import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
} from "@solana/web3.js";

const GUARDIAN_PUBKEY_LEN = 20;

export type GuardianPubkey = number[];

export class GuardianSet {
  index: number;
  keys: GuardianPubkey[];
  creationTime: number;
  expirationTime: number;

  private constructor(
    index: number,
    keys: GuardianPubkey[],
    creationTime: number,
    expirationTime: number
  ) {
    this.index = index;
    this.keys = keys;
    this.creationTime = creationTime;
    this.expirationTime = expirationTime;
  }

  static address(programId: PublicKey, guardianSetIndex: number): PublicKey {
    const guardianSetIndexBuf = Buffer.alloc(4);
    guardianSetIndexBuf.writeUInt32BE(guardianSetIndex, 0);
    return PublicKey.findProgramAddressSync(
      [Buffer.from("GuardianSet"), guardianSetIndexBuf],
      programId
    )[0];
  }

  static fromAccountInfo(info: AccountInfo<Buffer>): GuardianSet {
    return GuardianSet.deserialize(info.data);
  }

  static async fromAccountAddress(
    connection: Connection,
    address: PublicKey,
    commitmentOrConfig?: Commitment | GetAccountInfoConfig
  ): Promise<GuardianSet> {
    const accountInfo = await connection.getAccountInfo(address, commitmentOrConfig);
    if (accountInfo == null) {
      throw new Error(`Unable to find GuardianSet account at ${address}`);
    }
    return GuardianSet.fromAccountInfo(accountInfo);
  }

  static async fromPda(
    connection: Connection,
    programId: PublicKey,
    guardianSetIndex: number
  ): Promise<GuardianSet> {
    return GuardianSet.fromAccountAddress(
      connection,
      GuardianSet.address(programId, guardianSetIndex)
    );
  }

  static deserialize(data: Buffer): GuardianSet {
    const index = data.readUInt32LE(0);
    const keysLen = data.readUInt32LE(4);
    const keysEnd = 8 + keysLen * GUARDIAN_PUBKEY_LEN;
    const creationTime = data.readUInt32LE(keysEnd);
    const expirationTime = data.readUInt32LE(4 + keysEnd);

    const keys = [];
    for (let i = 0; i < keysLen; ++i) {
      const start = 8 + i * GUARDIAN_PUBKEY_LEN;
      keys.push(Array.from(data.subarray(start, start + GUARDIAN_PUBKEY_LEN)));
    }
    return new GuardianSet(index, keys, creationTime, expirationTime);
  }
}
