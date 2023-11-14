import { Buffer } from "buffer";
import { BinaryWriter } from "./BinaryWriter";
import { HexString } from "./consts";
import { ChainQueryType, ChainSpecificQuery } from "./request";
import { hexToUint8Array, isValidHexString } from "./utils";

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
