import { Buffer } from "buffer";
import base58 from "bs58";
import { BinaryWriter } from "./BinaryWriter";
import { ChainQueryType, ChainSpecificQuery } from "./request";
import { bigIntWithDef, coalesceUint8Array } from "./utils";
import { BinaryReader } from "./BinaryReader";
import { ChainSpecificResponse } from "./response";

export class SolanaAccountQueryRequest implements ChainSpecificQuery {
  commitment: string;
  accounts: Uint8Array[];
  minContextSlot: bigint;
  dataSliceOffset: bigint;
  dataSliceLength: bigint;

  constructor(
    commitment: "finalized",
    accounts: string[],
    minContextSlot?: bigint,
    dataSliceOffset?: bigint,
    dataSliceLength?: bigint
  ) {
    this.commitment = commitment;
    this.minContextSlot = bigIntWithDef(minContextSlot);
    this.dataSliceOffset = bigIntWithDef(dataSliceOffset);
    this.dataSliceLength = bigIntWithDef(dataSliceLength);

    this.accounts = [];
    accounts.forEach((account) => {
      if (account.startsWith("0x")) {
        // Should be 32 bytes.
        if (account.length != 66) {
          throw new Error(`Invalid account, must be 32 bytes: ${account}`);
        }
        this.accounts.push(
          Uint8Array.from(Buffer.from(account.substring(2), "hex"))
        );
      } else {
        // Should be base58.
        this.accounts.push(Uint8Array.from(base58.decode(account)));
      }
    });
  }

  type(): ChainQueryType {
    return ChainQueryType.SolanaAccount;
  }

  serialize(): Uint8Array {
    const writer = new BinaryWriter()
      .writeUint32(this.commitment.length)
      .writeUint8Array(Buffer.from(this.commitment))
      .writeUint64(this.minContextSlot)
      .writeUint64(this.dataSliceOffset)
      .writeUint64(this.dataSliceLength)
      .writeUint8(this.accounts.length);
    this.accounts.forEach((account) => {
      writer.writeUint8Array(account);
    });
    return writer.data();
  }

  static from(bytes: string | Uint8Array): SolanaAccountQueryRequest {
    const reader = new BinaryReader(coalesceUint8Array(bytes));
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): SolanaAccountQueryRequest {
    const commitmentLength = reader.readUint32();
    const commitment = reader.readString(commitmentLength);
    if (commitment !== "finalized") {
      throw new Error(`Invalid commitment: ${commitment}`);
    }
    const minContextSlot = reader.readUint64();
    const dataSliceOffset = reader.readUint64();
    const dataSliceLength = reader.readUint64();
    const numAccounts = reader.readUint8();
    const accounts: string[] = [];
    for (let idx = 0; idx < numAccounts; idx++) {
      const account = reader.readUint8Array(32);
      // Add the "0x" prefix so the constructor knows it's hex rather than base58.
      accounts.push("0x" + Buffer.from(account).toString("hex"));
    }
    return new SolanaAccountQueryRequest(
      commitment,
      accounts,
      minContextSlot,
      dataSliceOffset,
      dataSliceLength
    );
  }
}

export class SolanaAccountQueryResponse implements ChainSpecificResponse {
  slotNumber: bigint;
  blockTime: bigint;
  blockHash: Uint8Array;
  results: SolanaAccountResult[];

  constructor(
    slotNumber: bigint,
    blockTime: bigint,
    blockHash: Uint8Array,
    results: SolanaAccountResult[]
  ) {
    if (blockHash.length != 32) {
      throw new Error(
        `Invalid block hash, should be 32 bytes long: ${blockHash}`
      );
    }
    for (const result of results) {
      if (result.owner.length != 32) {
        throw new Error(
          `Invalid owner, should be 32 bytes long: ${result.owner}`
        );
      }
    }
    this.slotNumber = slotNumber;
    this.blockTime = blockTime;
    this.blockHash = blockHash;
    this.results = results;
  }

  type(): ChainQueryType {
    return ChainQueryType.SolanaAccount;
  }

  serialize(): Uint8Array {
    const writer = new BinaryWriter()
      .writeUint64(this.slotNumber)
      .writeUint64(this.blockTime)
      .writeUint8Array(this.blockHash)
      .writeUint8(this.results.length);
    for (const result of this.results) {
      writer
        .writeUint64(result.lamports)
        .writeUint64(result.rentEpoch)
        .writeUint8(result.executable ? 1 : 0)
        .writeUint8Array(result.owner)
        .writeUint32(result.data.length)
        .writeUint8Array(result.data);
    }
    return writer.data();
  }

  static from(bytes: string | Uint8Array): SolanaAccountQueryResponse {
    const reader = new BinaryReader(coalesceUint8Array(bytes));
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): SolanaAccountQueryResponse {
    const slotNumber = reader.readUint64();
    const blockTime = reader.readUint64();
    const blockHash = reader.readUint8Array(32);
    const resultsLength = reader.readUint8();
    const results: SolanaAccountResult[] = [];
    for (let idx = 0; idx < resultsLength; idx++) {
      const lamports = reader.readUint64();
      const rentEpoch = reader.readUint64();
      const executableU8 = reader.readUint8();
      const executable = executableU8 != 0;
      const owner = reader.readUint8Array(32);
      const dataLength = reader.readUint32();
      const data = reader.readUint8Array(dataLength);
      results.push({ lamports, rentEpoch, executable, owner, data });
    }
    return new SolanaAccountQueryResponse(
      slotNumber,
      blockTime,
      blockHash,
      results
    );
  }
}

export interface SolanaAccountResult {
  lamports: bigint;
  rentEpoch: bigint;
  executable: boolean;
  owner: Uint8Array;
  data: Uint8Array;
}
