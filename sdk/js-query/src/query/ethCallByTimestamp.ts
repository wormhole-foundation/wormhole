import { Buffer } from "buffer";
import { BinaryWriter } from "./BinaryWriter";
import { BlockTag, EthCallData } from "./ethCall";
import { ChainQueryType, ChainSpecificQuery } from "./request";
import { coalesceUint8Array, hexToUint8Array, isValidHexString } from "./utils";
import { BinaryReader } from "./BinaryReader";
import { ChainSpecificResponse } from "./response";

export class EthCallByTimestampQueryRequest implements ChainSpecificQuery {
  targetTimestamp: bigint;
  targetBlockHint: string;
  followingBlockHint: string;

  constructor(
    targetTimestamp: bigint,
    targetBlockHint: BlockTag,
    followingBlockHint: BlockTag,
    public callData: EthCallData[]
  ) {
    this.targetTimestamp = targetTimestamp;
    this.targetBlockHint = parseBlockHint(targetBlockHint);
    this.followingBlockHint = parseBlockHint(followingBlockHint);
  }

  type(): ChainQueryType {
    return ChainQueryType.EthCallByTimeStamp;
  }

  serialize(): Uint8Array {
    const writer = new BinaryWriter()
      .writeUint64(this.targetTimestamp)
      .writeUint32(this.targetBlockHint.length)
      .writeUint8Array(Buffer.from(this.targetBlockHint))
      .writeUint32(this.followingBlockHint.length)
      .writeUint8Array(Buffer.from(this.followingBlockHint))
      .writeUint8(this.callData.length);
    this.callData.forEach(({ to, data }) => {
      const dataArray = hexToUint8Array(data);
      writer
        .writeUint8Array(hexToUint8Array(to))
        .writeUint32(dataArray.length)
        .writeUint8Array(dataArray);
    });
    return writer.data();
  }

  static from(bytes: string | Uint8Array): EthCallByTimestampQueryRequest {
    const reader = new BinaryReader(coalesceUint8Array(bytes));
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): EthCallByTimestampQueryRequest {
    const targetTimestamp = reader.readUint64();
    const targetBlockHintLength = reader.readUint32();
    const targetBlockHint = reader.readString(targetBlockHintLength);
    const followingBlockHintLength = reader.readUint32();
    const followingBlockHint = reader.readString(followingBlockHintLength);
    const callDataLength = reader.readUint8();
    const callData: EthCallData[] = [];
    for (let idx = 0; idx < callDataLength; idx++) {
      const to = reader.readHex(20);
      const dataLength = reader.readUint32();
      const data = reader.readHex(dataLength);
      callData.push({ to, data });
    }
    return new EthCallByTimestampQueryRequest(
      targetTimestamp,
      targetBlockHint,
      followingBlockHint,
      callData
    );
  }
}

function parseBlockHint(blockHint: BlockTag): string {
  // Block hints are not required.
  if (blockHint !== "") {
    if (typeof blockHint === "number") {
      if (blockHint < 0) {
        throw new Error(`block tag must be positive`);
      }
      blockHint = `0x${blockHint.toString(16)}`;
    } else if (isValidHexString(blockHint)) {
      if (!blockHint.startsWith("0x")) {
        blockHint = `0x${blockHint}`;
      }
    } else {
      throw new Error(`Invalid block tag: ${blockHint}`);
    }
  }

  return blockHint;
}

export class EthCallByTimestampQueryResponse implements ChainSpecificResponse {
  constructor(
    public targetBlockNumber: bigint,
    public targetBlockHash: string,
    public targetBlockTime: bigint,
    public followingBlockNumber: bigint,
    public followingBlockHash: string,
    public followingBlockTime: bigint,
    public results: string[] = []
  ) {}

  type(): ChainQueryType {
    return ChainQueryType.EthCallByTimeStamp;
  }

  serialize(): Uint8Array {
    const writer = new BinaryWriter()
      .writeUint64(this.targetBlockNumber)
      .writeUint8Array(hexToUint8Array(this.targetBlockHash))
      .writeUint64(this.targetBlockTime)
      .writeUint64(this.followingBlockNumber)
      .writeUint8Array(hexToUint8Array(this.followingBlockHash))
      .writeUint64(this.followingBlockTime)
      .writeUint8(this.results.length);
    for (const result of this.results) {
      const arr = hexToUint8Array(result);
      writer.writeUint32(arr.length).writeUint8Array(arr);
    }
    return writer.data();
  }

  static from(bytes: string | Uint8Array): EthCallByTimestampQueryResponse {
    const reader = new BinaryReader(coalesceUint8Array(bytes));
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): EthCallByTimestampQueryResponse {
    const targetBlockNumber = reader.readUint64();
    const targetBlockHash = reader.readHex(32);
    const targetBlockTime = reader.readUint64();
    const followingBlockNumber = reader.readUint64();
    const followingBlockHash = reader.readHex(32);
    const followingBlockTime = reader.readUint64();
    const resultsLength = reader.readUint8();
    const results: string[] = [];
    for (let idx = 0; idx < resultsLength; idx++) {
      const resultLength = reader.readUint32();
      const result = reader.readHex(resultLength);
      results.push(result);
    }
    return new EthCallByTimestampQueryResponse(
      targetBlockNumber,
      targetBlockHash,
      targetBlockTime,
      followingBlockNumber,
      followingBlockHash,
      followingBlockTime,
      results
    );
  }
}
