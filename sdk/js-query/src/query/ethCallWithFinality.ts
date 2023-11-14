import { BinaryReader } from "./BinaryReader";
import { BinaryWriter } from "./BinaryWriter";
import {
  BlockTag,
  EthCallData,
  EthCallQueryResponse,
  parseBlockId,
} from "./ethCall";
import { ChainQueryType, ChainSpecificQuery } from "./request";
import { ChainSpecificResponse } from "./response";
import { coalesceUint8Array, hexToUint8Array, isValidHexString } from "./utils";

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

  static from(bytes: string | Uint8Array): EthCallWithFinalityQueryRequest {
    const reader = new BinaryReader(coalesceUint8Array(bytes));
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): EthCallWithFinalityQueryRequest {
    const blockTagLength = reader.readUint32();
    const blockTag = reader.readString(blockTagLength);
    const finalityLength = reader.readUint32();
    const finality = reader.readString(finalityLength);
    if (finality != "finalized" && finality != "safe") {
      throw new Error(`Unsupported finality: ${finality}`);
    }
    const callDataLength = reader.readUint8();
    const callData: EthCallData[] = [];
    for (let idx = 0; idx < callDataLength; idx++) {
      const to = reader.readHex(20);
      const dataLength = reader.readUint32();
      const data = reader.readHex(dataLength);
      callData.push({ to, data });
    }
    return new EthCallWithFinalityQueryRequest(blockTag, finality, callData);
  }
}

export class EthCallWithFinalityQueryResponse extends EthCallQueryResponse {
  type(): ChainQueryType {
    return ChainQueryType.EthCallWithFinality;
  }

  static from(bytes: string | Uint8Array): EthCallWithFinalityQueryResponse {
    const reader = new BinaryReader(coalesceUint8Array(bytes));
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): EthCallWithFinalityQueryResponse {
    const queryResponse = EthCallQueryResponse.fromReader(reader);
    return new EthCallWithFinalityQueryResponse(
      queryResponse.blockNumber,
      queryResponse.blockHash,
      queryResponse.blockTime,
      queryResponse.results
    );
  }
}
