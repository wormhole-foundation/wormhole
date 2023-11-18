// This file is intended to test an already deployed QueryPushPullDemo
// You can set this up locally with the following commands
//   anvil
//   cd ../deploy-core
//   npm ci
//   npm start
//   cd ../../ethereum
//   forge create \
//     --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
//     contracts/query/QueryPullAllDemo.sol:QueryPullAllDemo \
//     --constructor-args 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0 2
// where 0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0 is the core (Wormhole) address from the previous step

import { keccak256 } from "@ethersproject/keccak256";
import { JsonRpcProvider } from "@ethersproject/providers";
import {
  EthCallWithFinalityQueryRequest,
  PerChainQueryRequest,
  QueryProxyMock,
  QueryRequest,
  QueryResponse,
} from "@wormhole-foundation/wormhole-query-sdk";
import { Wallet } from "ethers";
import { QueryPullAllDemo__factory } from "./factories/QueryPullAllDemo__factory";
import { QueryPullAllDemo } from "./QueryPullAllDemo";

const rpc = process.env.RPC || "http://127.0.0.1:8545";
const pk = process.env.PRIVATE_KEY || "";
const mnemonic =
  process.env.MNEMONIC ||
  "test test test test test test test test test test test junk";
const address =
  process.env.ADDRESS || "0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9";

const provider = new JsonRpcProvider(rpc);
const mock = new QueryProxyMock({
  2: rpc,
});

async function sendPullMessage(contract: QueryPullAllDemo, message: string) {
  const pullTx = await contract.sendPullMessage(2, message);
  const pullTxResult = await pullTx.wait();
  // console.log(
  //   "Sent pull message:",
  //   pullTx.hash,
  //   "block:",
  //   pullTxResult.blockHash,
  //   "gasUsed:",
  //   pullTxResult.gasUsed.toString()
  // );
  const pullLogs = pullTxResult.logs;
  const log = contract.interface.decodeEventLog(
    contract.interface.events[
      "pullMessagePublished(bytes32,bytes32,uint16,uint8,uint16,string)"
    ],
    pullLogs[0].data
  );
  const {
    previousHash,
    latestHash,
    sourceChainID,
    payloadID,
    destinationChainID,
    message: sentMessage,
  } = log;
  // console.log(
  //   `Sent message: previousHash: ${previousHash}, latestHash: ${latestHash}, sourceChainID: ${sourceChainID}, payloadID: ${payloadID}, destination: ${destinationChainID}, message: ${sentMessage}`
  // );
  // this could be computed fully off-chain as well
  const encodedMessage = await contract.encodeMessage({
    payloadID,
    destinationChainID,
    message,
  });
  // console.log("digest:", messageDigest);
  return {
    message: encodedMessage,
    block: pullTxResult.blockNumber,
  };
}

async function sendAndReceivePullMessages(
  sendContract: QueryPullAllDemo,
  receiveContract: QueryPullAllDemo,
  n: number
) {
  const sent: { message: string; block: number }[] = [];
  for (let i = 0; i < n; i++) {
    sent.push(await sendPullMessage(sendContract, `test ${i + 1}`));
  }
  await provider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`
  const queryResultMulti = await mock.mock(
    new QueryRequest(0, [
      new PerChainQueryRequest(
        2,
        // TODO: support block hash
        new EthCallWithFinalityQueryRequest(
          sent[sent.length - 1].block,
          "finalized",
          [
            {
              to: address,
              data: sendContract.interface.encodeFunctionData(
                "latestSentMessage",
                [2]
              ),
            },
          ]
        )
      ),
    ])
  );
  // console.log(queryResultMulti);
  // const parsedQueryResponseMulti = QueryResponse.from(queryResultMulti.bytes);
  // console.log(parsedQueryResponseMulti.request.requests[0].query);
  // console.log(parsedQueryResponseMulti.responses[0].response);
  const receivePullTxMulti = await receiveContract.receivePullMessages(
    `0x${queryResultMulti.bytes}`,
    queryResultMulti.signatures.map((s) => ({
      r: `0x${s.substring(0, 64)}`,
      s: `0x${s.substring(64, 128)}`,
      v: `0x${(parseInt(s.substring(128, 130), 16) + 27).toString(16)}`,
      guardianIndex: `0x${s.substring(130, 132)}`,
    })),
    sent.map(({ message }) => message)
  );
  const receivePullResultMulti = await receivePullTxMulti.wait();
  console.log(
    `Received ${n} message${n > 1 ? "s" : ""}: tx: ${
      receivePullResultMulti.transactionHash
    }, gasUsed: ${receivePullResultMulti.gasUsed.toString()}`
  );
}

(async () => {
  const network = await provider.getNetwork();
  console.log("Connected to:", network);
  const signer = (pk ? new Wallet(pk) : Wallet.fromMnemonic(mnemonic)).connect(
    provider
  );
  console.log("Using wallet:", await signer.getAddress());
  const demo = QueryPullAllDemo__factory.connect(address, signer);
  const emitter = "0x000000000000000000000000" + address.substring(2);
  try {
    const registerTx = await demo.updateRegistration(2, emitter);
    console.log("Set registration:", registerTx.hash);
  } catch (e) {}
  await sendAndReceivePullMessages(demo, demo, 1);
  await sendAndReceivePullMessages(demo, demo, 1);
  await sendAndReceivePullMessages(demo, demo, 2);
  await sendAndReceivePullMessages(demo, demo, 10);
  await sendAndReceivePullMessages(demo, demo, 100);
  await sendAndReceivePullMessages(demo, demo, 1000);
})();
