// This file is intended to test an already deployed QueryPullAllDemo on Goerli

import { keccak256 } from "@ethersproject/keccak256";
import { JsonRpcProvider } from "@ethersproject/providers";
import {
  EthCallWithFinalityQueryRequest,
  PerChainQueryRequest,
  QueryProxyQueryResponse,
  QueryRequest,
} from "@wormhole-foundation/wormhole-query-sdk";
import axios from "axios";
import { QueryPullAllDemo } from "./QueryPullAllDemo";
import { QueryPullAllDemo__factory } from "./factories/QueryPullAllDemo__factory";

const RPC_URL = "https://ethereum-goerli.publicnode.com";
const QUERY_URL = "https://testnet.ccq.vaa.dev/v1/query";
const FINALITY = "finalized";
const ADDRESS = "0x44d0Da4e120FFCe28DD49b2fb3D835Fa12Badcd2";
const API_KEY = process.env.API_KEY;
const MESSAGES: QueryPullAllDemo.MessageStruct[] = [
  // {
  //   payloadID: 1,
  //   destinationChainID: 2,
  //   message: "This was a triumph.",
  // },
  // {
  //   payloadID: 1,
  //   destinationChainID: 2,
  //   message: "I'm making a note here:",
  // },
  // {
  //   payloadID: 1,
  //   destinationChainID: 2,
  //   message: "HUGE SUCCESS.",
  // },
  // {
  //   payloadID: 1,
  //   destinationChainID: 2,
  //   message: "It's hard to overstate",
  // },
  {
    payloadID: 1,
    destinationChainID: 2,
    message: "my satisfaction.",
  },
  {
    payloadID: 1,
    destinationChainID: 2,
    message: "Aperture Science",
  },
  {
    payloadID: 1,
    destinationChainID: 2,
    message: "We do what we must",
  },
  {
    payloadID: 1,
    destinationChainID: 2,
    message: "because we can.",
  },
  {
    payloadID: 1,
    destinationChainID: 2,
    message: "For the good of all of us.",
  },
  {
    payloadID: 1,
    destinationChainID: 2,
    message: "Except the ones who are dead.",
  },
  {
    payloadID: 1,
    destinationChainID: 2,
    message: "But there's no sense crying",
  },
  {
    payloadID: 1,
    destinationChainID: 2,
    message: "over every mistake.",
  },
  {
    payloadID: 1,
    destinationChainID: 2,
    message: "You just keep on trying",
  },
  {
    payloadID: 1,
    destinationChainID: 2,
    message: "till you run out of cake.",
  },
  // {
  //   payloadID: 1,
  //   destinationChainID: 2,
  //   message: "And the Science gets done.",
  // },
];

(async () => {
  const provider = new JsonRpcProvider(RPC_URL);
  const demo = QueryPullAllDemo__factory.connect(ADDRESS, provider);
  const encodedMessages: string[] = [];
  for (const message of MESSAGES) {
    const encodedMessage = await demo.encodeMessage(message);
    encodedMessages.push(encodedMessage);
  }
  console.log(encodedMessages);
  const block = (
    await axios.post(RPC_URL, {
      jsonrpc: "2.0",
      id: 1,
      method: "eth_getBlockByNumber",
      params: [FINALITY, false],
    })
  ).data.result.number;
  console.log(FINALITY, block);
  try {
    const queryResult = await axios.post<QueryProxyQueryResponse>(
      QUERY_URL,
      {
        bytes: Buffer.from(
          new QueryRequest(0, [
            new PerChainQueryRequest(
              2,
              new EthCallWithFinalityQueryRequest(block, FINALITY, [
                {
                  to: ADDRESS,
                  data: QueryPullAllDemo__factory.createInterface().encodeFunctionData(
                    "latestSentMessage",
                    [2]
                  ),
                },
              ])
            ),
          ]).serialize()
        ).toString("hex"),
      },
      { headers: { "X-API-Key": API_KEY } }
    );
    console.log("For etherscan receivePullMessages:");
    console.log("response:");
    console.log(`0x${queryResult.data.bytes}`);
    console.log("signatures:");
    console.log(
      `[${queryResult.data.signatures
        .map(
          (s) =>
            `["0x${s.substring(0, 64)}","0x${s.substring(64, 128)}","0x${(
              parseInt(s.substring(128, 130), 16) + 27
            ).toString(16)}","0x${s.substring(130, 132)}"]`
        )
        .join(",")}]`
    );
    console.log("messages:");
    console.log(`[${encodedMessages.join(",")}]`);
  } catch (e) {
    console.error(e?.toJSON ? e.toJSON() : e?.message);
  }
})();
