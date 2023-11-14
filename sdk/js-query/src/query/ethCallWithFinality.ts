import { BinaryWriter } from "./BinaryWriter";
import { BlockTag, EthCallData, parseBlockId } from "./ethCall";
import { ChainQueryType, ChainSpecificQuery } from "./request";
import { hexToUint8Array, isValidHexString } from "./utils";

export class EthCallWithFinalityQueryRequest implements ChainSpecificQuery {
  blockId: string;
  finality: "safe" | "finalized";

  constructor(
    blockId: BlockTag,
    finality: "safe" | "finalized",
    public callData: EthCallData[]
  ) {
    this.blockId = parseBlockId(blockId);
    this.finality = finality;
  }

  type(): ChainQueryType {
    return ChainQueryType.EthCallWithFinality;
  }

  serialize(): Uint8Array {
    const writer = new BinaryWriter()
      .writeUint32(this.blockId.length)
      .writeUint8Array(Buffer.from(this.blockId))
      .writeUint32(this.finality.length)
      .writeUint8Array(Buffer.from(this.finality))
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
