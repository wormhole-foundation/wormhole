import * as wh from "@certusone/wormhole-sdk";
//TODO address sdk version mismatch
//import { Implementation__factory, LogMessagePublishedEvent } from "@certusone/wormhole-sdk"
import {
  ChainInfo,
  getWormholeRelayer,
  getMockIntegration,
  getMockIntegrationAddress,
} from "../helpers/env";
import { ethers } from "ethers";

export async function sendMessage(
  sourceChain: ChainInfo,
  targetChain: ChainInfo,
  fetchSignedVaa: boolean = false,
  queryMessageOnTargetFlag: boolean = true
): Promise<boolean | undefined> {
  console.log(
    `Sending message from chain ${sourceChain.chainId} to ${targetChain.chainId}...`
  );

  const sourceRelayer = await getWormholeRelayer(sourceChain);
  const sourceProvider = await sourceRelayer.getDefaultDeliveryProvider();

  const relayQuote = await await sourceRelayer[
    "quoteEVMDeliveryPrice(uint16,uint256,uint256,address)"
  ](targetChain.chainId, 0, 2000000, sourceProvider);
  console.log("relay quote: " + relayQuote);

  const mockIntegration = getMockIntegration(sourceChain);
  const targetAddress = getMockIntegrationAddress(targetChain);

  const message = await mockIntegration.getMessage();
  console.log("got message from integration " + message);

  const sentMessage = "ID: " + String(Math.ceil(Math.random() * 10000));
  console.log(`Sending message: ${sentMessage}`);
  const tx = await mockIntegration.sendMessage(
    Buffer.from(sentMessage),
    targetChain.chainId,
    2000000,
    0,
    {
      gasLimit: 1000000,
      value: relayQuote[0],
    }
  );
  const rx = await tx.wait();
  const sequences = wh.parseSequencesFromLogEth(
    rx,
    sourceChain.wormholeAddress
  );
  console.log("Tx hash: ", rx.transactionHash);
  console.log(`Sequences: ${sequences}`);
  if (fetchSignedVaa) {
    for (let i = 0; i < 120; i++) {
      try {
        const vaa1 = await fetchVaaFromLog(rx.logs[0], sourceChain.chainId);
        console.log(vaa1);
        const vaa2 = await fetchVaaFromLog(rx.logs[1], sourceChain.chainId);
        console.log(vaa2);
        break;
      } catch (e) {
        console.error(`${i} seconds`);
        if (i === 0) {
          console.error(e);
        }
      }
      await new Promise((resolve) => setTimeout(resolve, 1_000));
    }
  }

  if (queryMessageOnTargetFlag) {
    return await queryMessageOnTarget(sentMessage, targetChain);
  }
  console.log("");
}

async function queryMessageOnTarget(
  sentMessage: string,
  targetChain: ChainInfo
): Promise<boolean> {
  let messageHistory: string[] = [];
  const targetIntegration = getMockIntegration(targetChain);

  let notFound = true;
  for (let i = 0; i < 20 && notFound; i++) {
    await new Promise<void>((resolve) => setTimeout(() => resolve(), 2000));
    const messageHistoryResp = await targetIntegration.getMessageHistory();
    messageHistory = messageHistoryResp.map((message) =>
      ethers.utils.toUtf8String(message)
    );
    notFound = !messageHistory
      .slice(messageHistory.length - 20)
      .find((msg) => msg === sentMessage);
    process.stdout.write("..");
  }
  console.log("");
  if (notFound) {
    console.log(`ERROR: Did not receive message!`);
    return false;
  }

  console.log(
    `Received message: ${messageHistory[messageHistory.length - 1][0]}`
  );
  console.log(`Received messageHistory: ${messageHistory.join(", ")}`);
  return true;
}

export async function encodeEmitterAddress(
  myChainId: wh.ChainId,
  emitterAddressStr: string
): Promise<string> {
  if (myChainId === wh.CHAIN_ID_SOLANA || myChainId === wh.CHAIN_ID_PYTHNET) {
    return wh.getEmitterAddressSolana(emitterAddressStr);
  }
  if (wh.isTerraChain(myChainId)) {
    return wh.getEmitterAddressTerra(emitterAddressStr);
  }
  if (wh.isEVMChain(myChainId)) {
    return wh.getEmitterAddressEth(emitterAddressStr);
  }
  throw new Error(`Unrecognized wormhole chainId ${myChainId}`);
}

function fetchVaaFromLog(
  bridgeLog: any,
  chainId: wh.ChainId
): Promise<wh.SignedVaa> {
  throw Error("fetchVAA unimplemented");
  // const iface = Implementation__factory.createInterface();
  // const log = (iface.parseLog(
  //   bridgeLog
  // ) as unknown) as LogMessagePublishedEvent;
  // const sequence = log.args.sequence.toString();
  // const emitter = wh.tryNativeToHexString(log.args.sender, "ethereum");
  // return wh
  //   .getSignedVAA(
  //     "https://wormhole-v2-testnet-api.certus.one",
  //     chainId,
  //     emitter,
  //     sequence,
  //     { transport: grpcWebNodeHttpTransport.NodeHttpTransport() }
  //   )
  //   .then((r) => r.vaaBytes);
}

export async function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms));
}
