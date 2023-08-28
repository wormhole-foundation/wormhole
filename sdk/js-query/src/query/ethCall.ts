import { BinaryWriter } from "./BinaryWriter";
import { ChainQueryType, ChainSpecificQuery } from "./request";
import { ChainSpecificResponse } from "./response";
import { hexToUint8Array } from "./utils";

export interface EthCallData {
  to: string;
  data: string;
}

export class EthCallQueryRequest implements ChainSpecificQuery {
  constructor(public blockId: string, public callData: EthCallData[]) {}

  type(): ChainQueryType {
    return ChainQueryType.EthCall;
  }

  serialize(): Uint8Array {
    const writer = new BinaryWriter()
      .writeUint32(this.blockId.length)
      .writeUint8Array(Buffer.from(this.blockId))
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

export class EthCallQueryResponse implements ChainSpecificResponse {
  constructor(
    public blockNumber: number,
    public hash: string,
    public time: string,
    public results: string[][]
  ) {}

  type(): ChainQueryType {
    return ChainQueryType.EthCall;
  }

  // static fromBytes(bytes: Uint8Array): EthCallQueryResponse {}
}
