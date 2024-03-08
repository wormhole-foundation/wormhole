import { Buffer } from "buffer";
import { BinaryWriter } from "./BinaryWriter";
import { ChainQueryType, ChainSpecificQuery } from "./request";
import { bigIntWithDef, coalesceUint8Array } from "./utils";
import { BinaryReader } from "./BinaryReader";
import { ChainSpecificResponse } from "./response";

export interface SolanaPdaEntry {
  programAddress: Uint8Array;
  seeds: Uint8Array[];
}

// According to the spec, there may be at most 16 seeds.
// https://github.com/gagliardetto/solana-go/blob/6fe3aea02e3660d620433444df033fc3fe6e64c1/keys.go#L559
export const SolanaMaxSeeds = 16;

// According to the spec, a seed may be at most 32 bytes.
// https://github.com/gagliardetto/solana-go/blob/6fe3aea02e3660d620433444df033fc3fe6e64c1/keys.go#L557
export const SolanaMaxSeedLen = 32;

export class SolanaPdaQueryRequest implements ChainSpecificQuery {
  commitment: string;
  minContextSlot: bigint;
  dataSliceOffset: bigint;
  dataSliceLength: bigint;

  constructor(
    commitment: "finalized",
    public pdas: SolanaPdaEntry[],
    minContextSlot?: bigint,
    dataSliceOffset?: bigint,
    dataSliceLength?: bigint
  ) {
    pdas.forEach((pda) => {
      if (pda.programAddress.length != 32) {
        throw new Error(
          `Invalid program address, must be 32 bytes: ${pda.programAddress}`
        );
      }
      if (pda.seeds.length == 0) {
        throw new Error(
          `Invalid pda, has no seeds: ${Buffer.from(
            pda.programAddress
          ).toString("hex")}`
        );
      }
      if (pda.seeds.length > SolanaMaxSeeds) {
        throw new Error(
          `Invalid pda, has too many seeds: ${Buffer.from(
            pda.programAddress
          ).toString("hex")}`
        );
      }
      pda.seeds.forEach((seed) => {
        if (seed.length == 0) {
          throw new Error(
            `Invalid pda, seed is null: ${Buffer.from(
              pda.programAddress
            ).toString("hex")}`
          );
        }
        if (seed.length > SolanaMaxSeedLen) {
          throw new Error(
            `Invalid pda, seed is too long: ${Buffer.from(
              pda.programAddress
            ).toString("hex")}`
          );
        }
      });
    });

    this.commitment = commitment;
    this.minContextSlot = bigIntWithDef(minContextSlot);
    this.dataSliceOffset = bigIntWithDef(dataSliceOffset);
    this.dataSliceLength = bigIntWithDef(dataSliceLength);
  }

  type(): ChainQueryType {
    return ChainQueryType.SolanaPda;
  }

  serialize(): Uint8Array {
    const writer = new BinaryWriter()
      .writeUint32(this.commitment.length)
      .writeUint8Array(Buffer.from(this.commitment))
      .writeUint64(this.minContextSlot)
      .writeUint64(this.dataSliceOffset)
      .writeUint64(this.dataSliceLength)
      .writeUint8(this.pdas.length);
    this.pdas.forEach((pda) => {
      writer.writeUint8Array(pda.programAddress).writeUint8(pda.seeds.length);
      pda.seeds.forEach((seed) => {
        writer.writeUint32(seed.length).writeUint8Array(seed);
      });
    });
    return writer.data();
  }

  static from(bytes: string | Uint8Array): SolanaPdaQueryRequest {
    const reader = new BinaryReader(coalesceUint8Array(bytes));
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): SolanaPdaQueryRequest {
    const commitmentLength = reader.readUint32();
    const commitment = reader.readString(commitmentLength);
    if (commitment !== "finalized") {
      throw new Error(`Invalid commitment: ${commitment}`);
    }
    const minContextSlot = reader.readUint64();
    const dataSliceOffset = reader.readUint64();
    const dataSliceLength = reader.readUint64();
    const numPdas = reader.readUint8();
    const pdas: SolanaPdaEntry[] = [];
    for (let idx = 0; idx < numPdas; idx++) {
      const programAddress = reader.readUint8Array(32);
      let seeds: Uint8Array[] = [];
      const numSeeds = reader.readUint8();
      for (let idx2 = 0; idx2 < numSeeds; idx2++) {
        const seedLen = reader.readUint32();
        const seed = reader.readUint8Array(seedLen);
        seeds.push(seed);
      }
      pdas.push({ programAddress, seeds });
    }
    return new SolanaPdaQueryRequest(
      commitment,
      pdas,
      minContextSlot,
      dataSliceOffset,
      dataSliceLength
    );
  }
}

export class SolanaPdaQueryResponse implements ChainSpecificResponse {
  slotNumber: bigint;
  blockTime: bigint;
  blockHash: Uint8Array;
  results: SolanaPdaResult[];

  constructor(
    slotNumber: bigint,
    blockTime: bigint,
    blockHash: Uint8Array,
    results: SolanaPdaResult[]
  ) {
    if (blockHash.length != 32) {
      throw new Error(
        `Invalid block hash, should be 32 bytes long: ${blockHash}`
      );
    }
    for (const result of results) {
      if (result.account.length != 32) {
        throw new Error(
          `Invalid account, should be 32 bytes long: ${result.account}`
        );
      }
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
    return ChainQueryType.SolanaPda;
  }

  serialize(): Uint8Array {
    const writer = new BinaryWriter()
      .writeUint64(this.slotNumber)
      .writeUint64(this.blockTime)
      .writeUint8Array(this.blockHash)
      .writeUint8(this.results.length);
    for (const result of this.results) {
      writer
        .writeUint8Array(result.account)
        .writeUint8(result.bump)
        .writeUint64(result.lamports)
        .writeUint64(result.rentEpoch)
        .writeUint8(result.executable ? 1 : 0)
        .writeUint8Array(result.owner)
        .writeUint32(result.data.length)
        .writeUint8Array(result.data);
    }
    return writer.data();
  }

  static from(bytes: string | Uint8Array): SolanaPdaQueryResponse {
    const reader = new BinaryReader(coalesceUint8Array(bytes));
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): SolanaPdaQueryResponse {
    const slotNumber = reader.readUint64();
    const blockTime = reader.readUint64();
    const blockHash = reader.readUint8Array(32);
    const resultsLength = reader.readUint8();
    const results: SolanaPdaResult[] = [];
    for (let idx = 0; idx < resultsLength; idx++) {
      const account = reader.readUint8Array(32);
      const bump = reader.readUint8();
      const lamports = reader.readUint64();
      const rentEpoch = reader.readUint64();
      const executableU8 = reader.readUint8();
      const executable = executableU8 != 0;
      const owner = reader.readUint8Array(32);
      const dataLength = reader.readUint32();
      const data = reader.readUint8Array(dataLength);
      results.push({
        account,
        bump,
        lamports,
        rentEpoch,
        executable,
        owner,
        data,
      });
    }
    return new SolanaPdaQueryResponse(
      slotNumber,
      blockTime,
      blockHash,
      results
    );
  }
}

export interface SolanaPdaResult {
  account: Uint8Array;
  bump: number;
  lamports: bigint;
  rentEpoch: bigint;
  executable: boolean;
  owner: Uint8Array;
  data: Uint8Array;
}
