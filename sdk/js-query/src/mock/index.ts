import axios from "axios";
import base58 from "bs58";
import { Buffer } from "buffer";
import {
  ChainQueryType,
  EthCallByTimestampQueryRequest,
  EthCallByTimestampQueryResponse,
  EthCallQueryRequest,
  EthCallQueryResponse,
  EthCallWithFinalityQueryRequest,
  EthCallWithFinalityQueryResponse,
  PerChainQueryResponse,
  QueryProxyQueryResponse,
  QueryRequest,
  QueryResponse,
  sign,
  SolanaAccountQueryRequest,
  SolanaAccountQueryResponse,
  SolanaAccountResult,
  SolanaPdaQueryRequest,
  SolanaPdaQueryResponse,
  SolanaPdaResult,
} from "../query";
import { BinaryWriter } from "../query/BinaryWriter";

import { PublicKey } from "@solana/web3.js";

// (2**64)-1
const SOLANA_MAX_RENT_EPOCH = BigInt("18446744073709551615");

interface SolanaGetMultipleAccountsOpts {
  commitment: string;
  minContextSlot?: number;
  dataSlice?: SolanaDataSlice;
}

interface SolanaDataSlice {
  offset: number;
  length: number;
}

type SolanaAccountData = {
  data: [string, string];
  executable: boolean;
  lamports: number;
  owner: string;
  rentEpoch: number;
  space: number;
};

type SolanaGetMultipleAccountsResponse = {
  result?: {
    context: { apiVersion: string; slot: number };
    value: SolanaAccountData[];
  };
};

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
  sign(serializedResponse: Uint8Array) {
    const digest = QueryResponse.digest(serializedResponse);
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
    const queryResponse = new QueryResponse(
      0, // source = off-chain
      Buffer.from(new Array(65)).toString("hex"), // empty signature for mock
      queryRequest
    );
    for (const perChainRequest of queryRequest.requests) {
      const rpc = this.rpcMap[perChainRequest.chainId];
      if (!rpc) {
        throw new Error(
          `Unregistered chain id for mock: ${perChainRequest.chainId}`
        );
      }
      const type = perChainRequest.query.type();
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
        queryResponse.responses.push(
          new PerChainQueryResponse(
            perChainRequest.chainId,
            new EthCallQueryResponse(
              BigInt(parseInt(blockResult.number.substring(2), 16)), // block number
              blockResult.hash, // hash
              BigInt(parseInt(blockResult.timestamp.substring(2), 16)) *
                BigInt("1000000"), // time in seconds -> microseconds
              callResults.map((callResult: any) => callResult.result)
            )
          )
        );
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
        queryResponse.responses.push(
          new PerChainQueryResponse(
            perChainRequest.chainId,
            new EthCallWithFinalityQueryResponse(
              BigInt(parseInt(blockResult.number.substring(2), 16)), // block number
              blockResult.hash, // hash
              BigInt(parseInt(blockResult.timestamp.substring(2), 16)) *
                BigInt("1000000"), // time in seconds -> microseconds
              callResults.map((callResult: any) => callResult.result)
            )
          )
        );
      } else if (type === ChainQueryType.EthCallByTimeStamp) {
        const query = perChainRequest.query as EthCallByTimestampQueryRequest;
        // Verify that the two block hints are consistent, either both set, or both unset.
        if (
          (query.targetBlockHint === "") !==
          (query.followingBlockHint === "")
        ) {
          throw new Error(
            `Invalid block id hints in eth_call_by_timestamp query request, both should be either set or unset`
          );
        }
        let targetBlock = query.targetBlockHint;
        let followingBlock = query.followingBlockHint;
        if (targetBlock === "") {
          let nextQueryBlock = "latest";
          let tries = 0;
          let targetTimestamp = BigInt(0);
          let followingTimestamp = BigInt(0);
          while (
            query.targetTimestamp < targetTimestamp ||
            query.targetTimestamp >= followingTimestamp
          ) {
            if (tries > 128) {
              throw new Error(`Timestamp was not within the last 128 blocks.`);
            }
            // TODO: batching
            const blockResult = (
              await axios.post(rpc, {
                jsonrpc: "2.0",
                id: 1,
                method: "eth_getBlockByNumber",
                params: [nextQueryBlock, false],
              })
            ).data?.result;
            if (!blockResult) {
              throw new Error(
                `Invalid block result while searching for timestamp of ${nextQueryBlock}`
              );
            }
            followingBlock = targetBlock;
            followingTimestamp = targetTimestamp;
            targetBlock = blockResult.number;
            targetTimestamp =
              BigInt(parseInt(blockResult.timestamp.substring(2), 16)) *
              BigInt("1000000"); // time in seconds -> microseconds
            nextQueryBlock = `0x${(
              BigInt(blockResult.number) - BigInt(1)
            ).toString(16)}`;
            tries++;
          }
        }
        const response = await axios.post(rpc, [
          ...query.callData.map((args, idx) => ({
            jsonrpc: "2.0",
            id: idx,
            method: "eth_call",
            params: [
              args,
              //TODO: support block hash
              targetBlock,
            ],
          })),
          {
            jsonrpc: "2.0",
            id: query.callData.length,
            method: "eth_getBlockByNumber",
            params: [targetBlock, false],
          },
          {
            jsonrpc: "2.0",
            id: query.callData.length,
            method: "eth_getBlockByNumber",
            params: [followingBlock, false],
          },
        ]);
        const callResults = response?.data?.slice(0, query.callData.length);
        const targetBlockResult =
          response?.data?.[query.callData.length]?.result;
        const followingBlockResult =
          response?.data?.[query.callData.length + 1]?.result;
        if (
          !targetBlockResult ||
          !targetBlockResult.number ||
          !targetBlockResult.timestamp ||
          !targetBlockResult.hash
        ) {
          throw new Error(
            `Invalid target block result for chain ${perChainRequest.chainId} block ${query.targetBlockHint}`
          );
        }
        if (
          !followingBlockResult ||
          !followingBlockResult.number ||
          !followingBlockResult.timestamp ||
          !followingBlockResult.hash
        ) {
          throw new Error(
            `Invalid following block result for chain ${perChainRequest.chainId} tag ${query.followingBlockHint}`
          );
        }
        /*
          target_block.timestamp <= target_time < following_block.timestamp
          and
          following_block_num - 1 == target_block_num
        */
        const targetBlockNumber = BigInt(
          parseInt(targetBlockResult.number.substring(2), 16)
        );
        const followingBlockNumber = BigInt(
          parseInt(followingBlockResult.number.substring(2), 16)
        );
        if (targetBlockNumber + BigInt(1) !== followingBlockNumber) {
          throw new Error(
            `eth_call_by_timestamp query blocks are not adjacent`
          );
        }
        const targetTimestamp =
          BigInt(parseInt(targetBlockResult.timestamp.substring(2), 16)) *
          BigInt("1000000"); // time in seconds -> microseconds
        const followingTimestamp =
          BigInt(parseInt(followingBlockResult.timestamp.substring(2), 16)) *
          BigInt("1000000"); // time in seconds -> microseconds
        if (
          query.targetTimestamp < targetTimestamp ||
          query.targetTimestamp >= followingTimestamp
        ) {
          throw new Error(
            `eth_call_by_timestamp desired timestamp falls outside of block range`
          );
        }
        queryResponse.responses.push(
          new PerChainQueryResponse(
            perChainRequest.chainId,
            new EthCallByTimestampQueryResponse(
              BigInt(parseInt(targetBlockResult.number.substring(2), 16)), // block number
              targetBlockResult.hash, // hash
              targetTimestamp,
              BigInt(parseInt(followingBlockResult.number.substring(2), 16)), // block number
              followingBlockResult.hash, // hash
              followingTimestamp,
              callResults.map((callResult: any) => callResult.result)
            )
          )
        );
      } else if (type === ChainQueryType.SolanaAccount) {
        const query = perChainRequest.query as SolanaAccountQueryRequest;
        // Validate the request.
        if (query.commitment !== "finalized") {
          throw new Error(
            `Invalid commitment in sol_account query request, must be "finalized"`
          );
        }
        if (
          query.dataSliceLength === BigInt(0) &&
          query.dataSliceOffset !== BigInt(0)
        ) {
          throw new Error(
            `data slice offset may not be set if data slice length is zero`
          );
        }
        if (query.accounts.length <= 0) {
          throw new Error(`does not contain any account entries`);
        }
        if (query.accounts.length > 255) {
          throw new Error(`too many account entries`);
        }

        let accounts: string[] = [];
        query.accounts.forEach((acct) => {
          if (acct.length != 32) {
            throw new Error(`invalid account length`);
          }
          accounts.push(base58.encode(acct));
        });

        let opts: SolanaGetMultipleAccountsOpts = {
          commitment: query.commitment,
        };
        if (query.minContextSlot != BigInt(0)) {
          opts.minContextSlot = Number(query.minContextSlot);
        }
        if (query.dataSliceLength !== BigInt(0)) {
          opts.dataSlice = {
            offset: Number(query.dataSliceOffset),
            length: Number(query.dataSliceLength),
          };
        }

        const response = await axios.post<SolanaGetMultipleAccountsResponse>(
          rpc,
          {
            jsonrpc: "2.0",
            id: 1,
            method: "getMultipleAccounts",
            params: [accounts, opts],
          }
        );

        if (!response.data.result) {
          throw new Error("Invalid result for getMultipleAccounts");
        }

        const slotNumber = response.data.result.context.slot;
        let results: SolanaAccountResult[] = [];
        response.data.result.value.forEach((val) => {
          const rentEpoch = BigInt(val.rentEpoch);
          results.push({
            lamports: BigInt(val.lamports),
            rentEpoch:
              // this is a band-aid for an axios / JSON.parse effect where numbers > Number.MAX_SAFE_INTEGER are not parsed correctly
              // e.g. 18446744073709551615 becomes 18446744073709552000
              // https://github.com/axios/axios/issues/4846
              rentEpoch > SOLANA_MAX_RENT_EPOCH
                ? SOLANA_MAX_RENT_EPOCH
                : rentEpoch,
            executable: Boolean(val.executable),
            owner: Uint8Array.from(base58.decode(val.owner.toString())),
            data: Uint8Array.from(
              Buffer.from(val.data[0].toString(), "base64")
            ),
          });
        });

        const response2 = await axios.post(rpc, {
          jsonrpc: "2.0",
          id: 1,
          method: "getBlock",
          params: [
            slotNumber,
            { commitment: query.commitment, transactionDetails: "none" },
          ],
        });

        const blockTime = response2.data.result.blockTime;
        const blockHash = base58.decode(response2.data.result.blockhash);

        queryResponse.responses.push(
          new PerChainQueryResponse(
            perChainRequest.chainId,
            new SolanaAccountQueryResponse(
              BigInt(slotNumber),
              BigInt(blockTime) * BigInt(1000000), // time in seconds -> microseconds,
              blockHash,
              results
            )
          )
        );
      } else if (type === ChainQueryType.SolanaPda) {
        const query = perChainRequest.query as SolanaPdaQueryRequest;
        // Validate the request and convert the PDAs into accounts.
        if (query.commitment !== "finalized") {
          throw new Error(
            `Invalid commitment in sol_account query request, must be "finalized"`
          );
        }
        if (
          query.dataSliceLength === BigInt(0) &&
          query.dataSliceOffset !== BigInt(0)
        ) {
          throw new Error(
            `data slice offset may not be set if data slice length is zero`
          );
        }
        if (query.pdas.length <= 0) {
          throw new Error(`does not contain any account entries`);
        }
        if (query.pdas.length > 255) {
          throw new Error(`too many account entries`);
        }

        let accounts: string[] = [];
        let bumps: number[] = [];
        query.pdas.forEach((pda) => {
          if (pda.programAddress.length != 32) {
            throw new Error(`invalid program address length`);
          }

          const [acct, bump] = PublicKey.findProgramAddressSync(
            pda.seeds,
            new PublicKey(pda.programAddress)
          );
          accounts.push(acct.toString());
          bumps.push(bump);
        });

        let opts: SolanaGetMultipleAccountsOpts = {
          commitment: query.commitment,
        };
        if (query.minContextSlot != BigInt(0)) {
          opts.minContextSlot = Number(query.minContextSlot);
        }
        if (query.dataSliceLength !== BigInt(0)) {
          opts.dataSlice = {
            offset: Number(query.dataSliceOffset),
            length: Number(query.dataSliceLength),
          };
        }

        const response = await axios.post<SolanaGetMultipleAccountsResponse>(
          rpc,
          {
            jsonrpc: "2.0",
            id: 1,
            method: "getMultipleAccounts",
            params: [accounts, opts],
          }
        );

        if (!response.data.result) {
          throw new Error("Invalid result for getMultipleAccounts");
        }

        const slotNumber = response.data.result.context.slot;
        let results: SolanaPdaResult[] = [];
        let idx = 0;
        response.data.result.value.forEach((val) => {
          const rentEpoch = BigInt(val.rentEpoch);
          results.push({
            account: Uint8Array.from(base58.decode(accounts[idx].toString())),
            bump: bumps[idx],
            lamports: BigInt(val.lamports),
            rentEpoch:
              // this is a band-aid for an axios / JSON.parse effect where numbers > Number.MAX_SAFE_INTEGER are not parsed correctly
              // e.g. 18446744073709551615 becomes 18446744073709552000
              // https://github.com/axios/axios/issues/4846
              rentEpoch > SOLANA_MAX_RENT_EPOCH
                ? SOLANA_MAX_RENT_EPOCH
                : rentEpoch,
            executable: Boolean(val.executable),
            owner: Uint8Array.from(base58.decode(val.owner.toString())),
            data: Uint8Array.from(
              Buffer.from(val.data[0].toString(), "base64")
            ),
          });
          idx += 1;
        });

        const response2 = await axios.post(rpc, {
          jsonrpc: "2.0",
          id: 1,
          method: "getBlock",
          params: [
            slotNumber,
            { commitment: query.commitment, transactionDetails: "none" },
          ],
        });

        const blockTime = response2.data.result.blockTime;
        const blockHash = base58.decode(response2.data.result.blockhash);

        queryResponse.responses.push(
          new PerChainQueryResponse(
            perChainRequest.chainId,
            new SolanaPdaQueryResponse(
              BigInt(slotNumber),
              BigInt(blockTime) * BigInt(1000000), // time in seconds -> microseconds,
              blockHash,
              results
            )
          )
        );
      } else {
        throw new Error(`Unsupported query type for mock: ${type}`);
      }
    }
    const serializedResponse = queryResponse.serialize();
    return {
      signatures: this.sign(serializedResponse),
      bytes: Buffer.from(serializedResponse).toString("hex"),
    };
  }
}
