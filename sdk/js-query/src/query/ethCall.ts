import { Buffer } from "buffer";
import { BinaryWriter } from "./BinaryWriter";
import { HexString } from "./consts";
import { ChainQueryType, ChainSpecificQuery } from "./request";
import { coalesceUint8Array, hexToUint8Array, isValidHexString } from "./utils";
import { BinaryReader } from "./BinaryReader";
import { ChainSpecificResponse } from "./response";

export interface EthCallData {
  to: string;
  data: string;
}

// Can be a block number or a block hash
export type BlockTag = number | HexString;

export class EthCallQueryRequest implements ChainSpecificQuery {
  blockTag: string;

  constructor(blockTag: BlockTag, public callData: EthCallData[]) {
    this.blockTag = parseBlockId(blockTag);
  }

  type(): ChainQueryType {
    return ChainQueryType.EthCall;
  }

  serialize(): Uint8Array {
    const writer = new BinaryWriter()
      .writeUint32(this.blockTag.length)
      .writeUint8Array(Buffer.from(this.blockTag))
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

  static from(bytes: string | Uint8Array): EthCallQueryRequest {
    const reader = new BinaryReader(coalesceUint8Array(bytes));
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): EthCallQueryRequest {
    const blockTagLength = reader.readUint32();
    const blockTag = reader.readString(blockTagLength);
    const callDataLength = reader.readUint8();
    const callData: EthCallData[] = [];
    for (let idx = 0; idx < callDataLength; idx++) {
      const to = reader.readHex(20);
      const dataLength = reader.readUint32();
      const data = reader.readHex(dataLength);
      callData.push({ to, data });
    }
    return new EthCallQueryRequest(blockTag, callData);
  }
}

export function parseBlockId(blockId: BlockTag): string {
  if (!blockId || blockId === "") {
    throw new Error(`block tag is required`);
  }

  if (typeof blockId === "number") {
    if (blockId < 0) {
      throw new Error(`block tag must be positive`);
    }
    blockId = `0x${blockId.toString(16)}`;
  } else if (isValidHexString(blockId)) {
    if (!blockId.startsWith("0x")) {
      blockId = `0x${blockId}`;
    }
  } else {
    throw new Error(`Invalid block tag: ${blockId}`);
  }

  return blockId;
}

export class EthCallQueryResponse implements ChainSpecificResponse {
  constructor(
    public blockNumber: bigint,
    public blockHash: string,
    public blockTime: bigint,
    public results: string[] = []
  ) {}

  type(): ChainQueryType {
    return ChainQueryType.EthCall;
  }

  serialize(): Uint8Array {
    const writer = new BinaryWriter()
      .writeUint64(this.blockNumber)
      .writeUint8Array(hexToUint8Array(this.blockHash))
      .writeUint64(this.blockTime)
      .writeUint8(this.results.length);
    for (const result of this.results) {
      const arr = hexToUint8Array(result);
      writer.writeUint32(arr.length).writeUint8Array(arr);
    }
    return writer.data();
  }

  static from(bytes: string | Uint8Array): EthCallQueryResponse {
    const reader = new BinaryReader(coalesceUint8Array(bytes));
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): EthCallQueryResponse {
    const blockNumber = reader.readUint64();
    const blockHash = reader.readHex(32);
    const blockTime = reader.readUint64();
    const resultsLength = reader.readUint8();
    const results: string[] = [];
    for (let idx = 0; idx < resultsLength; idx++) {
      const resultLength = reader.readUint32();
      const result = reader.readHex(resultLength);
      results.push(result);
    }
    return new EthCallQueryResponse(blockNumber, blockHash, blockTime, results);
  }
}
