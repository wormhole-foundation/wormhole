import { keccak256 } from "@ethersproject/keccak256";
import { Buffer } from "buffer";
import { coalesceUint8Array, hexToUint8Array } from "./utils";
import { BinaryReader } from "./BinaryReader";
import { ChainQueryType, ChainSpecificQuery, QueryRequest } from "./request";
import { BinaryWriter } from "./BinaryWriter";
import { EthCallQueryResponse } from "./ethCall";
import { EthCallByTimestampQueryResponse } from "./ethCallByTimestamp";
import { EthCallWithFinalityQueryResponse } from "./ethCallWithFinality";
import { SolanaAccountQueryResponse } from "./solanaAccount";
import { SolanaPdaQueryResponse } from "./solanaPda";

export const QUERY_RESPONSE_PREFIX = "query_response_0000000000000000000|";

const RESPONSE_VERSION = 1;

export class QueryResponse {
  constructor(
    public requestChainId: number = 0,
    public requestId: string,
    public request: QueryRequest,
    public responses: PerChainQueryResponse[] = [],
    public version: number = RESPONSE_VERSION
  ) {}

  serialize(): Uint8Array {
    const serializedRequest = this.request.serialize();
    const writer = new BinaryWriter()
      .writeUint8(this.version)
      .writeUint16(this.requestChainId)
      .writeUint8Array(hexToUint8Array(this.requestId)) // TODO: this only works for hex encoded signatures
      .writeUint32(serializedRequest.length)
      .writeUint8Array(serializedRequest)
      .writeUint8(this.responses.length);
    for (const response of this.responses) {
      writer.writeUint8Array(response.serialize());
    }
    return writer.data();
  }

  static digest(bytes: Uint8Array): Uint8Array {
    return hexToUint8Array(
      keccak256(
        Buffer.concat([
          Buffer.from(QUERY_RESPONSE_PREFIX),
          hexToUint8Array(keccak256(bytes)),
        ])
      )
    );
  }

  static from(bytes: string | Uint8Array): QueryResponse {
    const reader = new BinaryReader(coalesceUint8Array(bytes));
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): QueryResponse {
    const version = reader.readUint8();
    if (version != RESPONSE_VERSION) {
      throw new Error(`Unsupported message version: ${version}`);
    }
    const requestChainId = reader.readUint16();
    if (requestChainId !== 0) {
      // TODO: support reading off-chain and on-chain requests
      throw new Error(`Unsupported request chain: ${requestChainId}`);
    }

    const requestId = reader.readHex(65); // signature
    reader.readUint32(); // skip the query length
    const queryRequest = QueryRequest.fromReader(reader);
    const queryResponse = new QueryResponse(
      requestChainId,
      requestId,
      queryRequest
    );
    const numPerChainResponses = reader.readUint8();
    for (let idx = 0; idx < numPerChainResponses; idx++) {
      queryResponse.responses.push(PerChainQueryResponse.fromReader(reader));
    }
    return queryResponse;
  }
}

export class PerChainQueryResponse {
  constructor(public chainId: number, public response: ChainSpecificResponse) {}

  serialize(): Uint8Array {
    const writer = new BinaryWriter()
      .writeUint16(this.chainId)
      .writeUint8(this.response.type());
    const queryResponse = this.response.serialize();
    return writer
      .writeUint32(queryResponse.length)
      .writeUint8Array(queryResponse)
      .data();
  }

  static from(bytes: string | Uint8Array): PerChainQueryResponse {
    const reader = new BinaryReader(coalesceUint8Array(bytes));
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): PerChainQueryResponse {
    const chainId = reader.readUint16();
    const queryType = reader.readUint8();
    reader.readUint32(); // skip the query length
    let response: ChainSpecificResponse;
    if (queryType === ChainQueryType.EthCall) {
      response = EthCallQueryResponse.fromReader(reader);
    } else if (queryType === ChainQueryType.EthCallByTimeStamp) {
      response = EthCallByTimestampQueryResponse.fromReader(reader);
    } else if (queryType === ChainQueryType.EthCallWithFinality) {
      response = EthCallWithFinalityQueryResponse.fromReader(reader);
    } else if (queryType === ChainQueryType.SolanaAccount) {
      response = SolanaAccountQueryResponse.fromReader(reader);
    } else if (queryType === ChainQueryType.SolanaPda) {
      response = SolanaPdaQueryResponse.fromReader(reader);
    } else {
      throw new Error(`Unsupported response type: ${queryType}`);
    }
    return new PerChainQueryResponse(chainId, response);
  }
}

export interface ChainSpecificResponse extends ChainSpecificQuery {}
