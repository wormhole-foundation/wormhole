import axios from "axios";
import { Buffer } from "buffer";
import {
  ChainQueryType,
  EthCallQueryRequest,
  EthCallWithFinalityQueryRequest,
  QueryRequest,
  hexToUint8Array,
  sign,
} from "../query";
import { BinaryWriter } from "../query/BinaryWriter";
import { BytesLike } from "@ethersproject/bytes";
import { keccak256 } from "@ethersproject/keccak256";

export type QueryProxyQueryResponse = {
  signatures: string[];
  bytes: string;
};

const QUERY_RESPONSE_PREFIX = "query_response_0000000000000000000|";

/**
 * Usage:
 *
 * ```js
 * const mock = new QueryProxyMock({
 *   2: "http://localhost:8545",
 * });
 * ```
 *
 * If you are running an Anvil fork like
 *
 * ```bash
 * anvil -f https://ethereum-goerli.publicnode.com
 * ```
 *
 * You can use the following command to switch the guardian address to the devnet / mock guardian
 *
 * Where the `-a` parameter is the core bridge address on that chain
 *
 * https://docs.wormhole.com/wormhole/reference/constants#core-contracts
 *
 * ```bash
 * npx \@wormhole-foundation/wormhole-cli evm hijack -a 0x706abc4E45D419950511e474C7B9Ed348A4a716c -g 0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe
 * ```
 */
export class QueryProxyMock {
  constructor(
    public rpcMap: { [chainId: number]: string },
    public mockPrivateKeys = [
      "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0",
    ]
  ) {}
  sign(serializedResponse: BytesLike) {
    const digest = hexToUint8Array(
      keccak256(
        Buffer.concat([
          Buffer.from(QUERY_RESPONSE_PREFIX),
          hexToUint8Array(keccak256(serializedResponse)),
        ])
      )
    );
    return this.mockPrivateKeys.map(
      (key, idx) => `${sign(key, digest)}${idx.toString(16).padStart(2, "0")}`
    );
  }
  /**
   * Usage:
   *
   * ```js
   * const { bytes, signatures } = await mock.mock(new QueryRequest(nonce, [
   *   new PerChainQueryRequest(
   *     wormholeChainId,
   *     new EthCallQueryRequest(blockNumber, [
   *       { to: contract, data: abiEncodedData },
   *     ])
   *   ),
   * ]));
   * ```
   *
   * @param queryRequest an instance of the `QueryRequest` class
   * @returns a promise result matching the query proxy's query response
   */
  async mock(queryRequest: QueryRequest): Promise<QueryProxyQueryResponse> {
    const serializedRequest = queryRequest.serialize();
    const writer = new BinaryWriter()
      .writeUint8(1) // version
      .writeUint16(0) // source = off-chain
      .writeUint8Array(new Uint8Array(new Array(65))) // empty signature for mock
      .writeUint32(serializedRequest.length)
      .writeUint8Array(serializedRequest)
      .writeUint8(queryRequest.requests.length);
    for (const perChainRequest of queryRequest.requests) {
      const rpc = this.rpcMap[perChainRequest.chainId];
      if (!rpc) {
        throw new Error(
          `Unregistered chain id for mock: ${perChainRequest.chainId}`
        );
      }
      const type = perChainRequest.query.type();
      writer.writeUint16(perChainRequest.chainId).writeUint8(type);
      if (type === ChainQueryType.EthCall) {
        const query = perChainRequest.query as EthCallQueryRequest;
        const response = await axios.post(rpc, [
          ...query.callData.map((args, idx) => ({
            jsonrpc: "2.0",
            id: idx,
            method: "eth_call",
            params: [
              args,
              //TODO: support block hash
              query.blockTag,
            ],
          })),
          {
            jsonrpc: "2.0",
            id: query.callData.length,
            //TODO: support block hash
            method: "eth_getBlockByNumber",
            params: [query.blockTag, false],
          },
        ]);
        const callResults = response?.data?.slice(0, query.callData.length);
        const blockResult = response?.data?.[query.callData.length]?.result;
        if (
          !blockResult ||
          !blockResult.number ||
          !blockResult.timestamp ||
          !blockResult.hash
        ) {
          throw new Error(
            `Invalid block result for chain ${perChainRequest.chainId} block ${query.blockTag}`
          );
        }
        const results = callResults.map(
          (callResult: any) =>
            new Uint8Array(Buffer.from(callResult.result.substring(2), "hex"))
        );
        const perChainWriter = new BinaryWriter()
          .writeUint64(BigInt(parseInt(blockResult.number.substring(2), 16))) // block number
          .writeUint8Array(
            new Uint8Array(Buffer.from(blockResult.hash.substring(2), "hex"))
          ) // hash
          .writeUint64(
            BigInt(parseInt(blockResult.timestamp.substring(2), 16)) *
              BigInt("1000000")
          ) // time in seconds -> microseconds
          .writeUint8(results.length);
        for (const result of results) {
          perChainWriter.writeUint32(result.length).writeUint8Array(result);
        }
        const serialized = perChainWriter.data();
        writer.writeUint32(serialized.length).writeUint8Array(serialized);
      } else if (type === ChainQueryType.EthCallWithFinality) {
        const query = perChainRequest.query as EthCallWithFinalityQueryRequest;
        const response = await axios.post(rpc, [
          ...query.callData.map((args, idx) => ({
            jsonrpc: "2.0",
            id: idx,
            method: "eth_call",
            params: [
              args,
              //TODO: support block hash
              query.blockId,
            ],
          })),
          {
            jsonrpc: "2.0",
            id: query.callData.length,
            //TODO: support block hash
            method: "eth_getBlockByNumber",
            params: [query.blockId, false],
          },
          {
            jsonrpc: "2.0",
            id: query.callData.length,
            method: "eth_getBlockByNumber",
            params: [query.finality, false],
          },
        ]);
        const callResults = response?.data?.slice(0, query.callData.length);
        const blockResult = response?.data?.[query.callData.length]?.result;
        const finalityBlockResult =
          response?.data?.[query.callData.length + 1]?.result;
        if (
          !blockResult ||
          !blockResult.number ||
          !blockResult.timestamp ||
          !blockResult.hash
        ) {
          throw new Error(
            `Invalid block result for chain ${perChainRequest.chainId} block ${query.blockId}`
          );
        }
        if (
          !finalityBlockResult ||
          !finalityBlockResult.number ||
          !finalityBlockResult.timestamp ||
          !finalityBlockResult.hash
        ) {
          throw new Error(
            `Invalid tagged block result for chain ${perChainRequest.chainId} tag ${query.finality}`
          );
        }
        const blockNumber = BigInt(
          parseInt(blockResult.number.substring(2), 16)
        );
        const latestBlockNumberMatchingFinality = BigInt(
          parseInt(finalityBlockResult.number.substring(2), 16)
        );
        if (blockNumber > latestBlockNumberMatchingFinality) {
          throw new Error(
            `Requested block for eth_call_with_finality has not yet reached the requested finality. Block: ${blockNumber}, ${query.finality}: ${latestBlockNumberMatchingFinality}`
          );
        }
        const results = callResults.map(
          (callResult: any) =>
            new Uint8Array(Buffer.from(callResult.result.substring(2), "hex"))
        );
        const perChainWriter = new BinaryWriter()
          .writeUint64(blockNumber) // block number
          .writeUint8Array(
            new Uint8Array(Buffer.from(blockResult.hash.substring(2), "hex"))
          ) // hash
          .writeUint64(
            BigInt(parseInt(blockResult.timestamp.substring(2), 16)) *
              BigInt("1000000")
          ) // time in seconds -> microseconds
          .writeUint8(results.length);
        for (const result of results) {
          perChainWriter.writeUint32(result.length).writeUint8Array(result);
        }
        const serialized = perChainWriter.data();
        writer.writeUint32(serialized.length).writeUint8Array(serialized);
      } else {
        throw new Error(`Unsupported query type for mock: ${type}`);
      }
    }
    const serializedResponse = writer.data();
    return {
      signatures: this.sign(serializedResponse),
      bytes: Buffer.from(serializedResponse).toString("hex"),
    };
  }
}
