import { keccak256 } from "@ethersproject/keccak256";
import { Buffer } from "buffer";
import { BinaryWriter } from "./BinaryWriter";
import { Network } from "./consts";
import { coalesceUint8Array, hexToUint8Array, uint8ArrayToHex } from "./utils";
import { BinaryReader } from "./BinaryReader";
import { EthCallQueryRequest } from "./ethCall";
import { EthCallByTimestampQueryRequest } from "./ethCallByTimestamp";
import { EthCallWithFinalityQueryRequest } from "./ethCallWithFinality";
import { SolanaAccountQueryRequest } from "./solanaAccount";
import { SolanaPdaQueryRequest } from "./solanaPda";

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

const MSG_VERSION = 2;

export class QueryRequest {
  constructor(
    public nonce: number,
    public timestamp: number, // Unix timestamp in seconds
    public requests: PerChainQueryRequest[] = [],
    public stakerAddress?: string // Optional 20-byte Ethereum address for delegation
  ) { }

  serialize(): Uint8Array {
    const writer = new BinaryWriter()
      .writeUint8(MSG_VERSION)
      .writeUint32(this.nonce)
      .writeUint64(BigInt(this.timestamp));

    // Write optional staker address with length prefix
    if (this.stakerAddress) {
      const stakerBytes = hexToUint8Array(
        this.stakerAddress.replace(/^0x/, "")
      );
      if (stakerBytes.length !== 20) {
        throw new Error(
          `Invalid staker address length: expected 20 bytes, got ${stakerBytes.length}`
        );
      }
      writer.writeUint8(20);
      writer.writeUint8Array(stakerBytes);
    } else {
      writer.writeUint8(0);
    }

    writer.writeUint8(this.requests.length);
    this.requests.forEach((request) =>
      writer.writeUint8Array(request.serialize())
    );
    return writer.data();
  }

  static digest(network: Network, bytes: Uint8Array): Uint8Array {
    const prefix = getPrefix(network);
    const data = Buffer.concat([Buffer.from(prefix), Buffer.from(bytes)]);
    return hexToUint8Array(keccak256(data).slice(2));
  }

  static from(bytes: string | Uint8Array): QueryRequest {
    const reader = new BinaryReader(coalesceUint8Array(bytes).buffer);
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): QueryRequest {
    const version = reader.readUint8();
    if (version !== MSG_VERSION) {
      throw new Error(
        `Unsupported message version: ${version} (only v2 supported)`
      );
    }
    const nonce = reader.readUint32();
    const timestamp = Number(reader.readUint64());

    // Read optional staker address with length prefix
    let stakerAddress: string | undefined = undefined;
    const stakerLen = reader.readUint8();
    if (stakerLen > 0) {
      if (stakerLen !== 20) {
        throw new Error(
          `Invalid staker address length: expected 20, got ${stakerLen}`
        );
      }
      const stakerBytes = reader.readUint8Array(stakerLen);
      stakerAddress = uint8ArrayToHex(stakerBytes);
    }

    const queryRequest = new QueryRequest(nonce, timestamp, [], stakerAddress);
    const numPerChainQueries = reader.readUint8();
    for (let idx = 0; idx < numPerChainQueries; idx++) {
      queryRequest.requests.push(PerChainQueryRequest.fromReader(reader));
    }
    return queryRequest;
  }
}

export class PerChainQueryRequest {
  constructor(public chainId: number, public query: ChainSpecificQuery) { }

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

  static from(bytes: string | Uint8Array): PerChainQueryRequest {
    const reader = new BinaryReader(coalesceUint8Array(bytes).buffer);
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): PerChainQueryRequest {
    const chainId = reader.readUint16();
    const queryType = reader.readUint8();
    reader.readUint32(); // skip the query length
    let query: ChainSpecificQuery;
    if (queryType === ChainQueryType.EthCall) {
      query = EthCallQueryRequest.fromReader(reader);
    } else if (queryType === ChainQueryType.EthCallByTimeStamp) {
      query = EthCallByTimestampQueryRequest.fromReader(reader);
    } else if (queryType === ChainQueryType.EthCallWithFinality) {
      query = EthCallWithFinalityQueryRequest.fromReader(reader);
    } else if (queryType === ChainQueryType.SolanaAccount) {
      query = SolanaAccountQueryRequest.fromReader(reader);
    } else if (queryType === ChainQueryType.SolanaPda) {
      query = SolanaPdaQueryRequest.fromReader(reader);
    } else {
      throw new Error(`Unsupported query type: ${queryType}`);
    }
    return new PerChainQueryRequest(chainId, query);
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
  SolanaAccount = 4,
  SolanaPda = 5,
}
