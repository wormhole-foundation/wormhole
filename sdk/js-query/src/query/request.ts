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

const REQUEST_VERSION = 1;
const REQUEST_VERSION_V2 = 2;

export class QueryRequest {
  constructor(
    public nonce: number,
    public requests: PerChainQueryRequest[] = [],
    public version: number = REQUEST_VERSION,
    public stakerAddress?: string // Optional 20-byte Ethereum address for delegation (v2 only)
  ) {}

  serialize(): Uint8Array {
    // Use v2 format if stakerAddress is present, otherwise use v1 for backward compatibility
    const useV2 = this.stakerAddress !== undefined && this.stakerAddress.length > 0;
    const version = useV2 ? REQUEST_VERSION_V2 : REQUEST_VERSION;

    const writer = new BinaryWriter()
      .writeUint8(version)
      .writeUint32(this.nonce);

    // Write staker address only in v2 messages (delegation support)
    if (useV2 && this.stakerAddress) {
      const stakerBytes = hexToUint8Array(this.stakerAddress);
      if (stakerBytes.length !== 20) {
        throw new Error(`Staker address must be 20 bytes, got ${stakerBytes.length}`);
      }
      writer.writeUint8(stakerBytes.length);
      writer.writeUint8Array(stakerBytes);
    }

    writer.writeUint8(this.requests.length);
    this.requests.forEach((request) =>
      writer.writeUint8Array(request.serialize())
    );
    return writer.data();
  }

  static digest(network: Network, bytes: Uint8Array): Uint8Array {
    const prefix = getPrefix(network);
    const data = Buffer.concat([Buffer.from(prefix), Buffer.from(bytes)] as any);
    return hexToUint8Array(keccak256(data).slice(2));
  }

  static from(bytes: string | Uint8Array): QueryRequest {
    const reader = new BinaryReader(Buffer.from(coalesceUint8Array(bytes)) as any);
    return this.fromReader(reader);
  }

  static fromReader(reader: BinaryReader): QueryRequest {
    const version = reader.readUint8();
    if (version !== REQUEST_VERSION && version !== REQUEST_VERSION_V2) {
      throw new Error(`Unsupported message version: ${version}`);
    }
    const nonce = reader.readUint32();

    // Read optional staker address only in v2 messages (delegation support)
    let stakerAddress: string | undefined = undefined;
    if (version === REQUEST_VERSION_V2) {
      const stakerLen = reader.readUint8();
      if (stakerLen > 0) {
        if (stakerLen !== 20) {
          throw new Error(`Invalid staker address length: expected 20, got ${stakerLen}`);
        }
        const stakerBytes = reader.readUint8Array(stakerLen);
        stakerAddress = uint8ArrayToHex(stakerBytes);
      }
    }

    const queryRequest = new QueryRequest(nonce, [], version, stakerAddress);
    const numPerChainQueries = reader.readUint8();
    for (let idx = 0; idx < numPerChainQueries; idx++) {
      queryRequest.requests.push(PerChainQueryRequest.fromReader(reader));
    }
    return queryRequest;
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

  static from(bytes: string | Uint8Array): PerChainQueryRequest {
    const reader = new BinaryReader(Buffer.from(coalesceUint8Array(bytes)) as any);
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
