import { keccak256 } from "@ethersproject/keccak256";
import { Buffer } from "buffer";
import { BinaryWriter } from "./BinaryWriter";
import { Network } from "./consts";
import { hexToUint8Array } from "./utils";

export const MAINNET_QUERY_REQUEST_PREFIX =
  "mainnet_query_request_000000000000|";

export const TESTNET_QUERY_REQUEST_PREFIX =
  "testnet_query_request_000000000000|";

export const DEVNET_QUERY_REQUEST_PREFIX =
  "devnet_query_request_0000000000000|";

export function getPrefix(network: Network) {
  return network === "MAINNET"
    ? MAINNET_QUERY_REQUEST_PREFIX
    : network === "TESTNET"
    ? TESTNET_QUERY_REQUEST_PREFIX
    : DEVNET_QUERY_REQUEST_PREFIX;
}

export class QueryRequest {
  constructor(
    public nonce: number,
    public requests: PerChainQueryRequest[] = [],
    public version: number = 1
  ) {}

  serialize(): Uint8Array {
    const writer = new BinaryWriter()
      .writeUint8(this.version)
      .writeUint32(this.nonce)
      .writeUint8(this.requests.length);
    this.requests.forEach((request) =>
      writer.writeUint8Array(request.serialize())
    );
    return writer.data();
  }

  static digest(network: Network, bytes: Uint8Array): Uint8Array {
    const prefix = getPrefix(network);
    const data = Buffer.concat([Buffer.from(prefix), bytes]);
    return hexToUint8Array(keccak256(data).slice(2));
  }
}

export class PerChainQueryRequest {
  constructor(public chainId: number, public query: ChainSpecificQuery) {}

  serialize(): Uint8Array {
    const writer = new BinaryWriter()
      .writeUint16(this.chainId)
      .writeUint8(this.query.type());
    const queryData = this.query.serialize();
    return writer
      .writeUint32(queryData.length)
      .writeUint8Array(queryData)
      .data();
  }
}

export interface ChainSpecificQuery {
  type(): ChainQueryType;
  serialize(): Uint8Array;
}

export enum ChainQueryType {
  EthCall = 1,
  EthCallByTimeStamp = 2,
  EthCallWithFinality = 3,
}
