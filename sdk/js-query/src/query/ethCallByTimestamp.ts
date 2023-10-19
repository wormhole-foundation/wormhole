import { Buffer } from "buffer";
import { BinaryWriter } from "./BinaryWriter";
import { BlockTag, EthCallData } from "./ethCall";
import { ChainQueryType, ChainSpecificQuery } from "./request";
import { hexToUint8Array, isValidHexString } from "./utils";

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
}

function parseBlockHint(blockHint: BlockTag): string {
  // Block hints are not required.
  if (blockHint != "") {
    if (typeof blockHint === "number") {
      blockHint = `0x${blockHint.toString(16)}`;
    } else if (isValidHexString(blockHint)) {
      if (!blockHint.startsWith("0x")) {
        blockHint = `0x${blockHint}`;
      }
      blockHint = blockHint;
    } else {
      throw new Error(`Invalid block tag: ${blockHint}`);
    }
  }

  return blockHint;
}
