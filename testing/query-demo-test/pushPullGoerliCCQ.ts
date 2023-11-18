// This file is intended to test an already deployed QueryPushPullDemo on Goerli

import { keccak256 } from "@ethersproject/keccak256";
import { JsonRpcProvider } from "@ethersproject/providers";
import {
  EthCallWithFinalityQueryRequest,
  PerChainQueryRequest,
  QueryProxyQueryResponse,
  QueryRequest,
} from "@wormhole-foundation/wormhole-query-sdk";
import axios from "axios";
import { QueryPushPullDemo } from "./QueryPushPullDemo";
import { QueryPushPullDemo__factory } from "./factories/QueryPushPullDemo__factory";

const RPC_URL = "https://ethereum-goerli.publicnode.com";
const QUERY_URL = "https://testnet.ccq.vaa.dev/v1/query";
const FINALITY = "safe";
const ADDRESS = "0xfbCe310870a7D8Af3077125BcE8C125fb28d8C10";
const API_KEY = process.env.API_KEY;
const MESSAGES: QueryPushPullDemo.MessageStruct[] = [
  {
    payloadID: 1,
    sequence: 3,
    destinationChainID: 2,
    message: "HUGE SUCCESS.",
  },
  {
    payloadID: 1,
    sequence: 4,
    destinationChainID: 2,
    message: "It's hard to overstate",
  },
];

(async () => {
  const provider = new JsonRpcProvider(RPC_URL);
  const demo = QueryPushPullDemo__factory.connect(ADDRESS, provider);
  const encodedMessages: string[] = [];
  const hashes: string[] = [];
  for (const message of MESSAGES) {
    const encodedMessage = await demo.encodeMessage(message);
    encodedMessages.push(encodedMessage);
    const emitter = "0x000000000000000000000000" + ADDRESS.substring(2);
    const sendingInfo = `0x0002${emitter.substring(2)}`;
    const messageDigest = keccak256(
      `${sendingInfo}${keccak256(encodedMessage).substring(2)}`
    );
    const wasSent = await demo.hasSentMessage(messageDigest);
    if (!wasSent) {
      throw new Error(`Digest ${messageDigest} was not sent.`);
    }
    hashes.push(messageDigest);
  }
  console.log(hashes);
  const block = (
    await axios.post(RPC_URL, {
      jsonrpc: "2.0",
      id: 1,
      method: "eth_getBlockByNumber",
      params: [FINALITY, false],
    })
  ).data.result.number;
  console.log(FINALITY, block);
  console.log(
    hashes.map((hash) =>
      QueryPushPullDemo__factory.createInterface().encodeFunctionData(
        "hasSentMessage",
        [hash]
      )
    )
  );
  try {
    const queryResult = await axios.post<QueryProxyQueryResponse>(
      QUERY_URL,
      {
        bytes: Buffer.from(
          new QueryRequest(0, [
            new PerChainQueryRequest(
              2,
              new EthCallWithFinalityQueryRequest(
                block,
                FINALITY,
                hashes.map((hash) => ({
                  to: ADDRESS,
                  data: QueryPushPullDemo__factory.createInterface().encodeFunctionData(
                    "hasSentMessage",
                    [hash]
                  ),
                }))
              )
            ),
          ]).serialize()
        ).toString("hex"),
      },
      { headers: { "X-API-Key": API_KEY } }
    );
    console.log("For etherscan receivePullMessages:");
    console.log("response:", `0x${queryResult.data.bytes}`);
    console.log(
      "signatures:",
      `[${queryResult.data.signatures
        .map(
          (s) =>
            `["0x${s.substring(0, 64)}","0x${s.substring(64, 128)}","0x${(
              parseInt(s.substring(128, 130), 16) + 27
            ).toString(16)}","0x${s.substring(130, 132)}"]`
        )
        .join(",")}]`
    );
    console.log("messages:", `[${encodedMessages.join(",")}]`);
  } catch (e) {
    console.error(e?.toJSON ? e.toJSON() : e?.message);
  }
})();
