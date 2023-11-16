// This file is intended to test an already deployed QueryPushPullDemo
// You can set this up locally with the following commands
//   anvil
//   cd ../deploy-core
//   npm ci
//   npm start
//   cd ../../ethereum
//   forge create \
//     --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
//     contracts/query/QueryPushPullDemo.sol:QueryPushPullDemo \
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
import { QueryPushPullDemo__factory } from "./factories/QueryPushPullDemo__factory";

const rpc = process.env.RPC || "http://127.0.0.1:8545";
const pk = process.env.PRIVATE_KEY || "";
const mnemonic =
  process.env.MNEMONIC ||
  "test test test test test test test test test test test junk";
const address =
  process.env.ADDRESS || "0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9";

(async () => {
  const provider = new JsonRpcProvider(rpc);
  const network = await provider.getNetwork();
  console.log("Connected to:", network);
  const signer = (pk ? new Wallet(pk) : Wallet.fromMnemonic(mnemonic)).connect(
    provider
  );
  console.log("Using wallet:", await signer.getAddress());
  const demo = QueryPushPullDemo__factory.connect(address, signer);
  const emitter = "0x000000000000000000000000" + address.substring(2);
  const registerTx = await demo.updateRegistration(2, emitter);
  console.log("Updated registration:", registerTx.hash);
  const message = "this is a triumph";
  const pushTx = await demo.sendPushMessage(2, message);
  console.log("Sent push message:", pushTx.hash);
  console.log((await pushTx.wait()).logs);
  const pullTx = await demo.sendPullMessage(2, message);
  const pullTxResult = await pullTx.wait();
  console.log(
    "Sent pull message:",
    pullTx.hash,
    "block:",
    pullTxResult.blockHash
  );
  const pullLogs = pullTxResult.logs;
  const log = demo.interface.decodeEventLog(
    demo.interface.events["pullMessagePublished(uint8,uint64,uint16,string)"],
    pullLogs[0].data
  );
  const { payloadID, sequence, destinationChainID, message: sentMessage } = log;
  console.log(
    `Sent message: payloadID: ${payloadID}, sequence: ${sequence.toString()}, destination: ${destinationChainID}, message: ${sentMessage}`
  );
  // this could be computed fully off-chain as well
  const encodedMessage = await demo.encodeMessage({
    payloadID,
    sequence,
    destinationChainID,
    message,
  });
  const sendingInfo = `0x0002${emitter.substring(2)}`;
  const messageDigest = keccak256(
    `${sendingInfo}${keccak256(encodedMessage).substring(2)}`
  );
  console.log("digest:", messageDigest);
  const result = await demo.hasSentMessage(messageDigest);
  console.log(result);
  await provider.send("anvil_mine", ["0x20"]); // 32 blocks should get the above block to `safe`
  const mock = new QueryProxyMock({
    2: rpc,
  });
  const queryResult = await mock.mock(
    new QueryRequest(0, [
      new PerChainQueryRequest(
        2,
        // TODO: support block hash
        new EthCallWithFinalityQueryRequest(pullTxResult.blockNumber, "safe", [
          {
            to: address,
            // TODO: better generation here
            data: `0x8b9369e2${messageDigest.substring(2)}`,
          },
        ])
      ),
    ])
  );
  console.log(queryResult);
  const parsedQueryResponse = QueryResponse.from(queryResult.bytes);
  console.log(parsedQueryResponse.responses[0].response);
  console.log(
    "hasReceived [before]:",
    await demo.hasReceivedMessage(messageDigest)
  );
  const receivePullTx = await demo.receivePullMessages(
    `0x${queryResult.bytes}`,
    queryResult.signatures.map((s) => ({
      r: `0x${s.substring(0, 64)}`,
      s: `0x${s.substring(64, 128)}`,
      v: `0x${(parseInt(s.substring(128, 130), 16) + 27).toString(16)}`,
      guardianIndex: `0x${s.substring(130, 132)}`,
    })),
    [encodedMessage]
  );
  console.log((await receivePullTx.wait()).transactionHash);
  console.log(
    "hasReceived [after]:",
    await demo.hasReceivedMessage(messageDigest)
  );
})();
