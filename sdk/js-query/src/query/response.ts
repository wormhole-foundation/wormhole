import { ChainQueryType } from "./request";

// TODO: implement query response parsing

export class QueryResponse {
  signatures: string[] = [];
  bytes: string = "";

  //   constructor(signatures: string[], bytes: string) {

  //   }

  static fromBytes(bytes: Uint8Array): Uint8Array {
    //const reader: Reader = {
    //  buffer: Buffer.from(bytes),
    //  i: 0,
    //};
    //// Request
    //const requestChain = reader.buffer.readUint16BE(reader.i);
    //reader.i += 2;
    //const signature = buffer.toString("hex", offset, offset + 65);
    //offset += 65;
    //const request = null;
    //// Response
    //const numPerChainResponses = buffer.readUint8(offset);
    return new Uint8Array();
  }
}

export class PerChainQueryResponse {
  constructor(public chainId: number, public response: ChainSpecificResponse) {}
}

export interface ChainSpecificResponse {
  type(): ChainQueryType;
}
