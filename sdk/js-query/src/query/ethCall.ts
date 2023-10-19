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
    if (typeof blockTag === "number") {
      this.blockTag = `0x${blockTag.toString(16)}`;
    } else if (isValidHexString(blockTag)) {
      if (!blockTag.startsWith("0x")) {
        blockTag = `0x${blockTag}`;
      }
      this.blockTag = blockTag;
    } else {
      throw new Error(`Invalid block tag: ${blockTag}`);
    }
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
