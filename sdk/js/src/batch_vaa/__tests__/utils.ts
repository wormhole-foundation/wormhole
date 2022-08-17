import {ethers} from "ethers";
import {describe, it} from "@jest/globals";
import {WORMHOLE_MESSAGE_EVENT_ABI} from "./consts";

export async function parseWormholeEventsFromReceipt(
  receipt: ethers.ContractReceipt
): Promise<ethers.utils.LogDescription[]> {
  // create the wormhole message interface
  const wormholeMessageInterface = new ethers.utils.Interface(WORMHOLE_MESSAGE_EVENT_ABI);

  // loop through the logs and parse the events that were emitted
  const logDescriptions: ethers.utils.LogDescription[] = await Promise.all(
    receipt.logs.map(async (log) => {
      return wormholeMessageInterface.parseLog(log);
    })
  );

  return logDescriptions;
}

describe("Utils Should Exist", () => {
  it("Dummy Test", () => {
    return;
  });
});
