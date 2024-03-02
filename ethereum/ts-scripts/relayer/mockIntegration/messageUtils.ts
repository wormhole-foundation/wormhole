import {
  ChainInfo,
  getWormholeRelayer,
  getMockIntegration,
} from "../helpers/env";
import { ethers } from "ethers";

export async function sendMessage(
  sourceChain: ChainInfo,
  targetChain: ChainInfo,
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
  console.log("Tx hash: ", rx.transactionHash);

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

export async function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms));
}
